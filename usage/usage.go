package usage

import "time"

// Usage describes the time period of applicable charges to a subscription
type Usage struct {
	SubscriptionID string     `json:"subscriptionId"`
	Start          time.Time  `json:"start"`
	End            *time.Time `json:"end"`
}
