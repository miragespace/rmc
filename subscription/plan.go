package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	extErrors "github.com/pkg/errors"
	"github.com/stripe/stripe-go/v71"
	"github.com/stripe/stripe-go/v71/client"
)

// loadPlansFromFile will read from the plan JSON file to define what plans are availble for purchase.
// ID fields will be populated via EnsureExistence().
// Note, if you change any of these:
// Plan.Name
// Plan.Interval
// Plan.Currency
// Plan.[]Part.Name
// Plan.[]Part.AmountInCents
// Plan.[]Part.Period
// Plan.[]Part.Type
// Then a new Product and its associated Prices will be created on Stripe.
// If you want to change the Price of a Plan once it is created on Stripe, make a new Plan and mark the old ones Retired
func loadPlansFromFile(filename string) (map[string]Plan, error) {
	jsonBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, extErrors.Wrap(err, "Cannot open plans JSON file")
	}
	plans := make(map[string]Plan)
	if err := json.Unmarshal(jsonBytes, &plans); err != nil {
		return nil, extErrors.Wrap(err, "Invalid pla JSON file")
	}
	return plans, nil
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

// PartType is the custom type to identify what's the Type of this Part in the Plan
type PartType string

// Defining constants
const (
	FixedType    PartType = "Fixed"
	VariableType PartType = "Variable"
)

// Part describes each Part of a Plan
type Part struct {
	ID            string   `json:"-"`             // Corresponding to Stripe's PriceID
	Name          string   `json:"name"`          // Name to describe this Part
	AmountInCents float64  `json:"amountInCents"` // Amount in cents (e.g. 15.0 for $0.015/{period})
	Period        string   `json:"period"`        // How should the AmountInCents apply. If Type is FixedType, then this Part will be billed AmountInCents/month regardless. If Type is Variable, then this Part will be billed AmountInCents/{period} in a month
	Type          PartType `json:"type"`          // Either FixedType or VariableType
}

// Plan describes an Instance plan. This corresponds to Stripe's "Product"
type Plan struct {
	ID          string            `json:"-"`           // Corresponds to Stripe's Product ID
	Name        string            `json:"name"`        // Represent the name shown to the customer and on Stripe
	Description string            `json:"description"` // Shown to the customer
	Currency    string            `json:"currency"`    // The ISO currency code (e.g. usd)
	Interval    string            `json:"interval"`    // Billing Frequency (e.g. month)
	Parts       []Part            `json:"parts"`       // See Part struct above
	Parameters  map[string]string `json:"parameters"`  // Describes what this Plan will have (e.g. {Ram: 2GB, Players: 6})
	Retired     bool              `json:"retired"`     // Flag if the Plan is no longer valid
}

// EnsureExistence will ensure that corresponding Plan exist on Stripe, and it will populate the ID fields in the Plan object.
func (p *Plan) EnsureExistence(ctx context.Context, s *client.API) error {
	// we already have it
	if len(p.ID) > 0 {
		return nil
	}
	lookupMap := make(map[string]int)
	lookupKeys := make([]*string, 0, 2)
	for index, part := range p.Parts {
		key := p.lookupKey(part)
		lookupKeys = append(lookupKeys, stripe.String(key))
		lookupMap[key] = index + 1
	}
	lookupParams := &stripe.PriceListParams{
		ListParams: stripe.ListParams{
			Context: ctx,
		},
		Active:     stripe.Bool(true),
		LookupKeys: lookupKeys,
	}
	pricesIter := s.Prices.List(lookupParams)
	var prodID string
	var count int = 0
	for pricesIter.Next() {
		count++
		price := pricesIter.Price()
		if len(prodID) == 0 {
			prodID = price.Product.ID
		}
		if prodID != price.Product.ID {
			return fmt.Errorf("Price \"%s\" is in a different Product", price.Nickname)
		}
		index := lookupMap[price.LookupKey]
		if index > 0 {
			p.Parts[index-1].ID = price.ID
		}
	}
	if pricesIter.Err() != nil {
		return extErrors.Wrap(pricesIter.Err(), "Cannot ensure Plan existence on Stripe")
	}

	if count == len(p.Parts) {
		// Populate Prooduct ID
		fmt.Println("Found all Prices and populating all IDs")
		p.ID = prodID
		return nil
	}
	if count == 0 {
		fmt.Println("Plans do not exist, creating...")
		return p.CreateOnStripe(ctx, s)
	}
	return fmt.Errorf("Unexpected number of Prices for Plan \"%s\" (expected: %d, actual: %d)", p.Name, len(p.Parts), count)
}

func (p *Plan) lookupKey(part Part) string {
	planName := lookupKeyRegex.ReplaceAllString(p.Name, "-")
	partName := lookupKeyRegex.ReplaceAllString(part.Name, "-")
	amountPart := fmt.Sprintf("%f", part.AmountInCents)
	return strings.ToLower(fmt.Sprintf("%s_%s_%s_%s_%s_%s", planName, p.Interval, part.Type, partName, amountPart, part.Period))
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

	fixedRecurring := &stripe.PriceRecurringParams{
		AggregateUsage: nil,
		Interval:       stripe.String(p.Interval),
		IntervalCount:  stripe.Int64(1),
		UsageType:      stripe.String("licensed"),
	}
	variableRecurring := &stripe.PriceRecurringParams{
		AggregateUsage: stripe.String("sum"),
		Interval:       stripe.String(p.Interval),
		IntervalCount:  stripe.Int64(1),
		UsageType:      stripe.String("metered"),
	}

	for k, part := range p.Parts {
		pParams := &stripe.PriceParams{
			Params: stripe.Params{
				Context: ctx,
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
			return err
		}
		// Populate the ID back to the Part
		p.Parts[k].ID = partPrice.ID
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
			sParams.Quantity = stripe.Int64(1)
		case VariableType:
			sParams.Quantity = nil
		}
		sParams.Items = append(sParams.Items, pParams)
	}

	return sParams
}

var d = map[string]Plan{
	"Small": {
		Name:        "Small Minecraft Server",
		Description: "3 players slot with 3GB of RAM. Perfect for a small gathering",
		Currency:    "usd",
		Interval:    "month",
		Parts: []Part{
			{
				Name:          "Monthly Fixed Price",
				AmountInCents: 300.0,
				Period:        "month",
				Type:          FixedType,
			},
			{
				Name:          "Per Minute price",
				AmountInCents: 0.03,
				Period:        "miniute",
				Type:          VariableType,
			},
		},
		Parameters: map[string]string{
			"RAM":     "3072",
			"Players": "3",
		},
		Retired: false,
	},
	"Medium": {
		Name:        "Medium Minecraft Server",
		Description: "6 players slot with 6GB of RAM. It's a party!",
		Currency:    "usd",
		Interval:    "month",
		Parts: []Part{
			{
				Name:          "Monthly Fixed Price",
				AmountInCents: 300.0,
				Period:        "month",
				Type:          FixedType,
			},
			{
				Name:          "Per Minute price",
				AmountInCents: 0.06,
				Period:        "miniute",
				Type:          VariableType,
			},
		},
		Parameters: map[string]string{
			"RAM":     "6144",
			"Players": "6",
		},
		Retired: false,
	},
	"Spicy": {
		Name:        "Spicy Minecraft Server",
		Description: "12 players slot with 12GB of RAM. What are you doing!",
		Currency:    "usd",
		Interval:    "month",
		Parts: []Part{
			{
				Name:          "Monthly Fixed Price",
				AmountInCents: 300.0,
				Period:        "month",
				Type:          FixedType,
			},
			{
				Name:          "Per Minute price",
				AmountInCents: 0.12,
				Period:        "miniute",
				Type:          VariableType,
			},
		},
		Parameters: map[string]string{
			"RAM":     "12288",
			"Players": "12",
		},
		Retired: false,
	},
}
