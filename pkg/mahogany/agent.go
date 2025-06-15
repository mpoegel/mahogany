package mahogany

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"
	"time"

	dbus "github.com/coreos/go-systemd/v22/dbus"
	container "github.com/docker/docker/api/types/container"
	sources "github.com/mpoegel/mahogany/pkg/mahogany/sources"
	schema "github.com/mpoegel/mahogany/pkg/schema"
	grpc "google.golang.org/grpc"
	insecure "google.golang.org/grpc/credentials/insecure"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type Agent struct {
	config AgentConfig

	conn   *grpc.ClientConn
	client schema.UpdateServiceClient

	registration *schema.RegisterManifestResponse
}

func NewAgent(config AgentConfig) (*Agent, error) {
	conn, err := grpc.NewClient(config.ServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	a := &Agent{
		config: config,
		conn:   conn,
		client: schema.NewUpdateServiceClient(conn),
	}

	return a, nil
}

func (a *Agent) Run(ctx context.Context) error {
	tctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := a.register(tctx); err != nil {
		return err
	}

	metrics, err := NewMetrics(10*time.Second, map[string]string{"hostname": a.config.HostName})
	if err != nil {
		return err
	}
	go metrics.Collect(ctx)
	go a.reportServices(ctx)

	req := &schema.ReleaseStreamRequest{
		Hostname: a.config.HostName,
	}
	stream, err := a.client.ReleaseStream(ctx, req)
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err != nil && errors.Is(err, io.EOF) {
			slog.Info("stream closed")
			return nil
		}
		if err != nil {
			return err
		}
		slog.Info("got release notification", "resp", resp)
		// TODO reply with installation status
		a.installRelease(resp.Release)
	}
}

func (a *Agent) register(ctx context.Context) error {
	req := &schema.RegisterManifestRequest{
		Hostname:  a.config.HostName,
		Timestamp: timestamppb.Now(),
		Assets:    []*schema.Asset{},
	}
	resp, err := a.client.RegisterManifest(ctx, req)
	if err != nil {
		return err
	}
	slog.Info("registered with server", "resp", resp)
	a.registration = resp
	return nil
}

func (a *Agent) installRelease(release *schema.Release) bool {
	downloadDir := path.Join(a.config.DownloadDir, release.Name)
	if err := os.MkdirAll(downloadDir, 0777); err != nil {
		slog.Error("could not create download directory", "dir", downloadDir, "err", err)
		return false
	}
	for _, asset := range release.Assets {
		resp, err := http.Get(asset.SourceUrl)
		if err != nil {
			slog.Error("could not download asset", "asset", asset.Name, "source", asset.SourceUrl)
			return false
		}
		if resp.StatusCode != http.StatusOK {
			slog.Error("got bad response downloading asset", "asset", asset.Name, "status", resp.Status)
			return false
		}
		filename := path.Join(downloadDir, asset.Name)
		fp, err := os.Create(filename)
		if err != nil {
			slog.Error("could not create asset file", "file", filename, "err", err)
			return false
		}
		defer fp.Close()
		_, err = io.Copy(fp, resp.Body)
		if err != nil {
			slog.Error("could not write asset to file", "asset", asset.Name, "err", err)
			return false
		}
		slog.Info("asset downloaded", "file", filename)

		rawInstallCmd := strings.ReplaceAll(release.InstallCommand, "{}", filename)
		// TODO add command timeout
		cmds := strings.Split(rawInstallCmd, ";")
		for _, rawCmd := range cmds {
			cmdParts := strings.Split(rawCmd, " ")
			cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
			out := &strings.Builder{}
			cmd.Stdout = out
			cmd.Stderr = out
			if err = cmd.Run(); err != nil {
				slog.Error("install failed", "asset", asset.Name, "err", err, "output", out.String())
				return false
			}
		}
		slog.Info("install completed", "asset", asset.Name)
	}

	return true
}

func (a *Agent) reportServices(ctx context.Context) {
	stream, err := a.client.ServicesStream(ctx)
	if err != nil {
		slog.Error("failed to open services stream", "err", err)
		return
	}
	defer stream.CloseSend()

	const dockerVersion = "3"
	var dockerClient sources.DockerI
	if a.registration.SubscribeToDocker {
		dockerClient, err = sources.NewDocker("localhost", dockerVersion)
		if err != nil {
			slog.Error("failed to create docker client", "err", err)
			return
		}
	}

	var dbusConn *dbus.Conn
	if a.registration.SubscribeToSystemd {
		dbusConn, err = dbus.NewSystemdConnectionContext(ctx)
		if err != nil {
			slog.Error("failed to open dbus connection", "err", err)
			return
		}
		defer dbusConn.Close()
	}

	frequency := 30 * time.Second
	ticker := time.NewTicker(frequency)
	for {
		subCtx, cancel := context.WithTimeout(ctx, frequency)
		req := &schema.ServicesStreamRequest{
			Hostname:  a.config.HostName,
			Timestamp: timestamppb.Now(),
		}

		if a.registration.SubscribeToDocker {
			containers, err := dockerClient.ContainerList(subCtx, container.ListOptions{})
			if err != nil {
				slog.Warn("failed to list containers", "err", err)
			} else {
				for _, container := range containers {
					req.Services = append(req.Services, &schema.ServiceStatus{
						Name: container.Names[0],
						Service: &schema.ServiceStatus_DockerService{
							DockerService: &schema.ServiceDocker{
								Command: container.Command,
								Created: container.Created,
								Id:      container.ID,
								Image:   container.Image,
								ImageID: container.ImageID,
								Names:   container.Names,
								Ports: slices.Collect(func(yield func(uint32) bool) {
									for _, p := range container.Ports {
										if !yield(uint32(p.PublicPort)) {
											return
										}
									}
								}),
								State:  container.State,
								Status: container.Status,
							},
						},
					})
				}
			}
		}

		if a.registration.SubscribeToSystemd {
			units, err := dbusConn.ListUnitsByNamesContext(subCtx, a.registration.SubscribeToServices)
			if err != nil {
				slog.Warn("failed to list systemd units", "err", err)
			} else {
				for _, unit := range units {
					req.Services = append(req.Services, &schema.ServiceStatus{
						Name: unit.Name,
						Service: &schema.ServiceStatus_SystemdService{
							SystemdService: &schema.ServiceSystemd{
								Name:        unit.Name,
								Description: unit.Description,
								LoadState:   unit.LoadState,
								ActiveState: unit.ActiveState,
								Path:        string(unit.Path),
							},
						},
					})
					slog.Info("service", "unit", unit)
				}
			}
		}

		if err := stream.Send(req); err != nil {
			slog.Warn("failed to send services stream", "err", err, "msg", req)
		}

		cancel()
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (a *Agent) Close() {
	if a.conn != nil {
		a.conn.Close()
	}
}
