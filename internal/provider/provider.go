package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Resolution struct {
	Reader          io.ReadCloser
	ResolvedVersion string
	BinaryName      string
}

type Provider interface {
	Detect(src url.URL) bool
	Resolve(ctx context.Context, src url.URL, version string) (Resolution, error)
}

// buildReg generates the registry using constructors and inserts the httpclient
func buildReg(client *http.Client) (registry []Provider) {
	return []Provider{
		NewGithubProvider(client),
		NewURLProvider(client),
	}
}

// Dispatch checks which provider is needed, and returns it
func Dispatch(source string, client *http.Client) (Provider, error) {
	srcUrl, err := url.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}
	registry := buildReg(client)
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
