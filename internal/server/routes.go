package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/render"
)

// echo route for testing purposes
func echoRoute(w http.ResponseWriter, r *http.Request) {
	// Create a map to hold the request data
	var data map[string]any

	// Decode the request body into the data map
	if r.ContentLength != 0 {
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			if err := render.Decode(r, &data); err != nil {
				NewResponse().SetError("bad_request", err.Error()).BadRequest(w)

				return
			}
		} else {
			msg := fmt.Sprintf("Content-Type: %s", r.Header.Get("Content-Type"))

			NewResponse().SetError("bad_request", "Content-Type must be application/json", msg).BadRequest(w)

			return
		}
	}

	NewResponse().SetData(struct {
		URL     string         `json:"url"`
		Remote  string         `json:"remote"`
		Method  string         `json:"method"`
		Headers http.Header    `json:"headers"`
		Body    map[string]any `json:"body"`
	}{
		URL:     r.URL.String(),
		Remote:  r.RemoteAddr,
		Method:  r.Method,
		Headers: r.Header,
		Body:    data,
	}).Ok(w)
}
