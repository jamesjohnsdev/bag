package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesjohnsdev/bag/internal/manifest"
)

const validTOML = `
[commands]
lint = "golangci-lint run"
build = "go build ./..."

[issues]
active = "v0.1.0"

[issues.versions."v0.1.0"]
source = "github.com/jamesjohnsdev/issues"

[something]
active = "v1.23.0"

[something.versions."v1.23.0"]

[somethingelse]
active = "v1.0.0"
type = "script"

[somethingelse.versions."v1.0.0"]
source = "https://gist.github.com/be057f2959753ee7c8ab57b3ee6a87ab.git"
`

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "bag.toml")
	if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, m *manifest.Manifest)
	}{
		{
			name:  "valid full manifest",
			input: validTOML,
			check: func(t *testing.T, m *manifest.Manifest) {
				if got := m.Commands["lint"]; got != "golangci-lint run" {
					t.Errorf("commands[lint] = %q, want %q", got, "golangci-lint run")
				}
				issues, ok := m.Binaries["issues"]
				if !ok {
					t.Fatal("missing binary: issues")
				}
				if issues.Active != "v0.1.0" {
					t.Errorf("issues.active = %q", issues.Active)
				}
				issuesVer, ok := issues.Versions[issues.Active]
				if !ok {
					t.Fatal("missing active version entry for issues")
				}
				if issuesVer.Source != "github.com/jamesjohnsdev/issues" {
					t.Errorf("issues.source = %q", issuesVer.Source)
				}
				if m.Binaries["somethingelse"].Type != "script" {
					t.Errorf("somethingelse.type = %q, want script", m.Binaries["somethingelse"].Type)
				}
				if m.Binaries["something"].Type != "binary" {
					t.Errorf("something.type = %q, want binary (default)", m.Binaries["something"].Type)
				}
			},
		},
		{
			name: "commands only",
			input: `
[commands]
build = "go build ./..."
`,
			check: func(t *testing.T, m *manifest.Manifest) {
				if len(m.Binaries) != 0 {
					t.Errorf("expected no binaries, got %d", len(m.Binaries))
				}
				if m.Commands["build"] != "go build ./..." {
					t.Errorf("commands[build] = %q", m.Commands["build"])
				}
			},
		},
		{
			name:  "empty file",
			input: "",
			check: func(t *testing.T, m *manifest.Manifest) {
				if len(m.Binaries) != 0 {
					t.Errorf("expected no binaries, got %d", len(m.Binaries))
				}
				if len(m.Commands) != 0 {
					t.Errorf("expected no commands, got %d", len(m.Commands))
				}
			},
		},
		{
			name:    "malformed toml",
			input:   "[commands\nlint = \"golangci-lint run\"\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTemp(t, tt.input)
			m, err := manifest.Parse(path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.check != nil && m != nil {
				tt.check(t, m)
			}
		})
	}
}

func TestParseNonexistent(t *testing.T) {
	_, err := manifest.Parse(filepath.Join(t.TempDir(), "nonexistent.toml"))
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestWriteRoundTrip(t *testing.T) {
	original, err := manifest.Parse(writeTemp(t, validTOML))
	if err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "bag.toml")
	if err := manifest.Write(out, original); err != nil {
		t.Fatal(err)
	}

	got, err := manifest.Parse(out)
	if err != nil {
		t.Fatal(err)
	}

	for k, want := range original.Commands {
		if got.Commands[k] != want {
			t.Errorf("commands[%s] = %q, want %q", k, got.Commands[k], want)
		}
	}
	for k, want := range original.Binaries {
		entry, ok := got.Binaries[k]
		if !ok {
			t.Errorf("missing binary after round trip: %s", k)
			continue
		}
		if entry.Active != want.Active {
			t.Errorf("binary[%s].active = %q, want %q", k, entry.Active, want.Active)
		}
		if entry.Type != want.Type {
			t.Errorf("binary[%s].type = %q, want %q", k, entry.Type, want.Type)
		}
		for v, wantVer := range want.Versions {
			gotVer, ok := entry.Versions[v]
			if !ok {
				t.Errorf("binary[%s]: missing version after round trip: %s", k, v)
				continue
			}
			if gotVer.Source != wantVer.Source {
				t.Errorf("binary[%s][%s].source = %q, want %q", k, v, gotVer.Source, wantVer.Source)
			}
		}
	}
}

func TestRemoveBinary(t *testing.T) {
	path := writeTemp(t, validTOML)

	if err := manifest.RemoveBinary(path, "issues"); err != nil {
		t.Fatalf("RemoveBinary() error = %v", err)
	}

	m, err := manifest.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Binaries["issues"]; ok {
		t.Error("expected issues binary to be removed")
	}
	if _, ok := m.Binaries["something"]; !ok {
		t.Error("expected unrelated binary 'something' to remain")
	}
	if m.Commands["lint"] != "golangci-lint run" {
		t.Error("expected commands to be preserved")
	}
}

func TestRemoveBinaryNonexistent(t *testing.T) {
	path := writeTemp(t, validTOML)

	if err := manifest.RemoveBinary(path, "does-not-exist"); err != nil {
		t.Fatalf("RemoveBinary() on missing entry should be a no-op, got error = %v", err)
	}

	m, err := manifest.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Binaries) != 3 {
		t.Errorf("expected manifest untouched, got %d binaries", len(m.Binaries))
	}
}

func FuzzParse(f *testing.F) {
	f.Add([]byte(validTOML))
	f.Add([]byte(""))
	f.Add([]byte("[commands]\n"))
	f.Add([]byte("[foo]\nsource = \"bar\"\n"))
	f.Add([]byte("[foo]\ntype = \"script\"\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		tmp := filepath.Join(t.TempDir(), "fuzz.toml")
		if err := os.WriteFile(tmp, data, 0o600); err != nil {
			t.Skip()
		}
		m, err := manifest.Parse(tmp)
		if err != nil {
			return
		}
		_ = manifest.Write(filepath.Join(t.TempDir(), "out.toml"), m)
	})
}
