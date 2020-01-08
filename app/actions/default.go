package actions

import (
	"net/http"

	"github.com/lbryio/lbry.go/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/api"
)

// Root Handler is the default handler
func Root(r *http.Request) api.Response {
	if r.URL.Path == "/" {
		return api.Response{Data: "Welcome to Lighthouse!"}
	}
	return api.Response{Status: http.StatusNotFound, Error: errors.Err("404 Not Found")}
}

// Test Handler can be used for testing and triggering
func Test(r *http.Request) api.Response {
	return api.Response{Data: "ok"}
}
