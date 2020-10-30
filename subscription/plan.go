package subscription

import (
	"github.com/zllovesuki/rmc/spec"
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
	PlanID        string   `json:"-" gorm:"index"`
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

// -----------------------------------------------------------------------------
// 							Below is the boring part
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
