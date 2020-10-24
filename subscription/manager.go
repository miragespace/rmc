package subscription

import (
	"context"

	extErrors "github.com/pkg/errors"
	"github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/client"
	"go.uber.org/zap"
)

type Manager struct {
	stripeClient *client.API
	logger       *zap.Logger
}

func NewManager(logger *zap.Logger, s *client.API) (*Manager, error) {
	return &Manager{
		stripeClient: s,
		logger:       logger,
	}, nil
}

func (m *Manager) ValidSubscription(ctx context.Context, customerId, subscriptionId string) (bool, error) {
	params := &stripe.SubscriptionParams{
		Params: stripe.Params{
			Context: ctx,
		},
	}
	params.AddExpand("customer")
	params.AddExpand("pending_setup_intent")
	subcription, err := m.stripeClient.Subscriptions.Get(subscriptionId, params)
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

func (m *Manager) CancelSubscription(ctx context.Context, subscriptionId string) error {
	// TODO: report usage
	// TODO: Stripe API to cancel and collect
	panic("not implemented")
}
