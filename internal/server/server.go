package server

import (
	"context"
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
	"github.com/go-chi/render"
	"github.com/plugfox/foxy-gram-server/api"
	"github.com/plugfox/foxy-gram-server/internal/config"
	"github.com/plugfox/foxy-gram-server/internal/log"
)

type Server struct {
	router *chi.Mux
	public chi.Router
	admin  chi.Router
	server *http.Server
}

func New(config *config.Config, logger *slog.Logger) *Server { // Router for HTTP API and Websocket centrifuge protocol.
	middleware.DefaultLogger = middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: log.NewLogAdapter(logger)})
	router := chi.NewRouter()
	/* router.Use(middleware.Recoverer) */
	router.Use(middlewareErrorRecoverer(logger))
	router.Use(middleware.Logger)
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.URLFormat)
	router.Use(middleware.StripSlashes)
	router.Use(middleware.RedirectSlashes)
	router.Use(middleware.Timeout(config.API.Timeout))
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
	})

	// Admin API group
	const compressionLevel = 5

	fs := http.FileServer(http.Dir("./")) // File server

	admin := router.Group(func(r chi.Router) {
		// Middleware
		r.Use(middlewareAuthorization(config.Secret))

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
		Addr:         fmt.Sprintf("%s:%d", config.API.Host, config.API.Port),
		Handler:      router,
		WriteTimeout: config.API.WriteTimeout,
		ReadTimeout:  config.API.ReadTimeout,
		IdleTimeout:  config.API.IdleTimeout,
		ErrorLog:     log.NewLogAdapter(logger),
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

	srv.public.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		rsp := &api.Response{}
		ok, status := statusFunc()

		var memStats runtime.MemStats

		runtime.ReadMemStats(&memStats)

		data := map[string]any{
			"status": status,
			"uptime": time.Since(startedAt).String(),
			// Allocated memory / Reserved program memory
			"memory":     fmt.Sprintf("%v Mb / %v Mb", memStats.Alloc/bytesInMb, memStats.Sys/bytesInMb),
			"cpu":        runtime.NumCPU(),
			"goroutines": runtime.NumGoroutine(),
		}

		if ok {
			rsp.SetData(data)
			rsp.Ok(w)
		} else {
			rsp.SetError("status_error", "One or more services are not healthy", data)
			rsp.InternalServerError(w)
		}
	})
}

// echo route for testing purposes
func echoRoute(w http.ResponseWriter, r *http.Request) {
	// Create a new response object
	rsp := &api.Response{}

	// Create a map to hold the request data
	var data map[string]any

	// Decode the request body into the data map
	if r.ContentLength != 0 && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		err := render.Decode(r, &data)
		if err != nil {
			rsp.SetError("bad_request", err.Error())
			rsp.BadRequest(w)

			return
		}
	}

	rsp.SetData(struct {
		Remote  string         `json:"remote"`
		Method  string         `json:"method"`
		Headers http.Header    `json:"headers"`
		Body    map[string]any `json:"body"`
	}{
		Remote:  r.RemoteAddr,
		Method:  r.Method,
		Headers: r.Header,
		Body:    data,
	})
	rsp.Ok(w)
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
				rsp := &api.Response{}
				rsp.SetError("unauthorized", "Authorization header is required")
				rsp.Unauthorized(w)

				return
			}

			// Check if the Authorization header is not a Bearer token
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == authHeader { // If the Authorization header is not a Bearer token
				rsp := &api.Response{}
				rsp.SetError("unauthorized", "Bearer token is required")
				rsp.Unauthorized(w)

				return
			}

			// Check if the Bearer token is invalid
			if token != secret {
				rsp := &api.Response{}
				rsp.SetError("unauthorized", "Invalid Bearer token")
				rsp.Unauthorized(w)

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

					rsp := &api.Response{}

					rsp.SetError("internal_server_error",
						"Internal Server Error",
						map[string]any{
							"error": fmt.Sprintf("%v", err),
							"stack": string(debug.Stack()),
						},
					)
					rsp.InternalServerError(w)
				}
			}()

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}
