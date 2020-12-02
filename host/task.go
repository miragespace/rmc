package host

import (
	"context"
	"fmt"
	"time"

	"github.com/miragespace/rmc/spec"
	"github.com/miragespace/rmc/spec/broker"
	"github.com/miragespace/rmc/spec/protocol"

	"github.com/golang/protobuf/ptypes"
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
			timestamp, err := ptypes.Timestamp(hReply.GetTimestamp())
			if err != nil {
				t.Logger.Error("Cannot parse heartbeat timestamp",
					zap.Error(err),
				)
				continue
			}
			if time.Now().Sub(timestamp) > 2*spec.HeartbeatInterval {
				// discard old heartbeats
				t.Logger.Info("Discarding outdated heartbeat",
					zap.Time("HeartbeatTime", timestamp),
				)
				continue
			}
			if err := t.HostManager.ProcessHeartbeat(ctx, hReply); err != nil {
				t.Logger.Error("Cannot process heartbeat",
					zap.Error(err),
				)
			}
		}
	}
}

func (t *Task) HandleReply(ctx context.Context) error {
	hChan, err := t.Consumer.ReceiveHeartbeat(ctx, "hostTask")
	if err != nil {
		return extErrors.Wrap(err, "Cannot get heartbeat channel")
	}
	go t.handleHeartbeat(ctx, hChan)
	return nil
}
