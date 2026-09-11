package cmd

import "context"

type ToolCmd struct{}

func (c *ToolCmd) Run(ctx context.Context) error {
	return nil
}
