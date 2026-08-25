package provider

import (
	"context"
	"net/http"
	"net/url"
)

// GithubProvider will handle any links to Github repositories
// Base repo directory is preferred, and there is no need for address to binary
// Optionally, a token can be provided to authenticate with the API
type GithubProvider struct {
	Client *http.Client
	Token  string
}

var _ Provider = (*GithubProvider)(nil)

func NewGithubProvider(client *http.Client) GithubProvider {
	return GithubProvider{
		Client: client,
	}
}

func (github GithubProvider) Detect(src url.URL) bool {
	if !DirectURL(src) && src.Host == "github.com" {
		return true
	}
	return false
}

func (github GithubProvider) Resolve(ctx context.Context, src url.URL, version string) (Resolution, error) {
	// extract owner/repo from source
	// gh releases API: https://docs.github.com/en/rest/releases/releases#get-a-release-by-tag-name
	//  - version "" → latest otherwise tag

	// decode JSON, walk assets, score each against GOOS+GOARCH
	// download winning asset, stream body into Resolution.Reader

	// must contain OS string (linux, darwin, windows)
	// must contain arch string — normalize aliases (x86_64→amd64, aarch64→arm64)
	// skip checksums/sigs (.sha256, .asc, .sig)
	// prefer tarballs/zips over package formats (.deb, .rpm, .apk)
	// if still ambiguous, prefer shorter name (less junk in filename)
	panic("not implemented")
}
