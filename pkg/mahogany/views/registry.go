package views

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	sources "github.com/mpoegel/mahogany/pkg/mahogany/sources"
)

type RegistryView struct {
	Manifests []sources.RegistryManifest
	IsSuccess bool
	Err       error
	Status    *StatusView
}

func (v *RegistryView) Name() string         { return "RegistryView" }
func (v *RegistryView) Headers() http.Header { return http.Header{} }

func (v *ViewFinder) GetRegistry(ctx context.Context) *RegistryView {
	view := &RegistryView{
		Manifests: make([]sources.RegistryManifest, 0),
		IsSuccess: false,
		Status:    v.GetStatus(ctx),
	}
	catalog, err := v.registry.GetCatalog(ctx)
	if err != nil {
		slog.Error("failed to get registry catalog", "err", err)
		view.Err = err
		return view
	}
	for _, repository := range catalog.Repositories {
		tags, err := v.registry.GetTags(ctx, repository)
		if err != nil {
			slog.Error("failed to get registry tags", "err", err, "repository", repository)
			view.Err = err
			return view
		}
		for _, tag := range tags.Tags {
			manifest, err := v.registry.GetManifest(ctx, tags.Name, tag)
			if err != nil {
				slog.Error("failed to get registry manifest", "err", err, "repository", tags.Name, "tag", tag)
				view.Err = err
				return view
			}
			view.Manifests = append(view.Manifests, *manifest)
		}
	}
	view.IsSuccess = true
	return view
}

func (v *ViewFinder) DeleteRegistryImage(ctx context.Context, repository, tag string) *ActionResponseView {
	view := &ActionResponseView{
		IsSuccess: false,
	}
	if err := v.registry.DeleteImage(ctx, repository, tag); err != nil {
		view.Toast = fmt.Sprintf("Failed to delete image: %v", err)
	} else {
		view.IsSuccess = true
		view.Toast = fmt.Sprintf("Deleted image %s:%s", repository, tag)
	}
	return view
}
