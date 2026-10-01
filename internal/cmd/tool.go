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
	// Remove   RemoveToolCmd   `cmd:"" help:"Remove an executable"`
	// Update   UpdateToolCmd   `cmd:"" help:"Update an executable"`
	Use   UseToolCmd   `cmd:"" help:"Select a version of an executable to use"`
	View  ViewToolCmd  `cmd:"" help:"View information about the currently used executable in a project"`
	Which WhichToolCmd `cmd:"" help:"Find the stored path of the selected executable"`
}

func (c *ToolCmd) Run(ctx context.Context) error {
	return nil
}
