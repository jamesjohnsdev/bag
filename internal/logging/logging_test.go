package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotatingFileRotatesAndPrunesBackups(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bag.log")
	writer, err := newRotatingFile(path, 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := writer.Close(); err != nil {
			t.Error(err)
		}
	})

	for range 3 {
		if _, err := writer.Write([]byte("abc")); err != nil {
			t.Fatal(err)
		}
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "abc" {
		t.Errorf("active log = %q, want %q", contents, "abc")
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	backups := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "bag.log.") {
			backups++
		}
	}
	if backups != 1 {
		t.Errorf("backup count = %d, want 1", backups)
	}
}
