package views

import (
	"context"
	"log/slog"
	"time"

	db "github.com/mpoegel/mahogany/internal/db"
	sources "github.com/mpoegel/mahogany/pkg/mahogany/sources"
	vpn "github.com/mpoegel/mahogany/pkg/vpn"
)

type SettingsView struct {
	WatchtowerAddr    string
	WatchtowerToken   string
	WatchtowerTimeout string
	RegistryAddr      string
	RegistryTimeout   string
	TailscaleApiKey   string
	TailnetName       string
}

func (v *SettingsView) Name() string { return "SettingsView" }

func (v *ViewFinder) reload(ctx context.Context, query *db.Queries) error {
	if query == nil {
		query = v.query
	}
	registryTimeout, err := time.ParseDuration(v.getSetting(ctx, query, "RegistryTimeout"))
	if err != nil {
		return err
	}
	watchtowerTimeout, err := time.ParseDuration(v.getSetting(ctx, query, "WatchtowerTimeout"))
	if err != nil {
		return err
	}

	v.registry = sources.NewRegistry(v.getSetting(ctx, query, "RegistryAddr"), registryTimeout)
	v.watchtower = sources.NewWatchtower(v.getSetting(ctx, query, "WatchtowerAddr"), v.getSetting(ctx, query, "WatchtowerToken"), watchtowerTimeout)
	v.deviceFinder = vpn.NewClient(v.getSetting(ctx, query, "TailscaleApiKey"), v.getSetting(ctx, query, "TailnetName"))

	return nil
}

func (v *ViewFinder) getSetting(ctx context.Context, query *db.Queries, name string) string {
	row, err := query.GetSetting(ctx, name)
	if err != nil {
		slog.Warn("failed to get setting", "err", err, "name", name)
	}
	return row.Value
}

func (v *ViewFinder) GetSettings(ctx context.Context) *SettingsView {
	view := &SettingsView{
		WatchtowerAddr:    v.getSetting(ctx, v.query, "WatchtowerAddr"),
		WatchtowerToken:   v.getSetting(ctx, v.query, "WatchtowerToken"),
		WatchtowerTimeout: v.getSetting(ctx, v.query, "WatchtowerTimeout"),
		RegistryAddr:      v.getSetting(ctx, v.query, "RegistryAddr"),
		RegistryTimeout:   v.getSetting(ctx, v.query, "RegistryTimeout"),
		TailscaleApiKey:   v.getSetting(ctx, v.query, "TailscaleApiKey"),
		TailnetName:       v.getSetting(ctx, v.query, "TailnetName"),
	}
	return view
}

func (v *ViewFinder) PostSettings(ctx context.Context, params db.UpdateSettingParams) error {
	tx, err := v.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	query := v.query.WithTx(tx)
	if err := query.UpdateSetting(ctx, params); err != nil {
		return err
	}
	if err = v.reload(ctx, query); err != nil {
		if err := tx.Rollback(); err != nil {
			slog.Warn("failed to rollback settings transaction", "err", err)
		}
		return err
	}
	return tx.Commit()
}
