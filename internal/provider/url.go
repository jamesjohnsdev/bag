package provider

import (
	"context"
	"fmt"
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
	return DirectURL(src)
}

func (provider URLProvider) Resolve(ctx context.Context, src url.URL, binName, version string) (Resolution, error) {
	resp, err := provider.Client.Get(src.String())
	if err != nil {
		return Resolution{}, fmt.Errorf("unable to fetch %s: %w", src.String(), err)
	}
	extension, _, err := parseAssetName(src.Path)
	if err != nil {
		return Resolution{}, fmt.Errorf("parsing asset name: %w", err)
	}
	return handleAssetVariations(resp.Body, binName, version, extension)
}
