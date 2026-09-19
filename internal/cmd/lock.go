package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"

	"github.com/jamesjohnsdev/bag/internal/manifest"
)

type LockCmd struct {
	View  LockViewCmd  `cmd:"" help:"View the lock file"`
	Check LockCheckCmd `cmd:"" help:"Validate lock file matches expected format"`
	Tool  LockToolCmd  `cmd:"" help:"Lock commands for local project tools"`
}

type LockViewCmd struct{}

func (cmd *LockViewCmd) Run(ctx context.Context) error {
	ws, err := workSpace()
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}
	return viewLock(manifest.FindLock(ws.manPath))
}

type LockCheckCmd struct{}

func (cmd *LockCheckCmd) Run(ctx context.Context) error {
	ws, err := workSpace()
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}
	return checkLock(manifest.FindLock(ws.manPath))
}

type LockToolCmd struct {
	View  LockToolViewCmd  `cmd:"" name:"view" help:"View the project lock file"`
	Check LockToolCheckCmd `cmd:"" name:"check" help:"Validate project lock file matches expected format"`
}

type LockToolViewCmd struct{}

func (cmd *LockToolViewCmd) Run(ctx context.Context) error {
	manPath, global, err := manifest.Get(true)
	if err != nil {
		return fmt.Errorf("getting manifest: %w", err)
	}
	if global {
		fmt.Println("No local manifest found")
		return nil
	}
	return viewLock(manifest.FindLock(manPath))
}

type LockToolCheckCmd struct{}

func (cmd *LockToolCheckCmd) Run(ctx context.Context) error {
	manPath, global, err := manifest.Get(true)
	if err != nil {
		return fmt.Errorf("getting manifest: %w", err)
	}
	if global {
		fmt.Println("No local manifest found")
		return nil
	}
	return checkLock(manifest.FindLock(manPath))
}

func checkLock(lockPath string) error {
	_, err := manifest.ParseLock(lockPath)
	if err != nil {
		fmt.Printf("%s", color.RedString("Uh oh. There's a problem with your lock file"))
		fmt.Printf("Lock location: %s\n", lockPath)
		fmt.Printf("Error: %s", err.Error())
		return nil
	}
	fmt.Printf("%s", color.GreenString("Lock file looks good!"))
	return nil
}

func viewLock(lockPath string) error {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, lockPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
