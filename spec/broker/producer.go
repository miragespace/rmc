package broker

import (
	"context"

	"github.com/zllovesuki/rmc/host"
	"github.com/zllovesuki/rmc/spec/protocol"
)

// Producer defines a producer sending requests via message broker
type Producer interface {
	Close()
	SendControlRequest(host *host.Host, p *protocol.ControlRequest) error
	SendProvisionRequest(host *host.Host, p *protocol.ProvisionRequest) error
	ReceiveControlReply(ctx context.Context) (<-chan *protocol.ControlReply, error)
	ReceiveProvisionReply(ctx context.Context) (<-chan *protocol.ProvisionReply, error)
	ReceiveHeartbeat(ctx context.Context) (<-chan *protocol.Heartbeat, error)
}
