package views

import (
	"context"
	"fmt"
	"net/http"
)

type WatchtowerView struct {
	Status *StatusView
}

func (v *WatchtowerView) Name() string         { return "WatchtowerView" }
func (v *WatchtowerView) Headers() http.Header { return http.Header{} }

func (v *ViewFinder) GetWatchtower(ctx context.Context) *WatchtowerView {
	return &WatchtowerView{
		Status: v.GetStatus(ctx),
	}
}

func (v *ViewFinder) WatchtowerUpdate(ctx context.Context) *ActionResponseView {
	view := &ActionResponseView{
		IsSuccess: false,
	}
	if err := v.watchtower.Update(ctx); err != nil {
		view.Toast = fmt.Sprintf("Update request failed: %v", err)
	} else {
		view.IsSuccess = true
		view.Toast = "Update complete"
	}
	return view
}
