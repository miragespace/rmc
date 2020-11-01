package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/client"
	"github.com/zllovesuki/rmc/auth"
	"github.com/zllovesuki/rmc/broker"
	"github.com/zllovesuki/rmc/host"
	"github.com/zllovesuki/rmc/host/docker"
	"github.com/zllovesuki/rmc/host/worker"

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
			"component": "host",
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

	amqpBroker, err := broker.NewAMQPBroker(logger, os.Getenv("AMQP_URI"))
	if err != nil {
		logger.Fatal("Cannot connect to Broker",
			zap.Error(err),
		)
	}
	defer amqpBroker.Close()

	producer, err := amqpBroker.Producer()
	if err != nil {
		logger.Fatal("Cannot setup broker as producer",
			zap.Error(err),
		)
	}
	defer producer.Close()

	consumer, err := amqpBroker.Consumer()
	if err != nil {
		logger.Fatal("Cannot setup broker as consumer",
			zap.Error(err),
		)
	}
	defer consumer.Close()

	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Fatal("Cannot connect to docker daemon",
			zap.Error(err),
		)
	}

	docker, err := docker.NewClient(docker.Options{
		Client: dockerCli,
		Logger: logger,
	})
	if err != nil {
		logger.Fatal("Cannot initialize internal docker client",
			zap.Error(err),
		)
	}

	// TODO: define it from env variables
	// TODO: get server IP
	if len(os.Getenv("HOST_NAME")) == 0 {
		log.Fatal("Host Name must be specified")
	}
	currentHost := host.Host{
		Name:     os.Getenv("HOST_NAME"),
		Capacity: 20,
	}

	controller, err := worker.NewController(worker.Options{
		Docker:   docker,
		Logger:   logger,
		Producer: producer,
		Consumer: consumer,
		Host:     currentHost,
		HostIP:   "127.0.0.1",
	})
	if err != nil {
		logger.Fatal("Cannot initialize Controller",
			zap.Error(err),
		)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	controller.Run(ctx)

	<-c
	cancel()
}
