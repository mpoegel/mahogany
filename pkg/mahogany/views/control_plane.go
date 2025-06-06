package views

import "context"

type ControlPlaneView struct {
}

func (v *ControlPlaneView) Name() string { return "ControlPlaneView" }

func (v *ViewFinder) GetControlPlane(ctx context.Context) *ControlPlaneView {
	view := &ControlPlaneView{}
	return view
}
