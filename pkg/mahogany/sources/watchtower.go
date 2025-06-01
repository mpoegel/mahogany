package sources

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type WatchtowerI interface {
	Update(context.Context) error
}

type Watchtower struct {
	addr    string
	token   string
	timeout time.Duration
}

// NewWatchtower creates a client for the Watchtower HTTP API
// API spec: https://containrrr.dev/watchtower/http-api-mode/
func NewWatchtower(addr, token string, timeout time.Duration) WatchtowerI {
	return &Watchtower{addr: addr, token: token}
}

func (w *Watchtower) Update(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s/v1/update", w.addr), nil)
	if err != nil {
		return err
	}
	req.Header.Add("authorization", fmt.Sprintf("Bearer %s", w.token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	return nil
}
