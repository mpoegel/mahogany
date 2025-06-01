package views

import "context"

type ControlPlaneView struct {
}

func (v *ViewFinder) GetControlPlane(ctx context.Context) (*ControlPlaneView, error) {
	view := &ControlPlaneView{}
	return view, nil
}
