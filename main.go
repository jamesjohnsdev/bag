package main

import (
	"github.com/alecthomas/kong"
	"github.com/jamesjohnsdev/bag/cmd"
)

func main() {
	ctx := kong.Parse(&cmd.CLI{})
	ctx.FatalIfErrorf(ctx.Run())
}
