package provider

import (
	"context"
	"fmt"
	"io"
	"net/url"
)

type Resolution struct {
	Reader          io.ReadCloser
	ResolvedVersion string
	BinaryName      string
}

type Provider interface {
	Detect(src url.URL) bool
	Resolve(ctx context.Context, source, version string) (Resolution, error)
}

var registry = []Provider{}

// Dispatch checks which provider is needed, and returns it
func Dispatch(source string) (Provider, error) {
	srcUrl, err := url.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}
	for _, provider := range registry {
		if provider.Detect(*srcUrl) {
			return provider, nil
		}
	}
	return nil, fmt.Errorf("no provider for source %s", source)
}

// DirectURL is a helper which returns true if a direct url has been provided
func DirectURL(src url.URL) bool {
	if src.Scheme == "https" || src.Scheme == "http" {
		return true
	}
	return false
}
