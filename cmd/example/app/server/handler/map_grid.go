package handler

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/golang/geo/s2"

	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/dedup"
	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/server/response"
)

// MapGrid outputs a grid of S2 Cells for the map, using filter level.
func MapGrid(filter *dedup.SpatioTemporalFilter, w http.ResponseWriter, r *http.Request) error {
	type latLng struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}

	type bbox struct {
		Hi latLng `json:"hi"`
		Lo latLng `json:"lo"`
	}

	var b bbox
	p, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(p, &b); err != nil {
		return err
	}

	hi := s2.LatLngFromDegrees(b.Hi.Lat, b.Hi.Lng)
	lo := s2.LatLngFromDegrees(b.Lo.Lat, b.Lo.Lng)

	rect := s2.RectFromLatLng(hi).AddPoint(lo)
	rc := s2.RegionCoverer{
		MinLevel: filter.Level(),
		MaxLevel: filter.Level(),
	}
	cu := rc.Covering(rect)

	response.SendResponse(w, http.StatusOK, &response.Response{Data: makeGrid(cu)})
	return nil
}
