package provider

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"testing"

	gh "github.com/google/go-github/v89/github"
)

func ptr[T any](v T) *T { return &v }

func newTestProvider(t *testing.T, mux *http.ServeMux) GithubProvider {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client, err := gh.NewClient(
		gh.WithHTTPClient(srv.Client()),
		gh.WithEnterpriseURLs(srv.URL+"/", srv.URL+"/"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return GithubProvider{Client: client}
}

type tarEntry struct {
	name    string
	mode    int64
	content string
}

type zipEntry struct {
	name    string
	mode    fs.FileMode
	content string
}

func makeTarGz(t *testing.T, entries []tarEntry) io.ReadCloser {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Mode: e.mode, Size: int64(len(e.content))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(e.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return io.NopCloser(bytes.NewReader(buf.Bytes()))
}

func makeZip(t *testing.T, entries []zipEntry) io.ReadCloser {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		fh := &zip.FileHeader{Name: e.name}
		fh.SetMode(e.mode)
		w, err := zw.CreateHeader(fh)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return io.NopCloser(bytes.NewReader(buf.Bytes()))
}

func TestNormaliseArch(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"x86_64", "amd64"},
		{"aarch64", "arm64"},
		{"arm64v8", "arm64"},
		{"i386", "386"},
		{"i686", "386"},
		{"amd64", "amd64"},
		{"arm64", "arm64"},
		{"386", "386"},
		{"unknown_arch", "unknown_arch"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := normaliseArch(tt.in); got != tt.want {
				t.Errorf("normaliseArch(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestDetect(t *testing.T) {
	p := GithubProvider{}
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{"scheme-less github shorthand", "github.com/owner/repo", true},
		{"https github is direct url", "https://github.com/owner/repo", false},
		{"http github is direct url", "http://github.com/owner/repo", false},
		{"gitlab not github", "gitlab.com/owner/repo", false},
		{"direct https url", "https://example.com/binary", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.src)
			if err != nil {
				t.Fatal(err)
			}
			if got := p.Detect(*u); got != tt.want {
				t.Errorf("Detect(%q) = %v, want %v", tt.src, got, tt.want)
			}
		})
	}
}

func TestGetReleaseAsset(t *testing.T) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	makeRelease := func(names ...string) *gh.RepositoryRelease {
		assets := make([]*gh.ReleaseAsset, len(names))
		for i, n := range names {
			assets[i] = &gh.ReleaseAsset{Name: ptr(n)}
		}
		return &gh.RepositoryRelease{Assets: assets}
	}

	matchingTar := fmt.Sprintf("tool_0.1.0_%s_%s.tar.gz", goos, goarch)
	matchingZip := fmt.Sprintf("tool_0.1.0_%s_%s.zip", goos, goarch)

	tests := []struct {
		name    string
		release *gh.RepositoryRelease
		wantErr bool
		wantExt string
	}{
		{
			name:    "empty assets",
			release: makeRelease(),
			wantErr: true,
		},
		{
			name:    "only checksums.txt",
			release: makeRelease("checksums.txt"),
			wantErr: true,
		},
		{
			name:    "sha256 skipped",
			release: makeRelease(fmt.Sprintf("tool_0.1.0_%s_%s.sha256", goos, goarch)),
			wantErr: true,
		},
		{
			name:    "sig skipped",
			release: makeRelease(fmt.Sprintf("tool_0.1.0_%s_%s.sig", goos, goarch)),
			wantErr: true,
		},
		{
			name:    "no recognised extension",
			release: makeRelease(fmt.Sprintf("tool_0.1.0_%s_%s.deb", goos, goarch)),
			wantErr: true,
		},
		{
			name:    "too few underscore segments",
			release: makeRelease(fmt.Sprintf("tool_%s_%s.tar.gz", goos, goarch)),
			wantErr: true,
		},
		{
			name:    "wrong OS",
			release: makeRelease("tool_0.1.0_wrongos_amd64.tar.gz"),
			wantErr: true,
		},
		{
			name:    "wrong arch",
			release: makeRelease(fmt.Sprintf("tool_0.1.0_%s_wrongarch.tar.gz", goos)),
			wantErr: true,
		},
		{
			name:    "matching tar.gz",
			release: makeRelease(matchingTar),
			wantExt: "tar.gz",
		},
		{
			name:    "matching zip",
			release: makeRelease(matchingZip),
			wantExt: "zip",
		},
		{
			name:    "checksums.txt before match",
			release: makeRelease("checksums.txt", matchingTar),
			wantExt: "tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ext, err := getReleaseAsset(tt.release)
			if (err != nil) != tt.wantErr {
				t.Fatalf("getReleaseAsset() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && ext != tt.wantExt {
				t.Errorf("ext = %q, want %q", ext, tt.wantExt)
			}
		})
	}
}

func TestGetReleaseAssetArchAlias(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("alias test only meaningful on amd64")
	}
	release := &gh.RepositoryRelease{
		Assets: []*gh.ReleaseAsset{
			{Name: ptr(fmt.Sprintf("tool_0.1.0_%s_x86_64.tar.gz", runtime.GOOS))},
		},
	}
	_, _, err := getReleaseAsset(release)
	if err != nil {
		t.Fatalf("x86_64 should match amd64: %v", err)
	}
}

func TestExtractTarball(t *testing.T) {
	tests := []struct {
		name    string
		entries []tarEntry
		wantErr bool
		wantBin string
	}{
		{
			name:    "executable at root",
			entries: []tarEntry{{name: "mybinary", mode: 0o755, content: "ELF"}},
			wantBin: "mybinary",
		},
		{
			name:    "executable in subdir found",
			entries: []tarEntry{{name: "subdir/mybinary", mode: 0o755, content: "ELF"}},
			wantBin: "mybinary",
		},
		{
			name:    "non-executable ignored",
			entries: []tarEntry{{name: "README.md", mode: 0o644, content: "readme"}},
			wantErr: true,
		},
		{
			name: "non-executable before executable",
			entries: []tarEntry{
				{name: "README.md", mode: 0o644, content: "readme"},
				{name: "mybinary", mode: 0o755, content: "ELF"},
			},
			wantBin: "mybinary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := makeTarGz(t, tt.entries)
			res, err := extractTarball(rc, "", "v1.0.0")
			if (err != nil) != tt.wantErr {
				t.Fatalf("extractTarball() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			defer func() {
				if err := res.Reader.Close(); err != nil {
					t.Fatal(err)
				}
			}()
			if res.BinaryName != tt.wantBin {
				t.Errorf("BinaryName = %q, want %q", res.BinaryName, tt.wantBin)
			}
			if res.ResolvedVersion != "v1.0.0" {
				t.Errorf("ResolvedVersion = %q, want v1.0.0", res.ResolvedVersion)
			}
			content, err := io.ReadAll(res.Reader)
			if err != nil {
				t.Fatal(err)
			}
			if string(content) != "ELF" {
				t.Errorf("content = %q, want ELF", string(content))
			}
		})
	}
}

func TestExtractZip(t *testing.T) {
	tests := []struct {
		name    string
		entries []zipEntry
		wantErr bool
		wantBin string
	}{
		{
			name:    "executable at root",
			entries: []zipEntry{{name: "mybinary", mode: 0o755, content: "ELF"}},
			wantBin: "mybinary",
		},
		{
			name:    "executable in subdir found",
			entries: []zipEntry{{name: "subdir/mybinary", mode: 0o755, content: "ELF"}},
			wantBin: "mybinary",
		},
		{
			name:    "no executable",
			entries: []zipEntry{{name: "README.md", mode: 0o644, content: "readme"}},
			wantErr: true,
		},
		{
			name: "non-executable before executable",
			entries: []zipEntry{
				{name: "README.md", mode: 0o644, content: "readme"},
				{name: "mybinary", mode: 0o755, content: "ELF"},
			},
			wantBin: "mybinary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := makeZip(t, tt.entries)
			res, err := extractZip(rc, "", "v1.0.0")
			if (err != nil) != tt.wantErr {
				t.Fatalf("extractZip() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			defer func() {
				if err := res.Reader.Close(); err != nil {
					t.Fatal(err)
				}
			}()
			if res.BinaryName != tt.wantBin {
				t.Errorf("BinaryName = %q, want %q", res.BinaryName, tt.wantBin)
			}
			if res.ResolvedVersion != "v1.0.0" {
				t.Errorf("ResolvedVersion = %q, want v1.0.0", res.ResolvedVersion)
			}
			content, err := io.ReadAll(res.Reader)
			if err != nil {
				t.Fatal(err)
			}
			if string(content) != "ELF" {
				t.Errorf("content = %q, want ELF", string(content))
			}
		})
	}
}

func TestGetRelease(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/owner/repo/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		if err := json.NewEncoder(w).Encode(&gh.RepositoryRelease{TagName: "v2.0.0", ID: 2}); err != nil {
			t.Error(err)
		}
	})
	mux.HandleFunc("/api/v3/repos/owner/repo/releases/tags/v1.0.0", func(w http.ResponseWriter, _ *http.Request) {
		if err := json.NewEncoder(w).Encode(&gh.RepositoryRelease{TagName: "v1.0.0", ID: 1}); err != nil {
			t.Error(err)
		}
	})

	provider := newTestProvider(t, mux)

	t.Run("latest when version empty", func(t *testing.T) {
		release, err := provider.getRelease(context.Background(), "owner", "repo", "")
		if err != nil {
			t.Fatal(err)
		}
		if release.GetTagName() != "v2.0.0" {
			t.Errorf("tag = %q, want v2.0.0", release.GetTagName())
		}
	})

	t.Run("by tag when version set", func(t *testing.T) {
		release, err := provider.getRelease(context.Background(), "owner", "repo", "v1.0.0")
		if err != nil {
			t.Fatal(err)
		}
		if release.GetTagName() != "v1.0.0" {
			t.Errorf("tag = %q, want v1.0.0", release.GetTagName())
		}
	})
}
