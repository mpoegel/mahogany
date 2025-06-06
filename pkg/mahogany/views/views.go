package views

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	container "github.com/docker/docker/api/types/container"
	db "github.com/mpoegel/mahogany/internal/db"
	sources "github.com/mpoegel/mahogany/pkg/mahogany/sources"
	vpn "github.com/mpoegel/mahogany/pkg/vpn"
	_ "modernc.org/sqlite"
)

type IndexView struct {
	Containers []container.Summary
}

func (v *IndexView) Name() string { return "IndexView" }

type ActionResponseView struct {
	IsSuccess bool
	Message   string
}

func (v *ActionResponseView) Name() string { return "toast" }

type ViewFinder struct {
	docker       sources.DockerI
	registry     sources.RegistryI
	watchtower   sources.WatchtowerI
	deviceFinder vpn.VirtualNetworkClient
	db           *sql.DB
	query        *db.Queries
}

func NewViewFinder(dockerHost, dockerVersion, dbFile string) (*ViewFinder, error) {
	docker, err := sources.NewDocker(dockerHost, dockerVersion)
	if err != nil {
		return nil, err
	}

	dbConn, err := sql.Open("sqlite", dbFile)
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

func (v *ViewFinder) GetIndex(ctx context.Context) *IndexView {
	view := &IndexView{}
	opts := container.ListOptions{
		All: true,
	}
	containerList, err := v.docker.ContainerList(ctx, opts)
	if err != nil {
		slog.Error("failed to get docker container list", "err", err)
	} else {
		view.Containers = containerList
	}
	return view
}
