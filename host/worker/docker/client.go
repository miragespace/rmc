package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/zllovesuki/rmc/instance"
	"github.com/zllovesuki/rmc/spec"
	"github.com/zllovesuki/rmc/spec/protocol"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
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

	resp, err := c.Client.ContainerCreate(ctx,
		&container.Config{
			Image: spec.JavaMinecraftDockerImage,
			Env: []string{
				"EULA=true",
				"VERSION=1.16.3",
			},
			// TODO: map volumes for persistence
		},
		nil, // host config. TODO: map container port to host port
		nil, // network config
		"rmc-instance-"+p.GetInstanceID(),
	)

	if err != nil {
		return err
	}

	if err := c.Client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
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
