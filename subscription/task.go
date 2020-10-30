package subscription

import (
	"github.com/zllovesuki/rmc/spec/broker"

	"go.uber.org/zap"
)

type TaskOptions struct {
	SubscriptionManager *Manager
	Consumer            broker.Consumer
	Logger              *zap.Logger
}

type Task struct {
	TaskOptions
}

// This background task will:
// 1. Synchronize database state with Stripe
// 2. Report aggregate usage
