package views

import (
	"context"
	"fmt"

	sources "github.com/mpoegel/mahogany/pkg/mahogany/sources"
)

type RegistryView struct {
	Manifests []sources.RegistryManifest
	IsSuccess bool
	Err       error
}

func (v *ViewFinder) GetRegistry(ctx context.Context) (*RegistryView, error) {
	view := &RegistryView{
		Manifests: make([]sources.RegistryManifest, 0),
		IsSuccess: false,
	}
	catalog, err := v.registry.GetCatalog(ctx)
	if err != nil {
		view.Err = err
		return view, nil
	}
	for _, repository := range catalog.Repositories {
		tags, err := v.registry.GetTags(ctx, repository)
		if err != nil {
			view.Err = err
			return view, nil
		}
		for _, tag := range tags.Tags {
			manifest, err := v.registry.GetManifest(ctx, tags.Name, tag)
			if err != nil {
				view.Err = err
				return view, nil
			}
			view.Manifests = append(view.Manifests, *manifest)
		}
	}
	view.IsSuccess = true
	return view, nil
}

func (v *ViewFinder) DeleteRegistryImage(ctx context.Context, repository, tag string) (*ActionResponseView, error) {
	view := &ActionResponseView{
		IsSuccess: false,
	}
	if err := v.registry.DeleteImage(ctx, repository, tag); err != nil {
		view.Message = fmt.Sprintf("Failed to delete image: %v", err)
	} else {
		view.IsSuccess = true
		view.Message = fmt.Sprintf("Deleted image %s:%s", repository, tag)
	}
	return view, nil
}
