package cmd

import (
	"context"
)

type ToolCmd struct {
	Init     InitToolCmd     `cmd:"" name:"init" help:"Initialise a project-local bag"`
	Manifest ManifestToolCmd `cmd:"" help:"Manifest commands for local project tools"`
}

func (c *ToolCmd) Run(ctx context.Context) error {
	return nil
}
