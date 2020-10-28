package instance

import (
	"fmt"

	"github.com/zllovesuki/rmc/host"
	"github.com/zllovesuki/rmc/spec"
	"github.com/zllovesuki/rmc/spec/broker"
	"github.com/zllovesuki/rmc/spec/protocol"

	extErrors "github.com/pkg/errors"
)

type LifecycleManagerOption struct {
	Producer broker.Producer
}

type LifecycleOption struct {
	HostName   string
	InstanceID string
	Parameters *spec.Parameters
}

type LifecycleManager interface {
	Start(opt LifecycleOption) error
	Stop(opt LifecycleOption) error
	Create(opt LifecycleOption) error
	Delete(opt LifecycleOption) error
}

type lifecycleManager struct {
	LifecycleManagerOption
}

var _ LifecycleManager = &lifecycleManager{}

func NewLifecycleManager(option LifecycleManagerOption) (LifecycleManager, error) {
	if option.Producer == nil {
		return nil, fmt.Errorf("nil Producer is invalid")
	}
	return &lifecycleManager{
		LifecycleManagerOption: option,
	}, nil
}

func (o *LifecycleOption) Validate() error {
	if len(o.HostName) == 0 {
		return fmt.Errorf("empty HostName is invalid")
	}
	if len(o.InstanceID) == 0 {
		return fmt.Errorf("empty InstanceID is invalid")
	}
	return nil
}

func getIdentifier(hostName string) string {
	h := host.Host{
		Name: hostName,
	}
	return h.Identifier()
}

func (l *lifecycleManager) Start(opt LifecycleOption) error {
	if err := opt.Validate(); err != nil {
		return err
	}
	if err := l.Producer.SendControlRequest(
		getIdentifier(opt.HostName),
		&protocol.ControlRequest{
			Instance: &protocol.Instance{
				ID:         opt.InstanceID,
				Parameters: opt.Parameters.ToProto(),
			},
			Action: protocol.ControlRequest_START,
		}); err != nil {
		return extErrors.Wrap(err, "Cannot request to START instance")
	}
	return nil
}

func (l *lifecycleManager) Stop(opt LifecycleOption) error {
	if err := opt.Validate(); err != nil {
		return err
	}
	if err := l.Producer.SendControlRequest(
		getIdentifier(opt.HostName),
		&protocol.ControlRequest{
			Instance: &protocol.Instance{
				ID:         opt.InstanceID,
				Parameters: opt.Parameters.ToProto(),
			},
			Action: protocol.ControlRequest_STOP,
		}); err != nil {
		return extErrors.Wrap(err, "Cannot request to STOP instance")
	}
	return nil
}

func (l *lifecycleManager) Create(opt LifecycleOption) error {
	if err := opt.Validate(); err != nil {
		return err
	}
	if err := l.Producer.SendProvisionRequest(
		getIdentifier(opt.HostName),
		&protocol.ProvisionRequest{
			Instance: &protocol.Instance{
				ID:         opt.InstanceID,
				Parameters: opt.Parameters.ToProto(),
			},
			Action: protocol.ProvisionRequest_CREATE,
		}); err != nil {
		return extErrors.Wrap(err, "Cannot request to CREATE instance")
	}
	return nil
}

func (l *lifecycleManager) Delete(opt LifecycleOption) error {
	if err := opt.Validate(); err != nil {
		return err
	}
	if err := l.Producer.SendProvisionRequest(
		getIdentifier(opt.HostName),
		&protocol.ProvisionRequest{
			Instance: &protocol.Instance{
				ID:         opt.InstanceID,
				Parameters: opt.Parameters.ToProto(),
			},
			Action: protocol.ProvisionRequest_DELETE,
		}); err != nil {
		return extErrors.Wrap(err, "Cannot request to DELETE instance")
	}
	return nil
}
