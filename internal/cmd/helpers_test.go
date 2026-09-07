package cmd_test

import (
	"context"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/jamesjohnsdev/bag/internal/cmd"
)

func TestIsKnown(t *testing.T) {
	parser, err := kong.New(&cmd.CLI{}, kong.BindTo(context.Background(), (*context.Context)(nil)))
	if err != nil {
		t.Fatalf("kong.New() error = %v", err)
	}

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{name: "registered command is known", command: "add", want: true},
		{name: "another registered command is known", command: "view", want: true},
		{name: "unregistered command is unknown", command: "hello", want: false},
		{name: "empty string is unknown", command: "", want: false},
		{name: "command name is case sensitive", command: "Add", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cmd.IsKnown(parser, tt.command); got != tt.want {
				t.Errorf("IsKnown(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}
