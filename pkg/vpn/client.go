package vpn

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type VirtualNetworkClient interface {
	ListDevices(context.Context) ([]Device, error)
	GetDevice(ctx context.Context, deviceID string) (*Device, error)
	GetACL(ctx context.Context) (*NetPolicy, error)
}

type TailscaleClient struct {
	ApiKey  string
	Tailnet string
}

func NewClient(apiKey, tailnet string) *TailscaleClient {
	tc := &TailscaleClient{
		ApiKey:  apiKey,
		Tailnet: tailnet,
	}
	return tc
}

func (tc *TailscaleClient) newRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	url = fmt.Sprintf("https://api.tailscale.com/api/v2/%s", url)
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", tc.ApiKey))
	req.Header.Add("Accept", "application/json")
	return req, nil
}

func (tc *TailscaleClient) ListDevices(ctx context.Context) ([]Device, error) {
	req, err := tc.newRequest(ctx, http.MethodGet, fmt.Sprintf("tailnet/%s/devices", tc.Tailnet), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tailscale request failed: %s", resp.Status)
	}

	devicesResponse := struct {
		Devices []Device `json:"devices"`
	}{}
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&devicesResponse); err != nil {
		return nil, err
	}
	return devicesResponse.Devices, nil
}

func (tc *TailscaleClient) GetDevice(ctx context.Context, deviceID string) (*Device, error) {
	req, err := tc.newRequest(ctx, http.MethodGet, fmt.Sprintf("device/%s", deviceID), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tailscale request failed: %s", resp.Status)
	}

	var device Device
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&device); err != nil {
		return nil, err
	}
	return &device, nil
}

func (tc *TailscaleClient) GetACL(ctx context.Context) (*NetPolicy, error) {
	req, err := tc.newRequest(ctx, http.MethodGet, fmt.Sprintf("/tailnet/%s/acl", tc.Tailnet), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tailscale request failed: %s", resp.Status)
	}

	var netPolicy NetPolicy
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&netPolicy); err != nil {
		return nil, err
	}
	return &netPolicy, nil
}
