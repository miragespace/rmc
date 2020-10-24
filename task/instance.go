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

type InstanceOptions struct {
	InstanceManager *instance.Manager
	Producer        broker.Producer
	Logger          *zap.Logger
}

type InstanceTask struct {
	InstanceOptions
}

func NewInstanceTask(option InstanceOptions) (*InstanceTask, error) {
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
		InstanceOptions: option,
	}, nil
}

func (t *InstanceTask) handleControlReply(ctx context.Context, cChan <-chan *protocol.ControlReply) {
	for {
		select {
		case <-ctx.Done():
			return
		case cReply := <-cChan:
			if cReply.GetInstance() == nil {
				t.Logger.Error("Received nil protocol.Instance when processing control reply")
				continue
			}
			instanceID := cReply.GetInstance().GetInstanceID()
			if len(instanceID) == 0 {
				t.Logger.Error("Received empty protocol.InstanceID when processing control reply")
				continue
			}
			logger := t.Logger.With(
				zap.String("InstanceID", instanceID),
			)
			lambda := func(currentState *instance.Instance, desiredState *instance.Instance) (shouldSave bool) {
				if currentState == nil {
					logger.Error("nil Instance when processing provision reply")
					return
				}
				switch cReply.GetResult() {
				case protocol.ControlReply_SUCCESS:
					if currentState.State == instance.StateStopping {
						desiredState.State = instance.StateStopped
					} else {
						desiredState.State = instance.StateRunning
					}
				case protocol.ControlReply_FAILURE:
					logger.Error("Instance control replied failure")
					desiredState.State = instance.StateError
				default:
					logger.Error("Instance control replied undetermined result")
					desiredState.State = instance.StateUnknown
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
		case pReply := <-pChan:
			if pReply.GetInstance() == nil {
				t.Logger.Error("Received nil protocol.Instance when processing provision reply")
				continue
			}
			instanceID := pReply.GetInstance().GetInstanceID()
			if len(instanceID) == 0 {
				t.Logger.Error("Received empty protocol.InstanceID when processing provision reply")
				continue
			}
			logger := t.Logger.With(
				zap.String("InstanceID", instanceID),
			)
			lambda := func(currentState *instance.Instance, desiredState *instance.Instance) (shouldSave bool) {
				if currentState == nil {
					logger.Error("nil Instance when processing provision reply")
					return
				}
				switch pReply.GetResult() {
				case protocol.ProvisionReply_SUCCESS:
					if currentState.State == instance.StateProvisioning {
						desiredState.ServerAddr = pReply.GetInstance().GetAddr()
						desiredState.ServerPort = pReply.GetInstance().GetPort()
						desiredState.State = instance.StateRunning
					} else {
						desiredState.State = instance.StateStopped
						desiredState.Status = instance.StatusTerminated
					}
				case protocol.ProvisionReply_FAILURE:
					logger.Error("Provision replied failure")
					desiredState.State = instance.StateError
				default:
					logger.Error("Provision replied undetermined result")
					desiredState.State = instance.StateUnknown
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
