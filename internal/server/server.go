package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
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
	server *http.Server
}

func New(config *config.Config, logger *slog.Logger) *Server { // Router for HTTP API and Websocket centrifuge protocol.
	router := chi.NewRouter()

	// Public API group
	// fs := http.FileServer(http.Dir("./")) // File server
	public := router.Group(func(r chi.Router) {
		// Middleware
		r.Use(middleware.RequestID)
		r.Use(middleware.RealIP)
		/* r.Use(middleware.Logger) */
		/* r.Use(logger_mw.New(log)) */
		r.Use(middleware.Recoverer)
		r.Use(middleware.URLFormat)
		r.Use(middleware.NoCache)
		/* router.Use(middleware.Heartbeat("/ping"))
		r.Use(middleware.StripSlashes)
		r.Use(middleware.Compress(5))
		r.Use(middleware.RedirectSlashes)
		r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: log}))
		r.Use(middleware.Throttle(100))
		r.Use(middleware.Timeout(cfg.Server.Timeout * time.Second)) */
		/* r.Handle("/*", fs) */
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

		rsp.SetData(map[string]any{
			"status": status,
			"uptime": time.Since(startedAt).String(),
			// Allocated memory / Reserved program memory
			"memory":     fmt.Sprintf("%v Mb / %v Mb", memStats.Alloc/bytesInMb, memStats.Sys/bytesInMb),
			"cpu":        runtime.NumCPU(),
			"goroutines": runtime.NumGoroutine(),
		})

		if ok {
			rsp.Ok(w)
		} else {
			rsp.SetError("status_error", "One or more services are not healthy")
			rsp.InternalServerError(w)
		}
	})
}

// AddEcho adds an echo endpoint to the server.
func (srv *Server) AddEcho() {
	srv.public.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
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
	})
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

// middleware is a middleware function that adds CORS headers to the response before passing it to the next handler.
/* func middleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		method := r.Method
		url := r.URL.String()
		msg := fmt.Sprintf("[%s] %s", method, url)
		logger.InfoContext(r.Context(), msg, slog.String("method", method), slog.String("url", url))

		// Handle preflight requests
		if method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)

			return // Skip the next handler, as this is a preflight request.
		}

		next.ServeHTTP(w, r)
	})
} */
