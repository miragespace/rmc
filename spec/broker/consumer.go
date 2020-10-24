package broker

import (
	"context"

	"github.com/zllovesuki/rmc/host"
	"github.com/zllovesuki/rmc/spec/protocol"
)

// Consumer defines a consumer receiving requests via message broker
type Consumer interface {
	Close()
	SendControlReply(p *protocol.ControlReply) error
	SendProvisionReply(p *protocol.ProvisionReply) error
	ReceiveControlRequest(ctx context.Context, host *host.Host) (<-chan *protocol.ControlRequest, error)
	ReceiveProvisionRequest(ctx context.Context, host *host.Host) (<-chan *protocol.ProvisionRequest, error)
	SendHeartbeart(p *protocol.Heartbeat) error
}
