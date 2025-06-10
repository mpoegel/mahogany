package views

import "context"

type ControlPlaneView struct {
	Status *StatusView
}

func (v *ControlPlaneView) Name() string { return "ControlPlaneView" }

func (v *ViewFinder) GetControlPlane(ctx context.Context) *ControlPlaneView {
	view := &ControlPlaneView{
		Status: v.GetStatus(ctx),
	}
	return view
}
