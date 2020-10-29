package subscription

import (
	"context"
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

func (m *Manager) SynchronizeSubscriptionStatus(ctx context.Context, subscriptionId string) error {
	subscriptionParams := &stripe.SubscriptionParams{
		Params: stripe.Params{
			Context: ctx,
		},
	}
	subscriptionParams.AddExpand("latest_invoice.payment_intent")
	subscriptionParams.AddExpand("pending_setup_intent")
	sub, err := m.StripeClient.Subscriptions.Get(subscriptionId, subscriptionParams)
	if err != nil {
		return extErrors.Wrap(err, "Unable to fetch from Stripe to synchronize status")
	}
	// TODO: also synchronize cancelled/overdue
	if sub.Status == stripe.SubscriptionStatusActive && sub.PendingSetupIntent == nil {
		result := m.DB.WithContext(ctx).Model(&Subscription{}).Where("id = ?", subscriptionId).Update("state", StateActive)
		if result.Error != nil {
			return extErrors.Wrap(result.Error, "Unable to mark subscription as cancelled in database")
		}
	}
	return nil
}

func (m *Manager) CancelSubscription(ctx context.Context, subscriptionId string) error {
	updateParams := &stripe.SubscriptionParams{
		Params: stripe.Params{
			Context: ctx,
		},
		CancelAtPeriodEnd: stripe.Bool(true),
	}
	sub, err := m.StripeClient.Subscriptions.Update(subscriptionId, updateParams)
	if err != nil {
		return extErrors.Wrap(err, "Unable to cancel subscription on Stripe")
	}
	if sub.CancelAtPeriodEnd != true {
		return fmt.Errorf("Stripe did not mark subscription as cancel at end of period")
	}
	result := m.DB.WithContext(ctx).Model(&Subscription{}).Where("id = ?", subscriptionId).Update("state", StateInactive)
	if result.Error != nil {
		return extErrors.Wrap(result.Error, "Unable to mark subscription as cancelled in database")
	}
	return nil
}

// RecordUsageOption specifies the parameters of how to calculate usage
// See PDF file to understand why these parameters are needed
type RecordUsageOption struct {
	CustomerID     string
	SubscriptionID string

	CurrentlyRunning  bool
	PreviousStartDate *time.Time
	CurrentEndDate    *time.Time
	Primary           bool
}

// RecordUsage will insert an usage record, and submit a record to Stripe for billing
// See PDF file to understand all cases
func (m *Manager) RecordUsage(ctx context.Context, opt RecordUsageOption) error {
	// logger := m.Logger.With(
	// 	zap.String("CustomerID", opt.CustomerID),
	// 	zap.String("SubscriptionID", opt.SubscriptionID),
	// )

	// For now, we only care about primary subscription's variable usage
	if !opt.Primary {
		return nil
	}

	sub, err := m.Get(ctx, GetOption{
		CustomerID:     opt.CustomerID,
		SubscriptionID: opt.SubscriptionID,
	})
	if err != nil {
		return extErrors.Wrap(err, "Unable to lookup corresponding Subscription")
	}

	plan, ok := m.GetDefinedPlanByID(sub.PlanID)
	if !ok {
		return fmt.Errorf("Unable to find PlanID %s in defined plans", sub.PlanID)
	}
	// Find the subscriptionItemId where the part is variable
	var varialePart Part
	for _, part := range plan.Parts {
		if part.Primary && part.Type == VariableType {
			varialePart = part
		}
	}

	// the actual accounting can start now
	currentBillingStart := sub.SubscriptionItems[0].PeriodStart
	currentBillingEnd := sub.SubscriptionItems[0].PeriodEnd
	var actualStartDate time.Time
	var actualEndDate time.Time

	// Case 2, 4, and 5 are subjected to reporting window size
	// TODO: calculate the optimal window size
	// TODO: race condition between user action and background worker
	if opt.CurrentEndDate == nil {
		if opt.PreviousStartDate != nil && opt.PreviousStartDate.After(currentBillingStart) {
			// Case 2: User start, but no end date
			// TODO: double check with the diagram
			actualStartDate = *opt.PreviousStartDate
			actualEndDate = currentBillingEnd
			panic("not implemented")
		}
		if opt.CurrentlyRunning {
			// Case 4: No events for the current period but Running
			actualStartDate = currentBillingStart
			actualEndDate = currentBillingEnd
			panic("not implemented")
		}
		// Case 5: Instance is stopped for the current period
		return nil
	}

	// TODO: verify the math
	if opt.CurrentEndDate.After(currentBillingEnd) || opt.CurrentEndDate.Before(currentBillingStart) {
		return nil
	}

	if opt.CurrentlyRunning {
		return nil
	}

	if opt.PreviousStartDate.After(currentBillingStart) {
		// Case 1: Simple case
		actualStartDate = *opt.PreviousStartDate
		actualEndDate = *opt.CurrentEndDate
	} else {
		// Case 3: No Running event for current period
		actualStartDate = currentBillingStart
		actualEndDate = *opt.CurrentEndDate
	}

	duration := actualEndDate.Sub(actualStartDate)

	var subscriptionItemID string
	for _, item := range sub.SubscriptionItems {
		if item.PartID == varialePart.ID {
			subscriptionItemID = item.ID
		}
	}

	// TODO: figure out how to do ceiling instead of round up
	var quantity int64
	switch varialePart.Period {
	case "minute":
		quantity = int64(duration.Round(time.Minute).Minutes())
	default:
		return fmt.Errorf("Unsupported period: %s", varialePart.Period)
	}

	usage := &Usage{
		ID:                 shortuuid.New(),
		Unit:               "second", // for future use: allow for "per use"
		Amount:             int64(duration.Seconds()),
		StartDate:          actualStartDate,
		EndDate:            *opt.CurrentEndDate,
		SubscriptionID:     sub.ID,
		SubscriptionItemID: subscriptionItemID,
	}

	if saveRes := m.DB.WithContext(ctx).Create(&usage); saveRes.Error != nil {
		return extErrors.Wrap(saveRes.Error, "Unable to record usage to database")
	}

	params := &stripe.UsageRecordParams{
		SubscriptionItem: stripe.String(subscriptionItemID),
		Quantity:         stripe.Int64(quantity),
		Timestamp:        stripe.Int64(opt.CurrentEndDate.Unix()),
		Action:           stripe.String(string(stripe.UsageRecordActionSet)),
	}

	if _, err := m.StripeClient.UsageRecords.New(params); err != nil {
		return extErrors.Wrap(err, "Unable to record usage on Stripe")
	}

	return nil
}
