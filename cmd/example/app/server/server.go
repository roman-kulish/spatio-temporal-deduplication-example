package server

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/config"
	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/dedup"
	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/server/response"
)

// HandlerFunc is a wrapped handler functions.
type HandlerFunc func(*dedup.SpatioTemporalFilter, http.ResponseWriter, *http.Request) error

// WithSpatioTemporalFilter wraps handler function into http.HandlerFunc and
// injects dedup.Filter.
func WithSpatioTemporalFilter(f *dedup.SpatioTemporalFilter, fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(f, w, r); err != nil {
			response.SendError(w, err)
		}
	}
}

// New creates, configures and returns an instance of http.Server.
func New(cfg *config.Server, filter *dedup.SpatioTemporalFilter) (*http.Server, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	dir = filepath.Join(dir, "public")

	s := http.Server{
		Addr:              cfg.Addr,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}
	s.Handler = initRoutes(dir, filter)
	return &s, nil
}
