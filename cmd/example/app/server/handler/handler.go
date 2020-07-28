package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/geo/s2"

	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/dedup"
	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/server/handler/s2geojson"
	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/server/response"
)

func NotFound(w http.ResponseWriter, _ *http.Request) {
	response.SendError(w, &response.Error{
		StatusCode: http.StatusNotFound,
		Status:     response.NotFound,
	})
}

func MethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	response.SendError(w, &response.Error{
		StatusCode: http.StatusMethodNotAllowed,
		Status:     response.InvalidRequest,
	})
}

// Info returns filter configuration parameters.
func Info(filter *dedup.SpatioTemporalFilter, w http.ResponseWriter, r *http.Request) error {
	type resp struct {
		Distance string `json:"distance"`
		TTL      string `json:"ttl"`
	}

	response.SendResponse(w, http.StatusOK, &response.Response{Data: resp{
		Distance: fmt.Sprintf("%0.2f", filter.Distance()),
		TTL:      filter.Interval().String(),
	}})
	return nil
}

// AddLocation runs event location through the filter and returns result.
func AddLocation(filter *dedup.SpatioTemporalFilter, w http.ResponseWriter, r *http.Request) error {
	var ev dedup.Event

	p, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(p, &ev); err != nil {
		return err
	}

	isUnique, err := filter.Filter(ev)
	if err != nil {
		return err
	}

	response.SendResponse(w, http.StatusOK, &response.Response{Data: makePoint(ev.Lat, ev.Lng, map[string]interface{}{
		"type":   "location",
		"unique": isUnique,
		"radius": filter.Distance(),
	})})
	return nil
}

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

func makePoint(lat, lng float64, props map[string]interface{}) s2geojson.Feature {
	pt := s2geojson.NewPoint(lat, lng)
	ft := s2geojson.NewFeature(pt)
	if len(props) > 0 {
		for k := range props {
			ft.Properties[k] = props[k]
		}
	}
	return ft
}

func makeGrid(cu s2.CellUnion) *s2geojson.FeatureCollection {
	fc := s2geojson.NewFeatureCollection()
	for _, cell := range cu {
		c := s2.CellFromCellID(cell)
		pl := s2geojson.NewPolygon(s2.LoopFromCell(c))
		ft := s2geojson.NewFeature(pl)
		fc.Push(ft)
	}
	return fc
}
