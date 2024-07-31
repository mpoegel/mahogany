package mahogany

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type RegistryCatalog struct {
	Repositories []string `json:"repositories"`
}

type RegistryTags struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type RegistryManifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	Name          string `json:"name"`
	Tag           string `json:"tag"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"layers"`
}

type RegistryI interface {
	GetCatalog() (*RegistryCatalog, error)
	GetTags(repository string) (*RegistryTags, error)
	GetManifest(repository, tag string) (*RegistryManifest, error)
	DeleteImage(repository, tag string) error
}

type Registry struct {
	addr string
}

// NewRegistry creates a client for a docker registry
// API spec: https://distribution.github.io/distribution/spec/api/
func NewRegistry(registryAddr string) RegistryI {
	return &Registry{
		addr: registryAddr,
	}
}

func (r *Registry) GetCatalog() (*RegistryCatalog, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/v2/_catalog", r.addr))
	if err != nil {
		return nil, err
	}
	catalog := &RegistryCatalog{}
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(catalog); err != nil {
		r, _ := io.ReadAll(resp.Body)
		return nil, errors.New(string(r))
	}
	return catalog, nil
}

func (r *Registry) GetTags(repository string) (*RegistryTags, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/v2/%s/tags/list", r.addr, repository))
	if err != nil {
		return nil, err
	}
	tags := &RegistryTags{}
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(tags); err != nil {
		r, _ := io.ReadAll(resp.Body)
		return nil, errors.New(string(r))
	}
	return tags, nil
}

func (r *Registry) GetManifest(repository, tag string) (*RegistryManifest, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/v2/%s/manifests/%s", r.addr, repository, tag), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	manifest := &RegistryManifest{}
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(manifest); err != nil {
		r, _ := io.ReadAll(resp.Body)
		return nil, errors.New(string(r))
	}
	manifest.Name = repository
	manifest.Tag = tag
	return manifest, nil
}

func (r *Registry) DeleteImage(repository, digest string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://%s/v2/%s/manifests/%s", r.addr, repository, digest), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusAccepted {
		return errors.New(resp.Status)
	}
	return nil
}
