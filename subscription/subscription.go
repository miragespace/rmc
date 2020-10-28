package subscription

import (
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v71"
)

// State is the custom to define the current state of a subscription
type State string

// Defining different SubscriptionStates for a Subscription
const (
	StateActive    State = "Active"
	StateInactive        = "Inactive"
	StatePending         = "Pending"
	StateCancelled       = "Cancelled"
	StateOverdue         = "Overdue"
)

// Subscription is a local copy of a Stripe Subscription, with all the relations established
type Subscription struct {
	ID                string             `json:"id" gorm:"primaryKey"`
	PlanID            string             `json:"planId"`                  // Corresponds to Stripe's Product ID and Plan.ID
	CustomerID        string             `json:"customerId" gorm:"index"` // Corresponds to Stripe's Customer ID and Customer.ID
	State             State              `json:"state"`                   // Corresponds to Stripe's subscription.setup_intent.status. StateActive if status == 'succeeded'
	SubscriptionItems []SubscriptionItem `json:"subscriptionItems"`
	CreatedAt         time.Time          `json:"createdAt" gorm:"autoCreateTime"`
}

// SubscriptionItem is a local copy of a Stripe Subscription Item under a Subscription
type SubscriptionItem struct {
	ID             string     `json:"id" gorm:"primaryKey"`        // Corresponds to Stripe's Subscription Item ID
	PartID         string     `json:"partId"`                      // Corrsponds to Stripe's Price ID and Plan.[]Part.ID
	SubscriptionID string     `json:"subscriptionId" gorm:"index"` // Corresponds to the parent subscription ID that this item belongs to
	RunningUsage   int64      `json:"runningUsage"`                // Used for accounting purposes. This is the variable usage part of the subscription item. Round up to the nearest unit when reporting for usage
	PeriodStart    time.Time  `json:"periodStart"`                 // Used for accounting purposes, this signals when the RunningUsage stars
	PeriodEnd      time.Time  `json:"periodEnd"`                   // Used for accounting purposes, this signals when the RunningUsage end
	LastReportedAt *time.Time `json:"lastReportedAt"`              // Used for accounting purposes. This is when the the usage was last reported, if applicable
}

// FromStripeResponse will construct a local copy of Subscription from Stripe's response of subscription object
func (s *Subscription) FromStripeResponse(sub *stripe.Subscription, plan Plan) error {
	items := make([]SubscriptionItem, 0, 2)
	for _, subItem := range sub.Items.Data {
		partID := plan.lookupPartID(subItem.Price.LookupKey)
		if partID == "" {
			return fmt.Errorf("Inconsistent data: no corresponding Price ID")
		}
		item := SubscriptionItem{
			ID:             subItem.ID,
			PartID:         partID,
			SubscriptionID: sub.ID,
			RunningUsage:   0,
			PeriodStart:    time.Unix(sub.CurrentPeriodStart, 0), // TODO: revisit this
			PeriodEnd:      time.Unix(sub.CurrentPeriodEnd, 0),   // TODO: revisit this
		}
		items = append(items, item)
	}

	var subState State
	if sub.Status == stripe.SubscriptionStatusActive && sub.PendingSetupIntent == nil {
		subState = StateActive
	} else {
		subState = StatePending
	}

	*s = Subscription{
		ID:                sub.ID,
		PlanID:            plan.ID,
		CustomerID:        sub.Customer.ID,
		State:             subState,
		SubscriptionItems: items,
	}

	return nil
}
