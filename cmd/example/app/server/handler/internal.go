package handler

import (
	"net/http"

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
