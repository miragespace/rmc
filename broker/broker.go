package broker

import (
	"github.com/zllovesuki/rmc/host"
	"github.com/zllovesuki/rmc/spec"
)

// Broker defines the interface for publishing requests via message broker
type Broker interface {
	SendControlRequest(host *host.Host, p *spec.ControlRequest) error
	SendProvisionRequest(host *host.Host, p *spec.ProvisionRequest) error
}
