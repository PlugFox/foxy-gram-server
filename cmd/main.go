package main

import (
	"context"
	"fmt"
	logByDefault "log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	config "github.com/plugfox/foxy-gram-server/internal/config"
	"github.com/plugfox/foxy-gram-server/internal/httpclient"
	log "github.com/plugfox/foxy-gram-server/internal/log"
	"github.com/plugfox/foxy-gram-server/internal/model"
	storage "github.com/plugfox/foxy-gram-server/internal/storage"
	"github.com/plugfox/foxy-gram-server/internal/telegram"

	// This controls the maxprocs environment variable in container runtimes.
	// see https://martin.baillie.id/wrote/gotchas-in-the-go-network-packages-defaults/#bonus-gomaxprocs-containers-and-the-cfs
	"go.uber.org/automaxprocs/maxprocs"
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

	// Logger configuration
	logger := log.New(
		log.WithLevel(config.Verbose),
		log.WithSource(),
	)

	if err := run(config, logger); err != nil {
		logger.ErrorContext(context.Background(), "an error occurred", slog.String("error", err.Error()))
		os.Exit(1)
	}

	/* // Create a channel to shutdown the server.
	sigCh := make(chan os.Signal, 1)

	// Close after 1 sec to let response go to client.
	time.AfterFunc(time.Second, func() {
		sigCh <- syscall.SIGTERM // Close server.
	})

	waitExitSignal(sigCh) */

	os.Exit(0)
}

// waitExitSignal waits for the SIGINT or SIGTERM signal to shutdown the centrifuge node.
// It creates a channel to receive signals and a channel to indicate when the shutdown is complete.
// Then it notifies the channel for SIGINT and SIGTERM signals and starts a goroutine to wait for the signal.
// Once the signal is received, it shuts down the centrifuge node and indicates that the shutdown is complete.
func waitExitSignal(sigCh chan os.Signal, t *telegram.Telegram /* n *centrifuge.Node, s *http.Server */) {
	wg := sync.WaitGroup{}

	// Notify the channel for SIGINT and SIGTERM signals.
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start a goroutine to wait for the signal and handle graceful shutdown.
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Wait for the signal.
		<-sigCh
		_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		//_ = n.Shutdown(ctx)
		//_ = s.Shutdown(ctx)
	}()

	// Handle Telegram bot shutdown.
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Wait for the signal.
		<-sigCh

		// Stop the Telegram bot
		t.Stop()

		// Ensure the shutdown happens within 10 seconds.
		select {
		case <-time.After(10 * time.Second):
		}
	}()

	// Wait for both goroutines to complete before exiting.
	wg.Wait()
}

func run(config *config.Config, logger *slog.Logger) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := maxprocs.Set(maxprocs.Logger(func(s string, i ...interface{}) {
		logger.DebugContext(ctx, fmt.Sprintf(s, i...))
	}))
	if err != nil {
		return fmt.Errorf("setting max procs: %w", err)
	}

	// Setup hash function
	model.InitHashFunction()

	// Setup database connection
	db, err := storage.New(config, logger)
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	// Create a http client
	httpClient, err := httpclient.NewHTTPClient(&config.Proxy)
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	// Setup Telegram bot
	telegram, err := telegram.New(db, httpClient, config, logger)
	if err != nil {
		return fmt.Errorf("telegram bot setup error: %w", err)
	}

	// Update the bot user information
	if err := db.UpsertUser(telegram.Me().Seen()); err != nil {
		return fmt.Errorf("upserting user error: %w", err)
	}

	// TODO: Setup API server
	// - health
	// - metrics

	// TODO: Setup Centrifuge server

	// TODO: Setup InfluxDB metrics (if any)

	// Create a channel to shutdown the server.
	sigCh := make(chan os.Signal, 1)

	// Start the Telegram bot polling
	go func() {
		telegram.Start()
	}()

	logger.InfoContext(ctx, "Server started", slog.String("host", config.API.Host), slog.Int("port", config.API.Port))

	// Wait for the SIGINT or SIGTERM signal to shutdown the server.
	waitExitSignal(sigCh, telegram)
	close(sigCh)

	return nil
}
