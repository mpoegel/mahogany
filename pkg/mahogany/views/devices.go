package views

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	db "github.com/mpoegel/mahogany/internal/db"
	vpn "github.com/mpoegel/mahogany/pkg/vpn"
)

type DevicesView struct {
	Devices   []vpn.Device
	Policy    *vpn.NetPolicy
	IsSuccess bool
	Err       error
	Status    *StatusView
}

func (v *DevicesView) Name() string         { return "DevicesView" }
func (v *DevicesView) Headers() http.Header { return http.Header{} }

type DeviceView struct {
	Device       *vpn.Device
	SourcePolicy *vpn.NetPolicy
	DestPolicy   *vpn.NetPolicy
	Assets       []DeviceAsset
	AllPackages  []db.Package
	IsSuccess    bool
	Err          error
}

func (v *DeviceView) Name() string         { return "DeviceView" }
func (v *DeviceView) Headers() http.Header { return http.Header{} }

type DeviceAsset struct {
	Name    string
	Version string
}

func (v *ViewFinder) syncDevices(ctx context.Context, devices []vpn.Device) error {
	var allErrs error
	for _, device := range devices {
		_, err := v.query.AddDevice(ctx, device.Hostname)
		allErrs = errors.Join(allErrs, err)
	}
	return allErrs
}

func (v *ViewFinder) GetDevices(ctx context.Context) *DevicesView {
	view := &DevicesView{}
	devices, err := v.deviceFinder.ListDevices(ctx)
	if err != nil {
		slog.Error("list devices failed", "err", err)
		view.IsSuccess = false
		view.Err = err
		return view
	}
	v.syncDevices(ctx, devices)
	view.Devices = devices
	view.Status = v.GetStatus(ctx)
	policy, err := v.deviceFinder.GetACL(ctx)
	if err != nil {
		slog.Error("get policy failed", "err", err)
		view.IsSuccess = false
		view.Err = err
		return view
	}
	view.Policy = policy
	return view
}

func (v *ViewFinder) GetDevice(ctx context.Context, deviceID string) *DeviceView {
	view := &DeviceView{}
	device, err := v.deviceFinder.GetDevice(ctx, deviceID)
	if err != nil {
		slog.Error("get device failed", "err", err)
		view.IsSuccess = false
		view.Err = err
		return view
	}
	policy, err := v.deviceFinder.GetACL(ctx)
	if err != nil {
		slog.Error("get policy failed", "err", err)
		view.IsSuccess = false
		view.Err = err
		return view
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

	packages, err := v.query.ListPackages(ctx)
	if err != nil {
		slog.Error("list packages failed", "err", err)
		return view
	}
	view.AllPackages = packages

	return view
}
