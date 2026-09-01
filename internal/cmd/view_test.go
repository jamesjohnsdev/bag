package cmd_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/jamesjohnsdev/bag/internal/cmd"
	"github.com/jamesjohnsdev/bag/internal/manifest"
)

// captureStdout redirects os.Stdout for the duration of fn and returns
// everything written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("reading pipe: %v", err)
	}
	return buf.String()
}

func TestViewCmdRun(t *testing.T) {
	setupHome(t)

	manPath, _, err := manifest.Get(false)
	if err != nil {
		t.Fatal(err)
	}
	if err := manifest.AddBinary(manPath, "foo", manifest.BinaryEntry{
		Source:  "/tmp/src-bin",
		Version: "1.0.0",
		Type:    "binary",
	}); err != nil {
		t.Fatal(err)
	}

	viewCmd := &cmd.ViewCmd{Name: "foo"}

	var runErr error
	out := captureStdout(t, func() {
		runErr = viewCmd.Run(context.Background())
	})
	if runErr != nil {
		t.Fatalf("Run() error = %v", runErr)
	}

	for _, want := range []string{"/tmp/src-bin", "binary", "1.0.0"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got: %q", want, out)
		}
	}
}

func TestViewCmdRunNotInManifest(t *testing.T) {
	setupHome(t)

	viewCmd := &cmd.ViewCmd{Name: "does-not-exist"}
	if err := viewCmd.Run(context.Background()); err == nil {
		t.Fatal("expected error viewing binary absent from manifest")
	}
}
