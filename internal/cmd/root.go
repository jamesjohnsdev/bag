package cmd

import (
	"context"

	"github.com/alecthomas/kong"
)

type command interface {
	Run(ctx context.Context) error
}

type CLI struct {
	Version kong.VersionFlag `help:"Print version and exit"`

	ManInstall ManInstallCmd `cmd:"" name:"man-install" help:"Install man page for local use"`

	Init   InitCmd   `cmd:"" help:"Initialise a bag"`
	Add    AddCmd    `cmd:"" help:"Add a binary"`
	Update UpdateCmd `cmd:"" help:"Update a binary"`
	List   ListCmd   `cmd:"" help:"List all stored versions of a binary"`
	Remove RemoveCmd `cmd:"" help:"Remove an installed binary"`
	View   ViewCmd   `cmd:"" help:"View details of an installed binary"`
	Which  WhichCmd  `cmd:"" help:"Find the stored path of a binary"`
	Tool   ToolCmd   `cmd:"" help:"Manage project tools"`
}

// TODO: Consider changing definitions for commented out ones
var (
	// _ command = (*ManInstallCmd)(nil)
	_ command = (*InitCmd)(nil)
	_ command = (*AddCmd)(nil)
	_ command = (*UpdateCmd)(nil)
	_ command = (*ListCmd)(nil)
	_ command = (*RemoveCmd)(nil)
	_ command = (*ViewCmd)(nil)
	_ command = (*WhichCmd)(nil)
	_ command = (*ToolCmd)(nil)
)
