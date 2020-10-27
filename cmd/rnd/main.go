package main

import (
	"log"
	"os"
	"time"

	"github.com/TheZeroSlave/zapsentry"
	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
	"github.com/zllovesuki/rmc/auth"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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

	// Initialize sentry for error reporting
	if err := sentry.Init(sentry.ClientOptions{
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

	// Load configurations from dotFile
	if err := godotenv.Load(dotFile); err != nil {
		logger.Fatal("Cannot load configurations from .env",
			zap.Error(err),
		)
	}

	// a Stripe client will allow us to mock testing
	// stripeClient := external.NewStripeClient(os.Getenv("STRIPE_KEY"))

	// testPlan := subscription.Plans["Spicy"]

	// ctx := context.Background()

	// fmt.Printf("%+v\n", testPlan)
	// testPlan.EnsureExistence(ctx, stripeClient)
	// fmt.Printf("%+v\n", testPlan)
	// testPlan.EnsureExistence(ctx, stripeClient)
	// fmt.Printf("%+v\n", testPlan)
}
