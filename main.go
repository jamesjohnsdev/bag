package main

import (
	"context"
	"log"
	"runtime"

	"github.com/alecthomas/kong"
	"github.com/jamesjohnsdev/bag/internal/cmd"
	"github.com/jamesjohnsdev/bag/internal/config"
)

func main() {
	if err := config.Load(); err != nil {
		log.Fatalf("loading config: %s", err.Error())
	}
	ctx := kong.Parse(&cmd.CLI{}, kong.Name("bag"), kong.BindTo(context.Background(), (*context.Context)(nil)), cmd.Description)

	if runtime.GOOS != "windows" && ctx.Command() != "man-install" {
		cmd.EnsureManPage(ctx.Model)
	}

	ctx.FatalIfErrorf(ctx.Run())
}
