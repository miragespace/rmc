package docker

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zllovesuki/rmc/instance"
	"github.com/zllovesuki/rmc/spec"
	"github.com/zllovesuki/rmc/spec/protocol"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	extErrors "github.com/pkg/errors"
	"go.uber.org/zap"
)

type Options struct {
	Client *client.Client
	Logger *zap.Logger
}

type Client struct {
	Options
}

func NewClient(option Options) (*Client, error) {
	if option.Client == nil {
		return nil, fmt.Errorf("nil Client is invalid")
	}
	if option.Logger == nil {
		return nil, fmt.Errorf("nil Logger is invalid")
	}
	return &Client{
		Options: option,
	}, nil
}

func (c *Client) ProvisionInstance(ctx context.Context, p *protocol.Instance) error {
	// TESTING
	_, err := c.Client.ImagePull(ctx, spec.JavaMinecraftDockerImage, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	// Reference: https://medium.com/backendarmy/controlling-the-docker-engine-in-go-d25fc0fe2c45

	hostBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: strconv.Itoa(int(p.GetPort())),
	}

	containerPort, err := nat.NewPort("tcp", spec.JavaMinecraftTCPPort)
	if err != nil {
		return extErrors.Wrap(err, "Unable to create port")
	}

	portBinding := nat.PortMap{containerPort: []nat.PortBinding{hostBinding}}

	resp, err := c.Client.ContainerCreate(ctx,
		&container.Config{
			Image: spec.JavaMinecraftDockerImage,
			Env: []string{ // TODO: use a builder
				"EULA=true",
				"VERSION=" + p.GetVersion(),
			},
			// TODO: map volumes for persistence
		},
		&container.HostConfig{
			PortBindings: portBinding,
		},
		nil,                               // network config
		"rmc-instance-"+p.GetInstanceID(), // TODO: use a function to generate
	)

	if err != nil {
		return err
	}

	if err := c.Client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	return nil
}

func (c *Client) DeleteInstance(ctx context.Context, p *protocol.Instance) error {
	containerID, err := c.getContainerID(ctx, p.GetInstanceID())
	if err != nil {
		return extErrors.Wrap(err, "Cannot delete instance")
	}
	if err := c.Client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
	}); err != nil {
		return extErrors.Wrap(err, "Cannot delete container")
	}
	return nil
}

func (c *Client) getContainerID(ctx context.Context, instanceID string) (string, error) {
	var id string
	containers, err := c.Client.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return "", extErrors.Wrap(err, "Cannot get container id")
	}

	for _, container := range containers {
		for _, name := range container.Names {
			if name == "/rmc-instance-"+instanceID {
				id = container.ID
			}
		}
	}

	if len(id) == 0 {
		return "", fmt.Errorf("Cannot find instance with instance ID %s", instanceID)
	}

	return id, nil
}

func (c *Client) StopInstance(ctx context.Context, p *protocol.Instance) error {
	containerID, err := c.getContainerID(ctx, p.GetInstanceID())
	if err != nil {
		return extErrors.Wrap(err, "Cannot stop instance")
	}
	timeout := time.Second * 15
	if err := c.Client.ContainerStop(ctx, containerID, &timeout); err != nil {
		return extErrors.Wrap(err, "Cannot stop container")
	}
	return nil
}

func (c *Client) StartInstance(ctx context.Context, p *protocol.Instance) error {
	containerID, err := c.getContainerID(ctx, p.GetInstanceID())
	if err != nil {
		return extErrors.Wrap(err, "Cannot start instance")
	}
	if err := c.Client.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return extErrors.Wrap(err, "Cannot start container")
	}
	return nil
}

func (c *Client) ListInstances(ctx context.Context) ([]*instance.Instance, error) {
	containers, err := c.Client.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return nil, extErrors.Wrap(err, "Cannot list instances")
	}

	instances := make([]*instance.Instance, 0)
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.HasPrefix(name, "/rmc-instance-") {
				instances = append(instances, &instance.Instance{
					ID: "h",
				})
			}
		}
	}
	return instances, nil
}
