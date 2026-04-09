package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ask [question]",
		Short: "Ask the AI assistant an operations question",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "ask stub received: %s\n", args[0])
			return nil
		},
	}

	return cmd
}
