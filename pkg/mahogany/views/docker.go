package views

import (
	"context"
	"io"

	container "github.com/docker/docker/api/types/container"
)

type ContainerView struct {
	ContainerInfo container.InspectResponse
	IsSuccess     bool
	Err           error
}

type ContainerStartView struct {
	ID        string
	IsSuccess bool
	Err       error
}

type ContainerStopView struct {
	ID        string
	IsSuccess bool
	Err       error
}

type ContainerRestartView struct {
	ID        string
	IsSuccess bool
	Err       error
}

type ContainerRemoveView struct {
	ID        string
	IsSuccess bool
	Err       error
}

type ContainerLogsView struct {
	ID        string
	IsSuccess bool
	Err       error
	Logs      []string
}

func (v *ViewFinder) GetContainer(ctx context.Context, containerID string) (*ContainerView, error) {
	containerInfo, err := v.docker.ContainerInspect(ctx, containerID)
	return &ContainerView{
		ContainerInfo: containerInfo,
		IsSuccess:     err == nil,
		Err:           err,
	}, nil
}

func (v *ViewFinder) StartContainer(ctx context.Context, containerID string) (*ContainerStartView, error) {
	opts := container.StartOptions{}
	err := v.docker.ContainerStart(ctx, containerID, opts)
	return &ContainerStartView{
		ID:        containerID,
		IsSuccess: err == nil,
		Err:       err,
	}, nil
}

func (v *ViewFinder) StopContainer(ctx context.Context, containerID string) (*ContainerStopView, error) {
	opts := container.StopOptions{}
	err := v.docker.ContainerStop(ctx, containerID, opts)
	return &ContainerStopView{
		ID:        containerID,
		IsSuccess: err == nil,
		Err:       err,
	}, nil
}

func (v *ViewFinder) RestartContainer(ctx context.Context, containerID string) (*ContainerRestartView, error) {
	opts := container.StopOptions{}
	err := v.docker.ContainerRestart(ctx, containerID, opts)
	return &ContainerRestartView{
		ID:        containerID,
		IsSuccess: err == nil,
		Err:       err,
	}, nil
}

func (v *ViewFinder) RemoveContainer(ctx context.Context, containerID string) (*ContainerRemoveView, error) {
	opts := container.RemoveOptions{}
	err := v.docker.ContainerRemove(ctx, containerID, opts)
	return &ContainerRemoveView{
		ID:        containerID,
		IsSuccess: err == nil,
		Err:       err,
	}, err
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
