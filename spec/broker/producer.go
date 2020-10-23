package broker

import (
	"github.com/zllovesuki/rmc/host"
	"github.com/zllovesuki/rmc/spec/protocol"
)

// Producer defines a producer sending requests via message broker
type Producer interface {
	Close()
	SendControlRequest(host *host.Host, p *protocol.ControlRequest) error
	SendProvisionRequest(host *host.Host, p *protocol.ProvisionRequest) error
}
