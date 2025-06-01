package views

import (
	"context"
	"log/slog"
	"slices"
	"strings"

	"github.com/mpoegel/mahogany/pkg/vpn"
)

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
