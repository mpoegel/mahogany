package main

//go:generate protoc -I=./schema --go_out=./pkg/schema --go_opt=paths=source_relative --go-grpc_out=./pkg/schema --go-grpc_opt=paths=source_relative ./schema/update_service.proto
//go:generate sqlc generate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"time"

	db "github.com/mpoegel/mahogany/internal/db"
	mahogany "github.com/mpoegel/mahogany/pkg/mahogany"
	_ "modernc.org/sqlite"
)

func RunServer() {
	config := mahogany.LoadConfig()
	slog.Info("initialized", "config", config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, err := mahogany.NewServer(ctx, config)
	if err != nil {
		slog.Error("cannot create server", "err", err)
		return
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		slog.Info("shutting down")
		cancel()
		server.Stop()
	}()

	if err = server.Start(); err != nil {
		slog.Error("cannot start server", "err", err)
		return
	}
}

func RunAgent() {
	config := mahogany.LoadAgentConfig()
	slog.Info("initialized", "config", config)

	agent, err := mahogany.NewAgent(config)
	if err != nil {
		slog.Error("cannot create agent", "err", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	otelShutdown, err := mahogany.SetupOTelSDK(ctx, config.TelemetryEndpoint)
	if err != nil {
		slog.Error("cannot setup otel", "err", err)
		return
	}
	defer otelShutdown(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		slog.Info("shutting down")
		cancel()
	}()

	reset := time.NewTimer(60 * time.Second)
	for ctx.Err() == nil {
		if err := agent.Run(ctx); err != nil {
			slog.Error("agent failure", "err", err)
		}
		reset.Reset(60 * time.Second)
		select {
		case <-ctx.Done():
		case <-reset.C:
		}
	}
	agent.Close()
}

func exportData(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	dbFile := fs.String("db", "mahogany.db", "database file")
	filename := fs.String("file", "mahogany.json", "file to export data to")

	if err := fs.Parse(args); err != nil {
		slog.Error("failed to parse export args", "err", err)
		return
	}

	fp, err := os.OpenFile(*filename, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
	if err != nil {
		slog.Error("failed to open export file", "err", err)
		return
	}

	dbConn, err := sql.Open("sqlite", *dbFile)
	if err != nil {
		slog.Error("failed to open database file", "err", err)
		return
	}
	query := db.New(dbConn)
	data := mahogany.AppData{}

	data.Settings, err = query.ListSettings(context.Background())
	if err != nil {
		slog.Error("failed to list settings", "err", err)
		return
	}

	data.Packages, err = query.ListPackages(context.Background())
	if err != nil {
		slog.Error("failed to list packages", "err", err)
		return
	}

	encoder := json.NewEncoder(fp)
	if err := encoder.Encode(data); err != nil {
		slog.Error("failed to encode app data", "err", err)
		return
	}
	slog.Info("export complete")
}

func importData(args []string) {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	dbFile := fs.String("db", "mahogany.db", "database file")
	filename := fs.String("file", "mahogany.json", "file to import data from")

	if err := fs.Parse(args); err != nil {
		slog.Error("failed to parse import args", "err", err)
		return
	}

	fp, err := os.Open(*filename)
	if err != nil {
		slog.Error("failed to open import file", "err", err)
		return
	}

	dbConn, err := sql.Open("sqlite", *dbFile)
	if err != nil {
		slog.Error("failed to open database file", "err", err)
		return
	}
	query := db.New(dbConn)
	data := mahogany.AppData{}
	decoder := json.NewDecoder(fp)
	if err := decoder.Decode(&data); err != nil {
		slog.Error("failed to parse import file", "err", err)
		return
	}

	var allErrs error
	for _, pkg := range data.Packages {
		_, err := query.AddPackage(context.Background(), db.AddPackageParams{
			Name:       pkg.Name,
			InstallCmd: pkg.InstallCmd,
			UpdateCmd:  pkg.UpdateCmd,
			RemoveCmd:  pkg.RemoveCmd,
		})
		allErrs = errors.Join(allErrs, err)
	}

	for _, setting := range data.Settings {
		err := query.UpdateSetting(context.Background(), db.UpdateSettingParams{
			Name:  setting.Name,
			Value: setting.Value,
		})
		allErrs = errors.Join(allErrs, err)
	}

	if allErrs != nil {
		slog.Warn("import finished with errors", "err", err)
	}
	slog.Info("import complete")
}

func main() {
	args := os.Args
	if len(args) < 2 {
		slog.Error("missing argument [server, agent]")
		return
	}

	switch args[1] {
	case "server":
		RunServer()
	case "agent":
		RunAgent()
	case "export":
		exportData(args[2:])
	case "import":
		importData(args[2:])
	default:
		slog.Error("invalid argument")
	}
}
