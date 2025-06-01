package mahogany

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"time"

	types "github.com/docker/docker/api/types"
	container "github.com/docker/docker/api/types/container"
	db "github.com/mpoegel/mahogany/internal/db"
	vpn "github.com/mpoegel/mahogany/pkg/vpn"
	_ "modernc.org/sqlite"
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

type ControlPlaneView struct {
}

type SettingsView struct {
	WatchtowerAddr    string
	WatchtowerToken   string
	WatchtowerTimeout string
	RegistryAddr      string
	RegistryTimeout   string
	TailscaleApiKey   string
	TailnetName       string
}

type DevicesView struct {
	Devices   []vpn.Device
	Policy    *vpn.NetPolicy
	IsSuccess bool
	Err       error
}

type DeviceView struct {
	Device       *vpn.Device
	SourcePolicy *vpn.NetPolicy
	DestPolicy   *vpn.NetPolicy
	IsSuccess    bool
	Err          error
}

type ViewFinder struct {
	docker       DockerI
	registry     RegistryI
	watchtower   WatchtowerI
	deviceFinder vpn.VirtualNetworkClient
	db           *sql.DB
	query        *db.Queries
}

func NewViewFinder(config Config) (*ViewFinder, error) {
	docker, err := NewDocker(config.DockerHost, config.DockerVersion)
	if err != nil {
		return nil, err
	}

	dbConn, err := sql.Open("sqlite", config.DbFile)
	if err != nil {
		return nil, err
	}

	vf := &ViewFinder{
		docker: docker,
		db:     dbConn,
		query:  db.New(dbConn),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := vf.reload(ctx, vf.query); err != nil {
		return nil, err
	}

	return vf, nil
}

func (v *ViewFinder) reload(ctx context.Context, query *db.Queries) error {
	if query == nil {
		query = v.query
	}
	registryTimeout, err := time.ParseDuration(v.getSetting(ctx, query, "RegistryTimeout"))
	if err != nil {
		return err
	}
	watchtowerTimeout, err := time.ParseDuration(v.getSetting(ctx, query, "WatchtowerTimeout"))
	if err != nil {
		return err
	}

	v.registry = NewRegistry(v.getSetting(ctx, query, "RegistryAddr"), registryTimeout)
	v.watchtower = NewWatchtower(v.getSetting(ctx, query, "WatchtowerAddr"), v.getSetting(ctx, query, "WatchtowerToken"), watchtowerTimeout)
	v.deviceFinder = vpn.NewClient(v.getSetting(ctx, query, "TailscaleApiKey"), v.getSetting(ctx, query, "TailnetName"))

	return nil
}

func (v *ViewFinder) getSetting(ctx context.Context, query *db.Queries, name string) string {
	row, err := query.GetSetting(ctx, name)
	if err != nil {
		slog.Warn("failed to get setting", "err", err, "name", name)
	}
	return row.Value
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
	catalog, err := v.registry.GetCatalog(ctx)
	if err != nil {
		view.Err = err
		return view, nil
	}
	for _, repository := range catalog.Repositories {
		tags, err := v.registry.GetTags(ctx, repository)
		if err != nil {
			view.Err = err
			return view, nil
		}
		for _, tag := range tags.Tags {
			manifest, err := v.registry.GetManifest(ctx, tags.Name, tag)
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
	if err := v.registry.DeleteImage(ctx, repository, tag); err != nil {
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

func (v *ViewFinder) GetControlPlane(ctx context.Context) (*ControlPlaneView, error) {
	view := &ControlPlaneView{}
	return view, nil
}

func (v *ViewFinder) GetSettings(ctx context.Context) (*SettingsView, error) {
	view := &SettingsView{
		WatchtowerAddr:    v.getSetting(ctx, v.query, "WatchtowerAddr"),
		WatchtowerToken:   v.getSetting(ctx, v.query, "WatchtowerToken"),
		WatchtowerTimeout: v.getSetting(ctx, v.query, "WatchtowerTimeout"),
		RegistryAddr:      v.getSetting(ctx, v.query, "RegistryAddr"),
		RegistryTimeout:   v.getSetting(ctx, v.query, "RegistryTimeout"),
		TailscaleApiKey:   v.getSetting(ctx, v.query, "TailscaleApiKey"),
		TailnetName:       v.getSetting(ctx, v.query, "TailnetName"),
	}
	return view, nil
}

func (v *ViewFinder) PostSettings(ctx context.Context, params db.UpdateSettingParams) error {
	tx, err := v.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	query := v.query.WithTx(tx)
	if err := query.UpdateSetting(ctx, params); err != nil {
		return err
	}
	if err = v.reload(ctx, query); err != nil {
		if err := tx.Rollback(); err != nil {
			slog.Warn("failed to rollback settings transaction", "err", err)
		}
		return err
	}
	return tx.Commit()
}

func (v *ViewFinder) GetDevices(ctx context.Context) (*DevicesView, error) {
	view := &DevicesView{}
	devices, err := v.deviceFinder.ListDevices(ctx)
	if err != nil {
		slog.Error("list devices failed", "err", err)
		view.IsSuccess = false
		view.Err = err
		return view, nil
	}
	view.Devices = devices
	policy, err := v.deviceFinder.GetACL(ctx)
	if err != nil {
		slog.Error("get policy failed", "err", err)
		view.IsSuccess = false
		view.Err = err
		return view, nil
	}
	view.Policy = policy
	return view, nil
}

func (v *ViewFinder) GetDevice(ctx context.Context, deviceID string) (*DeviceView, error) {
	view := &DeviceView{}
	device, err := v.deviceFinder.GetDevice(ctx, deviceID)
	if err != nil {
		slog.Error("get device failed", "err", err)
		view.IsSuccess = false
		view.Err = err
		return view, nil
	}
	policy, err := v.deviceFinder.GetACL(ctx)
	if err != nil {
		slog.Error("get policy failed", "err", err)
		view.IsSuccess = false
		view.Err = err
		return view, nil
	}

	sourceACL := vpn.NetPolicy{
		ACLs: make([]vpn.ACL, 0),
	}
	for _, acl := range policy.ACLs {
		for _, tag := range device.Tags {
			if slices.Contains(acl.Source, tag) {
				sourceACL.ACLs = append(sourceACL.ACLs, acl)
			}
		}
	}

	destACL := vpn.NetPolicy{
		ACLs: make([]vpn.ACL, 0),
	}
	for _, acl := range policy.ACLs {
		for _, tag := range device.Tags {
			if slices.ContainsFunc(acl.Destination, func(dest string) bool { return strings.HasPrefix(dest, tag) }) {
				destACL.ACLs = append(destACL.ACLs, acl)
			}
		}
	}

	view.Device = device
	view.SourcePolicy = &sourceACL
	view.DestPolicy = &destACL
	return view, nil
}
