package provider

import (
	"context"
	"net/http"
	"net/url"
)

// URLProvider handles direct URLs that are provided with a `http` or `https` prefix
type URLProvider struct {
	Client *http.Client
}

var _ Provider = (*URLProvider)(nil)

func NewURLProvider(client *http.Client) URLProvider {
	return URLProvider{
		Client: client,
	}
}

func (provider URLProvider) Detect(src url.URL) bool {
	if DirectURL(src) {
		return true
	}
	return false
}

func (provder URLProvider) Resolve(ctx context.Context, src url.URL, version string) (Resolution, error) {
	panic("not implemented")
}
