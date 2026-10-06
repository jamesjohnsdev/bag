package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/fatih/color"

	"github.com/jamesjohnsdev/bag/internal/cmd"
	"github.com/jamesjohnsdev/bag/internal/config"
	"github.com/jamesjohnsdev/bag/internal/logging"
)

// version is set via -ldflags at build time (see .goreleaser.yaml); "dev" for local builds.
var (
	version   = "dev"
	commitSHA = "none"
	buildTime = "unknown"
)

func main() {
	closeLog, err := logging.Setup()
	if err != nil {
		log.Fatalf("setting up logging: %s", err.Error())
	}
	defer func() {
		_ = closeLog()
	}()
	slog.Info("bag started")

	if base := filepath.Base(os.Args[0]); base != "bag" && base != "bag.exe" {
		if err := cmd.RunExec(base, os.Args[1:]); err != nil {
			slog.Error("running managed executable", "binary", base, "error", err)
			log.Fatalf("%s", err.Error())
		}
		return
	}

	if err := config.Load(); err != nil {
		slog.Error("loading config", "error", err)
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
		slog.Error("building command parser", "error", err)
		log.Fatal(err)
	}

	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") && !cmd.IsKnown(parser, os.Args[1]) {
		handled, err := cmd.RunCustom(os.Args[1], os.Args[2:])
		if handled {
			if err != nil {
				slog.Error("running custom command", "command", os.Args[1], "error", err)
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
	if err != nil {
		slog.Error("parsing command", "error", err)
	}
	parser.FatalIfErrorf(err)

	if runtime.GOOS != "windows" && ctx.Command() != "man-install" {
		cmd.EnsureManPage(ctx.Model)
	}

	if err := ctx.Run(); err != nil {
		slog.Error("command failed", "command", ctx.Command(), "error", err)
		ctx.FatalIfErrorf(err)
	}
	slog.Info("command completed", "command", ctx.Command())
}
