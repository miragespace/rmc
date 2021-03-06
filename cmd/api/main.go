package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/miragespace/rmc/auth"
	"github.com/miragespace/rmc/broker"
	"github.com/miragespace/rmc/customer"
	"github.com/miragespace/rmc/db"
	"github.com/miragespace/rmc/external"
	"github.com/miragespace/rmc/host"
	"github.com/miragespace/rmc/instance"
	"github.com/miragespace/rmc/response"
	"github.com/miragespace/rmc/subscription"
	"github.com/miragespace/rmc/util"

	"github.com/TheZeroSlave/zapsentry"
	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-redis/redis/v7"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Build-time injected variables
var (
	Version = ""
)

func main() {
	var logger *zap.Logger
	var authEnvironment auth.Environment
	var dotFile string
	var err error

	// Determine running environment and initialize structural logger
	env := os.Getenv("ENV")
	if "production" == env {
		dotFile = ".env.production"
		authEnvironment = auth.EnvProduction
		logger, err = zap.NewProduction()
	} else {
		dotFile = ".env.development"
		authEnvironment = auth.EnvDevelopment
		logger, err = zap.NewDevelopment()
	}

	if err != nil {
		log.Fatalf("Cannot initialize logger: %v\n", err)
	}
	logger = logger.With(zap.String("Version", Version))

	// Load configurations from dotFile
	if err := godotenv.Load(dotFile); err != nil {
		logger.Fatal("Cannot load configurations from .env",
			zap.Error(err),
		)
	}

	// Initialize sentry for error reporting
	if err := sentry.Init(sentry.ClientOptions{
		Release:     Version,
		Environment: string(authEnvironment),
		Debug:       authEnvironment == auth.EnvDevelopment,
	}); err != nil {
		log.Fatal("Cannot initialize sentry",
			zap.Error(err),
		)
	}
	defer sentry.Flush(time.Second * 2)

	// Attach sentry to zap so we can do automatic error capturing
	cfg := zapsentry.Configuration{
		Level: zapcore.ErrorLevel,
		Tags: map[string]string{
			"component": "api",
		},
	}
	core, err := zapsentry.NewCore(cfg, zapsentry.NewSentryClientFromClient(sentry.CurrentHub().Client()))
	logger = zapsentry.AttachCoreToLogger(core, logger)

	defer logger.Sync()

	// a Stripe client will allow us to mock testing
	stripeClient := external.NewStripeClient(os.Getenv("STRIPE_KEY"))

	// Initialize backend connections
	db, err := db.New(db.Options{
		URI:    os.Getenv("POSTGRES_URI"),
		Logger: logger,
	})
	if err != nil {
		logger.Fatal("Cannot connect to Postgres",
			zap.Error(err),
		)
	}

	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    []string{os.Getenv("REDIS_URI")},
		Password: os.Getenv("REDIS_PW"),
		DB:       0,
	})
	if _, err := rdb.Ping().Result(); err != nil {
		logger.Fatal("Cannot connect to Redis",
			zap.Error(err),
		)
	}
	defer rdb.Close()

	if status := rdb.Ping(); status.Err() != nil {
		logger.Fatal("Cannot ping Redis",
			zap.Error(status.Err()),
		)
	}

	amqpBroker, err := broker.NewAMQPBroker(logger, os.Getenv("AMQP_URI"))
	if err != nil {
		logger.Fatal("Cannot connect to Broker",
			zap.Error(err),
		)
	}
	defer amqpBroker.Close()

	// frontendOrigin for email and API CORS
	frontendOrigin := os.Getenv("SITE_URL")
	if len(frontendOrigin) == 0 {
		logger.Fatal("SITE_URL must be set. For example: https://mysite.com (without trailing slash)")
	}

	// Initialize authentication manager
	smtpAuth := smtp.PlainAuth("", os.Getenv("SMTP_USERNAME"), os.Getenv("SMTP_PASSWORD"), os.Getenv("SMTP_HOST"))
	auth, err := auth.New(auth.Options{
		Redis:  rdb,
		Logger: logger,

		JWTSigningKey: os.Getenv("JWT_KEY"),

		Environment: authEnvironment,
		SMTPAuth:    smtpAuth,
		From:        os.Getenv("SMTP_FROM"),
		Hostname:    os.Getenv("SMTP_HOST") + ":" + os.Getenv("SMTP_PORT"),
		EmailOption: auth.EmailOption{
			Name: os.Getenv("SITE_NAME"),
			LinkGenerator: func(uid, token string) string {
				return fmt.Sprintf("%s/login/%s/%s", frontendOrigin, uid, token)
			},
		},
	})
	if err != nil {
		logger.Fatal("Cannot initialize AuthManager",
			zap.Error(err),
		)
	}

	// Initialize Managers
	customerManager, err := customer.NewManager(logger, db, stripeClient)
	if err != nil {
		logger.Fatal("Cannot initialize CustomerManager",
			zap.Error(err),
		)
	}

	hostManager, err := host.NewManager(logger, db)
	if err != nil {
		logger.Fatal("Cannot initialize HostManager",
			zap.Error(err),
		)
	}

	subscriptionProducer, err := amqpBroker.Producer()
	if err != nil {
		logger.Fatal("Cannot setup broker as producer",
			zap.Error(err),
		)
	}
	defer subscriptionProducer.Close()
	subscriptionManager, err := subscription.NewManager(subscription.ManagerOptions{
		StripeClient: stripeClient,
		Producer:     subscriptionProducer,
		DB:           db,
		Logger:       logger,
	})
	if err != nil {
		logger.Fatal("Cannot initialize SubscriptionManager",
			zap.Error(err),
		)
	}

	instanceManager, err := instance.NewManager(instance.ManagerOptions{
		DB:     db,
		Logger: logger,
	})
	if err != nil {
		logger.Fatal("Cannot initialize InstanceManager",
			zap.Error(err),
		)
	}

	// Initialize servce routers
	customerRouter, err := customer.NewService(customer.ServiceOptions{
		Auth:            auth,
		CustomerManager: customerManager,
		Logger:          logger,
	})
	if err != nil {
		logger.Fatal("Cannot initialize Customer Service Router",
			zap.Error(err),
		)
	}

	instanceProducer, err := amqpBroker.Producer()
	if err != nil {
		logger.Fatal("Cannot setup broker as producer",
			zap.Error(err),
		)
	}
	defer instanceProducer.Close()

	instanceLifecycleManager, err := instance.NewLifecycleManager(instance.LifecycleManagerOption{
		Producer: instanceProducer,
	})

	instanceRouter, err := instance.NewService(instance.ServiceOptions{
		SubscriptionManager: subscriptionManager,
		HostManager:         hostManager,
		InstanceManager:     instanceManager,
		LifecycleManager:    instanceLifecycleManager,
		Logger:              logger,
	})
	if err != nil {
		logger.Fatal("Cannot initialize Instance Service Router",
			zap.Error(err),
		)
	}

	subscriptionRouter, err := subscription.NewService(subscription.ServiceOptions{
		SubscriptionManager: subscriptionManager,
		Logger:              logger,
	})
	if err != nil {
		logger.Fatal("Cannot initialize Subscription Service Router",
			zap.Error(err),
		)
	}

	hostRouter, err := host.NewService(host.ServiceOptions{
		HostManager: hostManager,
		Logger:      logger,
	})
	if err != nil {
		logger.Fatal("Cannot initialize Host Service Router",
			zap.Error(err),
		)
	}

	// Initialize http/middlewares
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{frontendOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(util.Recovery(logger))

	r.Mount("/auth", customerRouter.AuthRouter())

	authMiddleware := chi.Chain(auth.Middleware(), auth.ClaimCheck())
	authenticated := r.With(authMiddleware...)

	authenticated.Mount("/customers", customerRouter.Router())
	authenticated.Mount("/instances", instanceRouter.Router())
	authenticated.Mount("/subscriptions", subscriptionRouter.Router())
	authenticated.Mount("/hosts", hostRouter.Router())

	// internal router listens to a different port
	internal := chi.NewRouter()
	internal.Mount("/instances", instanceRouter.AdminRouter())
	internal.Mount("/subscriptions", subscriptionRouter.AdminRouter())
	internal.Mount("/debug", middleware.Profiler())

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, os.Getenv("SITE_URL"), 302)
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		response.WriteError(w, r, response.ErrMethodNotAllowed())
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		response.WriteError(w, r, response.ErrNotFound())
	})

	srv := &http.Server{
		Handler: r,
		Addr:    ":42069",
	}

	internalSrv := &http.Server{
		Handler: internal,
		Addr:    ":8888",
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Unable to listen for request",
				zap.Error(err),
			)
		}
	}()

	go func() {
		if err := internalSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Unable to listen for internal routes",
				zap.Error(err),
			)
		}
	}()

	logger.Info("API Started")

	<-c
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server Shutdown Failed", zap.Error(err))
	}
	if err := internalSrv.Shutdown(ctx); err != nil {
		logger.Fatal("Internal Router Shutdown Failed", zap.Error(err))
	}
}
