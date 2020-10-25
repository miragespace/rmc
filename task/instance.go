package task

import (
	"context"
	"fmt"

	"github.com/zllovesuki/rmc/instance"
	"github.com/zllovesuki/rmc/spec/broker"
	"github.com/zllovesuki/rmc/spec/protocol"

	extErrors "github.com/pkg/errors"
	"go.uber.org/zap"
)

type InstanceTaskOptions struct {
	InstanceManager *instance.Manager
	Producer        broker.Producer
	Logger          *zap.Logger
}

type InstanceTask struct {
	InstanceTaskOptions
}

func NewInstanceTask(option InstanceTaskOptions) (*InstanceTask, error) {
	if option.InstanceManager == nil {
		return nil, fmt.Errorf("nil InstanceManager is invalid")
	}
	if option.Producer == nil {
		return nil, fmt.Errorf("nil Producer is invalid")
	}
	if option.Logger == nil {
		return nil, fmt.Errorf("nil Logger is invalid")
	}
	return &InstanceTask{
		InstanceTaskOptions: option,
	}, nil
}

func (t *InstanceTask) handleControlReply(ctx context.Context, cChan <-chan *protocol.ControlReply) {
	for {
		select {
		case <-ctx.Done():
			return
		case reply := <-cChan:
			if reply.GetInstance() == nil {
				t.Logger.Error("Received nil protocol.Instance when processing control reply")
				continue
			}
			if reply.GetInstance().GetInstanceID() == "" {
				t.Logger.Error("Received empty InstanceID when processing control reply")
				continue
			}

			repliedInstance := reply.GetInstance()
			instanceID := repliedInstance.GetInstanceID()
			logger := t.Logger.With(
				zap.String("InstanceID", instanceID),
			)

			lambda := func(current *instance.Instance, desired *instance.Instance) (shouldSave bool) {
				if current == nil {
					logger.Error("nil Instance when processing control reply")
					return
				}
				// save current state as previous state
				desired.PreviousState = current.State

				// mediation
				switch reply.GetRequestAction() {
				case protocol.ControlRequest_START:
					switch reply.GetResult() {
					case protocol.ControlReply_SUCCESS:
						desired.State = instance.StateRunning
					case protocol.ControlReply_FAILURE:
						logger.Error("Instance Control START was not successful")
						desired.State = instance.StateStopped
					default:
						logger.Error("Control START replied undetermined result")
						desired.State = instance.StateUnknown
					}
				case protocol.ControlRequest_STOP:
					switch reply.GetResult() {
					case protocol.ControlReply_SUCCESS:
						desired.State = instance.StateStopped
					case protocol.ControlReply_FAILURE:
						logger.Error("Instance Control STOP was not successful")
						desired.State = instance.StateRunning
					default:
						logger.Error("Control STOP replied undetermined result")
						desired.State = instance.StateUnknown
					}
				default:
					logger.Error("ControlRequest had undefined action")
					desired.State = instance.StateUnknown
				}

				shouldSave = true
				return
			}
			if _, err := t.InstanceManager.LambdaUpdate(ctx, instanceID, lambda); err != nil {
				logger.Error("Cannot update instance status",
					zap.Error(err),
				)
			}
		}
	}
}

func (t *InstanceTask) handleProvisionReply(ctx context.Context, pChan <-chan *protocol.ProvisionReply) {
	for {
		select {
		case <-ctx.Done():
			return
		case reply := <-pChan:
			if reply.GetInstance() == nil {
				t.Logger.Error("Received nil protocol.Instance when processing control reply")
				continue
			}
			if reply.GetInstance().GetInstanceID() == "" {
				t.Logger.Error("Received empty InstanceID when processing control reply")
				continue
			}

			repliedInstance := reply.GetInstance()
			instanceID := repliedInstance.GetInstanceID()
			logger := t.Logger.With(
				zap.String("InstanceID", instanceID),
			)

			lambda := func(current *instance.Instance, desired *instance.Instance) (shouldSave bool) {
				if current == nil {
					logger.Error("nil Instance when processing provision reply")
					return
				}
				// save current state as previous state
				desired.PreviousState = current.State

				// mediation
				switch reply.GetRequestAction() {
				case protocol.ProvisionRequest_CREATE:
					switch reply.GetResult() {
					case protocol.ProvisionReply_SUCCESS:
						desired.ServerAddr = repliedInstance.GetAddr()
						desired.ServerPort = repliedInstance.GetPort()
						desired.State = instance.StateRunning
					case protocol.ProvisionReply_FAILURE:
						logger.Error("Instance provision CREATE was not successful")
						desired.State = instance.StateError
					default:
						logger.Error("Provision CREATE replied undetermined result")
						desired.State = instance.StateUnknown
					}
				case protocol.ProvisionRequest_DELETE:
					switch reply.GetResult() {
					case protocol.ProvisionReply_SUCCESS:
						desired.State = instance.StateRemoved
						desired.Status = instance.StatusTerminated
					case protocol.ProvisionReply_FAILURE:
						logger.Error("Instance provision DELETE was not successful")
						desired.State = instance.StateError
					default:
						logger.Error("Provision DELETE replied undetermined result")
						desired.State = instance.StateUnknown
					}
				default:
					logger.Error("ProvisionRequest had undefined action")
					desired.State = instance.StateUnknown
				}

				shouldSave = true
				return
			}
			if _, err := t.InstanceManager.LambdaUpdate(ctx, instanceID, lambda); err != nil {
				logger.Error("Cannot update instance status",
					zap.Error(err),
				)
			}
		}
	}
}

func (t *InstanceTask) HandleReply(ctx context.Context) error {
	cChan, err := t.Producer.ReceiveControlReply(ctx)
	if err != nil {
		return extErrors.Wrap(err, "Cannot get control reply channel")
	}
	pChan, err := t.Producer.ReceiveProvisionReply(ctx)
	if err != nil {
		return extErrors.Wrap(err, "Cannot get provision reply channel")
	}
	go t.handleControlReply(ctx, cChan)
	go t.handleProvisionReply(ctx, pChan)
	return nil
}
