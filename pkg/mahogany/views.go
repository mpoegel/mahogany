package mahogany

import (
	"context"
	"fmt"
	"io"

	types "github.com/docker/docker/api/types"
	container "github.com/docker/docker/api/types/container"
)

type IndexView struct {
	Containers []types.Container
}

type ContainerView struct {
	ContainerInfo types.ContainerJSON
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

type RegistryView struct {
	Manifests []RegistryManifest
	IsSuccess bool
	Err       error
}

type ActionResponseView struct {
	IsSuccess bool
	Message   string
}

type WatchtowerView struct {
}

type ViewFinder struct {
	docker     DockerI
	registry   RegistryI
	watchtower WatchtowerI
}

func NewViewFinder(config Config) (*ViewFinder, error) {
	docker, err := NewDocker(config.DockerHost, config.DockerVersion)
	if err != nil {
		return nil, err
	}

	return &ViewFinder{
		docker:     docker,
		registry:   NewRegistry(config.RegistryAddr),
		watchtower: NewWatchtower(config.WatchtowerAddr, config.WatchtowerToken),
	}, nil
}

func (v *ViewFinder) GetIndex(ctx context.Context) (*IndexView, error) {
	opts := container.ListOptions{
		All: true,
	}
	containerList, err := v.docker.ContainerList(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &IndexView{
		Containers: containerList,
	}, nil
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

func (v *ViewFinder) GetRegistry(ctx context.Context) (*RegistryView, error) {
	view := &RegistryView{
		Manifests: make([]RegistryManifest, 0),
		IsSuccess: false,
	}
	catalog, err := v.registry.GetCatalog()
	if err != nil {
		view.Err = err
		return view, nil
	}
	for _, repository := range catalog.Repositories {
		tags, err := v.registry.GetTags(repository)
		if err != nil {
			view.Err = err
			return view, nil
		}
		for _, tag := range tags.Tags {
			manifest, err := v.registry.GetManifest(tags.Name, tag)
			if err != nil {
				view.Err = err
				return view, nil
			}
			view.Manifests = append(view.Manifests, *manifest)
		}
	}
	view.IsSuccess = true
	return view, nil
}

func (v *ViewFinder) DeleteRegistryImage(ctx context.Context, repository, tag string) (*ActionResponseView, error) {
	view := &ActionResponseView{
		IsSuccess: false,
	}
	if err := v.registry.DeleteImage(repository, tag); err != nil {
		view.Message = fmt.Sprintf("Failed to delete image: %v", err)
	} else {
		view.IsSuccess = true
		view.Message = fmt.Sprintf("Deleted image %s:%s", repository, tag)
	}
	return view, nil
}

func (v *ViewFinder) GetWatchtower(ctx context.Context) (*WatchtowerView, error) {
	return &WatchtowerView{}, nil
}

func (v *ViewFinder) WatchtowerUpdate(ctx context.Context) *ActionResponseView {
	view := &ActionResponseView{
		IsSuccess: false,
	}
	if err := v.watchtower.Update(ctx); err != nil {
		view.Message = fmt.Sprintf("Update request failed: %v", err)
	} else {
		view.IsSuccess = true
		view.Message = "Update complete"
	}
	return view
}
