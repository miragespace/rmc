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

type Manager struct {
	stripeClient *client.API
	db           *gorm.DB
	logger       *zap.Logger
}

func NewManager(logger *zap.Logger, db *gorm.DB, s *client.API) (*Manager, error) {
	if err := db.AutoMigrate(&Subscription{}, &SubscriptionItem{}); err != nil {
		return nil, extErrors.Wrap(err, "Cannot initilize subscription.Manager")
	}
	return &Manager{
		stripeClient: s,
		db:           db,
		logger:       logger,
	}, nil
}

func (m *Manager) Create(ctx context.Context, si *SubscriptionItem) error {
	result := m.db.WithContext(ctx).Create(si)
	if result.Error != nil {
		m.logger.Error("Unable to create new subscription item in database",
			zap.Error(result.Error),
		)
		return extErrors.Wrap(result.Error, "Cannot create subscription item")
	}
	return nil
}

type GetOption struct {
	CustomerID         string
	SubscriptionID     string
	SubscriptionItemID string
}

func (m *Manager) Get(ctx context.Context, opt GetOption) (*Subscription, error) {
	if len(opt.CustomerID) == 0 {
		return nil, fmt.Errorf("CustomerID is required")
	}
	baseQuery := m.db.WithContext(ctx).Where("customer_id = ?", opt.CustomerID)
	if len(opt.SubscriptionID) > 0 {
		baseQuery = baseQuery.Where("subscription_id = ?", opt.SubscriptionID)
	} else if len(opt.SubscriptionItemID) > 0 {
		baseQuery = baseQuery.Where("id = ?", opt.SubscriptionItemID)
	} else {
		return nil, fmt.Errorf("Either SubscriptionCheck.SubscriptionID or SubscriptionCheck.SubscriptionItemID is required")
	}

	var item Subscription

	result := baseQuery.Preload("SubscriptionItems").First(&item)
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
