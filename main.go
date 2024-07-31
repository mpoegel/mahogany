package main

import (
	"log/slog"
	"os"
	"os/signal"

	"github.com/mpoegel/mahogany/pkg/mahogany"
)

func main() {
	config := mahogany.LoadConfig()
	slog.Info("initialized", "config", config)
	server, err := mahogany.NewServer(config)
	if err != nil {
		slog.Error("cannot create server", "err", err)
		return
	}

	if err = server.Start(); err != nil {
		slog.Error("cannot start server", "err", err)
		return
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		slog.Info("shutting down")
		server.Stop()
	}()
}
