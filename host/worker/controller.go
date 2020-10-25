package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/zllovesuki/rmc/host"
	"github.com/zllovesuki/rmc/host/docker"
	"github.com/zllovesuki/rmc/spec"
	"github.com/zllovesuki/rmc/spec/broker"
	"github.com/zllovesuki/rmc/spec/protocol"
	"github.com/zllovesuki/rmc/util"

	"go.uber.org/zap"
)

type Options struct {
	Docker   *docker.Client
	Logger   *zap.Logger
	Producer broker.Producer
	Consumer broker.Consumer
	Host     host.Host
	HostIP   string
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
	if option.Producer == nil {
		return nil, fmt.Errorf("nil Producer is invalid")
	}
	if option.Consumer == nil {
		return nil, fmt.Errorf("nil Consumer is invalid")
	}
	if len(option.Host.Name) == 0 {
		return nil, fmt.Errorf("empty Host Name is invalid")
	}
	if option.Host.Capacity == 0 {
		return nil, fmt.Errorf("zero capacity is invalid")
	}
	if len(option.HostIP) == 0 {
		return nil, fmt.Errorf("empty host ip is invalid")
	}
	return &Controller{
		Options: option,
	}, nil
}

func (c *Controller) Run(ctx context.Context) error {
	crChan, err := c.Consumer.ReceiveControlRequest(ctx, c.Host.Identifier())
	if err != nil {
		return err
	}

	prChan, err := c.Consumer.ReceiveProvisionRequest(ctx, c.Host.Identifier())
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
			if d.GetInstance() == nil {
				c.Logger.Error("Received provision request with nil Instance")
				continue
			}
			if d.GetInstance().GetInstanceID() == "" {
				c.Logger.Error("Received provision request with empty InstanceID")
				continue
			}

			requestedInstance := d.GetInstance()
			requestedAction := d.GetAction()
			instanceID := requestedInstance.GetInstanceID()

			logger := c.Logger.With(
				zap.String("InstanceID", instanceID),
				zap.String("Action", requestedAction.String()),
			)

			var err error
			switch d.GetAction() {
			case protocol.ControlRequest_STOP:
				err = c.Docker.StopInstance(ctx, requestedInstance)
			case protocol.ControlRequest_START:
				err = c.Docker.StartInstance(ctx, requestedInstance)
			default:
				logger.Error("Received unknown request")
				continue
			}

			var result protocol.ControlReply_ControlResult
			if err != nil {
				logger.Error("Cannot control instance",
					zap.Error(err),
				)
				result = protocol.ControlReply_FAILURE
			} else {
				result = protocol.ControlReply_SUCCESS
			}

			if err := c.Producer.SendControlReply(&protocol.ControlReply{
				Instance:      requestedInstance,
				RequestAction: requestedAction,
				Result:        result,
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
			if d.GetInstance() == nil {
				c.Logger.Error("Received provision request with nil Instance")
				continue
			}
			if d.GetInstance().GetInstanceID() == "" {
				c.Logger.Error("Received provision request with empty InstanceID")
				continue
			}

			requestedInstance := d.GetInstance()
			requestedAction := d.GetAction()
			instanceID := requestedInstance.GetInstanceID()

			logger := c.Logger.With(
				zap.String("InstanceID", instanceID),
				zap.String("Action", requestedAction.String()),
			)
			var err error
			var freePort int
			switch requestedAction {
			case protocol.ProvisionRequest_DELETE:
				// TODO: timeout or force delete
				err = c.Docker.DeleteInstance(ctx, requestedInstance)
			case protocol.ProvisionRequest_CREATE:
				freePort, err = util.GetFreePort()
				if err != nil {
					logger.Error("Cannot get a random port for provisioning",
						zap.Error(err),
					)
					break
				}
				d.Instance.Port = uint32(freePort)
				err = c.Docker.ProvisionInstance(ctx, requestedInstance)
			default:
				logger.Error("Received unknown request")
				continue
			}

			reply := &protocol.ProvisionReply{
				Instance:      requestedInstance,
				RequestAction: requestedAction,
			}
			var result protocol.ProvisionReply_ProvisionResult
			if err != nil {
				logger.Error("Cannot provision instance",
					zap.Error(err),
				)
				result = protocol.ProvisionReply_FAILURE
			} else {
				result = protocol.ProvisionReply_SUCCESS
				reply.Instance.Addr = c.HostIP
				reply.Instance.Port = uint32(freePort)
			}

			reply.Result = result

			if err := c.Producer.SendProvisionReply(reply); err != nil {
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
			stats, err := c.Docker.StatsInstances(ctx)
			if err != nil {
				c.Logger.Error("Cannot get instance list",
					zap.Error(err),
				)
			}
			c.Producer.SendHeartbeat(&protocol.Heartbeat{
				Host: &protocol.Host{
					Name:     c.Host.Name,
					Running:  stats.Running,
					Stopped:  stats.Stopped,
					Capacity: c.Host.Capacity,
				},
			})
		}
	}
}
