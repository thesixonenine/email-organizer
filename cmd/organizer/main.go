package main

import (
	"flag"
	"log/slog"
	"os"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	if err := run(*configPath); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	slog.Info("starting email organizer", "config", configPath)
	// TODO: wire components after they're built
	return nil
}