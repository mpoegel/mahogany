package views

import (
	"context"
	"fmt"
)

type WatchtowerView struct {
	Status *StatusView
}

func (v *WatchtowerView) Name() string { return "WatchtowerView" }

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
		view.Message = fmt.Sprintf("Update request failed: %v", err)
	} else {
		view.IsSuccess = true
		view.Message = "Update complete"
	}
	return view
}
