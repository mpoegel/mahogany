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
	"strings"
	"time"

	schema "github.com/mpoegel/mahogany/pkg/schema"
	grpc "google.golang.org/grpc"
	insecure "google.golang.org/grpc/credentials/insecure"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type Agent struct {
	config AgentConfig

	conn   *grpc.ClientConn
	client schema.UpdateServiceClient
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

func (a *Agent) Close() {
	if a.conn != nil {
		a.conn.Close()
	}
}
