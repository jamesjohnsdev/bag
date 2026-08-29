package provider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func newTestURLProvider(t *testing.T, mux *http.ServeMux) (URLProvider, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return NewURLProvider(srv.Client()), srv
}

func TestURLProviderDetect(t *testing.T) {
	p := URLProvider{}
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{"https URL", "https://example.com/tool.tar.gz", true},
		{"http URL", "http://example.com/tool.tar.gz", true},
		{"scheme-less", "example.com/tool.tar.gz", false},
		{"github shorthand", "github.com/owner/repo", false},
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

func TestURLProviderResolve(t *testing.T) {
	ctx := context.Background()

	t.Run("tar.gz", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/tool.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
			rc := makeTarGz(t, []tarEntry{{name: "tool", mode: 0o755, content: "ELF"}})
			data, readErr := io.ReadAll(rc)
			if err := errors.Join(readErr, rc.Close()); err != nil {
				t.Error(err)
			}
			_, _ = w.Write(data)
		})
		p, srv := newTestURLProvider(t, mux)
		u, _ := url.Parse(srv.URL + "/tool.tar.gz")
		res, err := p.Resolve(ctx, *u, "tool", "v1.0.0")
		if err != nil {
			t.Fatal(err)
		}
		if res.BinaryName != "tool" {
			t.Errorf("BinaryName = %q, want tool", res.BinaryName)
		}
		if res.ResolvedVersion != "v1.0.0" {
			t.Errorf("ResolvedVersion = %q, want v1.0.0", res.ResolvedVersion)
		}
		content, readErr := io.ReadAll(res.Reader)
		if err := errors.Join(readErr, res.Reader.Close()); err != nil {
			t.Error(err)
		}
		if string(content) != "ELF" {
			t.Errorf("content = %q, want ELF", string(content))
		}
	})

	t.Run("zip", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/tool.zip", func(w http.ResponseWriter, _ *http.Request) {
			rc := makeZip(t, []zipEntry{{name: "tool", mode: 0o755, content: "ELF"}})
			data, readErr := io.ReadAll(rc)
			if err := errors.Join(readErr, rc.Close()); err != nil {
				t.Error(err)
			}
			_, _ = w.Write(data)
		})
		p, srv := newTestURLProvider(t, mux)
		u, _ := url.Parse(srv.URL + "/tool.zip")
		res, err := p.Resolve(ctx, *u, "tool", "v1.0.0")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := res.Reader.Close(); err != nil {
				t.Error(err)
			}
		}()
		if res.BinaryName != "tool" {
			t.Errorf("BinaryName = %q, want tool", res.BinaryName)
		}
		if res.ResolvedVersion != "v1.0.0" {
			t.Errorf("ResolvedVersion = %q, want v1.0.0", res.ResolvedVersion)
		}
	})

	t.Run("raw binary", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/tool", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("ELF"))
		})
		p, srv := newTestURLProvider(t, mux)
		u, _ := url.Parse(srv.URL + "/tool")
		res, err := p.Resolve(ctx, *u, "tool", "v1.0.0")
		if err != nil {
			t.Fatal(err)
		}
		content, readErr := io.ReadAll(res.Reader)
		if err := errors.Join(readErr, res.Reader.Close()); err != nil {
			t.Error(err)
		}
		if res.BinaryName != "tool" {
			t.Errorf("BinaryName = %q, want tool", res.BinaryName)
		}
		if string(content) != "ELF" {
			t.Errorf("content = %q, want ELF", string(content))
		}
	})

	t.Run("connection error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
		srvURL := srv.URL
		srv.Close()
		p := NewURLProvider(&http.Client{})
		u, _ := url.Parse(srvURL + "/tool.tar.gz")
		_, err := p.Resolve(ctx, *u, "tool", "v1.0.0")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("unsupported extension", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/tool.deb", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("deb content"))
		})
		p, srv := newTestURLProvider(t, mux)
		u, _ := url.Parse(srv.URL + "/tool.deb")
		_, err := p.Resolve(ctx, *u, "tool", "v1.0.0")
		if err == nil {
			t.Fatal("expected error for unsupported extension, got nil")
		}
	})
}
