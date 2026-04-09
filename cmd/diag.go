package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDiagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diag",
		Short: "Collect host diagnostics and request AI analysis",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "diag stub collecting system information...")
			return nil
		},
	}

	return cmd
}
