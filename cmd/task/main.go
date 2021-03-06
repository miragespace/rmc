package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/miragespace/rmc/auth"
	"github.com/miragespace/rmc/broker"
	"github.com/miragespace/rmc/db"
	"github.com/miragespace/rmc/external"
	"github.com/miragespace/rmc/host"
	"github.com/miragespace/rmc/instance"
	"github.com/miragespace/rmc/subscription"

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
			"component": "task",
		},
	}
	core, err := zapsentry.NewCore(cfg, zapsentry.NewSentryClientFromClient(sentry.CurrentHub().Client()))
	logger = zapsentry.AttachCoreToLogger(core, logger)

	defer logger.Sync()

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

	amqpBroker, err := broker.NewAMQPBroker(logger, os.Getenv("AMQP_URI"))
	if err != nil {
		log.Fatal("Cannot connect to Broker",
			zap.Error(err),
		)
	}
	defer amqpBroker.Close()

	subscriptionProducer, err := amqpBroker.Producer()
	if err != nil {
		logger.Fatal("Cannot setup producer for subscription",
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

	hostManager, err := host.NewManager(logger, db)
	if err != nil {
		logger.Fatal("Cannot initialize HostManager",
			zap.Error(err),
		)
	}

	instanceConsumer, err := amqpBroker.Consumer()
	if err != nil {
		logger.Fatal("Cannot setup consumer for instance",
			zap.Error(err),
		)
	}
	defer instanceConsumer.Close()

	instanceTask, err := instance.NewTask(instance.TaskOptions{
		InstanceManager:     instanceManager,
		SubscriptionManager: subscriptionManager,
		Consumer:            instanceConsumer,
		Logger:              logger,
	})
	if err != nil {
		logger.Fatal("Cannot get instance task",
			zap.Error(err),
		)
	}

	hostConsumer, err := amqpBroker.Consumer()
	if err != nil {
		logger.Fatal("Cannot setup consumer for host",
			zap.Error(err),
		)
	}
	defer hostConsumer.Close()

	hostTask, err := host.NewTask(host.TaskOptions{
		HostManager: hostManager,
		Consumer:    hostConsumer,
		Logger:      logger,
	})
	if err != nil {
		logger.Fatal("Cannot get host task",
			zap.Error(err),
		)
	}

	subscriptionConsumer, err := amqpBroker.Consumer()
	if err != nil {
		logger.Fatal("Cannot setup consumer for subscription",
			zap.Error(err),
		)
	}
	defer subscriptionConsumer.Close()

	subscriptionTask, err := subscription.NewTask(subscription.TaskOptions{
		StripeClient:        stripeClient,
		Consumer:            subscriptionConsumer,
		SubscriptionManager: subscriptionManager,
		Logger:              logger,
	})
	if err != nil {
		logger.Fatal("Cannot get subscription task",
			zap.Error(err),
		)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := instanceTask.HandleReply(ctx); err != nil {
		logger.Fatal("Cannot handle instance replies",
			zap.Error(err),
		)
	}
	if err := hostTask.HandleReply(ctx); err != nil {
		logger.Fatal("Cannot handle host replies",
			zap.Error(err),
		)
	}

	if err := subscriptionTask.HandleTask(ctx); err != nil {
		logger.Fatal("Cannot handle async task",
			zap.Error(err),
		)
	}

	logger.Info("API task started")

	<-c
	cancel()

}
