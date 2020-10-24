package task

import (
	"context"
	"fmt"

	"github.com/zllovesuki/rmc/instance"
	"github.com/zllovesuki/rmc/spec/broker"

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

func (t *InstanceTask) HandleReply(ctx context.Context) error {
	cChan, err := t.Producer.ReceiveControlReply(ctx)
	if err != nil {
		return extErrors.Wrap(err, "Cannot get control reply channel")
	}
	pChan, err := t.Producer.ReceiveProvisionReply(ctx)
	if err != nil {
		return extErrors.Wrap(err, "Cannot get provision reply channel")
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case cReply := <-cChan:
				// TODO: handle control reply from host
				fmt.Println(cReply)
			}
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case pReply := <-pChan:
				// TODO: handle provision reply from host
				fmt.Println(pReply)
			}
		}
	}()
	return nil
}
