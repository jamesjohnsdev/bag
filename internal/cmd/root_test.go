package cmd_test

import (
	"context"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/jamesjohnsdev/bag/internal/cmd"
)

func TestCLIBuildsValidGrammar(t *testing.T) {
	if _, err := kong.New(&cmd.CLI{}, kong.BindTo(context.Background(), (*context.Context)(nil))); err != nil {
		t.Fatalf("kong.New() error = %v", err)
	}
}

func TestCLIRegistersExpectedCommands(t *testing.T) {
	k, err := kong.New(&cmd.CLI{}, kong.BindTo(context.Background(), (*context.Context)(nil)))
	if err != nil {
		t.Fatalf("kong.New() error = %v", err)
	}

	want := []string{"man-install", "init", "add", "remove", "view", "tool"}
	got := make(map[string]bool)
	for _, node := range k.Model.Children {
		got[node.Name] = true
	}

	for _, name := range want {
		if !got[name] {
			t.Errorf("expected command %q to be registered, got commands: %v", name, got)
		}
	}
}
