package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/zllovesuki/rmc/host"
	"github.com/zllovesuki/rmc/host/worker/docker"
	"github.com/zllovesuki/rmc/spec"
	"github.com/zllovesuki/rmc/spec/broker"
	"github.com/zllovesuki/rmc/spec/protocol"

	"go.uber.org/zap"
)

type Options struct {
	Docker   *docker.Client
	Logger   *zap.Logger
	Consumer broker.Consumer
	Host     *host.Host
}

type Controller struct {
	Options

	controlRequest   <-chan *protocol.ControlRequest
	provisionRequest <-chan *protocol.ProvisionRequest
}

func NewController(option Options) (*Controller, error) {
	if option.Docker == nil {
		return nil, fmt.Errorf("nil docker.Client is invalid")
	}
	if option.Logger == nil {
		return nil, fmt.Errorf("nil Logger is invalid")
	}
	if option.Consumer == nil {
		return nil, fmt.Errorf("nil Consumer is invalid")
	}
	if option.Host == nil {
		return nil, fmt.Errorf("nil Host is invalid")
	}
	return &Controller{
		Options: option,
	}, nil
}

func (c *Controller) Run(ctx context.Context) error {
	crChan, err := c.Consumer.ReceiveControlRequest(ctx, c.Host)
	if err != nil {
		return err
	}

	prChan, err := c.Consumer.ReceiveProvisionRequest(ctx, c.Host)
	if err != nil {
		return err
	}

	c.controlRequest = crChan
	c.provisionRequest = prChan

	go c.sendHeartbeat(ctx)
	go c.processControlRequest(ctx)
	go c.processProvisionRequest(ctx)

	return nil
}

func (c *Controller) processControlRequest(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case d := <-c.controlRequest:
			fmt.Println(d)
			if err := c.Consumer.SendControlReply(&protocol.ControlReply{
				Instance: &protocol.Instance{
					InstanceID: "test-instance",
				},
				Result: protocol.ControlReply_SUCCESS,
			}); err != nil {
				c.Logger.Error("Cannot send control reply",
					zap.Error(err),
				)
			}
		}
	}
}

func (c *Controller) processProvisionRequest(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case d := <-c.provisionRequest:
			fmt.Println(d)
			if d.GetAction() != protocol.ProvisionRequest_CREATE {
				continue
			}
			reply := &protocol.ProvisionReply{
				Instance: &protocol.Instance{
					InstanceID: d.GetInstance().GetInstanceID(),
				},
			}
			if err := c.Docker.ProvisionInstance(ctx, d.GetInstance()); err != nil {
				c.Logger.Error("Cannot provision instance",
					zap.Error(err),
				)
				reply.Result = protocol.ProvisionReply_FAILURE
			} else {
				reply.Result = protocol.ProvisionReply_SUCCESS
			}

			if err := c.Consumer.SendProvisionReply(reply); err != nil {
				c.Logger.Error("Cannot send provision reply",
					zap.Error(err),
				)
			}
		}
	}
}

func (c *Controller) sendHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(spec.HeartbeatInterval)
	c.Logger.Info("Heartbeat interval: " + spec.HeartbeatInterval.String())
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			instances, err := c.Docker.ListInstances(ctx)
			if err != nil {
				c.Logger.Error("Cannot get instance list",
					zap.Error(err),
				)
			}
			c.Consumer.SendHeartbeat(&protocol.Heartbeat{
				Host: &protocol.Host{
					Name:     "test",
					Running:  int64(len(instances)),
					Stopped:  0,
					Capacity: 20,
				},
			})
		}
	}
}
