package provider

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GoProvider struct{}

var _ Provider = (*GoProvider)(nil)

func NewGoProvider() GoProvider {
	return GoProvider{}
}

func (provider GoProvider) Detect(src url.URL) bool {
	return strings.HasPrefix(src.String(), "go:")
}

func (provider GoProvider) Resolve(ctx context.Context, src url.URL, binName string, version string) (Resolution, error) {
	if _, err := exec.LookPath("go"); err != nil {
		return Resolution{}, fmt.Errorf("go not installed on path: %w", err)
	}

	// extract owner/repo from source
	rawPath := src.Opaque
	if rawPath == "" {
		rawPath = src.Path
	}
	srcPath := strings.TrimPrefix(rawPath, "go:")
	parts := strings.SplitN(srcPath, "@", 2)
	modPath := parts[0]
	if len(parts) == 2 {
		version = parts[1] // URL version tag will override param
	}
	if version == "" {
		version = "latest"
	}

	if binName == "" {
		binName = filepath.Base(modPath)
	}

	tmpDir, err := os.MkdirTemp("", "bag-go-install-*")
	if err != nil {
		return Resolution{}, fmt.Errorf("creating temp dir: %w", err)
	}
	cleanup := func() error { return os.RemoveAll(tmpDir) } // INFO: could I just defer

	cmd := exec.CommandContext(ctx, "go", "install", fmt.Sprintf("%s@%s", parts[0], version))
	cmd.Env = append(os.Environ(), "GOBIN="+tmpDir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return Resolution{}, fmt.Errorf("go install failed: %w: %s", err, stderr.String())
	}

	builtName := filepath.Base(modPath) // go install always names after module dir

	if version == "latest" {
		out, err := exec.CommandContext(ctx, "go", "version", "-m", filepath.Join(tmpDir, builtName)).Output()
		if err != nil {
			return Resolution{}, fmt.Errorf("getting version: %w", err)
		}
		modVers, err := parseGoModVersion(out)
		if err != nil {
			return Resolution{}, fmt.Errorf("parsing go mod version", err)
		}
		version = modVers
	}

	info, err := os.Stat(filepath.Join(tmpDir, builtName))
	if err != nil {
		return Resolution{}, fmt.Errorf("stat binary: %w", err)
	}

	return Resolution{
		Dir:             tmpDir,
		BinaryRelPath:   builtName,
		Cleanup:         cleanup,
		BinaryName:      binName,
		ResolvedVersion: version,
		Size:            info.Size(),
	}, nil
}

func parseGoModVersion(out []byte) (string, error) {
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "mod" {
			return fields[2], nil
		}
	}
	return "", fmt.Errorf("mod line not found in go version -m output")
}
