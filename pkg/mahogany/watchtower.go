package mahogany

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

type WatchtowerI interface {
	Update(context.Context) error
}

type Watchtower struct {
	addr  string
	token string
}

func NewWatchtower(addr, token string) WatchtowerI {
	return &Watchtower{addr: addr, token: token}
}

func (w *Watchtower) Update(ctx context.Context) error {
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
