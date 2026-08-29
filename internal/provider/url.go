package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.String(), nil)
	if err != nil {
		return Resolution{}, fmt.Errorf("creating request for %s: %w", src.String(), err)
	}
	resp, err := provider.Client.Do(req)
	if err != nil {
		return Resolution{}, fmt.Errorf("unable to fetch %s: %w", src.String(), err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = resp.Body.Close()
		return Resolution{}, fmt.Errorf("fetching %s: unexpected status %s", src.String(), resp.Status)
	}
	extension, _, err := parseAssetName(src.Path)
	if err != nil {
		_ = resp.Body.Close()
		return Resolution{}, fmt.Errorf("parsing asset name: %w", err)
	}
	if extension == "" && binName == "" {
		binName = path.Base(src.Path)
	}
	return handleAssetVariations(resp.Body, binName, version, extension)
}
