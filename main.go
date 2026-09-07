package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

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

	parser, err := kong.New(&cmd.CLI{}, kong.Name("bag"), kong.BindTo(context.Background(), (*context.Context)(nil)), cmd.Description, kong.Vars{
		"version": versionString,
	})
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") && !cmd.IsKnown(parser, os.Args[1]) {
		handled, err := cmd.RunCustom(os.Args[1], os.Args[2:])
		if handled {
			if err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					os.Exit(exitErr.ExitCode())
				}
				log.Fatalf("running command : %s", err.Error())
			}
			return
		}
		// silently pass through to kong handling if cust. doesn't work
	}

	ctx, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)

	if runtime.GOOS != "windows" && ctx.Command() != "man-install" {
		cmd.EnsureManPage(ctx.Model)
	}

	ctx.FatalIfErrorf(ctx.Run())
}
