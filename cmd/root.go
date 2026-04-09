package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"go-ops-agent/internal/config"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "ops-agent",
	Short: "AI powered ops assistant for terminal diagnostics",
	Long:  "ops-agent is a terminal-first SRE assistant that collects local diagnostics and asks an LLM for analysis.",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
	rootCmd.SilenceUsage = true

	rootCmd.AddCommand(newAskCmd())
	rootCmd.AddCommand(newDiagCmd())
}

func initConfig() {
	_, err := config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
	}
}
