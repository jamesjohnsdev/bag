package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"

	"github.com/jamesjohnsdev/bag/internal/manifest"
)

type ManifestCmd struct {
	List  ManifestListCmd  `cmd:"" help:"List executables in manifest"`
	View  ManifestViewCmd  `cmd:"" help:"View the manifest file"`
	Check ManifestCheckCmd `cmd:"" help:"Validate manifest matches expected format"`
}

type ManifestListCmd struct {
	Commands bool `flag:"" xor:"listFlags" help:"Only list commands"`
	Scripts  bool `flag:"" xor:"listFlags" help:"Only list scripts"`
	Binaries bool `flag:"" xor:"listFlags" help:"Only list binaries"`
}

func (cmd *ManifestListCmd) Run(ctx context.Context) error {
	ws, err := workSpace()
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}
	return listManifest(ws.manPath, cmd.Commands, cmd.Scripts, cmd.Binaries)
}

type ManifestViewCmd struct{}

func (cmd *ManifestViewCmd) Run(ctx context.Context) error {
	ws, err := workSpace()
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}
	return viewManifest(ws.manPath)
}

type ManifestCheckCmd struct{}

func (cmd *ManifestCheckCmd) Run(ctx context.Context) error {
	ws, err := workSpace()
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}
	return checkManifest(ws.manPath)
}

type ManifestToolCmd struct {
	List  ManifestToolListCmd  `cmd:"" name:"list" help:"List executables in the project manifest"`
	View  ManifestToolViewCmd  `cmd:"" name:"view" help:"View the project manifest file"`
	Check ManifestToolCheckCmd `cmd:"" name:"check" help:"Validate project manifest matches expected format"`
}

type ManifestToolListCmd struct {
	Commands bool `flag:"" xor:"listFlags" help:"Only list commands"`
	Scripts  bool `flag:"" xor:"listFlags" help:"Only list scripts"`
	Binaries bool `flag:"" xor:"listFlags" help:"Only list binaries"`
}

func (cmd *ManifestToolListCmd) Run(ctx context.Context) error {
	manPath, global, err := manifest.Get(true)
	if err != nil {
		return fmt.Errorf("getting manifest: %w", err)
	}
	if global {
		fmt.Println("No local manifest found")
		return nil
	}
	return listManifest(manPath, cmd.Commands, cmd.Scripts, cmd.Binaries)
}

type ManifestToolViewCmd struct{}

func (cmd *ManifestToolViewCmd) Run(ctx context.Context) error {
	manPath, global, err := manifest.Get(true)
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}
	if global {
		fmt.Println("No local manifest found")
		return nil
	}
	return viewManifest(manPath)
}

type ManifestToolCheckCmd struct{}

func (cmd *ManifestToolCheckCmd) Run(ctx context.Context) error {
	manPath, global, err := manifest.Get(true)
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}
	if global {
		fmt.Println("No local manifest found")
		return nil
	}
	return checkManifest(manPath)
}

func checkManifest(manPath string) error {
	_, err := manifest.Parse(manPath)
	if err != nil {
		fmt.Printf("%s", color.RedString("Uh oh. There's a problem with your manifest"))
		fmt.Printf("Manifest location: %s\n", manPath)
		fmt.Printf("Error: %s", err.Error())
		return nil
	}
	fmt.Printf("%s", color.GreenString("Manifest file looks good!\n"))
	return nil
}

func viewManifest(manPath string) error {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, manPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func listManifest(manPath string, commands, scripts, binaries bool) error {
	man, err := manifest.Parse(manPath)
	if err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}
	switch {
	case commands:
		if len(man.Commands) == 0 {
			fmt.Println("No commands found")
			return nil
		}
		printCommands(man.Commands)
		return nil
	case scripts:
		if len(man.Binaries) == 0 {
			fmt.Printf("No scripts found")
		}
		printScripts(man.Binaries)
		return nil
	case binaries:
		if len(man.Binaries) == 0 {
			fmt.Printf("No scripts found")
		}
		printBinaries(man.Binaries)
		return nil
	}

	fmt.Printf("%s\n", color.BlueString("Commands:"))
	if count := printCommands(man.Commands); count == 0 {
		fmt.Println("0 commands found")
	}
	fmt.Printf("%s\n", color.BlueString("Scripts:"))
	if count := printScripts(man.Binaries); count == 0 {
		fmt.Println("0 scripts found")
	}
	fmt.Printf("%s\n", color.BlueString("Binaries:"))
	if count := printBinaries(man.Binaries); count == 0 {
		fmt.Println("0 binaries found")
	}
	return nil
}

func printCommands(cmdEntries map[string]string) (commandCount int) {
	var count int

	for name, command := range cmdEntries {
		count++
		fmt.Printf("%s: %s\n", name, command)
	}
	return count
}

func printBinaries(binEntries map[string]manifest.BinaryEntry) (binaryCount int) {
	var count int
	for name, entry := range binEntries {
		if entry.Type == manifest.BinaryType {
			count++
			fmt.Println(name)
		}
	}
	return count
}

func printScripts(binEntries map[string]manifest.BinaryEntry) (scriptCount int) {
	var count int
	for name, entry := range binEntries {
		if entry.Type == manifest.ScriptType {
			count++
			fmt.Println(name)
		}
	}
	return count
}
