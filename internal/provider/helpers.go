package provider

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
)

// isSafeBinaryName reports whether name is safe to use as the filename of an
// installed binary. It is applied to the base name of an archive entry, so
// entries nested in subdirectories (e.g. a release tarball that wraps its
// contents in a "tool-1.0.0-linux-amd64/" directory) can still be matched.
func isSafeBinaryName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	return !strings.ContainsAny(name, "/\\")
}

func extractZip(rc io.ReadCloser, binName, version string) (res Resolution, err error) {
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
		if file.FileInfo().IsDir() || file.Mode()&0o111 == 0 {
			continue
		}
		base := path.Base(file.Name)
		if !isSafeBinaryName(base) {
			continue
		}
		if binName != "" {
			base = binName
		}
		rc, err := file.Open()
		if err != nil {
			return Resolution{}, fmt.Errorf("opening file %s: %w", file.Name, err)
		}
		cleanup = false
		return Resolution{
			Reader:          &tempFileCloser{rc, tmp},
			ResolvedVersion: version,
			BinaryName:      base,
		}, nil
	}
	return Resolution{}, errors.New("no executable found in archive")
}

func extractTarball(rc io.ReadCloser, binName, version string) (res Resolution, err error) {
	cleanup := true
	defer func() {
		if cleanup {
			if cerr := rc.Close(); cerr != nil {
				err = errors.Join(err, fmt.Errorf("closing archive reader: %w", cerr))
			}
		}
	}()
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
		if hdr.Typeflag != tar.TypeReg || hdr.FileInfo().Mode()&0o111 == 0 {
			continue
		}
		base := path.Base(hdr.Name)
		if !isSafeBinaryName(base) {
			continue
		}
		if binName != "" {
			base = binName
		}
		cleanup = false
		return Resolution{
			Reader:          &tarCloser{rc, tr},
			ResolvedVersion: version,
			BinaryName:      base,
		}, nil
	}
	return Resolution{}, errors.New("no executable found in archive")
}

func useRawBinary(rc io.ReadCloser, name string, version string) (Resolution, error) {
	if !isSafeBinaryName(name) {
		return Resolution{}, fmt.Errorf("unsafe binary name: %s", name)
	}
	return Resolution{
		Reader:          rc,
		ResolvedVersion: version,
		BinaryName:      name,
	}, nil
}

// matchesSystem reports whether base (a release asset name with its extension
// stripped) encodes the current OS/arch using sep as its segment separator.
// OS and arch are taken as the trailing two segments, so tool names/versions
// may themselves contain the separator character. Some arch aliases (e.g.
// "x86_64") contain an underscore of their own, which collides with "_" as
// the segment separator, so arch is also tried as the last two segments
// joined back together (e.g. "x86" + "_" + "64").
func matchesSystem(base, sep string) bool {
	segments := strings.Split(base, sep)
	if len(segments) < 4 {
		return false
	}
	OS, arch := segments[len(segments)-2], segments[len(segments)-1]
	if runtime.GOOS == strings.ToLower(OS) && runtime.GOARCH == normaliseArch(arch) {
		return true
	}
	if len(segments) < 5 {
		return false
	}
	OS = segments[len(segments)-3]
	arch = segments[len(segments)-2] + sep + segments[len(segments)-1]
	return runtime.GOOS == strings.ToLower(OS) && runtime.GOARCH == normaliseArch(arch)
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

// nameMatchesBinary parses a given string and returns true if recognised as a binary
// This does not mean the file is actually a binary, but just the naming matches
// Extensions should be checked prior to this running, as it's possible it will mistake more complex extension types
func nameMatchesBinary(name string) bool {
	if ext := path.Ext(name); ext == "" || strings.ContainsAny(ext, "_-") {
		return true
	}
	return false
}

// handleAssetVariations parses extension type and runs the relevant helper to generate a Resolution
func handleAssetVariations(rc io.ReadCloser, binName, version, extension string) (Resolution, error) {
	switch extension {
	case "":
		res, err := useRawBinary(rc, binName, version)
		if err != nil {
			return Resolution{}, fmt.Errorf("using raw binary %s: %w", binName, err)
		}
		return res, nil
	case "zip":
		res, err := extractZip(rc, binName, version)
		if err != nil {
			return Resolution{}, fmt.Errorf("extracting zip %s: %w", binName, err)
		}
		return res, nil
	case "tar.gz":
		res, err := extractTarball(rc, binName, version)
		if err != nil {
			return Resolution{}, fmt.Errorf("extracting tarball %s: %w", binName, err)
		}
		return res, nil
	default:
		return Resolution{}, fmt.Errorf("release asset type not supported: %s", extension)
	}
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

func parseAssetName(name string) (ext, base string, err error) {
	switch {
	case strings.HasSuffix(name, ".tar.gz"):
		ext, base = "tar.gz", name[:len(name)-7]
	case strings.HasSuffix(name, ".zip"):
		ext, base = "zip", name[:len(name)-4]
	case nameMatchesBinary(name): // keep at bottom - could mistake unusual extensions
		ext, base = "", name
	default:
		return "", "", fmt.Errorf("file %s doesn't match known type", name)
	}
	return ext, base, nil
}
