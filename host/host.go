package host

import "time"

// Host defines the physical/virtual server that will deploy Minecraft servers to Docker
type Host struct {
	Name          string `gorm:"primaryKey"`
	Running       int64
	Stopped       int64
	Capacity      int64
	LastHeartbeat time.Time
}

// Identifier will return a deterministic routing key for message broker
func (h *Host) Identifier() string {
	return "worker-" + h.Name
}
