package api

import (
	"encoding/json"
	"net/http"
)

// Error is a generic error structure that is used to send error responses to the client.
type Error struct {
	Code    string `json:"code,required"`
	Message string `json:"message,required"`
}

// Response is a generic response structure that is used to send responses to the client.
type Response struct {
	Status string      `json:"status,required"`
	Data   interface{} `json:"data,omitempty"`
	Error  *Error      `json:"error,omitempty"`
}

// Error message
func (e *Error) Error() string {
	return e.Message
}

// Set data to response
func (rsp *Response) SetData(data interface{}) {
	rsp.Data = data
	rsp.Error = nil
}

// Set error to response
func (rsp *Response) SetError(code string, message string) {
	rsp.Data = nil
	rsp.Error = &Error{
		Code:    code,
		Message: message,
	}
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
	rsp.Status = "error"
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
	rsp.Status = "error"
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
	rsp.Status = "error"
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
	rsp.Status = "error"
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
	rsp.Status = "error"
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
	rsp.Status = "error"
	if rsp.Error == nil {
		rsp.Error = &Error{
			Code:    "method_not_allowed",
			Message: "Method not allowed",
		}
	}
	_ = json.NewEncoder(w).Encode(rsp)
}
