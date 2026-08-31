package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	mangokong "github.com/alecthomas/mango-kong"
	"github.com/muesli/roff"
)

type ManInstallCmd struct{}

func (c *ManInstallCmd) Run(ctx *kong.Context) error {
	dest, err := installManPage(ctx.Model)
	if err != nil {
		return err
	}
	// this repeats the previous step but not worth worrying about refactor
	EnsureManPage(ctx.Model)

	fmt.Printf("man page installed to %s\n", dest)
	fmt.Println("run `mandb` (Linux) or `makewhatis` (macOS) to refresh index, then `man bag`")
	return nil
}

func installManPage(model *kong.Application) (dest string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return RenderManPage(model, filepath.Join(home, ".local", "share", "man", "man1"))
}

// RenderManPage renders the man page for model into dir/bag.1, creating dir
// if needed. Exported for use by the release build's man page generator,
// which has no user home directory to install into.
func RenderManPage(model *kong.Application, dir string) (dest string, err error) {
	man := mangokong.NewManPage(1, model)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("resolving man directory: %w", err)
	}

	dest = filepath.Join(dir, "bag.1")
	f, err := os.Create(dest)
	if err != nil {
		return "", fmt.Errorf("generating man file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			cerr = fmt.Errorf("closing man file: %w", cerr)
			err = errors.Join(err, cerr)
		}
	}()

	// Disable automatic hyphenation via the `HY` register: the man(7)
	// macros check it at every section/paragraph, so a bare `.nh` gets
	// overridden. Without this, a hyphenated bold/underlined word can
	// split its overstrike sequence across a line wrap, which crashes
	// some pagers (e.g. neovim's :Man highlighter).
	body := fmt.Sprint(man.Build(roff.NewDocument()))
	firstLine, rest, _ := strings.Cut(body, "\n")
	if _, err := fmt.Fprintln(f, firstLine); err != nil {
		return dest, fmt.Errorf("writing man file: %w", err)
	}
	if _, err := fmt.Fprintln(f, ".nr HY 0"); err != nil {
		return dest, fmt.Errorf("writing man file: %w", err)
	}
	if _, err := fmt.Fprint(f, rest); err != nil {
		return dest, fmt.Errorf("writing man file: %w", err)
	}

	return dest, err
}

// EnsureManPage installs the man page once, on first run, tracked via a
// marker file. It never fails the CLI: a broken install is silently
// skipped, not retried on every invocation.
func EnsureManPage(model *kong.Application) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	marker := filepath.Join(home, ".local", "state", "bag", "man-installed")
	if _, err := os.Stat(marker); err == nil {
		return
	}

	_, _ = installManPage(model)

	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(marker, nil, 0o644)
}
