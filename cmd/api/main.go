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

	"github.com/zllovesuki/rmc/auth"
	"github.com/zllovesuki/rmc/broker"
	"github.com/zllovesuki/rmc/customer"
	"github.com/zllovesuki/rmc/db"
	"github.com/zllovesuki/rmc/external"
	"github.com/zllovesuki/rmc/host"
	"github.com/zllovesuki/rmc/instance"
	"github.com/zllovesuki/rmc/subscription"

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
	env := os.Getenv("API_ENV")
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
	defer logger.Sync()

	// Load configurations from dotFile
	if err := godotenv.Load(dotFile); err != nil {
		logger.Fatal("Cannot load configurations from .env",
			zap.Error(err),
		)
	}

	// Initialize sentry for error reporting
	if err := sentry.Init(sentry.ClientOptions{
		Environment: string(authEnvironment),
		Debug:       authEnvironment == auth.EnvDevelopment,
	}); err != nil {
		logger.Fatal("Cannot initialize sentry",
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

	// a Stripe client will allow us to mock testing
	stripeClient := external.NewStripeClient(os.Getenv("STRIPE_KEY"))

	// Initialize backend connections
	db, err := db.New(logger, os.Getenv("POSTGRES_URI"))
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

	amqpBroker, err := broker.NewAMQPBroker(os.Getenv("AMQP_URI"))
	if err != nil {
		log.Fatal("Cannot connect to Broker",
			zap.Error(err),
		)
	}
	defer amqpBroker.Close()

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
				return fmt.Sprintf("%s/customers/%s/%s", os.Getenv("SITE_URL"), uid, token)
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

	instanceManager, err := instance.NewManager(logger, db)
	if err != nil {
		logger.Fatal("Cannot initialize InstanceManager",
			zap.Error(err),
		)
	}

	hostManager, err := host.NewManager(logger, db)
	if err != nil {
		logger.Fatal("Cannot initialize HostManager",
			zap.Error(err),
		)
	}

	subscriptionManager, err := subscription.NewManager(logger, stripeClient)
	if err != nil {
		logger.Fatal("Cannot initialize SubscriptionManager",
			zap.Error(err),
		)
	}

	// Initialize servce routers
	customerRouter, err := customer.NewService(customer.Options{
		Auth:            auth,
		CustomerManager: customerManager,
		Logger:          logger,
	})
	if err != nil {
		logger.Fatal("Cannot initialize Customer Service Router",
			zap.Error(err),
		)
	}

	instanceRouter, err := instance.NewService(instance.Options{
		SubscriptionManager: subscriptionManager,
		HostManager:         hostManager,
		InstanceManager:     instanceManager,
		Producer:            amqpBroker,
		Logger:              logger,
	})
	if err != nil {
		logger.Fatal("Cannot initialize Instance Service Router",
			zap.Error(err),
		)
	}

	subscriptionRouter, err := subscription.NewService(subscription.Options{
		Logger: logger,
	})
	if err != nil {
		logger.Fatal("Cannot initialize Subscription Service Router",
			zap.Error(err),
		)
	}

	// Initialize http/middlewares
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Mount("/customers", customerRouter.Router())

	authMiddleware := chi.Chain(auth.Middleware())
	authenticated := r.With(authMiddleware...)

	authenticated.Mount("/instances", instanceRouter.Router())
	authenticated.Mount("/subscriptions", subscriptionRouter.Router())

	// For application insights
	r.Mount("/debug", middleware.Profiler())

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// TODO: redirect user to frontend
		fmt.Fprintf(w, "Hello World!")
	})

	srv := &http.Server{
		Handler: r,
		Addr:    ":42069",
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Unable to listen for request",
				zap.Error(srv.ListenAndServe()),
			)
		}
	}()
	logger.Info("API Started")

	<-c
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server Shutdown Failed", zap.Error(err))
	}
}
