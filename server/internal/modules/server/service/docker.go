package service

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type DockerService struct{}

func NewDockerService() *DockerService {
	return &DockerService{}
}

func (ds *DockerService) Create(name string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()

	minecraftImage := "itzg/minecraft-server"

	reader, err := cli.ImagePull(ctx, minecraftImage, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	if _, err := io.Copy(io.Discard, reader); err != nil {
		return fmt.Errorf("error while pulling image: %w", err)
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: minecraftImage,
		Tty:   false,
		Env:   []string{"EULA=TRUE"},
		Labels: map[string]string{
			"paas.minecraft.server": "true",
		},
	}, &container.HostConfig{}, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"server_minecraft-network": {},
		},
	}, nil, name)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}

func (ds *DockerService) Restart(containerName string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()

	timeout := 10
	stopOptions := container.StopOptions{
		Timeout: &timeout,
	}

	if err := cli.ContainerRestart(ctx, containerName, stopOptions); err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	return nil
}
