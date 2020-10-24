package usage

import "time"

type Event string

const (
	Start Event = "Start"
	End   Event = "End"
)

// Usage describes the time period of applicable charges to a subscription
type Usage struct {
	SubscriptionID string     `json:"subscriptionId" gorm:"primaryKey"`
	Start          time.Time  `json:"start"`
	End            *time.Time `json:"end"`
}
