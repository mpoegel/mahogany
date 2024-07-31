package mahogany

import (
	"context"
	"io"

	types "github.com/docker/docker/api/types"
	container "github.com/docker/docker/api/types/container"
	client "github.com/docker/docker/client"
)

type DockerI interface {
	ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error)
	ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerRestart(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error
	ContainerLogs(ctx context.Context, container string, options container.LogsOptions) (io.ReadCloser, error)
}

func NewDocker(host, version string) (DockerI, error) {
	return client.NewClientWithOpts()
}
