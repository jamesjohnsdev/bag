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
source = "github.com/jamesjohnsdev/issues"
version = "v0.1.0"

[something]
version = "v1.23.0"

[somethingelse]
source = "https://gist.github.com/be057f2959753ee7c8ab57b3ee6a87ab.git"
type = "script"
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
				if issues.Source != "github.com/jamesjohnsdev/issues" {
					t.Errorf("issues.source = %q", issues.Source)
				}
				if issues.Version != "v0.1.0" {
					t.Errorf("issues.version = %q", issues.Version)
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
		if entry.Source != want.Source {
			t.Errorf("binary[%s].source = %q, want %q", k, entry.Source, want.Source)
		}
		if entry.Version != want.Version {
			t.Errorf("binary[%s].version = %q, want %q", k, entry.Version, want.Version)
		}
		if entry.Type != want.Type {
			t.Errorf("binary[%s].type = %q, want %q", k, entry.Type, want.Type)
		}
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
