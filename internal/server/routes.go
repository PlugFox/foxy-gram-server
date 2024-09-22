package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/render"
	"github.com/plugfox/foxy-gram-server/api"
)

// echo route for testing purposes
func echoRoute(w http.ResponseWriter, r *http.Request) {
	// Create a new response object
	rsp := &api.Response{}

	// Create a map to hold the request data
	var data map[string]any

	// Decode the request body into the data map
	if r.ContentLength != 0 && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		err := render.Decode(r, &data)
		if err != nil {
			rsp.SetError("bad_request", err.Error())
			rsp.BadRequest(w)

			return
		}
	}

	rsp.SetData(struct {
		Remote  string         `json:"remote"`
		Method  string         `json:"method"`
		Headers http.Header    `json:"headers"`
		Body    map[string]any `json:"body"`
	}{
		Remote:  r.RemoteAddr,
		Method:  r.Method,
		Headers: r.Header,
		Body:    data,
	})
	rsp.Ok(w)
}
