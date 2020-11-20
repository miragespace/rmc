package docker

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/zllovesuki/rmc/spec"
	"github.com/zllovesuki/rmc/spec/protocol"
	"github.com/zllovesuki/rmc/util"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	extErrors "github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	managedInstancePrefix = "rmc-instance-"
	dockerPrefix          = "/" + managedInstancePrefix
	dockerPrefixLen       = len(dockerPrefix)
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

func (c *Client) ProvisionInstance(ctx context.Context, p *protocol.Instance) (int, error) {

	// Reference: https://medium.com/backendarmy/controlling-the-docker-engine-in-go-d25fc0fe2c45
	var err error
	var mcServerPort string
	var mcServerImage string
	var mcPortType string
	var exposedPort int
	var instanceParams spec.Parameters

	instanceParams.FromProto(p.GetParameters())

	switch instanceParams["ServerEdition"] {
	case "java":
		mcServerPort = spec.JavaMinecraftTCPPort
		mcServerImage = spec.JavaMinecraftDockerImage
		mcPortType = "tcp"
		exposedPort, err = util.GetFreeTCPPort()
		if err != nil {
			return 0, extErrors.Wrap(err, "Cannot obtain free TCP port")
		}
	case "bedrock":
		mcServerPort = spec.BedrockMinecraftUDPPort
		mcServerImage = spec.BedrockMinecraftDockerImage
		mcPortType = "udp"
		exposedPort, err = util.GetFreeUDPPort()
		if err != nil {
			return 0, extErrors.Wrap(err, "Cannot obtain free UDP port")
		}
	default:
		return 0, fmt.Errorf("Unexpected ServerEdition: %s", instanceParams["ServerEdition"])
	}

	out, err := c.Client.ImagePull(ctx, mcServerImage, types.ImagePullOptions{})
	if err != nil {
		return 0, err
	}
	io.Copy(ioutil.Discard, out) // needed to make sure image pull was done. TODO: ensure image existence when starting host worker

	hostBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: strconv.Itoa(exposedPort),
	}

	containerPort, err := nat.NewPort(mcPortType, mcServerPort)
	if err != nil {
		return 0, extErrors.Wrap(err, "Unable to create port")
	}

	portBinding := nat.PortMap{containerPort: []nat.PortBinding{hostBinding}}

	resp, err := c.Client.ContainerCreate(ctx,
		&container.Config{
			Image: mcServerImage,
			Env: []string{ // TODO: use a builder
				"EULA=true",
				"VERSION=" + instanceParams["ServerVersion"],
				"MAX_PLAYERS=" + instanceParams["Players"],
				"MEMORY=" + instanceParams["RAM"] + "M",
			},
			// TODO: map volumes for persistence
		},
		&container.HostConfig{
			PortBindings: portBinding,
		},
		nil, // network config
		managedInstancePrefix+p.GetID(),
	)

	if err != nil {
		return 0, err
	}

	if err := c.Client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return 0, err
	}

	return exposedPort, nil
}

func (c *Client) DeleteInstance(ctx context.Context, p *protocol.Instance) error {
	containerID, err := c.getContainerID(ctx, p.GetID())
	if err != nil {
		return extErrors.Wrap(err, "Cannot delete instance")
	}
	if containerID == "" {
		// when the instance failed to provision
		return nil
	}
	if err := c.Client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
	}); err != nil {
		return extErrors.Wrap(err, "Cannot delete instance")
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
			if name == "/"+managedInstancePrefix+instanceID {
				id = container.ID
			}
		}
	}

	if len(id) == 0 {
		return "", nil
	}

	return id, nil
}

func (c *Client) StopInstance(ctx context.Context, p *protocol.Instance) error {
	containerID, err := c.getContainerID(ctx, p.GetID())
	if err != nil {
		return extErrors.Wrap(err, "Cannot stop instance")
	}
	if containerID == "" {
		// when the instance failed to provision
		return nil
	}
	timeout := time.Second * 15
	if err := c.Client.ContainerStop(ctx, containerID, &timeout); err != nil {
		return extErrors.Wrap(err, "Cannot stop container")
	}
	return nil
}

func (c *Client) StartInstance(ctx context.Context, p *protocol.Instance) error {
	containerID, err := c.getContainerID(ctx, p.GetID())
	if err != nil {
		return extErrors.Wrap(err, "Cannot start instance")
	}
	if containerID == "" {
		// when the instance failed to provision
		return nil
	}
	if err := c.Client.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return extErrors.Wrap(err, "Cannot start container")
	}
	return nil
}

type Stats struct {
	Running          int64
	Stopped          int64
	RunningInstances []string
}

func (c *Client) StatsInstances(ctx context.Context) (stats Stats, err error) {
	containers, err := c.Client.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return
	}

	runningInstances := make([]string, 0, 2)

	for _, container := range containers {
		for _, name := range container.Names {
			if strings.HasPrefix(name, dockerPrefix) {
				switch container.State {
				case "running":
					runningInstances = append(runningInstances, name[dockerPrefixLen:])
					stats.Running++
				case "removing", "restarting":
					stats.Running++
				default:
					stats.Stopped++
				}
			}
		}
	}

	stats.RunningInstances = runningInstances

	return
}
