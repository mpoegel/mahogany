package sources

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"

	db "github.com/mpoegel/mahogany/internal/db"
	schema "github.com/mpoegel/mahogany/pkg/schema"
	grpc "google.golang.org/grpc"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

const ALL_HOSTS = "*"

type UpdateServerI interface {
	GetNumConnections() int
}

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
	db            *sql.DB
	query         *db.Queries
}

func NewUpdateServer(topologyFile string, port int, dbConn *sql.DB) (*UpdateServer, error) {
	topo, err := schema.ReadTopology(topologyFile)
	if err != nil {
		return nil, err
	}

	s := &UpdateServer{
		topology:       topo,
		githubPackages: make(map[string]*schema.Package),
		packageToHost:  make(map[string]map[string]bool),
		port:           port,
		releaseBroker:  NewBroker[*schema.Release](),
		isClosed:       false,
		db:             dbConn,
		query:          db.New(dbConn),
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
	services, err := s.query.ListWatchedServices(ctx)
	resp := &schema.RegisterManifestResponse{
		SubscribeToDocker:  true,
		SubscribeToSystemd: true,
	}
	if err != nil {
		slog.Warn("cannot list watched services", "err", err)
	} else {
		resp.SubscribeToServices = services
	}
	return resp, nil
}

func (s *UpdateServer) ReleaseStream(req *schema.ReleaseStreamRequest, stream schema.UpdateService_ReleaseStreamServer) error {
	c := s.releaseBroker.Subscribe()
	if c == nil {
		return errors.New("subscription unavailable")
	}
	slog.Info("new release stream")
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

func (s *UpdateServer) ServicesStream(stream grpc.BidiStreamingServer[schema.ServicesStreamRequest, schema.ServicesStreamResponse]) error {
	slog.Info("new services stream")
	deviceID := int64(-1)
	trackedServices := map[string]int64{}
	for {
		msg, err := stream.Recv()
		if err != nil {
			slog.Warn("error receiving from services stream", "err", err)
			return nil
		}
		for _, svc := range msg.Services {
			if deviceID == -1 {
				deviceID, err = s.getDeviceID(stream.Context(), msg)
				if err != nil {
					slog.Warn("services stream from unregistered device", "hostname", msg.Hostname, "err", err)
					break
				}
			}
			slog.Info("got tracked service report", "svc", svc)
			serviceID, ok := trackedServices[svc.Name]
			if !ok {
				serviceID, err = s.query.GetTrackedServiceID(stream.Context(), db.GetTrackedServiceIDParams{Name: svc.Name, DeviceID: deviceID})
				if err != nil {
					serviceID, err = s.addTrackedService(stream.Context(), deviceID, svc)
					if err != nil {
						slog.Warn("cannot add tracked service", "err", err, "svc", svc)
						continue
					}
				}
				trackedServices[svc.Name] = serviceID
			} else {
				if err = s.updateTrackedService(stream.Context(), serviceID, svc); err != nil {
					slog.Warn("cannot update tracked service", "id", serviceID, "device", deviceID, "err", err, "svc", svc)
				}
			}
		}
	}
}

func (s *UpdateServer) getDeviceID(ctx context.Context, msg *schema.ServicesStreamRequest) (int64, error) {
	deviceID, err := s.query.GetDevice(ctx, msg.Hostname)
	if err != nil {
		return -1, err
	}
	return deviceID.ID, nil
}

func (s *UpdateServer) addTrackedService(ctx context.Context, deviceID int64, svc *schema.ServiceStatus) (int64, error) {
	args := db.AddTrackedServiceParams{
		DeviceID:    deviceID,
		Name:        svc.Name,
		LastUpdated: timestamppb.Now().Seconds,
	}
	switch s := svc.Service.(type) {
	case *schema.ServiceStatus_DockerService:
		args.ContainerID = sql.NullString{String: s.DockerService.Id, Valid: true}
		args.ContainerImage = sql.NullString{String: s.DockerService.Image, Valid: true}
		args.Status = s.DockerService.Status
	case *schema.ServiceStatus_SystemdService:
		args.Status = s.SystemdService.ActiveState
	}
	trackedSvc, err := s.query.AddTrackedService(ctx, args)
	if err != nil {
		return -1, err
	}
	slog.Info("tracked service added", "svc", trackedSvc)
	return trackedSvc.ID, nil
}

func (s *UpdateServer) updateTrackedService(ctx context.Context, serviceID int64, svc *schema.ServiceStatus) error {
	args := db.UpdateTrackedServiceParams{
		ID:          serviceID,
		LastUpdated: timestamppb.Now().Seconds,
	}
	switch s := svc.Service.(type) {
	case *schema.ServiceStatus_DockerService:
		args.Status = s.DockerService.Status
	case *schema.ServiceStatus_SystemdService:
		args.Status = s.SystemdService.ActiveState
	}
	slog.Info("updated tracked service", "args", args)
	err := s.query.UpdateTrackedService(ctx, args)
	return err
}

func (s *UpdateServer) GetNumConnections() int {
	return s.releaseBroker.Count()
}
