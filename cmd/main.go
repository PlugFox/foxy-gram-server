package main

import (
	"context"
	"fmt"
	logByDefault "log"
	"log/slog"
	"os"
	"time"

	// This controls the maxprocs environment variable in container runtimes.
	// see https://martin.baillie.id/wrote/gotchas-in-the-go-network-packages-defaults/#bonus-gomaxprocs-containers-and-the-cfs
	"go.uber.org/automaxprocs/maxprocs"

	config "github.com/plugfox/foxy-gram-server/internal/config"
	log "github.com/plugfox/foxy-gram-server/internal/log"
	storage "github.com/plugfox/foxy-gram-server/internal/storage"
	"github.com/plugfox/foxy-gram-server/internal/telegram"
)

func main() {
	// Set the local timezone to UTC
	time.Local = time.UTC

	// Initialize the configuration
	config, err := config.MustLoadConfig()
	if err != nil {
		logByDefault.Fatalf("Config load error: %v", err)
		os.Exit(1)
	}

	// log := logger.SetupLogger(config.Environment)

	// Logger configuration
	logger := log.New(
		log.WithLevel(config.Verbose),
		log.WithSource(),
	)

	if err := run(config, logger); err != nil {
		logger.ErrorContext(context.Background(), "an error occurred", slog.String("error", err.Error()))
		os.Exit(1)
	}

	os.Exit(0)
	// select {}
}

func run(config *config.Config, logger *slog.Logger) error {
	ctx := context.Background()

	_, err := maxprocs.Set(maxprocs.Logger(func(s string, i ...interface{}) {
		logger.DebugContext(ctx, fmt.Sprintf(s, i...))
	}))
	if err != nil {
		return fmt.Errorf("setting max procs: %w", err)
	}

	// Setup database connection
	_, err = storage.New(config, logger)
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	// Setup Telegram bot
	telegram, err := telegram.New(config, logger)
	if err != nil {
		return fmt.Errorf("telegram bot setup error: %w", err)
	}

	// TODO: Setup API server

	// TODO: Setup Centrifuge server

	// TODO: Setup InfluxDB metrics (if any)

	telegram.Start()

	logger.InfoContext(ctx, "Server started", slog.String("host", config.API.Host), slog.Int("port", config.API.Port))

	return nil
}
