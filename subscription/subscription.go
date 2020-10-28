package subscription

import (
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v71"
)

type SubscriptionState string

const (
	StateActive    SubscriptionState = "Active"
	StateInactive                    = "Inactive"
	StateCancelled                   = "Cancelled"
	StateOverdue                     = "Overdue"
)

type Subscription struct {
	ID                string             `json:"id" gorm:"primaryKey"`
	PlanID            string             `json:"-"`                       // Corresponds to Stripe's Product ID and Plan.ID
	CustomerID        string             `json:"customerId" gorm:"index"` // Corresponds to Stripe's Customer ID and Customer.ID
	State             SubscriptionState  `json:"state"`                   // Corresponds to Stripe's subscription.setup_intent.status. StateActive if status == 'succeeded'
	SubscriptionItems []SubscriptionItem `json:"subscriptionItems"`
	CreatedAt         time.Time          `json:"createdAt" gorm:"autoCreateTime"`
}

type SubscriptionItem struct {
	ID                 string    `json:"id" gorm:"primaryKey"`        // Corresponds to Stripe's Subscription Item ID
	PartID             string    `json:"-"`                           // Corrsponds to Stripe's Price ID and Plan.[]Part.ID
	SubscriptionID     string    `json:"subscriptionId" gorm:"index"` // Corresponds to the parent subscription ID that this item belongs to
	RunningUsage       int64     `json:"runningUsage"`                // Used for accounting purposes. This is the variable usage part of the subscription item. Round up to the nearest unit when reporting for usage
	CurrentPeriodStart time.Time `json:"periodStart"`                 // Used for accounting purposes, this signals when the TotalMinutes stars
}

func (s *Subscription) FromStripeResponse(sub *stripe.Subscription, plan Plan) error {
	items := make([]SubscriptionItem, 0, 2)
	for _, subItem := range sub.Items.Data {
		partID := plan.lookupPartID(subItem.Price.LookupKey)
		if partID == "" {
			return fmt.Errorf("Inconsistent data: no corresponding Price ID")
		}
		item := SubscriptionItem{
			ID:                 subItem.ID,
			PartID:             partID,
			SubscriptionID:     sub.ID,
			RunningUsage:       0,
			CurrentPeriodStart: time.Unix(sub.CurrentPeriodStart, 0),
		}
		items = append(items, item)
	}

	var subState SubscriptionState
	if sub.PendingSetupIntent == nil {
		subState = StateActive
	} else {
		subState = StateInactive
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
