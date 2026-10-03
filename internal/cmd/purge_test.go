package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/store"
)

func TestMergeActiveManifestVersions(t *testing.T) {
	tests := []struct {
		name string
		new  manifest.Manifest
		full manifest.Manifest
		want manifest.Manifest
	}{
		{
			name: "adds only active version for a new binary",
			new: manifest.Manifest{Binaries: map[string]manifest.BinaryEntry{
				"foo": testBinaryEntry("2.0.0", "1.0.0", "2.0.0"),
			}},
			full: manifest.Manifest{Binaries: map[string]manifest.BinaryEntry{}},
			want: manifest.Manifest{Binaries: map[string]manifest.BinaryEntry{
				"foo": testBinaryEntry("2.0.0", "2.0.0"),
			}},
		},
		{
			name: "retains active versions from both manifests",
			new: manifest.Manifest{Binaries: map[string]manifest.BinaryEntry{
				"foo": testBinaryEntry("2.0.0", "2.0.0"),
			}},
			full: manifest.Manifest{Binaries: map[string]manifest.BinaryEntry{
				"foo": testBinaryEntry("1.0.0", "1.0.0", "0.9.0"),
			}},
			want: manifest.Manifest{Binaries: map[string]manifest.BinaryEntry{
				"foo": testBinaryEntry("1.0.0", "1.0.0", "2.0.0"),
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeActiveManifestVersions(tt.new, tt.full)
			if err != nil {
				t.Fatalf("mergeActiveManifestVersions() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeActiveManifestVersions() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestMergeActiveManifestVersionsInvalidActiveVersion(t *testing.T) {
	_, err := mergeActiveManifestVersions(
		manifest.Manifest{Binaries: map[string]manifest.BinaryEntry{
			"foo": {Active: "missing", Versions: map[string]manifest.VersionEntry{}},
		}},
		manifest.Manifest{Binaries: map[string]manifest.BinaryEntry{}},
	)
	if err == nil {
		t.Fatal("expected error for missing active version")
	}
}

func TestCollectActiveManifests(t *testing.T) {
	home := t.TempDir()
	writeTestManifest(t, filepath.Join(home, "one", "bag.toml"), manifest.Manifest{Binaries: map[string]manifest.BinaryEntry{
		"foo": testBinaryEntry("1.0.0", "1.0.0"),
	}})
	writeTestManifest(t, filepath.Join(home, "two", "bag.toml"), manifest.Manifest{Binaries: map[string]manifest.BinaryEntry{
		"foo": testBinaryEntry("2.0.0", "2.0.0"),
		"bar": testBinaryEntry("3.0.0", "3.0.0"),
	}})

	got, skipped, err := collectActiveManifests(home)
	if err != nil {
		t.Fatalf("collectActiveManifests() error = %v", err)
	}
	if len(skipped) != 0 {
		t.Errorf("skipped = %v, want none", skipped)
	}
	want := manifest.Manifest{Binaries: map[string]manifest.BinaryEntry{
		"foo": testBinaryEntry("1.0.0", "1.0.0", "2.0.0"),
		"bar": testBinaryEntry("3.0.0", "3.0.0"),
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("collectActiveManifests() = %#v, want %#v", got, want)
	}
}

func TestPurgeStore(t *testing.T) {
	tests := []struct {
		name          string
		installed     map[string][]string
		fullManifest  manifest.Manifest
		wantDeleted   map[string][]string
		wantRemaining map[string][]string
	}{
		{
			name: "removes stale and unreferenced versions",
			installed: map[string][]string{
				"foo":    {"1.0.0", "2.0.0"},
				"orphan": {"1.0.0"},
			},
			fullManifest: manifest.Manifest{Binaries: map[string]manifest.BinaryEntry{
				"foo": testBinaryEntry("1.0.0", "1.0.0"),
			}},
			wantDeleted: map[string][]string{
				"foo":    {"2.0.0"},
				"orphan": {"1.0.0"},
			},
			wantRemaining: map[string][]string{"foo": {"1.0.0"}},
		},
		{
			name:          "keeps all referenced versions",
			installed:     map[string][]string{"foo": {"1.0.0", "2.0.0"}},
			fullManifest:  manifest.Manifest{Binaries: map[string]manifest.BinaryEntry{"foo": testBinaryEntry("2.0.0", "1.0.0", "2.0.0")}},
			wantDeleted:   map[string][]string{},
			wantRemaining: map[string][]string{"foo": {"1.0.0", "2.0.0"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupPurgeStore(t)
			for name, versions := range tt.installed {
				for _, version := range versions {
					installPurgeTestBinary(t, name, version)
				}
			}

			got, err := purgeStore(tt.fullManifest)
			if err != nil {
				t.Fatalf("purgeStore() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.wantDeleted) {
				t.Errorf("deleted = %#v, want %#v", got, tt.wantDeleted)
			}
			for name, versions := range tt.wantRemaining {
				for _, version := range versions {
					if !store.BinaryExists(name, version) {
						t.Errorf("expected %s %s to remain", name, version)
					}
				}
			}
			for name, versions := range tt.wantDeleted {
				for _, version := range versions {
					if store.BinaryExists(name, version) {
						t.Errorf("expected %s %s to be removed", name, version)
					}
				}
			}
		})
	}
}

func FuzzMergeActiveManifestVersions(f *testing.F) {
	f.Add("1.0.0", "2.0.0")
	f.Add("v1", "v1")

	f.Fuzz(func(t *testing.T, firstVersion, secondVersion string) {
		if firstVersion == "" || secondVersion == "" {
			t.Skip()
		}
		full := manifest.Manifest{Binaries: map[string]manifest.BinaryEntry{
			"foo": testBinaryEntry(firstVersion, firstVersion),
		}}
		newManifest := manifest.Manifest{Binaries: map[string]manifest.BinaryEntry{
			"foo": testBinaryEntry(secondVersion, secondVersion),
		}}

		got, err := mergeActiveManifestVersions(newManifest, full)
		if err != nil {
			t.Fatalf("mergeActiveManifestVersions() error = %v", err)
		}
		versions := got.Binaries["foo"].Versions
		if _, ok := versions[firstVersion]; !ok {
			t.Errorf("missing first active version %q", firstVersion)
		}
		if gotSource := versions[secondVersion].Source; gotSource != secondVersion {
			t.Errorf("second active version source = %q, want %q", gotSource, secondVersion)
		}
	})
}

func testBinaryEntry(active string, versions ...string) manifest.BinaryEntry {
	entries := make(map[string]manifest.VersionEntry, len(versions))
	for _, version := range versions {
		entries[version] = manifest.VersionEntry{Source: version}
	}
	return manifest.BinaryEntry{Type: manifest.BinaryType, Active: active, Versions: entries}
}

func writeTestManifest(t *testing.T, path string, man manifest.Manifest) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := manifest.Write(path, &man); err != nil {
		t.Fatal(err)
	}
}

func setupPurgeStore(t *testing.T) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)
	t.Cleanup(func() {
		_ = filepath.Walk(dataDir, func(path string, _ os.FileInfo, err error) error {
			if err == nil {
				_ = os.Chmod(path, 0o755)
			}
			return nil
		})
	})
}

func installPurgeTestBinary(t *testing.T, name, version string) {
	t.Helper()
	source := filepath.Join(t.TempDir(), "source")
	if err := os.WriteFile(source, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := store.InstallLocal(name, version, source); err != nil {
		t.Fatalf("installing %s %s: %v", name, version, err)
	}
}
