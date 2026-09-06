package main

import (
	"context"
	"fmt"
	"log"
	"runtime"

	"github.com/alecthomas/kong"
	"github.com/fatih/color"

	"github.com/jamesjohnsdev/bag/internal/cmd"
	"github.com/jamesjohnsdev/bag/internal/config"
)

// version is set via -ldflags at build time (see .goreleaser.yaml); "dev" for local builds.
var (
	version   = "dev"
	commitSHA = "none"
	buildTime = "unknown"
)

func main() {
	if err := config.Load(); err != nil {
		log.Fatalf("loading config: %s", err.Error())
	}
	versionString := fmt.Sprintf(
		"bag %s\ncommit: %s\nbuilt:  %s",
		color.BlueString(version), color.BlueString(commitSHA), color.BlueString(buildTime),
	)

	ctx := kong.Parse(&cmd.CLI{}, kong.Name("bag"), kong.BindTo(context.Background(), (*context.Context)(nil)), cmd.Description, kong.Vars{
		"version": versionString,
	})

	if runtime.GOOS != "windows" && ctx.Command() != "man-install" {
		cmd.EnsureManPage(ctx.Model)
	}

	ctx.FatalIfErrorf(ctx.Run())
}
