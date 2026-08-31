package cmd

type CLI struct {
	ManInstall ManInstallCmd `cmd:"" name:"man-install" help:"Install man page for local use"`

	Init InitCmd `cmd:"" help:"Initialise a bag"`
	Add  AddCmd  `cmd:"" help:"Add a binary"`
	Tool ToolCmd `cmd:"" help:"Manage project tools"`
}
