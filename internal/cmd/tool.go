package cmd

import (
	"context"
)

type ToolCmd struct {
	Manifest ManifestToolCmd `cmd:"" help:"Manifest commands for local project tools"`
}

func (c *ToolCmd) Run(ctx context.Context) error {
	return nil
}
