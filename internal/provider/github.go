package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	gh "github.com/google/go-github/v89/github"
	"github.com/jamesjohnsdev/bag/internal/config"
)

// GithubProvider will handle any links to Github repositories
// Base repo directory is preferred, and there is no need for address to binary
// Optionally, a token can be provided to authenticate with the API
type GithubProvider struct {
	Client *gh.Client
	Token  string
}

var _ Provider = (*GithubProvider)(nil)

func NewGithubProvider(client *http.Client) (GithubProvider, error) {
	token := config.Get().GhToken
	opts := []gh.ClientOptionsFunc{gh.WithHTTPClient(client)}
	if token != "" {
		opts = append(opts, gh.WithAuthToken(token))
	}
	ghClient, err := gh.NewClient(opts...)
	if err != nil {
		return GithubProvider{}, fmt.Errorf("creating github client: %w", err)
	}
	return GithubProvider{
		Client: ghClient,
		Token:  token,
	}, nil
}

func (github GithubProvider) Detect(src url.URL) bool {
	if src.Host == "" && strings.HasPrefix(src.Path, "github.com/") {
		return true
	}
	return false
}

func (provider GithubProvider) Resolve(ctx context.Context, src url.URL, binName, version string) (Resolution, error) {
	// extract owner/repo from source
	srcPath := strings.TrimPrefix(src.Path, "github.com/")
	path := strings.Split(srcPath, "/")
	if len(path) < 2 {
		return Resolution{}, errors.New("path does not include owner and respository")
	}
	owner, repo := path[0], path[1]
	parts := strings.SplitN(repo, "@", 2)
	repo = parts[0]
	if len(parts) == 2 {
		version = parts[1] // URL version tag will override param
	}

	release, err := provider.getRelease(ctx, owner, repo, version)
	if err != nil {
		return Resolution{}, fmt.Errorf("fetching release: %w", err)
	}
	asset, extension, err := getReleaseAsset(release)
	if err != nil {
		return Resolution{}, fmt.Errorf("getting release asset: %w", err)
	}
	// download winning asset, stream body into Resolution.Reader
	res, err := provider.downloadReleaseAsset(ctx, extension, owner, repo, asset.GetID(), asset.GetName(), release.GetTagName(), provider.Client.Client())
	if err != nil {
		return Resolution{}, fmt.Errorf("downloading asset %s: %w", asset.GetName(), err)
	}
	return res, nil
}

// getRelease retrieves the specified github repo release version, or otherwise defaults to latest
func (provider GithubProvider) getRelease(ctx context.Context, owner, repo, version string) (*gh.RepositoryRelease, error) {
	if version == "" {
		release, _, err := provider.Client.Repositories.GetLatestRelease(ctx, owner, repo)
		if err != nil {
			return release, fmt.Errorf("latest release: %w", err)
		}
		return release, nil
	} else {
		release, _, err := provider.Client.Repositories.GetReleaseByTag(ctx, owner, repo, version)
		if err != nil {
			return release, fmt.Errorf("release %s: %w", version, err)
		}
		return release, nil
	}
}

// getReleaseAsset parses and returns the most appropriate release asset based on the system attributes
// supports conventional asset naming with underscore or hyphen separators, e.g. `goanna_0.2.1_darwin_amd64.tar.gz`
// or `golangci-lint-2.13.1-linux-amd64.tar.gz`. OS and arch are taken as the last two separator-delimited
// segments so tool names/versions may themselves contain the separator character.
func getReleaseAsset(release *gh.RepositoryRelease) (asset gh.ReleaseAsset, extension string, err error) {
	for _, asset := range release.Assets {
		name := asset.GetName()
		extension, base, err := parseAssetName(name)
		if err != nil {
			continue
		}
		if matchesSystem(base, "_") || matchesSystem(base, "-") {
			return *asset, extension, nil
		}
	}
	return gh.ReleaseAsset{}, "", errors.New("no asset matching system")
}

// downloadReleaseAsset gets a GitHub asset, handles based on type, and returns a Resolution and error
// This is mostly a wrapper specific for GitHub releases around handleAssetVariations
func (provider GithubProvider) downloadReleaseAsset(
	ctx context.Context,
	extension,
	owner,
	repo string,
	assetID int64,
	binName, version string,
	client *http.Client,
) (Resolution, error) {
	rc, _, err := provider.Client.Repositories.DownloadReleaseAsset(ctx, owner, repo, assetID, client)
	if err != nil {
		return Resolution{}, fmt.Errorf("downloading release asset %s: %w", binName, err)
	}
	return handleAssetVariations(rc, binName, version, extension)
}
