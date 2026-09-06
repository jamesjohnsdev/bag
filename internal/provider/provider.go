package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Resolution struct {
	// Reader is set only when no archive extraction was involved (a raw
	// binary download) - the caller streams it directly into the store.
	Reader io.ReadCloser
	// Dir is set when an archive was extracted: the root of the fully
	// extracted tree, to be moved/copied into the store as-is.
	Dir string
	// BinaryRelPath is the path of the chosen executable relative to Dir.
	// Only meaningful when Dir is set.
	BinaryRelPath string
	// Cleanup removes Dir's temporary root. Nil when Reader is set.
	Cleanup         func() error
	ResolvedVersion string
	BinaryName      string
}

type Provider interface {
	Detect(src url.URL) bool
	Resolve(ctx context.Context, src url.URL, binName, version string) (Resolution, error)
}

// buildReg generates the registry using constructors and inserts the httpclient
func buildReg(client *http.Client) (registry []Provider, err error) {
	ghProvider, err := NewGithubProvider(client)
	if err != nil {
		return []Provider{}, fmt.Errorf("registering github provider: %w", err)
	}
	return []Provider{
		ghProvider,
		NewURLProvider(client),
	}, nil
}

// Dispatch checks which provider is needed, and returns it
func Dispatch(source string, client *http.Client) (Provider, error) {
	srcUrl, err := url.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}
	registry, err := buildReg(client)
	if err != nil {
		return nil, fmt.Errorf("building provider register: %w", err)
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

// StripVersionTag removes a trailing "@version" tag from a source's path, if present.
// Stored sources may carry a pinned version (e.g. "github.com/owner/repo@v1.2.3") from
// when the binary was added or previously updated; stripping it before resolving forces
// providers such as GithubProvider to fall back to their latest-release lookup instead of
// re-resolving the same pinned tag forever.
func StripVersionTag(src url.URL) url.URL {
	if idx := strings.LastIndex(src.Path, "@"); idx != -1 {
		src.Path = src.Path[:idx]
	}
	return src
}
