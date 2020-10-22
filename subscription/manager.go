package subscription

import (
	extErrors "github.com/pkg/errors"
	"github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/sub"
	"go.uber.org/zap"
)

type Manager struct {
	logger *zap.Logger
}

func NewManager(logger *zap.Logger) (*Manager, error) {
	return &Manager{
		logger: logger,
	}, nil
}

func (m *Manager) ValidSubscription(customerId, subscriptionId string) (bool, error) {
	params := &stripe.SubscriptionParams{}
	params.AddExpand("customer")
	params.AddExpand("pending_setup_intent")
	subcription, err := sub.Get(subscriptionId, params)
	if err != nil {
		return false, extErrors.Wrap(err, "Cannot get subscription from Stripe")
	}
	if subcription == nil {
		return false, nil
	}
	if subcription.Customer.ID != customerId {
		return false, nil
	}
	if subcription.PendingSetupIntent != nil {
		return false, nil
	}
	return true, nil
}

func (m *Manager) CancelSubscription(subscriptionId string) error {
	// TODO: report usage
	// TODO: Stripe API to cancel and collect
	panic("not implemented")
}
