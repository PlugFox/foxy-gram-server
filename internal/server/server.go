package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/plugfox/foxy-gram-server/internal/global"
	"github.com/plugfox/foxy-gram-server/internal/log"
	"github.com/plugfox/foxy-gram-server/internal/storage"
)

type Server struct {
	router *chi.Mux
	public chi.Router
	admin  chi.Router
	server *http.Server
}

func New() *Server { // Router for HTTP API and Websocket centrifuge protocol.
	middleware.DefaultLogger = middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: log.NewLogAdapter(global.Logger)})
	router := chi.NewRouter()
	/* router.Use(middleware.Recoverer) */
	router.Use(middlewareErrorRecoverer(global.Logger))
	router.Use(middleware.Logger)
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.URLFormat)
	router.Use(middleware.StripSlashes)
	router.Use(middleware.RedirectSlashes)
	router.Use(middleware.Timeout(global.Config.API.Timeout))
	router.Use(middleware.Heartbeat("/ping"))

	/*
		r.Use(middleware.StripSlashes)
		r.Use(middleware.Compress(5))
		r.Use(middleware.RedirectSlashes)
		r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: log}))
		r.Use(middleware.Throttle(100))
	*/

	// Public API group
	public := router.Group(func(r chi.Router) {
		// Middleware
		r.Use(middleware.NoCache)

		// Routes
		r.HandleFunc("/echo", echoRoute)
		r.HandleFunc("/echo/*", echoRoute)
	})

	// Admin API group
	const compressionLevel = 5

	fs := http.FileServer(http.Dir("./")) // File server

	admin := router.Group(func(r chi.Router) {
		// Middleware
		r.Use(middlewareAuthorization(global.Config.Secret))

		// File server
		r.Route("/admin", func(r chi.Router) {
			r.Route("/files", func(r chi.Router) {
				r.Use(middleware.NoCache)
				r.Use(middleware.Compress(compressionLevel))
				r.Handle("/*", http.StripPrefix("/admin/files", fs))
			})
		})
	})

	// Create a new HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", global.Config.API.Host, global.Config.API.Port),
		Handler:      router,
		WriteTimeout: global.Config.API.WriteTimeout,
		ReadTimeout:  global.Config.API.ReadTimeout,
		IdleTimeout:  global.Config.API.IdleTimeout,
		ErrorLog:     log.NewLogAdapter(global.Logger),
	}

	return &Server{
		router: router,
		public: public,
		admin:  admin,
		server: server,
	}
}

// AddHealthCheck adds a health check endpoint to the server.
// The statusFunc function should return a map of status information.
// The map keys will be used as the status names in the response.
// The map values will be used as the status values in the response.
func (srv *Server) AddHealthCheck(statusFunc func() (bool, map[string]string)) {
	const bytesInMb = 1024 * 1024

	startedAt := time.Now() // Start time

	handler := func(w http.ResponseWriter, _ *http.Request) {
		ok, status := statusFunc()

		var memStats runtime.MemStats

		runtime.ReadMemStats(&memStats)

		uptime := time.Since(startedAt)
		allocatedMemory := memStats.Alloc / bytesInMb
		reservedMemory := memStats.Sys / bytesInMb
		cpu := runtime.NumCPU()
		goroutines := runtime.NumGoroutine()

		data := map[string]any{
			"status": status,
			"uptime": uptime.String(),
			// Allocated memory / Reserved program memory
			"memory":     fmt.Sprintf("%v Mb / %v Mb", allocatedMemory, reservedMemory),
			"cpu":        cpu,
			"goroutines": goroutines,
		}

		defer global.Metrics.LogEvent("health_check", nil, map[string]any{
			"status":           ok,
			"uptime":           uptime,
			"allocated_memory": allocatedMemory,
			"reserved_memory":  reservedMemory,
			"cpu":              cpu,
			"goroutines":       goroutines,
		})

		if ok {
			NewResponse().SetData(data).Ok(w)
		} else {
			NewResponse().SetError("status_error", "One or more services are not healthy", data).InternalServerError(w)
		}
	}

	srv.public.Get("/health", handler)
	srv.public.Get("/status", handler)
	srv.public.Get("/healthz", handler)
	srv.public.Get("/statusz", handler)
	srv.public.Get("/metrics", handler)
	srv.public.Get("/info", handler)
}

func (srv *Server) AddVerifyUsers(db *storage.Storage) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		var requestBody struct {
			IDs    []int  `json:"ids"`
			Reason string `json:"reason,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			NewResponse().SetError("bad_request", err.Error()).BadRequest(w)
		} else if (requestBody.IDs == nil) || (len(requestBody.IDs) == 0) {
			NewResponse().SetError("bad_request", "IDs are required").BadRequest(w)
		} else {
			if requestBody.Reason == "" {
				requestBody.Reason = "Verified from API"
			}

			if err := db.VerifyUsers(requestBody.Reason, requestBody.IDs); err != nil {
				NewResponse().SetError("internal_server_error", err.Error()).InternalServerError(w)
			} else {
				NewResponse().Ok(w)
			}
		}
	}

	srv.admin.Post("/admin/verify", handler)
}

// AddPublicRoute adds a public route to the server.
func (srv *Server) AddPublicRoute(method string, path string, handler http.HandlerFunc) {
	srv.public.Method(method, path, handler)
}

// AddAdminRoute adds an admin route to the server.
func (srv *Server) AddAdminRoute(method string, path string, handler http.HandlerFunc) {
	srv.admin.Method(method, path, handler)
}

// Status returns the server status.
func (srv *Server) Status() (string, error) {
	return "ok", nil
}

// ListenAndServe starts the server and listens for incoming requests.
func (srv *Server) ListenAndServe() error {
	return srv.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server without interrupting any active connections.
func (srv *Server) Shutdown(ctx context.Context) error {
	return srv.server.Shutdown(ctx)
}

// Close closes the server immediately.
func (srv *Server) Close() error {
	return srv.server.Close()
}

// middlewareAuthorization is a middleware function that checks the Authorization header for a Bearer token.
func middlewareAuthorization(secret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			// Check if the Authorization header is missing
			if authHeader == "" {
				NewResponse().SetError("unauthorized", "Authorization header is required").Unauthorized(w)

				return
			}

			// Check if the Authorization header is not a Bearer token
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == authHeader { // If the Authorization header is not a Bearer token
				NewResponse().SetError("unauthorized", "Bearer token is required").Unauthorized(w)

				return
			}

			// Check if the Bearer token is invalid
			if token != secret {
				NewResponse().SetError("unauthorized", "Invalid Bearer token").Unauthorized(w)

				return
			}

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// middlewareErrorRecoverer is a middleware function that recovers from panics and returns an error response.
func middlewareErrorRecoverer(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					if e, ok := err.(error); ok {
						if errors.Is(e, http.ErrAbortHandler) {
							// we don't recover http.ErrAbortHandler so the response
							// to the client is aborted, this should not be logged
							panic(err)
						}
					}

					if r.Header.Get("Connection") == "Upgrade" {
						return
					}

					// Log the error
					logger.ErrorContext(context.Background(), "Recovered from panic", slog.String("error", fmt.Sprintf("%v", err)))

					NewResponse().SetError("internal_server_error",
						"Internal Server Error",
						map[string]any{
							"error": fmt.Sprintf("%v", err),
							"stack": string(debug.Stack()),
						},
					).InternalServerError(w)
				}
			}()

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}
