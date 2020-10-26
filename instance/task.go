package instance

import (
	"context"
	"fmt"

	"github.com/zllovesuki/rmc/spec/broker"
	"github.com/zllovesuki/rmc/spec/protocol"

	extErrors "github.com/pkg/errors"
	"go.uber.org/zap"
)

type TaskOptions struct {
	InstanceManager *Manager
	Consumer        broker.Consumer
	Logger          *zap.Logger
}

type Task struct {
	TaskOptions
}

func NewTask(option TaskOptions) (*Task, error) {
	if option.InstanceManager == nil {
		return nil, fmt.Errorf("nil InstanceManager is invalid")
	}
	if option.Consumer == nil {
		return nil, fmt.Errorf("nil Consumer is invalid")
	}
	if option.Logger == nil {
		return nil, fmt.Errorf("nil Logger is invalid")
	}
	return &Task{
		TaskOptions: option,
	}, nil
}

func (t *Task) handleControlReply(ctx context.Context, reply *protocol.ControlReply) {
	if reply == nil {
		t.Logger.Error("Received nil protocol.ControlReply when processing control reply")
		return
	}
	if reply.GetInstance() == nil {
		t.Logger.Error("Received nil protocol.Instance when processing control reply")
		return
	}
	if reply.GetInstance().GetInstanceID() == "" {
		t.Logger.Error("Received empty InstanceID when processing control reply")
		return
	}

	repliedInstance := reply.GetInstance()
	instanceID := repliedInstance.GetInstanceID()
	logger := t.Logger.With(
		zap.String("InstanceID", instanceID),
		zap.String("Action", reply.GetRequestAction().String()),
	)

	lambda := func(current *Instance, desired *Instance) (shouldSave bool, returnError interface{}) {
		if current == nil {
			returnError = "nil Instance when processing control reply"
			return
		}
		switch reply.GetRequestAction() {
		case protocol.ControlRequest_START:
			if current.State != StateStarting {
				returnError = "Invalid Instance.State when processing control reply (expected: " + StateStarting + ", actual: " + current.State + ")"
				return
			}
			switch reply.GetResult() {
			case protocol.ControlReply_SUCCESS:
				desired.State = StateRunning
			case protocol.ControlReply_FAILURE:
				returnError = "Instance Control START was not successful"
				desired.State = StateStopped
			default:
				returnError = "Control START replied undetermined result"
				desired.State = StateUnknown
			}
		case protocol.ControlRequest_STOP:
			if current.State != StateStopping {
				returnError = "Invalid Instance.State when processing control reply (expected: " + StateStopping + ", actual: " + current.State + ")"
				return
			}
			switch reply.GetResult() {
			case protocol.ControlReply_SUCCESS:
				desired.State = StateStopped
			case protocol.ControlReply_FAILURE:
				returnError = "Instance Control STOP was not successful"
				desired.State = StateRunning
			default:
				returnError = "Control STOP replied undetermined result"
				desired.State = StateUnknown
			}
		default:
			returnError = "ControlRequest had undefined action"
			desired.State = StateUnknown
		}

		// trigger history insertion
		desired.PreviousState = current.State
		shouldSave = true
		return
	}
	lambdaResult := t.InstanceManager.LambdaUpdate(ctx, instanceID, lambda)
	if lambdaResult.ReturnValue != nil {
		logger.Error(lambdaResult.ReturnValue.(string))
	}
	if lambdaResult.TxError != nil {
		logger.Error("Cannot update instance status",
			zap.Error(lambdaResult.TxError),
		)
	}
}

func (t *Task) handleProvisionReply(ctx context.Context, reply *protocol.ProvisionReply) {
	if reply == nil {
		t.Logger.Error("Received nil protocol.ProvisionReply when processing provision reply")
		return
	}
	if reply.GetInstance() == nil {
		t.Logger.Error("Received nil protocol.Instance when processing provision reply")
		return
	}
	if reply.GetInstance().GetInstanceID() == "" {
		t.Logger.Error("Received empty InstanceID when processing provision reply")
		return
	}

	repliedInstance := reply.GetInstance()
	instanceID := repliedInstance.GetInstanceID()
	logger := t.Logger.With(
		zap.String("InstanceID", instanceID),
	)

	lambda := func(current *Instance, desired *Instance) (shouldSave bool, returnError interface{}) {
		if current == nil {
			returnError = "nil Instance when processing provision reply"
			return
		}
		switch reply.GetRequestAction() {
		case protocol.ProvisionRequest_CREATE:
			if current.State != StateProvisioning {
				returnError = "Invalid Instance.State when processing provision reply (expected: " + StateProvisioning + ", actual: " + current.State + ")"
				return
			}
			switch reply.GetResult() {
			case protocol.ProvisionReply_SUCCESS:
				desired.ServerAddr = repliedInstance.GetAddr()
				desired.ServerPort = repliedInstance.GetPort()
				desired.State = StateRunning
			case protocol.ProvisionReply_FAILURE:
				returnError = "Instance provision CREATE was not successful"
				desired.State = StateError
			default:
				returnError = "Provision CREATE replied undetermined result"
				desired.State = StateUnknown
			}
		case protocol.ProvisionRequest_DELETE:
			if current.State != StateRemoving {
				returnError = "Invalid Instance.State when processing provision reply (expected: " + StateRemoving + ", actual: " + current.State + ")"
				return
			}
			switch reply.GetResult() {
			case protocol.ProvisionReply_SUCCESS:
				desired.State = StateRemoved
				desired.Status = StatusTerminated
			case protocol.ProvisionReply_FAILURE:
				returnError = "Instance provision DELETE was not successful"
				desired.State = StateError
			default:
				returnError = "Provision DELETE replied undetermined result"
				desired.State = StateUnknown
			}
		default:
			returnError = "ProvisionRequest had undefined action"
			desired.State = StateUnknown
		}

		// trigger history insertion
		desired.PreviousState = current.State
		shouldSave = true
		return
	}
	lambdaResult := t.InstanceManager.LambdaUpdate(ctx, instanceID, lambda)
	if lambdaResult.ReturnValue != nil {
		logger.Error(lambdaResult.ReturnValue.(string))
	}
	if lambdaResult.TxError != nil {
		logger.Error("Cannot update instance status",
			zap.Error(lambdaResult.TxError),
		)
	}
}

func (t *Task) HandleReply(ctx context.Context) error {
	cChan, err := t.Consumer.ReceiveControlReply(ctx)
	if err != nil {
		return extErrors.Wrap(err, "Cannot get control reply channel")
	}
	pChan, err := t.Consumer.ReceiveProvisionReply(ctx)
	if err != nil {
		return extErrors.Wrap(err, "Cannot get provision reply channel")
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case reply := <-cChan:
				t.handleControlReply(ctx, reply)
			}
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case reply := <-pChan:
				t.handleProvisionReply(ctx, reply)
			}
		}
	}()
	return nil
}
