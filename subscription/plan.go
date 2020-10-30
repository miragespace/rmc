package subscription

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/zllovesuki/rmc/spec"

	extErrors "github.com/pkg/errors"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/client"
)

// PartType is the custom type to identify what's the Type of this Part in the Plan
type PartType string

// Defining constants
const (
	FixedType    PartType = "Fixed"
	VariableType PartType = "Variable"
)

// Part describes each Part of a Plan. This corresponds to Stripe's "Price"
type Part struct {
	ID            string   `json:"id" gorm:"primaryKey"` // Corresponding to Stripe's PriceID
	Name          string   `json:"name"`                 // Name to describe this Part
	AmountInCents float64  `json:"amountInCents"`        // Amount in cents (e.g. 15.0 for $0.015/{period})
	Unit          string   `json:"unit"`                 // How should the AmountInCents apply. If Type is FixedType, then this Part will be billed AmountInCents/month regardless. If Type is Variable, then this Part will be billed Usage * AmountInCents/{unit} in a month
	Type          PartType `json:"type"`                 // Either FixedType or VariableType
	Primary       bool     `json:"primary"`              // Indicate if this Part is the Primary part (e.g. Instance, not Addon) or not
	PlanID        string   `json:"-"`
}

// Plan describes an Instance plan. This corresponds to Stripe's "Product"
type Plan struct {
	ID          string          `json:"id" gorm:"primaryKey"` // Corresponds to Stripe's Product ID
	Name        string          `json:"name"`                 // Represent the name shown to the customer and on Stripe
	Description string          `json:"description"`          // Shown to the customer
	Currency    string          `json:"currency"`             // The ISO currency code (e.g. usd)
	Interval    string          `json:"interval"`             // Billing Frequency (e.g. month)
	Parts       []Part          `json:"parts"`                // See Part struct above
	Parameters  spec.Parameters `json:"parameters"`           // Describes what this Plan will have (e.g. {Ram: 2GB, Players: 6})
	Retired     bool            `json:"retired"`              // Flag if the Plan is no longer valid (Archived on Stripe)
}

func (m *Manager) CreatePlans(ctx context.Context, plans []*Plan) error {
	for _, plan := range plans {
		if err := plan.createPlanOnStripe(ctx, m.StripeClient); err != nil {
			return err
		}
	}
	return m.DB.WithContext(ctx).Create(&plans).Error
}

func (m *Manager) ListPlans(ctx context.Context) ([]Plan, error) {
	plans := make([]Plan, 0, 1)
	if lookupRes := m.DB.WithContext(ctx).
		Preload("Parts").
		Find(&plans); lookupRes.Error != nil {
		return nil, lookupRes.Error
	}
	return plans, nil
}

func (m *Manager) GetPlan(ctx context.Context, planID string) (*Plan, error) {
	var plan Plan
	if lookupRes := m.DB.WithContext(ctx).
		Preload("Parts").
		First(&plan, "id = ?", planID); lookupRes.Error != nil {
		return nil, lookupRes.Error
	}
	return &plan, nil
}

// -----------------------------------------------------------------------------
// 							Before is the boring part
// -----------------------------------------------------------------------------

// Each subscription has two price part:
// Fixed: $2/month
// Variable: $0.02/hr

// Therefore, you have plans like such:
// 3 Players Server: $2/month + $0.02/hr
// 6 Players Server: $2/month + $0.04/hr
// etc

// To simplify the usage part of the subscription, we will add two SubscriptionItems to each Subscription in Stripe.
// The fixed part will have `usage_type` of "licensed" in the Price API
// The variable part will have `usage_type` of "metered" in the Price API
// See https://stripe.com/docs/api/prices/create for details
// When creating a new subscription, we will bill the Fixed part with Quantity of 1 (and "renew" it every new billing period)
// When reporting usage, we will report the Variable part with Subscription.RunningTotalMinutes/60, rounded *up* to the nearest hour

var lookupKeyRegex = regexp.MustCompile("[^a-zA-Z0-9]+")

// lookupKey will generate a unique LookupKey on stripe to identify each Part of the Plan
func (p *Plan) lookupKey(part Part) string {
	planName := lookupKeyRegex.ReplaceAllString(p.Name, "-")
	partName := lookupKeyRegex.ReplaceAllString(part.Name, "-")
	amountPart := fmt.Sprintf("%f", part.AmountInCents)
	return strings.ToLower(fmt.Sprintf("%s_%s_%s_%s_%s_%s", planName, p.Interval, part.Type, partName, amountPart, part.Unit))
}

// createPlanOnStripe will create missing Plan as Product on Stripe
func (p *Plan) createPlanOnStripe(ctx context.Context, s *client.API) error {
	if len(p.ID) == 0 {
		// Create corresponding Product
		prodParams := &stripe.ProductParams{
			Params: stripe.Params{
				Context:  ctx,
				Metadata: p.Parameters,
			},
			Active:      stripe.Bool(true),
			Name:        stripe.String(p.Name),
			Description: stripe.String(p.Description),
		}
		stripeProduct, err := s.Products.New(prodParams)
		if err != nil {
			return extErrors.Wrap(err, "Cannot create Plan as Product on Stripe")
		}
		// Populate the Product.ID
		p.ID = stripeProduct.ID
	}

	return p.createPartsOnStripe(ctx, s)
}

// createPartsOnStripe will create missing Parts as Price on Stripe
func (p *Plan) createPartsOnStripe(ctx context.Context, s *client.API) error {

	if len(p.ID) == 0 {
		return fmt.Errorf("Plan ID doesn't exist, please create Plan before Parts")
	}

	fixedRecurring := &stripe.PriceRecurringParams{
		AggregateUsage: nil,
		Interval:       stripe.String(p.Interval),
		IntervalCount:  stripe.Int64(1),
		UsageType:      stripe.String("licensed"),
	}
	variableRecurring := &stripe.PriceRecurringParams{
		AggregateUsage: stripe.String(string(stripe.PriceRecurringAggregateUsageLastDuringPeriod)),
		Interval:       stripe.String(p.Interval),
		IntervalCount:  stripe.Int64(1),
		UsageType:      stripe.String("metered"),
	}

	for k, part := range p.Parts {
		if len(part.ID) > 0 {
			// already exist, don't create
			continue
		}

		pParams := &stripe.PriceParams{
			Params: stripe.Params{
				Context: ctx,
				Metadata: map[string]string{
					"Type":    string(part.Type),
					"Primary": strconv.FormatBool(part.Primary),
				},
			},
			Active:            stripe.Bool(true),
			Nickname:          stripe.String(part.Name),
			BillingScheme:     stripe.String("per_unit"),
			Currency:          stripe.String(p.Currency),
			UnitAmountDecimal: stripe.Float64(part.AmountInCents),
			Product:           stripe.String(p.ID),
			LookupKey:         stripe.String(p.lookupKey(part)),
		}
		switch part.Type {
		case FixedType:
			pParams.Recurring = fixedRecurring
		case VariableType:
			pParams.Recurring = variableRecurring
		}
		partPrice, err := s.Prices.New(pParams)
		if err != nil {
			return extErrors.Wrap(err, "Cannot create Part as Price on Stripe")
		}
		// Populate the ID back to the Part
		p.Parts[k].ID = partPrice.ID
	}
	return nil
}

func (p *Plan) lookupPartByLookupKey(lookupKey string) Part {
	for _, part := range p.Parts {
		if lookupKey == p.lookupKey(part) {
			return part
		}
	}
	return Part{}
}

func (p *Plan) lookupPartByID(partID string) Part {
	for _, part := range p.Parts {
		if part.ID == partID {
			return part
		}
	}
	return Part{}
}

func (p *Plan) findVariablePart(primary bool, partID *string) *Part {
	if !primary && partID == nil {
		return nil
	}
	if p == nil || p.ID == "" || len(p.Parts) == 0 {
		return nil
	}
	for _, part := range p.Parts {
		if primary {
			if part.Primary && part.Type == VariableType {
				return &part
			}
		} else {
			if !part.Primary && part.Type == VariableType && part.ID == *partID {
				return &part
			}
		}
	}
	return nil
}

// GetStripeSubscriptionParams will generate SubscriptionParams for used with Stripe from a Plan
func (p *Plan) GetStripeSubscriptionParams(ctx context.Context, customerID string) *stripe.SubscriptionParams {
	sParams := &stripe.SubscriptionParams{
		Params: stripe.Params{
			Context: ctx,
		},
		Customer: stripe.String(customerID),
		Items:    []*stripe.SubscriptionItemsParams{},
	}

	for _, part := range p.Parts {
		pParams := &stripe.SubscriptionItemsParams{
			Price: stripe.String(part.ID),
		}
		switch part.Type {
		case FixedType:
			pParams.Quantity = stripe.Int64(1)
		case VariableType:
			pParams.Quantity = nil
		}
		sParams.Items = append(sParams.Items, pParams)
	}

	return sParams
}
