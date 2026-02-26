package api

import (
	"context"
	"net/http"
	"time"
)

// MetricsServer is an HTTP server that only serves /metrics.
type MetricsServer struct {
	srv *http.Server
}

// NewMetricsServer creates a server that serves handler at path (e.g. "/metrics") and GET /healthz.
func NewMetricsServer(listen, path string, handler http.Handler) *MetricsServer {
	if path == "" {
		path = "/metrics"
	}
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return &MetricsServer{
		srv: &http.Server{
			Addr:    listen,
			Handler: mux,
		},
	}
}

// ListenAndServe starts the server (blocks).
func (s *MetricsServer) ListenAndServe() error {
	return s.srv.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *MetricsServer) Shutdown(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}
	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.srv.Shutdown(ctx2)
}
