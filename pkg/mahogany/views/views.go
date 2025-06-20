package views

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	types "github.com/docker/docker/api/types"
	container "github.com/docker/docker/api/types/container"
	db "github.com/mpoegel/mahogany/internal/db"
	sources "github.com/mpoegel/mahogany/pkg/mahogany/sources"
	vpn "github.com/mpoegel/mahogany/pkg/vpn"
	_ "modernc.org/sqlite"
)

type StatusView struct {
	NumAgents           int
	NumDevices          int64
	RegistryConnected   bool
	WatchtowerConnected bool
}

type IndexView struct {
	Status     *StatusView
	Containers []types.Container
}

func (v *IndexView) Name() string         { return "IndexView" }
func (v *IndexView) Headers() http.Header { return http.Header{} }

type ActionResponseView struct {
	IsSuccess bool
	Toast     string

	headers http.Header
}

func (v *ActionResponseView) Name() string         { return "toast" }
func (v *ActionResponseView) Headers() http.Header { return v.headers }

type ViewFinder struct {
	docker       sources.DockerI
	registry     sources.RegistryI
	watchtower   sources.WatchtowerI
	updateServer sources.UpdateServerI
	deviceFinder vpn.VirtualNetworkClient
	db           *sql.DB
	query        *db.Queries
}

func NewViewFinder(dockerHost, dockerVersion string, dbConn *sql.DB, updateServer sources.UpdateServerI) (*ViewFinder, error) {
	docker, err := sources.NewDocker(dockerHost, dockerVersion)
	if err != nil {
		return nil, err
	}

	vf := &ViewFinder{
		docker:       docker,
		db:           dbConn,
		query:        db.New(dbConn),
		updateServer: updateServer,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := vf.reload(ctx, vf.query); err != nil {
		return nil, err
	}

	return vf, nil
}

func (v *ViewFinder) GetIndex(ctx context.Context) *IndexView {
	view := &IndexView{
		Status: v.GetStatus(ctx),
	}
	opts := container.ListOptions{
		All: true,
	}
	containerList, err := v.docker.ContainerList(ctx, opts)
	if err != nil {
		slog.Error("failed to get docker container list", "err", err)
	} else {
		view.Containers = containerList
	}
	slog.Info("loaded index", "view", view.Status)
	return view
}

func (v *ViewFinder) GetStatus(ctx context.Context) *StatusView {
	view := &StatusView{
		NumAgents:           v.updateServer.GetNumConnections(),
		NumDevices:          0,
		RegistryConnected:   v.registry.Status(ctx) == nil,
		WatchtowerConnected: v.watchtower.Status(ctx) == nil,
	}
	view.NumDevices, _ = v.query.CountDevices(ctx)
	return view
}
