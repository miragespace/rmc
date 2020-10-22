package host

import "time"

// Host defines the physical/virtual server that will deploy Minecraft servers to Docker
type Host struct {
	Name          string `gorm:"primaryKey"`
	Load          int64
	Capacity      int64
	LastHeartbeat time.Time
}

// Exchange returns a deterministic exchange name for use in Message Broker
func (h *Host) Exchange() string {
	return "exchange-" + h.Name
}
