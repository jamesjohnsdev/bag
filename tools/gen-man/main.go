// Command gen-man renders bag's man page to a directory, for the release
// build to bundle into the tar.gz archives (auto-install at runtime only
// covers the per-user home directory case, see internal/cmd/man.go).
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/alecthomas/kong"
	"github.com/jamesjohnsdev/bag/internal/cmd"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <output-dir>", os.Args[0])
	}

	k, err := kong.New(&cmd.CLI{}, kong.Name("bag"), kong.BindTo(context.Background(), (*context.Context)(nil)), cmd.Description)
	if err != nil {
		log.Fatalf("building kong app: %s", err)
	}

	dest, err := cmd.RenderManPage(k.Model, os.Args[1])
	if err != nil {
		log.Fatalf("generating man page: %s", err)
	}

	fmt.Println(dest)
}
