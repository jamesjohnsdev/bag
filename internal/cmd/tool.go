package cmd

import (
	"context"
)

type ToolCmd struct {
	Init     InitToolCmd     `cmd:"" name:"init" help:"Initialise a project-local bag"`
	Add      AddToolCmd      `cmd:"" help:"add a project-scoped executable"`
	List     ListToolCmd     `cmd:"" help:"list all stored versions of a binary"`
	Manifest ManifestToolCmd `cmd:"" help:"Manifest commands for local project tools"`
	Lock     LockToolCmd     `cmd:"" help:"Lock commands for local project tools"`
}

func (c *ToolCmd) Run(ctx context.Context) error {
	return nil
}
