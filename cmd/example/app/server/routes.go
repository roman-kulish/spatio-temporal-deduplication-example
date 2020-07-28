package server

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"

	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/dedup"
	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/server/handler"
)

func initRoutes(publicDir string, filter *dedup.SpatioTemporalFilter) chi.Router {
	mux := chi.NewRouter()
	mux.Use(
		middleware.NoCache,
		cors.Handler(cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "OPTIONS"},
			AllowedHeaders: []string{"Accept", "Content-Type"},
			MaxAge:         300,
		}),
	)

	// internal routes
	mux.NotFound(handler.NotFound)
	mux.MethodNotAllowed(handler.MethodNotAllowed)
	mux.Mount("/debug", middleware.Profiler())

	// public routes
	mux.Get("/info", WithSpatioTemporalFilter(filter, handler.Info))
	mux.Post("/grid", WithSpatioTemporalFilter(filter, handler.MapGrid))
	mux.Get("/locations", WithSpatioTemporalFilter(filter, handler.IndexedLocations))
	mux.Post("/locations", WithSpatioTemporalFilter(filter, handler.AddLocation))
	mux.Method(http.MethodGet, "/*", http.FileServer(http.Dir(publicDir)))

	return mux
}
