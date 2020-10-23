package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/zllovesuki/rmc/auth"
	"github.com/zllovesuki/rmc/broker"
	"github.com/zllovesuki/rmc/host"

	"github.com/TheZeroSlave/zapsentry"
	"github.com/getsentry/sentry-go"
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
			"component": "host",
		},
	}
	core, err := zapsentry.NewCore(cfg, zapsentry.NewSentryClientFromClient(sentry.CurrentHub().Client()))
	logger = zapsentry.AttachCoreToLogger(core, logger)

	amqpBroker, err := broker.NewAMQPBroker(os.Getenv("AMQP_URI"))
	if err != nil {
		log.Fatal("Cannot connect to Broker",
			zap.Error(err),
		)
	}
	defer amqpBroker.Close()

	ctx := context.Background()

	msgChan, err := amqpBroker.ReceiveControlRequest(ctx, &host.Host{
		Name: "test",
	})
	if err != nil {
		log.Fatal("Cannot get message channel",
			zap.Error(err),
		)
	}

	for {
		select {
		case d := <-msgChan:
			fmt.Println(d)
		}
	}
}
