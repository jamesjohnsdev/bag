package cmd

type CLI struct {
	Init InitCmd `cmd:"" help:"Initialise a bag"`
	Add  AddCmd  `cmd:"" help:"Add a binary"`
	Tool ToolCmd `cmd:"" help:"Manage project tools"`
}
