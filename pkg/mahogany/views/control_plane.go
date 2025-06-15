package views

import (
	"context"
	"net/http"
)

type ControlPlaneView struct {
	Status *StatusView
}

func (v *ControlPlaneView) Name() string         { return "ControlPlaneView" }
func (v *ControlPlaneView) Headers() http.Header { return http.Header{} }

func (v *ViewFinder) GetControlPlane(ctx context.Context) *ControlPlaneView {
	view := &ControlPlaneView{
		Status: v.GetStatus(ctx),
	}
	return view
}
