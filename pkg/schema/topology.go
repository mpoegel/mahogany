package schema

import (
	"os"
	"regexp"

	validator "github.com/go-playground/validator/v10"
	toml "github.com/pelletier/go-toml/v2"
)

type Topology struct {
	Baseline     []Package      `toml:"baseline"`
	HostPackages []HostPackages `toml:"host_packages"`
}

type Package struct {
	ID             string `toml:"id" validate:"required"`
	InstallCommand string `toml:"install_command"`

	GithubPackage *GithubPackage `toml:"github_package" validate:"required_without=AptPackage DockerPackage LocalPackage"`
	AptPackage    *AptPackage    `toml:"apt_package" validate:"required_without=GithubPackage DockerPackage LocalPackage"`
	DockerPackage *DockerPackage `toml:"docker_package" validate:"required_without=GithubPackage AptPackage LocalPackage"`
	LocalPackage  *LocalPackage  `toml:"local_package" validate:"required_without=GithubPackage AptPackage DockerPackage"`
}

type HostPackages struct {
	HostName string    `toml:"hostname" validate:"required"`
	Packages []Package `toml:"packages" validate:"required"`
	Skipped  []string  `toml:"skipped"`
}

type GithubPackage struct {
	Name       string         `toml:"name" validate:"required"`
	AssetRegex string         `toml:"asset_regex" validate:"required"`
	Regex      *regexp.Regexp `toml:"-"`
}

type AptPackage struct {
	Name string `toml:"name" validate:"required"`
}

type DockerPackage struct {
	Name string `toml:"name" validate:"required"`
}

type LocalPackage struct {
	Name        string `toml:"name" validate:"required"`
	Source      string `toml:"source" validate:"required"`
	Destination string `toml:"destination" validate:"required"`
}

func ReadTopology(filename string) (*Topology, error) {
	fp, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	var topo Topology
	decoder := toml.NewDecoder(fp)
	if err = decoder.Decode(&topo); err != nil {
		return nil, err
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err = validate.Struct(topo); err != nil {
		return nil, err
	}

	for i, pack := range topo.Baseline {
		if pack.GithubPackage != nil {
			if re, err := regexp.Compile(pack.GithubPackage.AssetRegex); err != nil {
				return nil, err
			} else {
				topo.Baseline[i].GithubPackage.Regex = re
			}
		}
	}
	for i, host := range topo.HostPackages {
		for k, pack := range host.Packages {
			if pack.GithubPackage != nil {
				if re, err := regexp.Compile(pack.GithubPackage.AssetRegex); err != nil {
					return nil, err
				} else {
					topo.HostPackages[i].Packages[k].GithubPackage.Regex = re
				}
			}
		}
	}

	return &topo, nil
}
