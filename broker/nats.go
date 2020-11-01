package broker

import (
	"context"

	"github.com/zllovesuki/rmc/spec"
	"github.com/zllovesuki/rmc/spec/broker"
	"github.com/zllovesuki/rmc/spec/protocol"
)

type NATSBroker struct {
}

var _ broker.Producer = &NATSBroker{}
var _ broker.Consumer = &NATSBroker{}

func NewNATSBroker() (*NATSBroker, error) {
	return nil, nil
}

func (n *NATSBroker) Close() {

}

func (n *NATSBroker) SendControlRequest(hostIdentifier string, p *protocol.ControlRequest) error {
	return nil
}

func (n *NATSBroker) SendControlReply(p *protocol.ControlReply) error {
	return nil
}

func (n *NATSBroker) SendProvisionRequest(hostIdentifier string, p *protocol.ProvisionRequest) error {
	return nil
}

func (n *NATSBroker) SendProvisionReply(p *protocol.ProvisionReply) error {
	return nil
}

func (n *NATSBroker) SendHeartbeat(p *protocol.Heartbeat) error {
	return nil
}

func (n *NATSBroker) ReceiveControlRequest(ctx context.Context, hostIdentifier string) (<-chan *protocol.ControlRequest, error) {
	return nil, nil
}

func (n *NATSBroker) ReceiveProvisionRequest(ctx context.Context, hostIdentifier string) (<-chan *protocol.ProvisionRequest, error) {
	return nil, nil
}

func (n *NATSBroker) ReceiveControlReply(ctx context.Context) (<-chan *protocol.ControlReply, error) {
	return nil, nil
}

func (n *NATSBroker) ReceiveProvisionReply(ctx context.Context) (<-chan *protocol.ProvisionReply, error) {
	return nil, nil
}

func (n *NATSBroker) ReceiveHeartbeat(ctx context.Context, processor string) (<-chan *protocol.Heartbeat, error) {
	return nil, nil
}

func (n *NATSBroker) SendTask(taskType spec.TaskType, p *protocol.Task) error {
	return nil
}

func (n *NATSBroker) ReceiveTask(ctx context.Context, taskType spec.TaskType) (<-chan *protocol.Task, error) {
	return nil, nil
}
