package host

import (
	"context"
	"fmt"

	"github.com/zllovesuki/rmc/spec/broker"
	"github.com/zllovesuki/rmc/spec/protocol"

	extErrors "github.com/pkg/errors"
	"go.uber.org/zap"
)

type TaskOptions struct {
	HostManager *Manager
	Consumer    broker.Consumer
	Logger      *zap.Logger
}

type Task struct {
	TaskOptions
}

func NewTask(option TaskOptions) (*Task, error) {
	if option.HostManager == nil {
		return nil, fmt.Errorf("nil HostManager is invalid")
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

func (t *Task) handleHeartbeat(ctx context.Context, hChan <-chan *protocol.Heartbeat) {
	for {
		select {
		case <-ctx.Done():
			return
		case hReply := <-hChan:
			if err := t.HostManager.ProcessHeartbeat(ctx, hReply); err != nil {
				t.Logger.Error("Cannot process heartbeat",
					zap.Error(err),
				)
			}
		}
	}
}

func (t *Task) HandleReply(ctx context.Context) error {
	hChan, err := t.Consumer.ReceiveHeartbeat(ctx)
	if err != nil {
		return extErrors.Wrap(err, "Cannot get heartbeat channel")
	}
	go t.handleHeartbeat(ctx, hChan)
	return nil
}
