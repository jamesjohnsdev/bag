package provider

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
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

func (provider GithubProvider) Resolve(ctx context.Context, src url.URL, version string) (Resolution, error) {
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
	res, err := provider.downloadReleaseAsset(ctx, extension, owner, repo, asset.GetID(), release, provider.Client.Client())
	if err != nil {
		return Resolution{}, fmt.Errorf("downloading asset: %w", err)
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
		var ext, base string
		switch {
		case strings.HasSuffix(name, ".tar.gz"):
			ext, base = "tar.gz", name[:len(name)-7]
		case strings.HasSuffix(name, ".zip"):
			ext, base = "zip", name[:len(name)-4]
		default:
			continue
		}
		segments := strings.Split(base, "_")
		if len(segments) < 4 {
			segments = strings.Split(base, "-")
			if len(segments) < 4 {
				continue
			}
		}
		OS, arch := segments[len(segments)-2], segments[len(segments)-1]
		if runtime.GOOS != strings.ToLower(OS) {
			continue
		}
		if runtime.GOARCH == normaliseArch(arch) {
			return *asset, ext, nil
		}
	}
	return gh.ReleaseAsset{}, "", errors.New("no asset matching system")
}

func (provider GithubProvider) downloadReleaseAsset(
	ctx context.Context,
	extension,
	owner,
	repo string,
	id int64,
	release *gh.RepositoryRelease,
	client *http.Client,
) (Resolution, error) {
	rc, _, err := provider.Client.Repositories.DownloadReleaseAsset(ctx, owner, repo, id, client)
	if err != nil {
		return Resolution{}, fmt.Errorf("downloading release asset: %w", err)
	}
	if extension == "zip" {
		res, err := extractZip(rc, release)
		if err != nil {
			return Resolution{}, fmt.Errorf("extracting zip: %w", err)
		}
		return res, nil
	}
	if extension == "tar.gz" {
		res, err := extractTarball(rc, release)
		if err != nil {
			return Resolution{}, fmt.Errorf("extracting tarball: %w", err)
		}
		return res, nil
	}

	return Resolution{}, fmt.Errorf("release asset type not supported: %s", extension)
}

type tempFileCloser struct {
	rc  io.ReadCloser
	tmp *os.File
}

type tarCloser struct {
	rc io.ReadCloser
	tr *tar.Reader
}

func (t *tarCloser) Read(p []byte) (int, error) { return t.tr.Read(p) }
func (t *tarCloser) Close() error               { return t.rc.Close() }

func (t *tempFileCloser) Read(p []byte) (int, error) {
	return t.rc.Read(p)
}

func (t *tempFileCloser) Close() error {
	err := t.rc.Close()
	if cerr := t.tmp.Close(); cerr != nil {
		err = errors.Join(err, fmt.Errorf("closing temp file: %w", cerr))
	}
	if rerr := os.Remove(t.tmp.Name()); rerr != nil {
		err = errors.Join(err, fmt.Errorf("removing temp file: %w", rerr))
	}
	return err
}

func isSafeArchiveBinaryName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}
	return filepath.Base(name) == name
}

func extractZip(rc io.ReadCloser, release *gh.RepositoryRelease) (res Resolution, err error) {
	tmp, err := os.CreateTemp("", "bag-*.zip")
	if err != nil {
		return Resolution{}, fmt.Errorf("creating temp zip: %w", err)
	}
	cleanup := true
	defer func() {
		if cerr := rc.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("closing archive reader: %w", cerr))
		}
		if cleanup {
			if cerr := tmp.Close(); cerr != nil {
				err = errors.Join(err, fmt.Errorf("closing temp zip: %w", cerr))
			}
			if rerr := os.Remove(tmp.Name()); rerr != nil {
				err = errors.Join(err, fmt.Errorf("removing temp zip: %w", rerr))
			}
		}
	}()
	_, err = io.Copy(tmp, rc)
	if err != nil {
		return Resolution{}, fmt.Errorf("copying to tmp: %w", err)
	}

	info, err := tmp.Stat()
	if err != nil {
		return Resolution{}, fmt.Errorf("extracting info from temp .zip: %w", err)
	}
	zr, err := zip.NewReader(tmp, info.Size())
	if err != nil {
		return Resolution{}, fmt.Errorf("reading temp .zip: %w", err)
	}

	for _, file := range zr.File {
		if file.Mode()&0o111 != 0 && isSafeArchiveBinaryName(file.Name) {
			rc, err := file.Open()
			if err != nil {
				return Resolution{}, err
			}
			cleanup = false
			return Resolution{
				Reader:          &tempFileCloser{rc, tmp},
				ResolvedVersion: release.GetTagName(),
				BinaryName:      file.Name,
			}, nil
		}
	}
	return Resolution{}, errors.New("no executable found in archive")
}

func extractTarball(rc io.ReadCloser, release *gh.RepositoryRelease) (Resolution, error) {
	gz, err := gzip.NewReader(rc)
	if err != nil {
		return Resolution{}, fmt.Errorf("gzip reader: %w", err)
	}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Resolution{}, fmt.Errorf("reading tar: %w", err)
		}
		if hdr.FileInfo().Mode()&0o111 != 0 && isSafeArchiveBinaryName(hdr.Name) {
			return Resolution{
				Reader:          &tarCloser{rc, tr},
				ResolvedVersion: release.GetTagName(),
				BinaryName:      hdr.Name,
			}, nil
		}
	}
	return Resolution{}, errors.New("no executable found in archive")
}

// normaliseArch takes an arch string and normalises alternative cases to be consistent with GOARCH
func normaliseArch(arch string) string {
	switch arch {
	case "x86_64":
		return "amd64"
	case "aarch64", "arm64v8":
		return "arm64"
	case "i386", "i686":
		return "386"
	default:
		return arch
	}
}
