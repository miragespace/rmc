package subscription

import (
	"context"
	"fmt"

	extErrors "github.com/pkg/errors"
	"github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/client"
	"github.com/zllovesuki/rmc/auth"
	"go.uber.org/zap"
)

type ManagerOptions struct {
	Auth         *auth.Auth
	StripeClient *client.API
	Logger       *zap.Logger
}

type Manager struct {
	ManagerOptions
}

func NewManager(option ManagerOptions) (*Manager, error) {
	if option.Auth == nil {
		return nil, fmt.Errorf("nil AuthManager is invalid")
	}
	if option.StripeClient == nil {
		return nil, fmt.Errorf("nil StripeClient is invalid")
	}
	if option.Logger == nil {
		return nil, fmt.Errorf("nil Logger is invalid")
	}
	return &Manager{
		ManagerOptions: option,
	}, nil
}

func (m *Manager) ValidSubscription(ctx context.Context, customerId, subscriptionId string) (bool, error) {
	if m.Auth.Environment == auth.EnvDevelopment {
		return true, nil
	}

	params := &stripe.SubscriptionParams{
		Params: stripe.Params{
			Context: ctx,
		},
	}
	params.AddExpand("customer")
	params.AddExpand("pending_setup_intent")
	subcription, err := m.StripeClient.Subscriptions.Get(subscriptionId, params)
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
	if m.Auth.Environment == auth.EnvDevelopment {
		return nil
	}
	// TODO: report usage
	// TODO: Stripe API to cancel and collect
	panic("not implemented")
}
