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
	Detect(source url.URL) bool
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
