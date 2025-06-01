package sources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
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
	GetCatalog(ctx context.Context) (*RegistryCatalog, error)
	GetTags(ctx context.Context, repository string) (*RegistryTags, error)
	GetManifest(ctx context.Context, repository, tag string) (*RegistryManifest, error)
	DeleteImage(ctx context.Context, repository, tag string) error
}

type Registry struct {
	addr    string
	timeout time.Duration
}

// NewRegistry creates a client for a docker registry
// API spec: https://distribution.github.io/distribution/spec/api/
func NewRegistry(registryAddr string, timeout time.Duration) RegistryI {
	return &Registry{
		addr:    registryAddr,
		timeout: timeout,
	}
}

func (r *Registry) GetCatalog(ctx context.Context) (*RegistryCatalog, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s/v2/_catalog", r.addr), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
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

func (r *Registry) GetTags(ctx context.Context, repository string) (*RegistryTags, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s/v2/%s/tags/list", r.addr, repository), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
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

func (r *Registry) GetManifest(ctx context.Context, repository, tag string) (*RegistryManifest, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s/v2/%s/manifests/%s", r.addr, repository, tag), nil)
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

func (r *Registry) DeleteImage(ctx context.Context, repository, digest string) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("http://%s/v2/%s/manifests/%s", r.addr, repository, digest), nil)
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
