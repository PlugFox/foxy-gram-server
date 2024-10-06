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
	"github.com/plugfox/foxy-gram-server/internal/err"
	"github.com/plugfox/foxy-gram-server/internal/global"
	"github.com/plugfox/foxy-gram-server/internal/httpclient"
	log "github.com/plugfox/foxy-gram-server/internal/log"
	"github.com/plugfox/foxy-gram-server/internal/metrics"
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

	// Metrics configuration
	if config.Metrics.IsValid() {
		global.Metrics = metrics.NewMetricsImpl(
			config.Metrics.URL,
			config.Metrics.Token,
			config.Metrics.Org,
			config.Metrics.Bucket,
			map[string]string{},
		)
	} else {
		global.Metrics = metrics.NewMetricsFake()
	}

	global.Config = config
	global.Logger = logger

	// Run the server
	if err := run(); err != nil {
		logger.ErrorContext(context.Background(), "an error occurred", slog.String("error", err.Error()))
		os.Exit(1)
	}

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

	// Flush and close the metrics logger.
	global.Metrics.Close()
}

// Starts the server and waits for the SIGINT or SIGTERM signal to shutdown the server.
func run() error {
	if global.Config == nil || global.Logger == nil {
		return err.ErrorGlobalVariablesNotInitialized
	}

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	// Set the maxprocs environment variable in container runtimes.
	_, err := maxprocs.Set(maxprocs.Logger(func(s string, i ...interface{}) {
		global.Logger.DebugContext(ctx, fmt.Sprintf(s, i...))
	}))
	if err != nil {
		return fmt.Errorf("setting max procs: %w", err)
	}

	// Setup hash function
	model.InitHashFunction()

	// Setup database connection
	db := initStorage()

	// Create a http client
	httpClient := initHTTPClient()

	// Setup Telegram bot
	tg := initTelegram(db, httpClient)

	// Update the bot user information
	if err := db.UpsertUser(tg.Me().Seen()); err != nil {
		return fmt.Errorf("upserting user error: %w", err)
	}

	// Setup API srv
	srv := initServer(db, tg)

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

	// Track outdated captchas
	go func() {
		for {
			select {
			case <-time.After(global.Config.Captcha.Expiration / 10): //nolint:mnd
				captchas := db.GetOutdatedCaptchas()
				for _, captcha := range captchas {
					if err := db.DeleteCaptchaByID(captcha.ID); err != nil {
						global.Logger.ErrorContext(ctx, "database: deleting outdated captcha error", slog.String("error", err.Error()), slog.Int64("id", captcha.ID))
						continue
					}
					if err := tg.DeleteMessage(captcha.ChatID, captcha.MessageID); err != nil {
						global.Logger.ErrorContext(ctx, "telegram: deleting outdated captcha error", slog.String("error", err.Error()), slog.Int64("id", captcha.ID))
						continue
					}

					global.Logger.InfoContext(ctx, "outdated captcha deleted", slog.Int64("id", captcha.ID))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Log the server start
	global.Logger.InfoContext(
		ctx,
		"Server started",
		slog.String("host", global.Config.API.Host),
		slog.Int("port", global.Config.API.Port),
	)

	global.Metrics.LogEvent("server_started", nil, map[string]interface{}{
		"host": global.Config.API.Host,
		"port": global.Config.API.Port,
	})

	// Wait for the SIGINT or SIGTERM signal to shutdown the server.
	waitExitSignal(sigCh, tg, srv)
	close(sigCh)

	return nil
}

// initStorage initializes the database connection.
func initStorage() *storage.Storage {
	db, err := storage.New()
	if err != nil {
		panic(fmt.Sprintf("database connection error: %v", err))
	}

	return db
}

// Create a new HTTP client
func initHTTPClient() *http.Client {
	httpClient, err := httpclient.NewHTTPClient(&global.Config.Proxy)
	if err != nil {
		panic(fmt.Sprintf("http client error: %v", err))
	}

	return httpClient
}

// Initialize the Telegram bot
func initTelegram(db *storage.Storage, httpClient *http.Client) *telegram.Telegram {
	tg, err := telegram.New(db, httpClient)
	if err != nil {
		panic(fmt.Sprintf("telegram bot setup error: %v", err))
	}

	// Start the Telegram bot polling
	go func() {
		tg.Start()
	}()

	return tg
}

// Initialize the API server
func initServer(db *storage.Storage, tg *telegram.Telegram) *server.Server {
	srv := server.New()

	srv.AddHealthCheck(
		func() (bool, map[string]string) {
			dbStatus, dbErr := db.Status()
			srvStatus, srvErr := srv.Status()
			tgStatus, tgErr := tg.Status()

			isHealthy := dbErr == nil && srvErr == nil && tgErr == nil

			return isHealthy, map[string]string{
				"database": dbStatus,
				"server":   srvStatus,
				"telegram": tgStatus,
			}
		},
	) // Add health check endpoint
	srv.AddVerifyUsers(db) // Add verify users endpoint [POST] /admin/verify

	// Start the server
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			global.Logger.Error("Server error", slog.String("error", err.Error()))
			os.Exit(1) // Exit the program if the server fails to start.
		}
	}()

	return srv
}
