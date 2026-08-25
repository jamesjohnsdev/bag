package main

import (
	"github.com/alecthomas/kong"
	"github.com/jamesjohnsdev/bag/internal/cmd"
	"github.com/jamesjohnsdev/bag/internal/config"
)

func main() {
	config.Load()
	ctx := kong.Parse(&cmd.CLI{})
	ctx.FatalIfErrorf(ctx.Run())
}
