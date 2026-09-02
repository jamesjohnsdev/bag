package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesjohnsdev/bag/internal/manifest"
)

const validLock = `
[issues."v0.1.0"]
hash = "sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abc1"

[something."v1.23.0"]
hash = "sha256:789012ghi345789012ghi345789012ghi345789012ghi345789012ghi345789"
`

func writeLockTemp(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), ".bag-lock")
	if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestParseLock(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, lf *manifest.LockFile)
	}{
		{
			name:  "valid lock file",
			input: validLock,
			check: func(t *testing.T, lf *manifest.LockFile) {
				versions, ok := lf.Entries["issues"]
				if !ok {
					t.Fatal("missing entry: issues")
				}
				entry, ok := versions["v0.1.0"]
				if !ok {
					t.Fatal("missing version: issues v0.1.0")
				}
				if entry.Hash == "" {
					t.Error("issues.hash is empty")
				}
				if _, ok := lf.Entries["something"]; !ok {
					t.Error("missing entry: something")
				}
			},
		},
		{
			name:  "empty file",
			input: "",
			check: func(t *testing.T, lf *manifest.LockFile) {
				if len(lf.Entries) != 0 {
					t.Errorf("expected empty entries, got %d", len(lf.Entries))
				}
			},
		},
		{
			name:    "malformed lock",
			input:   "[issues\nversion = \"v0.1.0\"\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeLockTemp(t, tt.input)
			lf, err := manifest.ParseLock(path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseLock() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.check != nil && lf != nil {
				tt.check(t, lf)
			}
		})
	}
}

func TestParseLockNonexistent(t *testing.T) {
	lf, err := manifest.ParseLock(filepath.Join(t.TempDir(), "nonexistent.lock"))
	if err != nil {
		t.Fatalf("expected nil error for nonexistent lock, got %v", err)
	}
	if len(lf.Entries) != 0 {
		t.Errorf("expected empty entries, got %d", len(lf.Entries))
	}
}

func TestWriteLockRoundTrip(t *testing.T) {
	original, err := manifest.ParseLock(writeLockTemp(t, validLock))
	if err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), ".bag-lock")
	if err := manifest.WriteLock(out, original); err != nil {
		t.Fatal(err)
	}

	got, err := manifest.ParseLock(out)
	if err != nil {
		t.Fatal(err)
	}

	for k, wantVersions := range original.Entries {
		gotVersions, ok := got.Entries[k]
		if !ok {
			t.Errorf("missing entry after round trip: %s", k)
			continue
		}
		for v, want := range wantVersions {
			entry, ok := gotVersions[v]
			if !ok {
				t.Errorf("entry[%s]: missing version after round trip: %s", k, v)
				continue
			}
			if entry.Hash != want.Hash {
				t.Errorf("entry[%s][%s].hash = %q, want %q", k, v, entry.Hash, want.Hash)
			}
		}
	}
}

func TestRemoveLockEntry(t *testing.T) {
	path := writeLockTemp(t, validLock)

	if err := manifest.RemoveLockEntry(path, "issues"); err != nil {
		t.Fatalf("RemoveLockEntry() error = %v", err)
	}

	lf, err := manifest.ParseLock(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := lf.Entries["issues"]; ok {
		t.Error("expected issues entry to be removed")
	}
	if _, ok := lf.Entries["something"]; !ok {
		t.Error("expected unrelated entry 'something' to remain")
	}
}

func TestRemoveLockEntryNonexistent(t *testing.T) {
	path := writeLockTemp(t, validLock)

	if err := manifest.RemoveLockEntry(path, "does-not-exist"); err != nil {
		t.Fatalf("RemoveLockEntry() on missing entry should be a no-op, got error = %v", err)
	}

	lf, err := manifest.ParseLock(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(lf.Entries) != 2 {
		t.Errorf("expected lockfile untouched, got %d entries", len(lf.Entries))
	}
}

func TestRemoveLockEntryMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.lock")

	if err := manifest.RemoveLockEntry(path, "issues"); err != nil {
		t.Fatalf("expected no error removing from nonexistent lockfile, got %v", err)
	}
}

func FuzzParseLock(f *testing.F) {
	f.Add([]byte(validLock))
	f.Add([]byte(""))
	f.Add([]byte("[foo.\"v1.0.0\"]\nhash = \"sha256:abc\"\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		tmp := filepath.Join(t.TempDir(), "fuzz.lock")
		if err := os.WriteFile(tmp, data, 0o600); err != nil {
			t.Skip()
		}
		lf, err := manifest.ParseLock(tmp)
		if err != nil {
			return
		}
		_ = manifest.WriteLock(filepath.Join(t.TempDir(), "out.lock"), lf)
	})
}
