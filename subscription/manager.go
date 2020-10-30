package subscription

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lithammer/shortuuid/v3"
	extErrors "github.com/pkg/errors"
	"github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/client"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ManagerOptions struct {
	StripeClient   *client.API
	DB             *gorm.DB
	Logger         *zap.Logger
	PathToPlanJSON string
}

type Manager struct {
	ManagerOptions
	planArray      []Plan
	planIDIndexMap map[string]int
}

func NewManager(option ManagerOptions) (*Manager, error) {
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
	if err := option.DB.AutoMigrate(&Subscription{}, &SubscriptionItem{}, &Usage{}); err != nil {
		return nil, extErrors.Wrap(err, "Cannot initilize subscription.Manager")
	}

	plans, err := loadPlansFromFile(option.PathToPlanJSON)
	if err != nil {
		return nil, extErrors.Wrap(err, "Cannot populate defined Plans")
	}

	planMap := make(map[string]int)
	for index, p := range plans {
		if err := p.ensureExistence(context.Background(), option.StripeClient); err != nil {
			return nil, extErrors.Wrap(err, "Cannot ensure Plan existence on Stripe")
		}
		planMap[p.ID] = index + 1
		plans[index] = p
	}

	return &Manager{
		ManagerOptions: option,
		planIDIndexMap: planMap,
		planArray:      plans,
	}, nil
}

func (m *Manager) ListDefinedPlans() []Plan {
	return m.planArray
}

func (m *Manager) GetDefinedPlanByID(planID string) (Plan, bool) {
	index := m.planIDIndexMap[planID]
	if index == 0 {
		return Plan{}, false
	}

	plan := m.planArray[index-1]

	plan.Parameters = plan.Parameters.Clone()
	return plan, true
}

func (m *Manager) Create(ctx context.Context, si *Subscription) error {
	result := m.DB.WithContext(ctx).Create(si)
	if result.Error != nil {
		m.Logger.Error("Unable to create new subscription in database",
			zap.Error(result.Error),
		)
		return extErrors.Wrap(result.Error, "Cannot create subscription")
	}
	return nil
}

type ListOption struct {
	CustomerID string
	Before     time.Time
	Limit      int
}

func (m *Manager) List(ctx context.Context, opt ListOption) ([]Subscription, error) {
	baseQuery := m.DB.WithContext(ctx).Order("created_at desc")
	if len(opt.CustomerID) > 0 {
		baseQuery = baseQuery.Where("customer_id = ?", opt.CustomerID)
	} else {
		return nil, fmt.Errorf("Either ListOption.CustomerID is required")
	}
	if opt.Limit > 0 {
		baseQuery = baseQuery.Limit(opt.Limit)
	}
	if !opt.Before.IsZero() {
		baseQuery = baseQuery.Where("created_at < ?", opt.Before)
	}

	results := make([]Subscription, 0, 1)
	result := baseQuery.Preload("SubscriptionItems").Find(&results)

	if result.Error != nil {
		m.Logger.Error("Database returned error",
			zap.Error(result.Error),
		)
		return nil, result.Error
	}
	return results, nil
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
	var sub Subscription
	result := m.DB.WithContext(ctx).
		Preload("SubscriptionItems").
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

func (m *Manager) GetUsage(ctx context.Context, opt GetOption) ([]Usage, error) {
	if len(opt.CustomerID) == 0 {
		return nil, fmt.Errorf("CustomerID is required")
	}
	if len(opt.SubscriptionID) == 0 {
		return nil, fmt.Errorf("Either SubscriptionCheck.SubscriptionID is required")
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

	usages := make([]Usage, 0, 2)
	result = m.DB.WithContext(ctx).
		Preload("Subscription.SubscriptionItems").
		Preload(clause.Associations).
		Order("usages.end_date desc").
		Where("subscription_id = ?", opt.SubscriptionID).
		Find(&usages)

	if result.Error != nil {
		m.Logger.Error("Database returned error",
			zap.Error(result.Error),
		)
		return nil, result.Error
	}
	return usages, nil
}

type AttachPaymentOptions struct {
	CustomerID      string
	PaymentMethodID string
}

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

type CreateFromPlanOption struct {
	CustomerID string
	Plan       Plan
}

func (m *Manager) CreateSubscriptionFromPlan(ctx context.Context, opt CreateFromPlanOption) (*stripe.Subscription, error) {
	if len(opt.CustomerID) == 0 {
		return nil, fmt.Errorf("SetupOptions.CustomerID is required")
	}
	if len(opt.Plan.ID) == 0 {
		return nil, fmt.Errorf("SetupOptions.Plan needs to be a synchronized Plan")
	}

	subscriptionParams := opt.Plan.GetStripeSubscriptionParams(ctx, opt.CustomerID)
	subscriptionParams.AddExpand("latest_invoice.payment_intent")
	subscriptionParams.AddExpand("pending_setup_intent")

	sub, err := m.StripeClient.Subscriptions.New(subscriptionParams)

	if err != nil {
		return nil, err
	}

	return sub, nil
}

func (m *Manager) SynchronizeSubscriptionStatus(ctx context.Context, subscriptionID string) error {
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
	// TODO: also synchronize billing start/end
	if sub.Status == stripe.SubscriptionStatusActive && sub.PendingSetupIntent == nil {
		result := m.DB.WithContext(ctx).Model(&Subscription{}).Where("id = ?", subscriptionID).Update("state", StateActive)
		if result.Error != nil {
			return extErrors.Wrap(result.Error, "Unable to mark subscription as active in database")
		}
	}
	return nil
}

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

// TODO: support non-primary increment

type newUsageOption struct {
	Amount  int64
	Primary bool
}

func (m *Manager) newUsage(tx *gorm.DB, sub *Subscription, opt newUsageOption) error {
	plan, ok := m.GetDefinedPlanByID(sub.PlanID)
	if !ok {
		return fmt.Errorf("Unable to find Plan with ID %s in defined plans", sub.PlanID)
	}
	var variablePart Part
	for _, part := range plan.Parts {
		if part.Primary == opt.Primary && part.Type == VariableType {
			variablePart = part
		}
	}
	var subscriptionItemID string
	for _, item := range sub.SubscriptionItems {
		if item.PartID == variablePart.ID {
			subscriptionItemID = item.ID
		}
	}
	currentBillingStart := sub.SubscriptionItems[0].PeriodStart
	currentBillingEnd := sub.SubscriptionItems[0].PeriodEnd
	usage := &Usage{
		ID:                 shortuuid.New(),
		Amount:             opt.Amount,
		StartDate:          currentBillingStart,
		EndDate:            currentBillingEnd,
		IsPrimary:          opt.Primary,
		SubscriptionID:     sub.ID,
		SubscriptionItemID: subscriptionItemID,
	}
	return tx.Create(&usage).Error
}

// PrimaryUsageOption specifies which primary subscription to increment and by how much
type PrimaryUsageOption struct {
	SubscriptionID string
	ReferenceTime  time.Time
	Amount         int64
}

// IncrementPrimaryUsage will increment the primary usage record for billing
func (m *Manager) IncrementPrimaryUsage(ctx context.Context, opts []PrimaryUsageOption) error {
	if len(opts) == 0 {
		return nil
	}
	return m.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, aggr := range opts {
			// try updating current period usage record
			res := tx.Model(&Usage{}).
				Where("subscription_id = ?", aggr.SubscriptionID).
				Where("start_date < ? AND ? <= end_date", aggr.ReferenceTime, aggr.ReferenceTime).
				Where("is_primary = ?", true).
				UpdateColumn("amount", gorm.Expr("amount + ?", aggr.Amount))
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected > 1 {
				m.Logger.Error("Primary usage update affected more than 1 row",
					zap.String("SubscriptionID", aggr.SubscriptionID),
				)
				// fail through
			}
			if res.RowsAffected > 0 {
				continue
			}
			// new usage record
			var sub Subscription
			lookupRes := tx.Preload("SubscriptionItems").
				Where("id = ?", aggr.SubscriptionID).
				First(&sub)
			if lookupRes.Error != nil {
				return lookupRes.Error
			}
			if err := m.newUsage(tx, &sub, newUsageOption{
				Amount:  aggr.Amount,
				Primary: true,
			}); err != nil {
				return err
			}
		}
		return nil
	}, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
}

type SecondaryUsageOption struct {
	SubscriptionID string
	PartID         string
	ReferenceTime  time.Time
	Amount         int64
}

func (m *Manager) IncrementSecondaryUsage(ctx context.Context, opts []SecondaryUsageOption) error {
	// find the item with given part id, then increment that usage
	// should be something like this:
	// res := tx.Model(&Usage{}).
	// 	Where("subscription_item_id = ?", subscriptionItemID). // notice it is using item id, not subscription id
	// 	Where("start_date < ? AND ? <= end_date", aggr.ReferenceTime, aggr.ReferenceTime).
	// 	Where("is_primary = ?", false). // notice it is is_primary = false
	// 	UpdateColumn("amount", gorm.Expr("amount + ?", aggr.Amount))
	return nil
}
