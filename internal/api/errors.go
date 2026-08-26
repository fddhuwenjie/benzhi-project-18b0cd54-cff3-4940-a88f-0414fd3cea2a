package api

import (
	"anaerobic-release/internal/domain"
	"errors"
	"net/http"
)

type requestError struct {
	Message string
	Status  int
}

func (e *requestError) Error() string { return e.Message }

type errorEnvelope struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"request_id,omitempty"`
	} `json:"error"`
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := 500, "internal_error", "服务内部错误"
	var de *domain.Error
	var re *requestError
	if errors.As(err, &de) {
		status, code, message = de.Status, de.Code, de.Message
	} else if errors.As(err, &re) {
		status, code, message = re.Status, "invalid_request", re.Message
	}
	var body errorEnvelope
	body.Error.Code = code
	body.Error.Message = message
	body.Error.RequestID = requestID(r)
	respond(w, status, body)
}
