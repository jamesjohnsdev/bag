package provider

import (
	"context"
	"net/url"
)

// URLProvider handles direct URLs that are provided with a `http` or `https` prefix
type URLProvider struct{}

var _ Provider = (*URLProvider)(nil)

func (provider URLProvider) Detect(src url.URL) bool {
	if DirectURL(src) {
		return true
	}
	return false
}

func (provder URLProvider) Resolve(ctx context.Context, source, version string) (Resolution, error) {
	panic("not implemented")
}
