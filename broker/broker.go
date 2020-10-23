package broker

import (
	"context"

	"github.com/zllovesuki/rmc/host"
	"github.com/zllovesuki/rmc/spec"
)

// Broker defines the interface for publishing requests via message broker
type Broker interface {
	Close()
	SendControlRequest(host *host.Host, p *spec.ControlRequest) error
	SendProvisionRequest(host *host.Host, p *spec.ProvisionRequest) error
	ReceiveControlRequest(ctx context.Context, host *host.Host) (<-chan *spec.ControlRequest, error)
	ReceiveProvisionRequest(ctx context.Context, host *host.Host) (<-chan *spec.ProvisionRequest, error)
	Heartbeart(host *host.Host, p *spec.Heartbeat) error
}
