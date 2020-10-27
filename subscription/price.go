package subscription

import (
	"context"
	"fmt"
	"regexp"

	extErrors "github.com/pkg/errors"
	"github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/client"
)

// Plans defines what plans are availble for purchase
var Plans = map[string]Plan{
	"Small": {
		Name:        "Small Minecraft Server",
		Description: "3 players slot with 3GB of RAM. Perfect for a small gathering",
		Currency:    "usd",
		Interval:    "month",
		Parts: PlanParts{
			Fixed: PartDetail{
				Amount: 2000.0, // $2/month
			},
			Variable: PartDetail{
				Amount: 0.02, // $0.02/hr
			},
		},
		Parameters: map[string]string{
			"RAM":     "3072",
			"Players": "3",
		},
	},
	"Medium": {
		Name:        "Medium Minecraft Server",
		Description: "6 players slot with 6GB of RAM. It's a party!",
		Currency:    "usd",
		Interval:    "month",
		Parts: PlanParts{
			Fixed: PartDetail{
				Amount: 2000.0, // $2/month
			},
			Variable: PartDetail{
				Amount: 0.03, // $0.03/hr
			},
		},
		Parameters: map[string]string{
			"RAM":     "6144",
			"Players": "6",
		},
	},
	"Spicy": {
		Name:        "Spicy Minecraft Server",
		Description: "12 players slot with 12GB of RAM. What are you doing!",
		Currency:    "usd",
		Interval:    "month",
		Parts: PlanParts{
			Fixed: PartDetail{
				Amount: 2000.0, // $2/month
			},
			Variable: PartDetail{
				Amount: 0.06, // $0.06/hr
			},
		},
		Parameters: map[string]string{
			"RAM":     "12288",
			"Players": "12",
		},
	},
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

// PartDetail describes each Part of a Plan
type PartDetail struct {
	ID     string  `json:"id"`     // Corresponding to Stripe's PriceID
	Amount float64 `json:"amount"` // Amount in cents (e.g. 0.15 for $0.0015/Hour)
}

// PlanParts describes the fixed and variable parts of a Plan
type PlanParts struct {
	Fixed    PartDetail `json:"fixed"`    // Represent the fixed part of the Subscription (e.g. $2/month)
	Variable PartDetail `json:"variable"` // Represent the variable part of the Subscription (e.g. $0.02/hr)
}

// Plan describes an Instance plan. This corresponds to Stripe's "Product"
type Plan struct {
	ID          string            `json:"id"`          // Corresponds to Stripe's Product ID
	Name        string            `json:"name"`        // Represent the name shown to the customer
	Description string            `json:"description"` // Shown to the customer
	Currency    string            `json:"currency"`    // The ISO currency code (e.g. usd)
	Interval    string            `json:"interval"`    // Billing Frequency (e.g. month)
	Parts       PlanParts         `json:"parts"`       // See PlanParts above
	Parameters  map[string]string `json:"parameters"`  // Describes what this Plan will have (e.g. {Ram: 2GB, Players: 6})
}

// EnsureExistence will ensure that corresponding Plan exist on Stripe
func (p *Plan) EnsureExistence(ctx context.Context, s *client.API) error {
	// we already have it
	if len(p.ID) > 0 {
		return nil
	}
	lookupParams := &stripe.PriceListParams{
		ListParams: stripe.ListParams{
			Context: ctx,
		},
		Active: stripe.Bool(true),
		LookupKeys: []*string{
			stripe.String(p.LookupKey()),
		},
	}
	pricesIter := s.Prices.List(lookupParams)
	var count int = 0
	for pricesIter.Next() {
		count++
	}
	if pricesIter.Err() != nil {
		return extErrors.Wrap(pricesIter.Err(), "Cannot list prices for plans")
	}
	if count == 2 {
		return nil
	}
	if count == 0 {
		return p.CreateOnStripe(ctx, s)
	}
	return fmt.Errorf("Inconsistent number of Prices (expected: 2, actual: %d", count)
}

// LookupKey will generate a unique lookup key based on the name of the Plan for Stripe
func (p *Plan) LookupKey() string {
	return fmt.Sprintf("%s_%s", lookupKeyRegex.ReplaceAllString(p.Name, "_"), p.Interval)
}

// CreateOnStripe will create corresponding prices and Products/Prices on Stripe for a given plan
func (p *Plan) CreateOnStripe(ctx context.Context, s *client.API) error {
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
		return err
	}
	// Populate the Product.ID
	p.ID = stripeProduct.ID

	// Create the Fixed part for Price
	fixedPart := &stripe.PriceParams{
		Params: stripe.Params{
			Context: ctx,
		},
		Active:            stripe.Bool(true),
		Nickname:          stripe.String("Fixed"),
		BillingScheme:     stripe.String("per_unit"),
		Currency:          stripe.String(p.Currency),
		UnitAmountDecimal: stripe.Float64(p.Parts.Fixed.Amount),
		Product:           stripe.String(p.ID),
		LookupKey:         stripe.String(p.LookupKey()),
		Recurring: &stripe.PriceRecurringParams{
			AggregateUsage: nil,
			Interval:       stripe.String(p.Interval),
			IntervalCount:  stripe.Int64(1),
			UsageType:      stripe.String("licensed"),
		},
	}
	stripeFixedPrice, err := s.Prices.New(fixedPart)
	if err != nil {
		return err
	}
	// Populate the PriceID
	p.Parts.Fixed.ID = stripeFixedPrice.ID

	// Create the Variable part for Price
	varPart := &stripe.PriceParams{
		Params: stripe.Params{
			Context: ctx,
		},
		Active:            stripe.Bool(true),
		Nickname:          stripe.String("Variable"),
		BillingScheme:     stripe.String("per_unit"),
		Currency:          stripe.String(p.Currency),
		UnitAmountDecimal: stripe.Float64(p.Parts.Variable.Amount),
		Product:           stripe.String(p.ID),
		LookupKey:         stripe.String(p.LookupKey()),
		Recurring: &stripe.PriceRecurringParams{
			AggregateUsage: stripe.String("sum"),
			Interval:       stripe.String(p.Interval),
			IntervalCount:  stripe.Int64(1),
			UsageType:      stripe.String("metered"),
		},
	}
	stripeVarPrice, err := s.Prices.New(varPart)
	if err != nil {
		return err
	}
	// Populate the PriceID
	p.Parts.Variable.ID = stripeVarPrice.ID

	return nil
}

// GetStripeSubscriptionParams will generate SubscriptionParams for used with Stripe from a Plan
func (p *Plan) GetStripeSubscriptionParams(ctx context.Context, customerID string) *stripe.SubscriptionParams {
	return &stripe.SubscriptionParams{
		Params: stripe.Params{
			Context: ctx,
		},
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price:    stripe.String(p.Parts.Fixed.ID),
				Quantity: stripe.Int64(1),
			},
			{
				Price:    stripe.String(p.Parts.Variable.ID),
				Quantity: nil,
			},
		},
	}
}
