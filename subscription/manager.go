package subscription

import (
	"context"
	"errors"
	"fmt"

	extErrors "github.com/pkg/errors"
	"github.com/stripe/stripe-go/v71/client"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SubscriptionManagerOptions struct {
	StripeClient   *client.API
	DB             *gorm.DB
	Logger         *zap.Logger
	PathToPlanJSON string
}

type Manager struct {
	SubscriptionManagerOptions
	definedPlans map[string]Plan
}

func NewManager(option SubscriptionManagerOptions) (*Manager, error) {
	if option.StripeClient == nil {
		return nil, fmt.Errorf("nil StripeClient is invalid")
	}
	if option.DB == nil {
		return nil, fmt.Errorf("nil DB is invalid")
	}
	if option.Logger == nil {
		return nil, fmt.Errorf("nil Logger is invalid")
	}
	if len(option.PathToPlanJSON) == 0 {
		return nil, fmt.Errorf("empty PathToPlanJSON is invalid")
	}
	if err := option.DB.AutoMigrate(&Subscription{}, &SubscriptionItem{}); err != nil {
		return nil, extErrors.Wrap(err, "Cannot initilize subscription.Manager")
	}

	plans, err := loadPlansFromFile(option.PathToPlanJSON)
	if err != nil {
		return nil, extErrors.Wrap(err, "Cannot populate defined Plans")
	}

	definedPlans := make(map[string]Plan)
	for key, p := range plans {
		if err := p.EnsureExistence(context.Background(), option.StripeClient); err != nil {
			return nil, extErrors.Wrap(err, "Cannot ensure plan existence on Stripe")
		}
		definedPlans[key] = p
	}

	return &Manager{
		SubscriptionManagerOptions: option,
		definedPlans:               definedPlans,
	}, nil
}

func (m *Manager) ListDefinedPlans() map[string]Plan {
	return m.definedPlans
}

func (m *Manager) Create(ctx context.Context, si *SubscriptionItem) error {
	result := m.DB.WithContext(ctx).Create(si)
	if result.Error != nil {
		m.Logger.Error("Unable to create new subscription item in database",
			zap.Error(result.Error),
		)
		return extErrors.Wrap(result.Error, "Cannot create subscription item")
	}
	return nil
}

type GetOption struct {
	CustomerID     string
	SubscriptionID string
}

func (m *Manager) Get(ctx context.Context, opt GetOption) (*Subscription, error) {
	if len(opt.CustomerID) == 0 {
		return nil, fmt.Errorf("CustomerID is required")
	}
	if len(opt.SubscriptionID) == 0 {
		return nil, fmt.Errorf("Either SubscriptionCheck.SubscriptionID is required")
	}
	var item Subscription
	result := m.DB.WithContext(ctx).
		Preload("SubscriptionItems").
		Where("customer_id = ?", opt.CustomerID).
		Where("id = ?", opt.SubscriptionID).
		First(&item)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return &item, nil
	// params := &stripe.SubscriptionParams{
	// 	Params: stripe.Params{
	// 		Context: ctx,
	// 	},
	// }
	// params.AddExpand("customer")
	// params.AddExpand("pending_setup_intent")
	// subcription, err := m.StripeClient.Subscriptions.Get(subscriptionId, params)
	// if err != nil {
	// 	return false, extErrors.Wrap(err, "Cannot get subscription from Stripe")
	// }
	// if subcription == nil {
	// 	return false, nil
	// }
	// if subcription.Customer.ID != customerId {
	// 	return false, nil
	// }
	// if subcription.PendingSetupIntent != nil {
	// 	return false, nil
	// }
	// return true, nil
}

func (m *Manager) CancelSubscription(ctx context.Context, subscriptionId string) error {
	// TODO: report usage
	// TODO: Stripe API to cancel and collect
	return nil
}
