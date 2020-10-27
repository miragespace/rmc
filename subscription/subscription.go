package subscription

import "time"

type ItemType string

type Subscription struct {
	ID                string             `json:"id" gorm:"primaryKey"`
	CustomerID        string             `json:"customerId" gorm:"index"` // Corresponds to Stripe's Customer ID and Customer.ID
	Ready             bool               `json:"ready"`                   // Corresponds to Stripe's subscription.setup_intent.status. True if status == 'succeeded'
	SubscriptionItems []SubscriptionItem `json:"subscriptionItems"`
}

const (
	SubscriptionFixed    ItemType = "Fixed"
	SubscriptionVariable          = "Variable"
	SubscriptionAddon             = "Addon"
)

type SubscriptionItem struct {
	ID                  string    `json:"id" gorm:"primaryKey"`        // Corresponds to Stripe's Subscription Item ID
	SubscriptionID      string    `json:"subscriptionId" gorm:"index"` // Corresponds to the parent subscription ID that this item belongs to
	RunningTotalMinutes int64     `json:"totalMinutes"`                // Used for accounting purposes. This is the variable usage part of the subscription item. Round up to the nearest hour when reporting for usage
	CurrentPeriodStart  time.Time `json:"periodStart"`                 // Used for accounting purposes, this signals when the TotalMinutes stars
	Type                ItemType  `json:"type"`                        // Used to identified if this is the parent item (tied to an instance)
}
