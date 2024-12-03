package main

//go:generate protoc -I=./schema --go_out=./pkg/schema --go_opt=paths=source_relative --go-grpc_out=./pkg/schema --go-grpc_opt=paths=source_relative ./schema/update_service.proto

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	mahogany "github.com/mpoegel/mahogany/pkg/mahogany"
)

func RunServer() {
	config := mahogany.LoadConfig()
	slog.Info("initialized", "config", config)

	updateServer, err := mahogany.NewUpdateServer(config)
	if err != nil {
		slog.Error("cannot create update server", "err", err)
		return
	}

	server, err := mahogany.NewServer(config, updateServer)
	if err != nil {
		slog.Error("cannot create server", "err", err)
		return
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		slog.Info("shutting down")
		updateServer.Stop()
		server.Stop()
	}()

	go func() {
		if err = updateServer.Start(context.Background()); err != nil {
			slog.Error("cannot start update server", "err", err)
			c <- os.Interrupt
		}
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
	default:
		slog.Error("invalid argument")
	}
}
