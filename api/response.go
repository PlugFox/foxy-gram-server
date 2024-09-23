package api

import (
	"encoding/json"
	"net/http"
)

const (
	okStatus  = "ok"
	errStatus = "error"
)

// Error is a generic error structure that is used to send error responses to the client.
type Error struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Extra   interface{} `json:"extra,omitempty"`
}

// Response is a generic response structure that is used to send responses to the client.
type Response struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  *Error      `json:"error,omitempty"`
}

// NewResponse creates a new response object.
func NewResponse() *Response {
	return &Response{
		Status: okStatus,
	}
}

// Error message
func (e *Error) Error() string {
	return e.Message
}

// Set data to response
func (rsp *Response) SetData(data interface{}) *Response {
	rsp.Status = okStatus
	rsp.Error = nil
	rsp.Data = data

	return rsp
}

// Set error to response
func (rsp *Response) SetError(code string, message string, extra ...interface{}) *Response {
	rsp.Status = errStatus
	rsp.Data = nil

	var extraData interface{}
	if len(extra) > 0 {
		extraData = extra[0] // Берем первый переданный аргумент, если он есть
	} else {
		extraData = nil // Если аргумент не был передан, оставляем nil
	}

	rsp.Error = &Error{
		Code:    code,
		Message: message,
		Extra:   extraData,
	}

	return rsp
}

// Send success response to client
func (rsp *Response) Ok(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	rsp.Status = "ok"

	_ = json.NewEncoder(w).Encode(rsp)
}

// Send error response to client
func (rsp *Response) BadRequest(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	rsp.Status = errStatus

	if rsp.Error == nil {
		rsp.Error = &Error{
			Code:    "bad_request",
			Message: "Bad request",
		}
	}

	_ = json.NewEncoder(w).Encode(rsp)
}

// Send error response to client
func (rsp *Response) InternalServerError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)

	rsp.Status = errStatus

	if rsp.Error == nil {
		rsp.Error = &Error{
			Code:    "internal_server_error",
			Message: "Internal server error",
		}
	}

	_ = json.NewEncoder(w).Encode(rsp)
}

// Send error response to client
func (rsp *Response) NotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)

	rsp.Status = errStatus

	if rsp.Error == nil {
		rsp.Error = &Error{
			Code:    "not_found",
			Message: "Not found",
		}
	}

	_ = json.NewEncoder(w).Encode(rsp)
}

// Send error response to client
func (rsp *Response) Unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	rsp.Status = errStatus

	if rsp.Error == nil {
		rsp.Error = &Error{
			Code:    "unauthorized",
			Message: "Unauthorized",
		}
	}

	_ = json.NewEncoder(w).Encode(rsp)
}

// Send error response to client
func (rsp *Response) Forbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)

	rsp.Status = errStatus

	if rsp.Error == nil {
		rsp.Error = &Error{
			Code:    "forbidden",
			Message: "Forbidden",
		}
	}

	_ = json.NewEncoder(w).Encode(rsp)
}

// Send error response to client
func (rsp *Response) MethodNotAllowed(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)

	rsp.Status = errStatus

	if rsp.Error == nil {
		rsp.Error = &Error{
			Code:    "method_not_allowed",
			Message: "Method not allowed",
		}
	}

	_ = json.NewEncoder(w).Encode(rsp)
}
