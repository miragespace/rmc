package broker

import (
	"github.com/miragespace/rmc/spec"
	"github.com/miragespace/rmc/spec/protocol"
)

// Producer defines a producer sending requests via message broker
type Producer interface {
	Close()
	SendControlRequest(hostIdentifier string, p *protocol.ControlRequest) error
	SendControlReply(p *protocol.ControlReply) error
	SendProvisionRequest(hostIdentifier string, p *protocol.ProvisionRequest) error
	SendProvisionReply(p *protocol.ProvisionReply) error
	SendHeartbeat(p *protocol.Heartbeat) error
	SendTask(taskType spec.TaskType, p *protocol.Task) error
}
