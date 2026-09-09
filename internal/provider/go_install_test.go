package provider

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGoScript is a POSIX sh stand-in for the `go` binary. It understands just
// enough of `go install <mod>@<version>` and `go version -m <path>` to drive
// GoProvider.Resolve, with behavior steered by env vars so each test can
// control success/failure without touching the real network or toolchain.
const fakeGoScript = `#!/bin/sh
case "$1" in
  install)
    if [ -n "$FAKE_GO_INSTALL_EXIT" ] && [ "$FAKE_GO_INSTALL_EXIT" != "0" ]; then
      printf '%s' "$FAKE_GO_INSTALL_STDERR" >&2
      exit "$FAKE_GO_INSTALL_EXIT"
    fi
    if [ -z "$FAKE_GO_SKIP_BINARY" ]; then
      modpart="${2%@*}"
      name=$(basename "$modpart")
      : > "$GOBIN/$name"
      chmod +x "$GOBIN/$name"
    fi
    exit 0
    ;;
  version)
    if [ -n "$FAKE_DISALLOW_VERSION" ]; then
      printf 'version subcommand should not be called\n' >&2
      exit 1
    fi
    if [ -n "$FAKE_GO_VERSION_EXIT" ] && [ "$FAKE_GO_VERSION_EXIT" != "0" ]; then
      exit "$FAKE_GO_VERSION_EXIT"
    fi
    printf '%s\n' "$FAKE_GO_VERSION_OUTPUT"
    exit 0
    ;;
  *)
    exit 1
    ;;
esac
`

// installFakeGo puts fakeGoScript on PATH ahead of the real PATH so
// exec.LookPath("go") and subsequent `go` invocations hit our stand-in, while
// shell builtins the script itself needs (basename, chmod, printf) still
// resolve normally.
func installFakeGo(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "go")
	if err := os.WriteFile(scriptPath, []byte(fakeGoScript), 0o755); err != nil {
		t.Fatalf("writing fake go script: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// removeGoFromPATH points PATH at an empty directory so exec.LookPath("go")
// fails, simulating a machine without the Go toolchain installed.
func removeGoFromPATH(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
}

func mustParseURL(t *testing.T, raw string) url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parsing URL %q: %v", raw, err)
	}
	return *u
}

func TestGoProviderDetect(t *testing.T) {
	p := GoProvider{}
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{"go scheme", "go:golang.org/x/tools/cmd/goimports", true},
		{"go scheme with version", "go:example.com/foo/bar@v1.2.3", true},
		{"github shorthand", "github.com/owner/repo", false},
		{"https URL", "https://example.com/install.sh", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := mustParseURL(t, tt.src)
			if got := p.Detect(u); got != tt.want {
				t.Errorf("Detect(%q) = %v, want %v", tt.src, got, tt.want)
			}
		})
	}
}

func TestGoProviderResolve(t *testing.T) {
	ctx := context.Background()
	p := NewGoProvider()

	t.Run("go not on PATH", func(t *testing.T) {
		removeGoFromPATH(t)
		u := mustParseURL(t, "go:example.com/foo/bar")
		_, err := p.Resolve(ctx, u, "", "")
		if err == nil || !strings.Contains(err.Error(), "go not installed on path") {
			t.Fatalf("err = %v, want \"go not installed on path\"", err)
		}
	})

	t.Run("explicit version skips go version -m", func(t *testing.T) {
		installFakeGo(t)
		t.Setenv("FAKE_DISALLOW_VERSION", "1")
		u := mustParseURL(t, "go:example.com/foo/bar")
		res, err := p.Resolve(ctx, u, "", "v1.2.3")
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		defer func() {
			_ = res.Cleanup()
		}()
		if res.ResolvedVersion != "v1.2.3" {
			t.Errorf("ResolvedVersion = %q, want v1.2.3", res.ResolvedVersion)
		}
		if res.BinaryName != "bar" {
			t.Errorf("BinaryName = %q, want bar (defaulted from module path)", res.BinaryName)
		}
		if res.BinaryRelPath != "bar" {
			t.Errorf("BinaryRelPath = %q, want bar", res.BinaryRelPath)
		}
	})

	t.Run("src version tag overrides version param", func(t *testing.T) {
		installFakeGo(t)
		t.Setenv("FAKE_DISALLOW_VERSION", "1")
		u := mustParseURL(t, "go:example.com/foo/bar@v9.9.9")
		res, err := p.Resolve(ctx, u, "", "v1.0.0")
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		defer func() {
			_ = res.Cleanup()
		}()
		if res.ResolvedVersion != "v9.9.9" {
			t.Errorf("ResolvedVersion = %q, want v9.9.9 (src tag should win)", res.ResolvedVersion)
		}
	})

	t.Run("custom binName kept separate from module-derived BinaryRelPath", func(t *testing.T) {
		installFakeGo(t)
		u := mustParseURL(t, "go:example.com/foo/bar@v1.0.0")
		res, err := p.Resolve(ctx, u, "custom-tool", "")
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		defer func() {
			_ = res.Cleanup()
		}()
		if res.BinaryName != "custom-tool" {
			t.Errorf("BinaryName = %q, want custom-tool", res.BinaryName)
		}
		if res.BinaryRelPath != "bar" {
			t.Errorf("BinaryRelPath = %q, want bar (go install always names after module dir)", res.BinaryRelPath)
		}
	})

	t.Run("empty version resolves latest via go version -m", func(t *testing.T) {
		installFakeGo(t)
		t.Setenv("FAKE_GO_VERSION_OUTPUT", "mod\texample.com/foo/bar\tv3.4.5\th1:abcdef=")
		u := mustParseURL(t, "go:example.com/foo/bar")
		res, err := p.Resolve(ctx, u, "", "")
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		defer func() {
			_ = res.Cleanup()
		}()
		if res.ResolvedVersion != "v3.4.5" {
			t.Errorf("ResolvedVersion = %q, want v3.4.5", res.ResolvedVersion)
		}
	})

	t.Run("go install failure surfaces stderr", func(t *testing.T) {
		installFakeGo(t)
		t.Setenv("FAKE_GO_INSTALL_EXIT", "1")
		t.Setenv("FAKE_GO_INSTALL_STDERR", "boom: module not found")
		u := mustParseURL(t, "go:example.com/foo/bar@v1.0.0")
		_, err := p.Resolve(ctx, u, "", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "go install failed") || !strings.Contains(err.Error(), "boom: module not found") {
			t.Errorf("err = %v, want it to mention \"go install failed\" and stderr", err)
		}
	})

	t.Run("go version -m command failure", func(t *testing.T) {
		installFakeGo(t)
		t.Setenv("FAKE_GO_VERSION_EXIT", "1")
		u := mustParseURL(t, "go:example.com/foo/bar")
		_, err := p.Resolve(ctx, u, "", "")
		if err == nil || !strings.Contains(err.Error(), "getting version") {
			t.Fatalf("err = %v, want \"getting version\"", err)
		}
	})

	t.Run("go version -m output missing mod line", func(t *testing.T) {
		installFakeGo(t)
		t.Setenv("FAKE_GO_VERSION_OUTPUT", "garbage\nnot a mod line")
		u := mustParseURL(t, "go:example.com/foo/bar")
		_, err := p.Resolve(ctx, u, "", "")
		if err == nil || !strings.Contains(err.Error(), "parsing go mod version") {
			t.Fatalf("err = %v, want \"parsing go mod version\"", err)
		}
	})

	t.Run("missing binary after install reports stat error", func(t *testing.T) {
		installFakeGo(t)
		t.Setenv("FAKE_GO_SKIP_BINARY", "1")
		u := mustParseURL(t, "go:example.com/foo/bar@v1.0.0")
		_, err := p.Resolve(ctx, u, "", "")
		if err == nil || !strings.Contains(err.Error(), "stat binary") {
			t.Fatalf("err = %v, want \"stat binary\"", err)
		}
	})

	t.Run("cleanup removes temp dir", func(t *testing.T) {
		installFakeGo(t)
		u := mustParseURL(t, "go:example.com/foo/bar@v1.0.0")
		res, err := p.Resolve(ctx, u, "", "")
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if err := res.Cleanup(); err != nil {
			t.Fatalf("Cleanup: %v", err)
		}
		if _, err := os.Stat(res.Dir); !os.IsNotExist(err) {
			t.Errorf("Dir %q still exists after Cleanup", res.Dir)
		}
	})
}

func TestParseGoModVersion(t *testing.T) {
	tests := []struct {
		name    string
		out     string
		want    string
		wantErr bool
	}{
		{
			name: "simple mod line",
			out:  "path\texample.com/foo/bar\ndep\texample.com/foo/bar\ngo1.21\nmod\texample.com/foo/bar\tv3.4.5\th1:abcdef=",
			want: "v3.4.5",
		},
		{
			name: "mod line is first",
			out:  "mod\texample.com/foo/bar\tv1.0.0\th1:xxx=",
			want: "v1.0.0",
		},
		{
			name:    "no mod line",
			out:     "path\texample.com/foo/bar\ndep\texample.com/baz\tv1.0.0\th1:xxx=",
			wantErr: true,
		},
		{
			name:    "empty input",
			out:     "",
			wantErr: true,
		},
		{
			name:    "mod line with too few fields is ignored",
			out:     "mod\texample.com/foo/bar",
			wantErr: true,
		},
		{
			name: "extra whitespace between fields still splits",
			out:  "mod   example.com/foo/bar   v2.0.0   h1:xxx=",
			want: "v2.0.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGoModVersion([]byte(tt.out))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseGoModVersion(%q) = %q, want error", tt.out, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseGoModVersion(%q) unexpected error: %v", tt.out, err)
			}
			if got != tt.want {
				t.Errorf("parseGoModVersion(%q) = %q, want %q", tt.out, got, tt.want)
			}
		})
	}
}

// FuzzParseGoModVersion checks that parseGoModVersion never panics on
// arbitrary `go version -m` output, and that it is deterministic (same input
// always yields the same result).
func FuzzParseGoModVersion(f *testing.F) {
	seeds := []string{
		"",
		"mod\texample.com/foo/bar\tv1.0.0\th1:xxx=",
		"garbage",
		"mod\ttooshort",
		"mod\texample.com/foo\tv1.0.0",
		"\x00\x01\x02",
		"mod \n mod\ta\tb\tc",
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		gotVer, gotErr := parseGoModVersion(data)
		againVer, againErr := parseGoModVersion(data)
		if gotVer != againVer || (gotErr == nil) != (againErr == nil) {
			t.Fatalf("parseGoModVersion(%q) not deterministic: (%q,%v) vs (%q,%v)", data, gotVer, gotErr, againVer, againErr)
		}
		if gotErr == nil && gotVer == "" {
			t.Fatalf("parseGoModVersion(%q) returned nil error with empty version", data)
		}
	})
}
