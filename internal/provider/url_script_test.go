package provider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"

	"github.com/fatih/color"
)

// trackedBody wraps a body so tests can assert whether it was closed, and how
// many times.
type trackedBody struct {
	io.ReadCloser
	closes int
}

func (b *trackedBody) Close() error {
	b.closes++
	return b.ReadCloser.Close()
}

// trackingTransport swaps every response body for a *trackedBody, stashing a
// pointer to it so the test can inspect close counts after Resolve returns.
type trackingTransport struct {
	base    http.RoundTripper
	tracked *trackedBody
}

func (t *trackingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	t.tracked = &trackedBody{ReadCloser: resp.Body}
	resp.Body = t.tracked
	return resp, nil
}

func newTrackedURLScriptProvider(t *testing.T, mux *http.ServeMux) (URLScriptProvider, *httptest.Server, *trackingTransport) {
	t.Helper()
	// WrapProgress short-circuits to a no-op wrapper only when color.NoColor is
	// set; force it so the close-tracking assertions below don't depend on
	// whether the test runs under a tty.
	orig := color.NoColor
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = orig })

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	transport := &trackingTransport{base: srv.Client().Transport}
	client := srv.Client()
	client.Transport = transport
	return NewURLScriptProvider(client), srv, transport
}

func TestURLScriptProviderDetect(t *testing.T) {
	p := URLScriptProvider{}
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{"https URL", "https://example.com/install.sh", true},
		{"http URL", "http://example.com/install.sh", true},
		{"scheme-less", "example.com/install.sh", false},
		{"github shorthand", "github.com/owner/repo", false},
		{"ftp scheme", "ftp://example.com/install.sh", false},
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

func TestURLScriptProviderResolve(t *testing.T) {
	ctx := context.Background()

	t.Run("version passed through unchanged", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/install.sh", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("#!/bin/sh\necho hi"))
		})
		p, srv, transport := newTrackedURLScriptProvider(t, mux)
		u, _ := url.Parse(srv.URL + "/install.sh")
		res, err := p.Resolve(ctx, *u, "myscript", "v1.2.3")
		if err != nil {
			t.Fatal(err)
		}
		if res.ResolvedVersion != "v1.2.3" {
			t.Errorf("ResolvedVersion = %q, want v1.2.3", res.ResolvedVersion)
		}
		if res.BinaryName != "myscript" {
			t.Errorf("BinaryName = %q, want myscript", res.BinaryName)
		}
		content, readErr := io.ReadAll(res.Reader)
		if err := errors.Join(readErr, res.Reader.Close()); err != nil {
			t.Error(err)
		}
		if string(content) != "#!/bin/sh\necho hi" {
			t.Errorf("content = %q", string(content))
		}
		if transport.tracked.closes != 1 {
			t.Errorf("body closed %d times, want 1", transport.tracked.closes)
		}
	})

	t.Run("empty version defaults to unknown", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/install.sh", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("body"))
		})
		p, srv, _ := newTrackedURLScriptProvider(t, mux)
		u, _ := url.Parse(srv.URL + "/install.sh")
		res, err := p.Resolve(ctx, *u, "myscript", "")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if cerr := res.Reader.Close(); cerr != nil {
				err = errors.Join(err, cerr)
			}
		}()
		if res.ResolvedVersion != "unknown" {
			t.Errorf("ResolvedVersion = %q, want unknown", res.ResolvedVersion)
		}
	})

	t.Run("scriptName defaults to url path basename", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/tools/install.sh", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("body"))
		})
		p, srv, _ := newTrackedURLScriptProvider(t, mux)
		u, _ := url.Parse(srv.URL + "/tools/install.sh")
		res, err := p.Resolve(ctx, *u, "", "v1")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if cerr := res.Reader.Close(); cerr != nil {
				err = errors.Join(err, cerr)
			}
		}()
		if res.BinaryName != "install.sh" {
			t.Errorf("BinaryName = %q, want install.sh", res.BinaryName)
		}
	})

	t.Run("unsafe binary name closes body and errors", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/install.sh", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("body"))
		})
		p, srv, transport := newTrackedURLScriptProvider(t, mux)
		u, _ := url.Parse(srv.URL + "/install.sh")
		_, err := p.Resolve(ctx, *u, "../escape", "v1")
		if err == nil {
			t.Fatal("expected error for unsafe binary name, got nil")
		}
		if transport.tracked == nil {
			t.Fatal("request never reached server")
		}
		if transport.tracked.closes != 1 {
			t.Errorf("body closed %d times on error path, want 1 (leak)", transport.tracked.closes)
		}
	})

	t.Run("non-2xx status returns error and closes body", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/install.sh", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		p, srv, transport := newTrackedURLScriptProvider(t, mux)
		u, _ := url.Parse(srv.URL + "/install.sh")
		_, err := p.Resolve(ctx, *u, "myscript", "v1")
		if err == nil {
			t.Fatal("expected error for 404, got nil")
		}
		if transport.tracked.closes != 1 {
			t.Errorf("body closed %d times, want 1", transport.tracked.closes)
		}
	})

	t.Run("connection error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
		srvURL := srv.URL
		srv.Close()
		p := NewURLScriptProvider(&http.Client{})
		u, _ := url.Parse(srvURL + "/install.sh")
		_, err := p.Resolve(ctx, *u, "myscript", "v1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

// FuzzURLScriptProviderResolve checks that Resolve never panics for arbitrary
// binary names, and that its result is consistent with isSafeBinaryName: a
// name it rejects must produce an error (with the response body still closed
// exactly once), and a name it accepts must round-trip as BinaryName.
func FuzzURLScriptProviderResolve(f *testing.F) {
	seeds := []string{
		"", ".", "..", "tool", "../etc/passwd", "a/b", `a\b`, "tool.sh",
		"-", " ", "tool ", "\x00", "😀script",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/install.sh", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("body"))
	})
	orig := color.NoColor
	color.NoColor = true
	f.Cleanup(func() { color.NoColor = orig })
	srv := httptest.NewServer(mux)
	f.Cleanup(srv.Close)

	f.Fuzz(func(t *testing.T, name string) {
		transport := &trackingTransport{base: http.DefaultTransport}
		client := &http.Client{Transport: transport}
		p := NewURLScriptProvider(client)
		u, err := url.Parse(srv.URL + "/install.sh")
		if err != nil {
			t.Fatal(err)
		}

		res, err := p.Resolve(context.Background(), *u, name, "v1")

		// Resolve defaults an empty scriptName to the URL's path basename
		// before the safety check ever sees it.
		effectiveName := name
		if effectiveName == "" {
			effectiveName = path.Base(u.Path)
		}
		wantSafe := isSafeBinaryName(effectiveName)

		if wantSafe && err != nil {
			t.Fatalf("Resolve(%q) errored unexpectedly: %v", name, err)
		}
		if !wantSafe && err == nil {
			t.Fatalf("Resolve(%q) succeeded, want error for unsafe name %q", name, effectiveName)
		}
		if err != nil {
			// error path: Resolve owns the body and must have closed it itself.
			if transport.tracked != nil && transport.tracked.closes != 1 {
				t.Errorf("Resolve(%q): body closed %d times on error path, want 1", name, transport.tracked.closes)
			}
			return
		}
		// success path: ownership passes to the caller via res.Reader.
		if res.BinaryName != effectiveName {
			t.Errorf("BinaryName = %q, want %q", res.BinaryName, effectiveName)
		}
		_ = res.Reader.Close()
		if transport.tracked.closes != 1 {
			t.Errorf("Resolve(%q): body closed %d times after caller closed reader, want 1", name, transport.tracked.closes)
		}
	})
}
