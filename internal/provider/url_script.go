package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/jamesjohnsdev/bag/internal/output"
)

type URLScriptProvider struct {
	Client *http.Client
}

var _ Provider = (*URLScriptProvider)(nil)

func NewURLScriptProvider(client *http.Client) URLScriptProvider {
	return URLScriptProvider{
		Client: client,
	}
}

func (provider *URLScriptProvider) Detect(src url.URL) bool {
	return DirectURL(src)
}

// TODO: version is always "unknown" here since script sources have nothing to
// resolve a version from. Supporting `update` for scripts will need comparing
// fetched content by hash against the installed hash, and likely prompting the
// user to supply/bump a version manually when it changes (see update.go TODO).
func (provider *URLScriptProvider) Resolve(ctx context.Context, src url.URL, scriptName, version string) (Resolution, error) {
	if version == "" {
		version = "unknown"
	}
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

	if scriptName == "" {
		scriptName = path.Base(src.Path)
	}
	output.Statusf("%s %s (%s)", scriptName, version, output.HumanSize(resp.ContentLength))
	wrappedRC := output.WrapProgress(resp.Body, resp.ContentLength, scriptName)
	res, err := useRawBinary(wrappedRC, scriptName, version)
	if err != nil {
		_ = wrappedRC.Close()
		return Resolution{}, err
	}
	return res, nil
}
