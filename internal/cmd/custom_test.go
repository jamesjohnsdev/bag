package cmd_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesjohnsdev/bag/internal/cmd"
)

// writeGlobalManifest points $HOME at a fresh temp dir and writes the given
// TOML content to the global manifest path RunCustom will resolve to.
func writeGlobalManifest(t *testing.T, content string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)

	manDir := filepath.Join(home, ".config", "bag")
	if err := os.MkdirAll(manDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manPath := filepath.Join(manDir, "bag.toml")
	if err := os.WriteFile(manPath, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

// chdirEmpty chdirs the test into a fresh temp dir with no bag.toml
// anywhere in its ancestry, so local manifest resolution reliably falls
// through to global regardless of the real repo's working tree state.
func chdirEmpty(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
}

// writeLocalManifest chdirs the test into a fresh temp dir and writes the
// given TOML content to bag.toml there, so RunCustom resolves it as the
// local manifest.
func writeLocalManifest(t *testing.T, content string) {
	t.Helper()
	dir := t.TempDir()
	manPath := filepath.Join(dir, "bag.toml")
	if err := os.WriteFile(manPath, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Chdir(dir)
}

func TestRunCustom(t *testing.T) {
	t.Run("unknown command is not handled", func(t *testing.T) {
		writeGlobalManifest(t, "[commands]\nhello = \"echo hi\"\n")
		chdirEmpty(t)

		handled, err := cmd.RunCustom("goodbye", nil)
		if handled {
			t.Error("handled = true, want false")
		}
		if err != nil {
			t.Errorf("err = %v, want nil", err)
		}
	})

	t.Run("known command is handled and runs", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "out.txt")
		writeGlobalManifest(t, "[commands]\nhello = \"echo hi > "+outFile+"\"\n")
		chdirEmpty(t)

		handled, err := cmd.RunCustom("hello", nil)
		if !handled {
			t.Fatal("handled = false, want true")
		}
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		got, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}
		if want := "hi\n"; string(got) != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})

	t.Run("args are forwarded as positional parameters", func(t *testing.T) {
		// RunCustom appends `"$@"` to custCmd itself, so the manifest entry
		// below must not reference $@ directly.
		outFile := filepath.Join(t.TempDir(), "out.txt")
		writeGlobalManifest(t, "[commands]\ngreet = \"echo > "+outFile+"\"\n")
		chdirEmpty(t)

		handled, err := cmd.RunCustom("greet", []string{"foo", "bar baz"})
		if !handled {
			t.Fatal("handled = false, want true")
		}
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		got, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}
		if want := "foo bar baz\n"; string(got) != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})

	t.Run("nonzero exit surfaces as ExitError", func(t *testing.T) {
		writeGlobalManifest(t, "[commands]\nfail = \"exit 7\"\n")
		chdirEmpty(t)

		handled, err := cmd.RunCustom("fail", nil)
		if !handled {
			t.Fatal("handled = false, want true")
		}
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("err = %v, want *exec.ExitError", err)
		}
		if exitErr.ExitCode() != 7 {
			t.Errorf("ExitCode() = %d, want 7", exitErr.ExitCode())
		}
	})

	t.Run("invalid manifest surfaces parse error", func(t *testing.T) {
		writeGlobalManifest(t, "not valid toml [[[")
		chdirEmpty(t)

		handled, err := cmd.RunCustom("hello", nil)
		if handled {
			t.Error("handled = true, want false")
		}
		if err == nil || !strings.Contains(err.Error(), "parsing manifest") {
			t.Errorf("err = %v, want error containing %q", err, "parsing manifest")
		}
	})

	t.Run("local command overrides global", func(t *testing.T) {
		globalOut := filepath.Join(t.TempDir(), "global-out.txt")
		writeGlobalManifest(t, "[commands]\nhello = \"echo global > "+globalOut+"\"\n")

		localOut := filepath.Join(t.TempDir(), "local-out.txt")
		writeLocalManifest(t, "[commands]\nhello = \"echo local > "+localOut+"\"\n")

		handled, err := cmd.RunCustom("hello", nil)
		if !handled {
			t.Fatal("handled = false, want true")
		}
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}

		got, err := os.ReadFile(localOut)
		if err != nil {
			t.Fatalf("ReadFile(local) error = %v", err)
		}
		if want := "local\n"; string(got) != want {
			t.Errorf("output = %q, want %q", got, want)
		}
		if _, err := os.Stat(globalOut); !os.IsNotExist(err) {
			t.Error("global command ran, want only local to run")
		}
	})

	t.Run("falls back to global when no local manifest defines the command", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "out.txt")
		writeGlobalManifest(t, "[commands]\nhello = \"echo global > "+outFile+"\"\n")
		chdirEmpty(t)

		handled, err := cmd.RunCustom("hello", nil)
		if !handled {
			t.Fatal("handled = false, want true")
		}
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		got, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}
		if want := "global\n"; string(got) != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})

	t.Run("not handled when neither local nor global define the command", func(t *testing.T) {
		writeGlobalManifest(t, "[commands]\nhello = \"echo hi\"\n")
		chdirEmpty(t)

		handled, err := cmd.RunCustom("missing", nil)
		if handled {
			t.Error("handled = true, want false")
		}
		if err != nil {
			t.Errorf("err = %v, want nil", err)
		}
	})
}

// FuzzRunCustomArgs asserts args survive the "$@" shell passthrough verbatim,
// regardless of shell metacharacters, so a custom command can never be
// injected with attacker-controlled argument content.
func FuzzRunCustomArgs(f *testing.F) {
	seeds := []string{
		"plain",
		"has spaces",
		`$(rm -rf /)`,
		"; echo pwned",
		"`whoami`",
		`"quoted"`,
		`'single quoted'`,
		"back\\slash",
		"new\nline",
		"tab\ttab",
		"unicode: 漢字 🎒",
		"",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, arg string) {
		if strings.ContainsRune(arg, 0) {
			t.Skip("NUL byte is not a valid os/exec argument")
		}

		// RunCustom appends `"$@"` to custCmd itself, so the manifest entry
		// below must not reference $@ directly.
		outFile := filepath.Join(t.TempDir(), "out")
		writeGlobalManifest(t, "[commands]\necho = \"printf '%s' > "+outFile+"\"\n")
		chdirEmpty(t)

		handled, err := cmd.RunCustom("echo", []string{arg})
		if !handled {
			t.Fatalf("handled = false, want true")
		}
		if err != nil {
			t.Fatalf("RunCustom() error = %v", err)
		}

		got, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}
		if string(got) != arg {
			t.Errorf("output = %q, want %q", got, arg)
		}
	})
}
