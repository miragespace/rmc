package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"net/smtp"
	"os"
	"time"

	"github.com/zllovesuki/rmc/auth"
	"github.com/zllovesuki/rmc/customer"
	"github.com/zllovesuki/rmc/db"
	"github.com/zllovesuki/rmc/instance"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-redis/redis/v7"
	"github.com/joho/godotenv"
	"github.com/stripe/stripe-go/v71"
	"go.uber.org/zap"
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
	defer logger.Sync()

	if err != nil {
		log.Fatalf("Cannot initialize logger: %v\n", err)
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

	// Load configurations from dotFile
	if err := godotenv.Load(dotFile); err != nil {
		logger.Fatal("Cannot load configurations from .env",
			zap.Error(err),
		)
	}

	stripe.Key = os.Getenv("STRIPE_KEY")

	// Initialize backend connections
	db, err := db.New(os.Getenv("POSTGRES_URI"))
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

	rdb.Ping()

	// TODO: Initialize RabbitMQ here

	// Initialize authentication manager
	smtpAuth := smtp.PlainAuth("", os.Getenv("SMTP_USERNAME"), os.Getenv("SMTP_PASSWORD"), os.Getenv("SMTP_HOST"))
	auth, err := auth.New(auth.Options{
		Redis: rdb,

		Environment: authEnvironment,
		SMTPAuth:    smtpAuth,
		From:        os.Getenv("SMTP_FROM"),
		Hostname:    os.Getenv("SMTP_HOST") + ":" + os.Getenv("SMTP_PORT"),
		EmailOption: auth.EmailOption{
			Name: os.Getenv("SITE_NAME"),
			LinkGenerator: func(uid, token string) string {
				return fmt.Sprintf("%s/customer/%s/%s", os.Getenv("SITE_URL"), uid, token)
			},
		},
	})
	if err != nil {
		logger.Error("Cannot initialize AuthManager",
			zap.Error(err),
		)
	}

	// Initialize Managers
	customerManager, err := customer.NewManager(logger, db)
	if err != nil {
		logger.Error("Cannot initialize CustomerManager",
			zap.Error(err),
		)
	}

	instanceManager, err := instance.NewManager(logger, db)
	if err != nil {
		logger.Error("Cannot initialize InstanceManager",
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
		logger.Error("Cannot initialize Customer Service Router",
			zap.Error(err),
		)
	}

	instanceRouter, err := instance.NewService(instance.Options{
		Auth:            auth,
		InstanceManager: instanceManager,
		Logger:          logger,
	})
	if err != nil {
		logger.Error("Cannot initialize Instance Service Router",
			zap.Error(err),
		)
	}

	// Initialize http/middlewares
	r := chi.NewRouter()

	sentryMiddleware := sentryhttp.New(sentryhttp.Options{
		Repanic: true,
	})

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(sentryMiddleware.Handle)

	r.Mount("/customers", customerRouter.Router())
	r.Mount("/instances", instanceRouter.Router())

	// For application insights
	r.HandleFunc("/pprof/*", pprof.Index)
	r.HandleFunc("/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/pprof/profile", pprof.Profile)
	r.HandleFunc("/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/pprof/trace", pprof.Trace)

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// TODO: redirect user to frontend
		fmt.Fprintf(w, "Hello World!")
	})

	srv := &http.Server{
		Handler: r,
		Addr:    ":42069",
	}

	if err := srv.ListenAndServe(); err != nil {
		logger.Fatal("Unable to listen for request",
			zap.Error(srv.ListenAndServe()),
		)
	}
}
