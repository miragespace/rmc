package subscription

import (
	"time"
)

// State is the custom to define the current state of a subscription
type State string

// Defining different SubscriptionStates for a Subscription
const (
	StateActive    State = "Active"
	StateInactive  State = "Inactive"
	StatePending   State = "Pending"
	StateCancelled State = "Cancelled"
	StateOverdue   State = "Overdue"
)

// Subscription is a local copy of a Stripe Subscription, with all the relations established
type Subscription struct {
	ID                string             `json:"id" gorm:"primaryKey"`
	CustomerID        string             `json:"customerId" gorm:"index;not null"` // Corresponds to Stripe's Customer ID and Customer.ID
	State             State              `json:"state"`                            // Corresponds to Stripe's subscription.setup_intent.status. StateActive if status == 'succeeded'
	Plan              Plan               `json:"plan"`
	SubscriptionItems []SubscriptionItem `json:"subscriptionItems"`
	CreatedAt         time.Time          `json:"createdAt" gorm:"autoCreateTime"`
	PlanID            string             `json:"-" gorm:"not null"` // Corresponds to Stripe's Product ID and Plan.ID
}

// SubscriptionItem is a local copy of a Stripe Subscription Item under a Subscription
type SubscriptionItem struct {
	ID             string    `json:"id" gorm:"primaryKey"`                 // Corresponds to Stripe's Subscription Item ID
	SubscriptionID string    `json:"subscriptionId" gorm:"index;not null"` // Corresponds to the parent subscription ID that this item belongs to
	PeriodStart    time.Time `json:"periodStart" gorm:"not null"`          // Used for accounting purposes, this signals when the RunningUsage stars
	PeriodEnd      time.Time `json:"periodEnd" gorm:"not null"`            // Used for accounting purposes, this signals when the RunningUsage end
	PartID         string    `json:"-" gorm:"not null"`                    // Corrsponds to Stripe's Price ID and Plan.[]Part.ID
	Part           Part      `json:"part"`
}

// Usage describes the aggregate usage amount within a billing period
type Usage struct {
	SubscriptionItemID string           `json:"-" gorm:"primaryKey"`
	SubscriptionItem   SubscriptionItem `json:"subscriptionItem"`
	StartDate          time.Time        `json:"startDate"`
	EndDate            time.Time        `json:"endDate" gorm:"primaryKey"`
	AggregateTotal     int64            `json:"aggregateTotal"`
}
