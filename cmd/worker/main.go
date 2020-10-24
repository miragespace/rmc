package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zllovesuki/rmc/auth"
	"github.com/zllovesuki/rmc/broker"
	"github.com/zllovesuki/rmc/host"
	"github.com/zllovesuki/rmc/spec"
	"github.com/zllovesuki/rmc/spec/protocol"

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

	// TESTING
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	crChan, err := amqpBroker.ReceiveControlRequest(ctx, &host.Host{
		Name: "test",
	})
	if err != nil {
		log.Fatal("Cannot get message channel",
			zap.Error(err),
		)
	}

	prChan, err := amqpBroker.ReceiveProvisionRequest(ctx, &host.Host{
		Name: "test",
	})
	if err != nil {
		log.Fatal("Cannot get message channel",
			zap.Error(err),
		)
	}

	go func() {
		tick := time.Tick(spec.HeartbeatInterval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick:
				amqpBroker.SendHeartbeart(&protocol.Heartbeat{
					Host: &protocol.Host{
						Name:     "test",
						Running:  0,
						Stopped:  0,
						Capacity: 20,
					},
				})
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d := <-crChan:
				fmt.Println(d)
				if err := amqpBroker.SendControlReply(&protocol.ControlReply{
					Instance: &protocol.Instance{
						InstanceID: "test-instance",
					},
					Result: protocol.ControlReply_SUCCESS,
				}); err != nil {
					logger.Error("Cannot send control reply",
						zap.Error(err),
					)
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d := <-prChan:
				fmt.Println(d)
				if err := amqpBroker.SendProvisionReply(&protocol.ProvisionReply{
					Instance: &protocol.Instance{
						InstanceID: "test-instance",
					},
					Result: protocol.ProvisionReply_SUCCESS,
				}); err != nil {
					logger.Error("Cannot send provision reply",
						zap.Error(err),
					)
				}
			}
		}
	}()

	logger.Info("Heartbeat interval: " + spec.HeartbeatInterval.String())

	<-c
	cancel()
}
