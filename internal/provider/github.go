package provider

import (
	"context"
	"net/url"
)

// GithubProvider will handle any links to Github repositories
// Base repo directory is preferred, and there is no need for address to binary
type GithubProvider struct{}

var _ Provider = (*GithubProvider)(nil)

func (github GithubProvider) Detect(src url.URL) bool {
	if !DirectURL(src) && src.Host == "github.com" {
		return true
	}
	return false
}

func (github GithubProvider) Resolve(ctx context.Context, source, version string) (Resolution, error) {
	panic("not implemented")
}
