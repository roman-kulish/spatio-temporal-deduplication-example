package handler

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/dedup"
	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app/server/response"
)

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
		"radius": filter.DistanceMeters(),
	})})
	return nil
}
