package broker

import (
	"context"

	"github.com/miragespace/rmc/spec"
	"github.com/miragespace/rmc/spec/protocol"
)

// Consumer defines a consumer receiving requests via message broker
type Consumer interface {
	Close()
	ReceiveControlRequest(ctx context.Context, hostIdentifier string) (<-chan *protocol.ControlRequest, error)
	ReceiveProvisionRequest(ctx context.Context, hostIdentifier string) (<-chan *protocol.ProvisionRequest, error)
	ReceiveControlReply(ctx context.Context) (<-chan *protocol.ControlReply, error)
	ReceiveProvisionReply(ctx context.Context) (<-chan *protocol.ProvisionReply, error)
	ReceiveHeartbeat(ctx context.Context, processor string) (<-chan *protocol.Heartbeat, error)
	ReceiveTask(ctx context.Context, taskType spec.TaskType) (<-chan *protocol.Task, error)
}
