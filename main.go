package main

import (
	"context"

	"github.com/alecthomas/kong"
	"github.com/jamesjohnsdev/bag/internal/cmd"
	"github.com/jamesjohnsdev/bag/internal/config"
)

func main() {
	config.Load()
	ctx := kong.Parse(&cmd.CLI{}, kong.BindTo(context.Background(), (*context.Context)(nil)))
	ctx.FatalIfErrorf(ctx.Run())
}
