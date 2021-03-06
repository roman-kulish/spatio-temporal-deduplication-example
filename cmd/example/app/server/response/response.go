package response

import (
	"encoding/json"
	"errors"
	"net/http"
)

const (
	OK             Status = "OK"
	InternalError  Status = "ERROR"
	NotFound       Status = "NOT_FOUND"
	InvalidRequest Status = "INVALID_REQUEST"
)

type Status string

// Response is the generic server response struct.
type Response struct {
	Status  Status      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type Error struct {
	StatusCode int
	Status     Status
	Err        error
}

func (r Error) Error() string {
	if r.Err != nil {
		return r.Err.Error()
	}
	return "unknown error"
}

func SendResponse(w http.ResponseWriter, statusCode int, r *Response) {
	if r.Status == "" {
		r.Status = OK
	}
	sendJSON(w, statusCode, r)
}

func SendError(w http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	res := Response{
		Status:  InternalError,
		Message: err.Error(),
	}
	var e *Error
	if errors.As(err, &e) {
		if e.StatusCode != 0 {
			statusCode = e.StatusCode
		}
		if e.Status != "" {
			res.Status = e.Status
		}
	}
	sendJSON(w, statusCode, &res)
}

func sendJSON(w http.ResponseWriter, statusCode int, r *Response) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(r)
}
