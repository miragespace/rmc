package task

import (
	"context"
	"fmt"

	"github.com/zllovesuki/rmc/host"
	"github.com/zllovesuki/rmc/spec/broker"

	extErrors "github.com/pkg/errors"
	"go.uber.org/zap"
)

type HostOptions struct {
	HostManager *host.Manager
	Producer    broker.Producer
	Logger      *zap.Logger
}

type HostTask struct {
	HostOptions
}

func NewHostTask(option HostOptions) (*HostTask, error) {
	if option.HostManager == nil {
		return nil, fmt.Errorf("nil InstanceManager is invalid")
	}
	if option.Producer == nil {
		return nil, fmt.Errorf("nil Producer is invalid")
	}
	if option.Logger == nil {
		return nil, fmt.Errorf("nil Logger is invalid")
	}
	return &HostTask{
		HostOptions: option,
	}, nil
}

func (t *HostTask) HandleReply(ctx context.Context) error {
	hChan, err := t.Producer.ReceiveHeartbeat(ctx)
	if err != nil {
		return extErrors.Wrap(err, "Cannot get heartbeat channel")
	}
	go func() {
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
	}()
	return nil
}
