package host

import (
	"time"

	"github.com/zllovesuki/rmc/spec"
)

// Host defines the physical/virtual server that will deploy Minecraft servers to Docker
type Host struct {
	Name          string `gorm:"primaryKey"`
	Running       int64
	Stopped       int64
	Capacity      int64
	LastHeartbeat time.Time
	FirstSeen     time.Time
	// TODO: Server location?
}

// Identifier will return a deterministic routing key for message broker
func (h *Host) Identifier() string {
	return "worker-" + h.Name
}

// Alive will return true if the host's last heartbeat was sent within 2 spec.HeartbeatInterval
func (h *Host) Alive() bool {
	return time.Now().Sub(h.LastHeartbeat) <= (2 * spec.HeartbeatInterval)
}
