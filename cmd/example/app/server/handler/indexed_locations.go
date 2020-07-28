package handler

import (
	"net/http"

	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/dedup"
	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/server/handler/s2geojson"
	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/server/response"
)

// IndexedLocations outputs a list of indexed locations from the filter.
func IndexedLocations(filter *dedup.SpatioTemporalFilter, w http.ResponseWriter, _ *http.Request) error {
	fc := s2geojson.NewFeatureCollection()

	err := filter.IndexedLocations(func(lat, lng float64) error {
		fc.Push(makePoint(lat, lng, nil))
		return nil
	})
	if err != nil {
		return err
	}

	response.SendResponse(w, http.StatusOK, &response.Response{Data: fc})
	return nil
}
