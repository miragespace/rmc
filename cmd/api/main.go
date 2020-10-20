package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"net/smtp"
	"os"

	"github.com/zllovesuki/rmc/auth"
	"github.com/zllovesuki/rmc/customer"
	"github.com/zllovesuki/rmc/db"
	"github.com/zllovesuki/rmc/instance"

	"github.com/go-chi/chi"
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

	if err := godotenv.Load(dotFile); err != nil {
		logger.Fatal("Cannot load configurations from .env",
			zap.Error(err),
		)
	}

	stripe.Key = os.Getenv("STRIPE_KEY")

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

	customerManager, err := customer.NewManager(logger, db)
	if err != nil {
		logger.Error("Cannot initialize CustomerManager",
			zap.Error(err),
		)
	}

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

	instanceManager, err := instance.NewManager(logger, db)
	if err != nil {
		logger.Error("Cannot initialize InstanceManager",
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

	rootRouter := chi.NewRouter()

	rootRouter.Mount("/customers", customerRouter.Router())
	rootRouter.Mount("/instances", instanceRouter.Router())

	rootRouter.HandleFunc("/pprof/*", pprof.Index)
	rootRouter.HandleFunc("/pprof/cmdline", pprof.Cmdline)
	rootRouter.HandleFunc("/pprof/profile", pprof.Profile)
	rootRouter.HandleFunc("/pprof/symbol", pprof.Symbol)
	rootRouter.HandleFunc("/pprof/trace", pprof.Trace)

	rootRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// TODO: redirect user to frontend
		fmt.Fprintf(w, "Hello World!")
	})

	srv := &http.Server{
		Handler: rootRouter,
		Addr:    ":42069",
	}

	log.Fatalln(srv.ListenAndServe())

}
