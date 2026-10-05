package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManPageMarkerMatchesCurrentBuildOnly(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "state", "man-installed")
	const buildID = "/path/to/bag:123:456"

	if manPageMarkerMatches(marker, buildID) {
		t.Fatal("missing marker matched")
	}

	writeManPageMarkerFor(marker, buildID)
	if !manPageMarkerMatches(marker, buildID) {
		t.Fatal("current marker did not match")
	}
	if manPageMarkerMatches(marker, "/path/to/bag:124:456") {
		t.Fatal("different build matched")
	}

	contents, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("reading marker: %v", err)
	}
	if got, want := string(contents), buildID+"\n"; got != want {
		t.Errorf("marker contents = %q, want %q", got, want)
	}
}
