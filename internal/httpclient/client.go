package httpclient

import "net/http"

// New creates a new *http.Client
// A new client should be created per package that imports it
func New() *http.Client {
	return &http.Client{
		Transport: http.DefaultTransport,
	}
}
