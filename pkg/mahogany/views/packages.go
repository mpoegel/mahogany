package views

import (
	"context"
	"strconv"

	db "github.com/mpoegel/mahogany/internal/db"
)

type PackagesView struct {
	IsSuccess  bool
	Err        string
	Packages   []db.Package
	AddPackage db.AddPackageParams
	Toast      string
}

func (vf *ViewFinder) GetPackages(ctx context.Context) (*PackagesView, error) {
	view := &PackagesView{
		IsSuccess: true,
	}
	packages, err := vf.query.ListPackages(ctx)
	if err != nil {
		view.IsSuccess = false
		view.Err = err.Error()
		return view, nil
	}
	view.Packages = packages

	return view, nil
}

func (vf *ViewFinder) AddPackage(ctx context.Context, params db.AddPackageParams) (*PackagesView, error) {
	view, err := vf.GetPackages(ctx)
	if err != nil {
		return view, err
	}
	view.AddPackage = params

	if len(params.Name) == 0 {
		view.Toast = "Name required"
		return view, nil
	}
	if len(params.InstallCmd) == 0 {
		view.Toast = "Install command required"
		return view, nil
	}
	if len(params.UpdateCmd) == 0 {
		view.Toast = "Update command required"
		return view, nil
	}

	pkg, err := vf.query.AddPackage(ctx, params)
	if err != nil {
		return view, err
	}
	view.AddPackage = db.AddPackageParams{}
	view.Packages = append(view.Packages, pkg)
	return view, nil
}

func (vf *ViewFinder) DeletePackage(ctx context.Context, id string) (*PackagesView, error) {
	view, err := vf.GetPackages(ctx)
	if err != nil {
		return view, err
	}

	packageID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		view.Toast = err.Error()
		return view, err
	}

	err = vf.query.DeletePackage(ctx, packageID)
	if err != nil {
		view.Toast = err.Error()
		return view, err
	}
	return vf.GetPackages(ctx)
}
