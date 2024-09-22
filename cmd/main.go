package main

import (
	"context"
	"errors"
	"fmt"
	logByDefault "log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	config "github.com/plugfox/foxy-gram-server/internal/config"
	"github.com/plugfox/foxy-gram-server/internal/httpclient"
	log "github.com/plugfox/foxy-gram-server/internal/log"
	"github.com/plugfox/foxy-gram-server/internal/model"
	"github.com/plugfox/foxy-gram-server/internal/server"
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
func waitExitSignal(sigCh chan os.Signal, t *telegram.Telegram, s *server.Server /* n *centrifuge.Node */) {
	wg := sync.WaitGroup{}

	// Notify the channel for SIGINT and SIGTERM signals.
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	const timeout = 10 * time.Second

	// Start a goroutine to wait for the signal and handle graceful shutdown.
	wg.Add(1)

	go func() {
		defer wg.Done()

		// Wait for the signal.
		<-sigCh

		ctx, cancel := context.WithTimeout(context.Background(), timeout)

		defer cancel()

		// _ = n.Shutdown(ctx)

		_ = s.Shutdown(ctx)
	}()

	// Handle Telegram bot shutdown.
	wg.Add(1)

	go func() {
		defer wg.Done()

		// Wait for the signal.
		<-sigCh

		// Create a channel to indicate when the shutdown is complete.
		done := make(chan struct{})

		// Stop the Telegram bot
		go func() {
			defer close(done)
			t.Stop()
		}()

		// Ensure the shutdown happens within 10 seconds.
		select {
		case <-done: // Done
		case <-time.After(timeout): // Timeout
		}
	}()

	// Wait for both goroutines to complete before exiting.
	wg.Wait()
}

func run(config *config.Config, logger *slog.Logger) error {
	startedAt := time.Now()
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

	// Setup API server
	server := server.New(config, logger)
	server.AddHealthCheck(
		func() map[string]string {
			dbStatus, _ := db.Status()

			return map[string]string{
				"db":     dbStatus,
				"uptime": time.Since(startedAt).String(),
			}
		},
	) // Add health check endpoint
	server.AddEcho() // Add echo endpoint

	// TODO: Setup Centrifuge server

	// TODO: Setup InfluxDB metrics (if any)

	// Create a channel to shutdown the server.
	sigCh := make(chan os.Signal, 1)

	// Create a function to stop the server.
	// Call this function when the server needs to be closed.
	/* stop := func(sigCh chan os.Signal) func() {
		return func() {
			sigCh <- syscall.SIGTERM // Close server.
		}
	} */

	// Start the Telegram bot polling
	go func() {
		telegram.Start()
	}()

	// Start the server
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.ErrorContext(ctx, "Server error", slog.String("error", err.Error()))
			os.Exit(1) // Exit the program if the server fails to start.
		}
	}()

	// Log the server start
	logger.InfoContext(ctx, "Server started", slog.String("host", config.API.Host), slog.Int("port", config.API.Port))

	// Wait for the SIGINT or SIGTERM signal to shutdown the server.
	waitExitSignal(sigCh, telegram, server)
	close(sigCh)

	return nil
}
