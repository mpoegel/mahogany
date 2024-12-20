package mahogany

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"

	schema "github.com/mpoegel/mahogany/pkg/schema"
	grpc "google.golang.org/grpc"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

const ALL_HOSTS = "*"

type UpdateServer struct {
	schema.UnimplementedUpdateServiceServer

	port int

	topology *schema.Topology
	// map of github full name to package
	githubPackages map[string]*schema.Package
	// map of package name to set of host names
	packageToHost map[string]map[string]bool

	releaseBroker *Broker[*schema.Release]
	ln            net.Listener
	isClosed      bool
}

func NewUpdateServer(config Config) (*UpdateServer, error) {
	topo, err := schema.ReadTopology(config.TopologyFile)
	if err != nil {
		return nil, err
	}

	s := &UpdateServer{
		topology:       topo,
		githubPackages: make(map[string]*schema.Package),
		packageToHost:  make(map[string]map[string]bool),
		port:           config.Port + 1,
		releaseBroker:  NewBroker[*schema.Release](),
		isClosed:       false,
	}

	for _, pack := range s.topology.Baseline {
		if pack.GithubPackage != nil {
			s.githubPackages[pack.ID] = &pack
		}
		s.packageToHost[pack.ID] = map[string]bool{ALL_HOSTS: true}
	}
	for _, host := range s.topology.HostPackages {
		for _, pack := range host.Packages {
			if pack.GithubPackage != nil {
				s.githubPackages[pack.ID] = &pack
			}
			packOnHost, ok := s.packageToHost[pack.ID]
			if !ok {
				s.packageToHost[pack.ID] = map[string]bool{}
				packOnHost = s.packageToHost[pack.ID]
			}
			packOnHost[host.HostName] = true
		}
		for _, packName := range host.Skipped {
			packOnHost, ok := s.packageToHost[packName]
			if !ok {
				s.packageToHost[packName] = map[string]bool{}
				packOnHost = s.packageToHost[packName]
			}
			packOnHost[host.HostName] = true
		}
	}

	return s, nil
}

func (s *UpdateServer) Start(ctx context.Context) error {
	lnConfig := net.ListenConfig{}

	addr := fmt.Sprintf(":%d", s.port)
	ln, err := lnConfig.Listen(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	s.ln = ln
	slog.Info("update server listening", "addr", addr)

	go s.releaseBroker.Start()

	grpcServer := grpc.NewServer()
	schema.RegisterUpdateServiceServer(grpcServer, s)
	if err := grpcServer.Serve(ln); err != nil && !s.isClosed {
		return err
	}
	return nil
}

func (s *UpdateServer) Stop() {
	s.isClosed = true
	if s.ln != nil {
		s.ln.Close()
	}
	s.releaseBroker.Stop()
}

func (s *UpdateServer) PropagateGithubRelease(event *GithubReleaseEvent) {
	pack, ok := s.githubPackages[*event.Repo.Name]
	if !ok {
		slog.Warn("github package not in topology", "name", *event.Repo.Name)
		return
	}

	release := &schema.Release{
		Name:           *event.Repo.Name,
		Version:        *event.Release.Name,
		RepositoryName: *event.Repo.FullName,
		Assets:         make([]*schema.Asset, 0),
		InstallCommand: pack.InstallCommand,
	}

	for _, asset := range event.Release.Assets {
		if pack.GithubPackage.Regex.MatchString(*asset.Name) {
			asset := &schema.Asset{
				Name:      *asset.Name,
				SourceUrl: *asset.BrowserDownloadURL,
			}
			sourceMask := fmt.Sprintf("https://github.com/%s/releases", pack.GithubPackage.Name)
			if !strings.HasPrefix(asset.SourceUrl, sourceMask) {
				slog.Warn("asset has suspicious download url", "url", asset.SourceUrl)
				continue
			}
			release.Assets = append(release.Assets, asset)
		}
	}
	if len(release.Assets) == 0 {
		slog.Warn("no release assets matched", "name", *event.Repo.Name, "version", *event.Release.Name)
		return
	}

	s.releaseBroker.Broadcast(release)
	slog.Info("release broadcasted", "name", *event.Repo.Name, "version", *event.Release.Name)
}

func (s *UpdateServer) RegisterManifest(ctx context.Context, req *schema.RegisterManifestRequest) (*schema.RegisterManifestResponse, error) {
	slog.Info("got register manifest request", "hostname", req.Hostname)
	resp := &schema.RegisterManifestResponse{}
	return resp, nil
}

func (s *UpdateServer) ReleaseStream(req *schema.ReleaseStreamRequest, stream schema.UpdateService_ReleaseStreamServer) error {
	c := s.releaseBroker.Subscribe()
	if c == nil {
		return errors.New("subscription unavailable")
	}
	defer s.releaseBroker.Unsubscribe(c)
	for {
		release := <-c
		if release == nil {
			return nil
		}
		slog.Info("checking release", "hostname", req.Hostname, "release", release.Name)
		pack, ok := s.packageToHost[release.Name]
		if ok && (pack[req.Hostname] || pack[ALL_HOSTS]) {
			resp := &schema.ReleaseStreamResponse{
				Release:   release,
				Timestamp: timestamppb.Now(),
			}
			if err := stream.Send(resp); err != nil {
				slog.Warn("failed to send release stream response", "err", err)
				return err
			} else {
				slog.Info("sent out release", "name", release.Name, "hostname", req.Hostname)
			}
		}
	}
}
