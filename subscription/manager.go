package subscription

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zllovesuki/rmc/spec"
	"github.com/zllovesuki/rmc/spec/broker"
	"github.com/zllovesuki/rmc/spec/protocol"

	"github.com/golang/protobuf/ptypes"
	"github.com/jackc/pgconn"
	extErrors "github.com/pkg/errors"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/client"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ManagerOptions is used to setup SubscriptionManager's dependencies
type ManagerOptions struct {
	StripeClient *client.API
	Producer     broker.Producer
	DB           *gorm.DB
	Logger       *zap.Logger
}

// Manager struct is used to manage Subscriptions and Plans
type Manager struct {
	ManagerOptions
}

// NewManager returns a new SubscriptionManager
func NewManager(option ManagerOptions) (*Manager, error) {
	if option.StripeClient == nil {
		return nil, fmt.Errorf("nil StripeClient is invalid")
	}
	if option.Producer == nil {
		return nil, fmt.Errorf("nil Producer is invalid")
	}
	if option.DB == nil {
		return nil, fmt.Errorf("nil DB is invalid")
	}
	if option.Logger == nil {
		return nil, fmt.Errorf("nil Logger is invalid")
	}
	if err := option.DB.AutoMigrate(&Plan{}, &Part{}, &Subscription{}, &SubscriptionItem{}, &Usage{}); err != nil {
		return nil, extErrors.Wrap(err, "Cannot initilize subscription.Manager")
	}

	return &Manager{
		ManagerOptions: option,
	}, nil
}

// Create inserts a new Subscription record in the database
func (m *Manager) Create(ctx context.Context, si *Subscription) error {
	result := m.DB.WithContext(ctx).Omit("Plan", "SubscriptionItems.Part").Create(si)
	if result.Error != nil {
		m.Logger.Error("Unable to create new subscription in database",
			zap.Error(result.Error),
		)
		return extErrors.Wrap(result.Error, "Cannot create subscription")
	}
	return nil
}

// ListOption specifies the parameters for subscription listing
type ListOption struct {
	CustomerID string
	Before     time.Time
	Limit      int
}

// List will return a list of subscriptions for a customer
func (m *Manager) List(ctx context.Context, opt ListOption) ([]Subscription, error) {
	baseQuery := m.DB.WithContext(ctx).
		Preload("SubscriptionItems").
		Preload("SubscriptionItems.Part").
		Preload("Plan").
		Preload("Plan.Parts").
		Order("created_at desc")
	if len(opt.CustomerID) > 0 {
		baseQuery = baseQuery.Where("customer_id = ?", opt.CustomerID)
	} else {
		return nil, fmt.Errorf("CustomerID is required")
	}
	if opt.Limit > 0 {
		baseQuery = baseQuery.Limit(opt.Limit)
	}
	if !opt.Before.IsZero() {
		baseQuery = baseQuery.Where("created_at < ?", opt.Before)
	}

	results := make([]Subscription, 0, 1)
	result := baseQuery.Find(&results)

	if result.Error != nil {
		m.Logger.Error("Database returned error",
			zap.Error(result.Error),
		)
		return nil, result.Error
	}
	return results, nil
}

// GetOption specifies the paremeters for getting a single Subscription
type GetOption struct {
	CustomerID     string
	SubscriptionID string
}

// Get will return a subscription for a customer
func (m *Manager) Get(ctx context.Context, opt GetOption) (*Subscription, error) {
	if len(opt.CustomerID) == 0 {
		return nil, fmt.Errorf("CustomerID is required")
	}
	if len(opt.SubscriptionID) == 0 {
		return nil, fmt.Errorf("SubscriptionID is required")
	}
	var sub Subscription
	result := m.DB.WithContext(ctx).
		Preload("SubscriptionItems").
		Preload("SubscriptionItems.Part").
		Preload("Plan").
		Preload("Plan.Parts").
		Where("customer_id = ?", opt.CustomerID).
		Where("id = ?", opt.SubscriptionID).
		First(&sub)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return &sub, nil
}

func (m *Manager) GetStripe(ctx context.Context, opt GetOption) (*stripe.Subscription, error) {
	if len(opt.SubscriptionID) == 0 {
		return nil, fmt.Errorf("SubscriptionID is required")
	}

	return m.StripeClient.Subscriptions.Get(opt.SubscriptionID, nil)
}

// GetUsage will return the aggregate total usage for a single subscription
func (m *Manager) GetUsage(ctx context.Context, opt GetOption) ([]Usage, error) {
	if len(opt.CustomerID) == 0 {
		return nil, fmt.Errorf("CustomerID is required")
	}
	if len(opt.SubscriptionID) == 0 {
		return nil, fmt.Errorf("SubscriptionID is required")
	}
	var sub Subscription
	result := m.DB.WithContext(ctx).
		Preload("SubscriptionItems").
		Where("customer_id = ?", opt.CustomerID).
		Where("id = ?", opt.SubscriptionID).
		First(&sub)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	subItemsID := make([]string, 0, 2)
	for _, item := range sub.SubscriptionItems {
		subItemsID = append(subItemsID, item.ID)
	}

	usages := make([]Usage, 0, 2)
	result = m.DB.WithContext(ctx).
		Clauses(clause.Locking{Strength: "SHARE"}).
		Preload("SubscriptionItem").
		Preload("SubscriptionItem.Part").
		Order("usages.end_date desc").
		Where("subscription_item_id IN ?", subItemsID).
		Find(&usages)

	if result.Error != nil {
		m.Logger.Error("Database returned error",
			zap.Error(result.Error),
		)
		return nil, result.Error
	}
	return usages, nil
}

// AttachPaymentOptions is used for AttachmentPayment
type AttachPaymentOptions struct {
	CustomerID      string
	PaymentMethodID string
}

// AttachPayment will attach a default payment method to a customer
func (m *Manager) AttachPayment(ctx context.Context, opt AttachPaymentOptions) error {
	if len(opt.CustomerID) == 0 {
		return fmt.Errorf("AttachPaymentOptions.CustomerID is required")
	}
	if len(opt.PaymentMethodID) == 0 {
		return fmt.Errorf("AttachPaymentOptions.PaymentMethodID is required")
	}
	params := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(opt.CustomerID),
	}
	pm, err := m.StripeClient.PaymentMethods.Attach(
		opt.PaymentMethodID,
		params,
	)
	if err != nil {
		return err
	}

	customerParams := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(pm.ID),
		},
	}
	if _, err := m.StripeClient.Customers.Update(
		opt.CustomerID,
		customerParams,
	); err != nil {
		return err
	}

	return nil
}

// CreateFromPlanOption is used to specify parameters for creating a subscription
type CreateFromPlanOption struct {
	CustomerID string
	Plan       Plan
}

// CreateSubscriptionFromPlan will create a subscription with Stripe from an existing Plan
func (m *Manager) CreateSubscriptionFromPlan(ctx context.Context, opt CreateFromPlanOption) (*stripe.Subscription, error) {
	if len(opt.CustomerID) == 0 {
		return nil, fmt.Errorf("SetupOptions.CustomerID is required")
	}
	if len(opt.Plan.ID) == 0 {
		return nil, fmt.Errorf("SetupOptions.Plan needs to be a synchronized Plan")
	}

	subscriptionParams := opt.Plan.toStripeSubscriptionParams(ctx, opt.CustomerID)
	subscriptionParams.AddExpand("latest_invoice.payment_intent")
	subscriptionParams.AddExpand("pending_setup_intent")

	sub, err := m.StripeClient.Subscriptions.New(subscriptionParams)

	if err != nil {
		return nil, err
	}

	return sub, nil
}

func (m *Manager) synchronizeSubscriptionPeriod(ctx context.Context, subscriptionID string) error {
	subscriptionParams := &stripe.SubscriptionParams{
		Params: stripe.Params{
			Context: ctx,
		},
	}
	sub, err := m.StripeClient.Subscriptions.Get(subscriptionID, subscriptionParams)
	if err != nil {
		return extErrors.Wrap(err, "Unable to fetch from Stripe to synchronize billing period")
	}
	fmt.Printf("%+v\n", sub)
	result := m.DB.WithContext(ctx).
		Model(&Subscription{}).
		Where("id = ?", subscriptionID).
		Updates(map[string]interface{}{
			"period_start": time.Unix(sub.CurrentPeriodStart, 0),
			"period_end":   time.Unix(sub.CurrentPeriodEnd, 0),
		})
	if result.Error != nil {
		return extErrors.Wrap(result.Error, "Unable to update subscription billing period in database")
	}
	return nil
}

func (m *Manager) synchronizeSubscriptionStatus(ctx context.Context, subscriptionID string) error {
	subscriptionParams := &stripe.SubscriptionParams{
		Params: stripe.Params{
			Context: ctx,
		},
	}
	subscriptionParams.AddExpand("latest_invoice.payment_intent")
	subscriptionParams.AddExpand("pending_setup_intent")
	sub, err := m.StripeClient.Subscriptions.Get(subscriptionID, subscriptionParams)
	if err != nil {
		return extErrors.Wrap(err, "Unable to fetch from Stripe to synchronize status")
	}
	// TODO: also synchronize cancelled/overdue
	if sub.Status == stripe.SubscriptionStatusActive && sub.PendingSetupIntent == nil {
		result := m.DB.WithContext(ctx).
			Model(&Subscription{}).
			Where("id = ?", subscriptionID).
			Update("state", StateActive)
		if result.Error != nil {
			return extErrors.Wrap(result.Error, "Unable to mark subscription as active in database")
		}
	}
	return nil
}

// CancelSubscription will mark the subscription to be cancelled in the next billing period
func (m *Manager) CancelSubscription(ctx context.Context, subscriptionID string) error {
	updateParams := &stripe.SubscriptionParams{
		Params: stripe.Params{
			Context: ctx,
		},
		CancelAtPeriodEnd: stripe.Bool(true),
	}
	sub, err := m.StripeClient.Subscriptions.Update(subscriptionID, updateParams)
	if err != nil {
		return extErrors.Wrap(err, "Unable to cancel subscription on Stripe")
	}
	if sub.CancelAtPeriodEnd != true {
		return fmt.Errorf("Stripe did not mark subscription as cancel at end of period")
	}
	result := m.DB.WithContext(ctx).Model(&Subscription{}).Where("id = ?", subscriptionID).Update("state", StateInactive)
	if result.Error != nil {
		return extErrors.Wrap(result.Error, "Unable to mark subscription as inactive in database")
	}
	return nil
}

func (m *Manager) createPlans(ctx context.Context, plans []Plan) error {
	for k := range plans {
		if err := plans[k].createPlanOnStripe(ctx, m.StripeClient); err != nil {
			return err
		}
	}
	if createRes := m.DB.WithContext(ctx).Create(&plans); createRes.Error != nil {
		return createRes.Error
	}
	return nil
}

func (m *Manager) listPlans(ctx context.Context) ([]Plan, error) {
	plans := make([]Plan, 0, 1)
	if lookupRes := m.DB.WithContext(ctx).
		Preload("Parts").
		Find(&plans); lookupRes.Error != nil {
		return nil, lookupRes.Error
	}
	return plans, nil
}

// GetPlan will return a defined Plan
func (m *Manager) GetPlan(ctx context.Context, planID string) (*Plan, error) {
	var plan Plan
	lookupRes := m.DB.WithContext(ctx).
		Preload("Parts").
		First(&plan, "id = ?", planID)
	if errors.Is(lookupRes.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if lookupRes.Error != nil {
		return nil, lookupRes.Error
	}
	return &plan, nil
}

// PrimaryUsageOption specifies which primary subscription to increment and by how much
type PrimaryUsageOption struct {
	SubscriptionID string
	ReferenceTime  time.Time
	Amount         int64
}

func (p *PrimaryUsageOption) validate() error {
	if len(p.SubscriptionID) == 0 {
		return fmt.Errorf("empty SubscriptionID is invalid")
	}
	if p.ReferenceTime.IsZero() {
		return fmt.Errorf("invalid ReferenceTime")
	}
	if p.Amount < 0 {
		return fmt.Errorf("negative Amount is invalid")
	}
	return nil
}

// IncrementPrimaryUsage will increment the primary usage record for billing
func (m *Manager) IncrementPrimaryUsage(ctx context.Context, opts []PrimaryUsageOption) error {
	if len(opts) == 0 {
		return nil
	}
	for _, opt := range opts {
		if err := opt.validate(); err != nil {
			return err
		}
	}

	// TODO: make this configurable
	concurrentSemaphore := make(chan struct{}, 5)
	for _, opt := range opts {
		concurrentSemaphore <- struct{}{}
		go func(aggr PrimaryUsageOption) {
			if err := m.txIncrementOrNew(ctx, usageOption{
				SubscriptionID: aggr.SubscriptionID,
				PartID:         nil,
				ReferenceTime:  aggr.ReferenceTime,
				Amount:         aggr.Amount,
			}); err != nil {
				m.Logger.Error("Error incrementing primary usage",
					zap.String("SubscriptionID", aggr.SubscriptionID),
					zap.Error(err),
				)
			}
			<-concurrentSemaphore
		}(opt)
	}
	return nil
}

// SecondaryUsageOption specifies which secondary subscription to increment and by how much
type SecondaryUsageOption struct {
	SubscriptionID string
	PartID         string
	ReferenceTime  time.Time
	Amount         int64
}

func (s *SecondaryUsageOption) validate() error {
	if len(s.SubscriptionID) == 0 {
		return fmt.Errorf("empty SubscriptionID is invalid")
	}
	if len(s.PartID) == 0 {
		return fmt.Errorf("empty PartID is invalid")
	}
	if s.ReferenceTime.IsZero() {
		return fmt.Errorf("invalid ReferenceTime")
	}
	if s.Amount < 0 {
		return fmt.Errorf("negative Amount is invalid")
	}
	return nil
}

// IncrementSecondaryUsage will increment the secondary usage record for billing
func (m *Manager) IncrementSecondaryUsage(ctx context.Context, opts []SecondaryUsageOption) error {
	if len(opts) == 0 {
		return nil
	}
	for _, opt := range opts {
		if err := opt.validate(); err != nil {
			return err
		}
	}

	// TODO: make this configurable
	concurrentSemaphore := make(chan struct{}, 5)
	for _, opt := range opts {
		concurrentSemaphore <- struct{}{}
		go func(aggr SecondaryUsageOption) {
			if err := m.txIncrementOrNew(ctx, usageOption{
				SubscriptionID: aggr.SubscriptionID,
				PartID:         &aggr.PartID,
				ReferenceTime:  aggr.ReferenceTime,
				Amount:         aggr.Amount,
			}); err != nil {
				m.Logger.Error("Error incrementing secondary usage",
					zap.String("SubscriptionID", aggr.SubscriptionID),
					zap.String("PartID", aggr.PartID),
					zap.Error(err),
				)
			}
			<-concurrentSemaphore
		}(opt)
	}
	return nil
}

type usageOption struct {
	SubscriptionID string
	PartID         *string
	ReferenceTime  time.Time
	Amount         int64
	retryCount     int
	lastError      error
}

// TODO: break up this giant function
func (m *Manager) txIncrementOrNew(ctx context.Context, aggr usageOption) error {
	if aggr.retryCount > 3 {
		return fmt.Errorf("Transaction retry exceeded: %w", aggr.lastError)
	}

	var sub Subscription
	var variableItem *SubscriptionItem
	err := m.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lookupRes := tx.Clauses(clause.Locking{Strength: "SHARE"}).
			Preload("SubscriptionItems").
			Preload("SubscriptionItems.Part").
			Preload("Plan").
			Preload("Plan.Parts").
			Where("id = ?", aggr.SubscriptionID).
			First(&sub)
		if lookupRes.Error != nil {
			return lookupRes.Error
		}

		if aggr.PartID == nil {
			variableItem = sub.findPrimaryVariableItem()
		} else {
			variablePart := sub.Plan.findVariablePart(false, aggr.PartID)
			if variablePart == nil {
				return fmt.Errorf("No secondary variable Part found for subscription with ID %s", aggr.SubscriptionID)
			}
			variableItem = sub.findSubscriptionItemByPartID(variablePart.ID)
		}

		if variableItem == nil {
			return fmt.Errorf("No variable item found for subscription with ID %s", aggr.SubscriptionID)
		}

		// try updating current period usage record
		res := tx.Model(&Usage{}).
			Where("subscription_item_id = ?", variableItem.ID).
			Where("start_date < ? AND ? <= end_date", aggr.ReferenceTime, aggr.ReferenceTime).
			UpdateColumn("aggregate_total", gorm.Expr("aggregate_total + ?", aggr.Amount))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected > 1 {
			m.Logger.Error("Primary usage update affected more than 1 row",
				zap.String("SubscriptionID", aggr.SubscriptionID),
			)
			return fmt.Errorf("Primary usage update affected more than 1 row")
		}
		if res.RowsAffected > 0 {
			return nil
		}

		// new usage record
		return m.newUsage(tx, newUsageOption{
			Amount:           aggr.Amount,
			Subscription:     &sub,
			SubscriptionItem: variableItem,
		})
	}, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})

	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			if pgErr.Code == "40001" {
				// CockroachDB write contention retry
				aggr.retryCount++
				aggr.lastError = err
				return m.txIncrementOrNew(ctx, aggr)
			} else if pgErr.Code == "23505" {
				// PrimaryKey constraint violation
				// this only happens when PeriodStart/PeriodEnd has not been updated yet
				timestamp, _ := ptypes.TimestampProto(aggr.ReferenceTime)
				err := m.Producer.SendTask(spec.SubscriptionTask, &protocol.Task{
					Timestamp: timestamp,
					SubscriptionTask: &protocol.SubscriptionTask{
						Function:       protocol.SubscriptionTask_Synchronize,
						SubscriptionID: sub.ID,
					},
					Type: protocol.Task_Subscription,
				})
				if err != nil {
					m.Logger.Error("Unable to send async task to subscription",
						zap.Error(err),
					)
					return err
				}
				return nil
			}
		}
		return err
	}

	timestamp, _ := ptypes.TimestampProto(aggr.ReferenceTime)
	if err := m.Producer.SendTask(spec.SubscriptionTask, &protocol.Task{
		Timestamp: timestamp,
		SubscriptionTask: &protocol.SubscriptionTask{
			Function:           protocol.SubscriptionTask_ReportUsage,
			SubscriptionItemID: variableItem.ID,
		},
		Type: protocol.Task_Subscription,
	}); err != nil {
		m.Logger.Error("Unable to send async task to report usage",
			zap.Error(err),
		)
		return err
	}

	return nil
}

type newUsageOption struct {
	Amount           int64
	Subscription     *Subscription
	SubscriptionItem *SubscriptionItem
}

func (m *Manager) newUsage(tx *gorm.DB, opt newUsageOption) error {
	usage := &Usage{
		AggregateTotal:     opt.Amount,
		StartDate:          opt.Subscription.PeriodStart,
		EndDate:            opt.Subscription.PeriodEnd,
		SubscriptionItemID: opt.SubscriptionItem.ID,
	}
	return tx.Create(&usage).Error
}

func (m *Manager) getSubscriptionItem(ctx context.Context, subscriptionItemID string) (*SubscriptionItem, error) {
	var item SubscriptionItem
	err := m.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.
			Clauses(clause.Locking{Strength: "SHARE"}).
			Preload("Part").
			First(&item, "id = ?", subscriptionItemID).Error
	})
	if err != nil {
		return nil, err
	}
	return &item, nil
}

type usageLookupOption struct {
	ReferenceTime      time.Time
	SubscriptionItemID string
}

func (m *Manager) getUsageBySubscriptionItemID(ctx context.Context, opt usageLookupOption) (*Usage, error) {
	var usage Usage
	err := m.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.
			Clauses(clause.Locking{Strength: "SHARE"}).
			Preload("SubscriptionItem").
			Preload("SubscriptionItem.Part").
			Order("end_date asc").
			Where("start_date < ? AND ? < end_date", opt.ReferenceTime, opt.ReferenceTime).
			Where("subscription_item_id = ?", opt.SubscriptionItemID)

		return query.First(&usage).Error
	})
	if err != nil {
		return nil, err
	}
	return &usage, nil
}
