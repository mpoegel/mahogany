package views

import (
	"context"
	"io"
	"log/slog"

	container "github.com/docker/docker/api/types/container"
)

type ContainerView struct {
	TemplateName  string
	ContainerInfo container.InspectResponse
	IsSuccess     bool
	Err           error
}

func (v *ContainerView) Name() string { return v.TemplateName }
func (v *ContainerView) WithName(name string) *ContainerView {
	v.TemplateName = name
	return v
}

type ContainerStartView struct {
	ID        string
	IsSuccess bool
	Err       error
}

func (v *ContainerStartView) Name() string { return "container-start" }

type ContainerStopView struct {
	ID        string
	IsSuccess bool
	Err       error
}

func (v *ContainerStopView) Name() string { return "container-stop" }

type ContainerRestartView struct {
	ID        string
	IsSuccess bool
	Err       error
}

func (v *ContainerRestartView) Name() string { return "container-restart" }

type ContainerRemoveView struct {
	ID        string
	IsSuccess bool
	Err       error
}

func (v *ContainerRemoveView) Name() string { return "container-delete" }

type ContainerLogsView struct {
	ID        string
	IsSuccess bool
	Err       error
	Logs      []string
}

func (v *ViewFinder) GetContainer(ctx context.Context, containerID string) *ContainerView {
	containerInfo, err := v.docker.ContainerInspect(ctx, containerID)
	if err != nil {
		slog.Error("failed to inspect container", "id", containerID, "err", err)
	}
	return &ContainerView{
		ContainerInfo: containerInfo,
		IsSuccess:     err == nil,
		Err:           err,
	}
}

func (v *ViewFinder) StartContainer(ctx context.Context, containerID string) *ContainerStartView {
	opts := container.StartOptions{}
	err := v.docker.ContainerStart(ctx, containerID, opts)
	return &ContainerStartView{
		ID:        containerID,
		IsSuccess: err == nil,
		Err:       err,
	}
}

func (v *ViewFinder) StopContainer(ctx context.Context, containerID string) *ContainerStopView {
	opts := container.StopOptions{}
	err := v.docker.ContainerStop(ctx, containerID, opts)
	return &ContainerStopView{
		ID:        containerID,
		IsSuccess: err == nil,
		Err:       err,
	}
}

func (v *ViewFinder) RestartContainer(ctx context.Context, containerID string) *ContainerRestartView {
	opts := container.StopOptions{}
	err := v.docker.ContainerRestart(ctx, containerID, opts)
	return &ContainerRestartView{
		ID:        containerID,
		IsSuccess: err == nil,
		Err:       err,
	}
}

func (v *ViewFinder) RemoveContainer(ctx context.Context, containerID string) *ContainerRemoveView {
	opts := container.RemoveOptions{}
	err := v.docker.ContainerRemove(ctx, containerID, opts)
	return &ContainerRemoveView{
		ID:        containerID,
		IsSuccess: err == nil,
		Err:       err,
	}
}

func (v *ViewFinder) GetContainerLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
	}
	return v.docker.ContainerLogs(ctx, containerID, opts)
}
