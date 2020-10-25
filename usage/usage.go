package usage

import "time"

type Action string

const (
	Start Action = "Start"
	Stop  Action = "Stop"
)

// Usage describes the actions taken on an instance
type Usage struct {
	InstanceID string `gorm:"not null;index"`
	Action     Action
	When       time.Time
	Accounted  bool `gorm:"not null;default:false"`
}
