package views

import (
	"context"
	"log/slog"
	"strconv"

	db "github.com/mpoegel/mahogany/internal/db"
)

type PackagesView struct {
	TemplateName string
	IsSuccess    bool
	Err          error
	Packages     []db.Package
	AddPackage   db.AddPackageParams
	Toast        string
}

func (v *PackagesView) Name() string { return v.TemplateName }
func (v *PackagesView) WithName(name string) *PackagesView {
	v.TemplateName = name
	return v
}

func (vf *ViewFinder) GetPackages(ctx context.Context) *PackagesView {
	view := &PackagesView{
		IsSuccess: true,
	}
	packages, err := vf.query.ListPackages(ctx)
	if err != nil {
		slog.Error("failed to list packages", "err", err)
		view.IsSuccess = false
		view.Err = err
	} else {
		view.Packages = packages
	}

	return view
}

func (vf *ViewFinder) AddPackage(ctx context.Context, params db.AddPackageParams) *PackagesView {
	view := vf.GetPackages(ctx)
	if !view.IsSuccess {
		return view
	}
	view.AddPackage = params

	if len(params.Name) == 0 {
		view.Toast = "Name required"
		return view
	}
	if len(params.InstallCmd) == 0 {
		view.Toast = "Install command required"
		return view
	}
	if len(params.UpdateCmd) == 0 {
		view.Toast = "Update command required"
		return view
	}

	pkg, err := vf.query.AddPackage(ctx, params)
	if err != nil {
		slog.Error("failed to add package", "params", params, "err", err)
		view.IsSuccess = false
		view.Err = err
		return view
	} else {
		view.AddPackage = db.AddPackageParams{}
		view.Packages = append(view.Packages, pkg)
	}
	return view
}

func (vf *ViewFinder) DeletePackage(ctx context.Context, id string) *PackagesView {
	view := vf.GetPackages(ctx)
	if !view.IsSuccess {
		view.Toast = view.Err.Error()
		return view
	}

	packageID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		view.IsSuccess = false
		view.Toast = err.Error()
		return view
	}

	if err = vf.query.DeletePackage(ctx, packageID); err != nil {
		slog.Error("failed to delete package", "id", packageID, "err", err)
		view.IsSuccess = false
		view.Toast = err.Error()
		return view
	}
	return vf.GetPackages(ctx)
}
