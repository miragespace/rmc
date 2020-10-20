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
)

func main() {
	var authEnvironment auth.Environment
	env := os.Getenv("API_ENV")
	if "" == env {
		env = "development"
		authEnvironment = auth.EnvDevelopment
	} else {
		authEnvironment = auth.EnvProduction
	}

	if err := godotenv.Load(".env." + env); err != nil {
		panic(err)
	}

	stripe.Key = os.Getenv("STRIPE_KEY")

	db, err := db.New(os.Getenv("POSTGRES_URI"))
	if err != nil {
		panic(err)
	}

	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    []string{os.Getenv("REDIS_URI")},
		Password: os.Getenv("REDIS_PW"),
		DB:       0,
	})
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
		panic(err)
	}

	customerManager, err := customer.NewManager(db)
	if err != nil {
		panic(err)
	}

	instanceManager, err := instance.NewManager(db)
	if err != nil {
		panic(err)
	}

	log.Println(instanceManager)

	customerRouter, err := customer.NewService(customer.Options{
		Auth:            auth,
		CustomerManager: customerManager,
	})

	rootRouter := chi.NewRouter()

	rootRouter.Mount("/customer", customerRouter.Router())

	rootRouter.HandleFunc("/pprof/*", pprof.Index)
	rootRouter.HandleFunc("/pprof/cmdline", pprof.Cmdline)
	rootRouter.HandleFunc("/pprof/profile", pprof.Profile)
	rootRouter.HandleFunc("/pprof/symbol", pprof.Symbol)
	rootRouter.HandleFunc("/pprof/trace", pprof.Trace)

	rootRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello World!")
	})

	srv := &http.Server{
		Handler: rootRouter,
		Addr:    ":42069",
	}

	log.Fatalln(srv.ListenAndServe())

}
