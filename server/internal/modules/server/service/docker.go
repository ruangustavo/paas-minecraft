package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
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

func (ds *DockerService) Delete(name string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()

	if err := cli.ContainerRemove(ctx, name, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("failed to remove container: %w", err)
		}
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

func (ds *DockerService) Start(containerName string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()

	if err := cli.ContainerStart(ctx, containerName, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}

func (ds *DockerService) Stop(containerName string) error {
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

	if err := cli.ContainerStop(ctx, containerName, stopOptions); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}

	return nil
}

func (ds *DockerService) GetContainerStatus(containerName string) (string, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()

	containerJSON, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return "not_found", nil
		}
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	if containerJSON.State.Running {
		return "running", nil
	}
	return "stopped", nil
}

type ContainerInfo struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Status  string    `json:"status"`
	Created time.Time `json:"created"`
}

func (ds *DockerService) List() ([]ContainerInfo, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()

	filterArgs := filters.NewArgs()
	filterArgs.Add("label", "paas.minecraft.server=true")

	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	result := make([]ContainerInfo, 0, len(containers))
	for _, c := range containers {
		name := c.Names[0]
		if len(name) > 0 && name[0] == '/' {
			name = name[1:]
		}

		status := "stopped"
		if c.State == "running" {
			status = "running"
		}

		result = append(result, ContainerInfo{
			ID:      c.ID,
			Name:    name,
			Status:  status,
			Created: time.Unix(c.Created, 0),
		})
	}

	return result, nil
}
