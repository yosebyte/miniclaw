// MIT License - Copyright (c) 2026 yosebyte
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "miniclaw",
	Short: "miniclaw — minimal personal AI assistant",
	Long:  "miniclaw is a minimal personal AI assistant powered by Claude, delivered via Telegram.",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(onboardCmd)
	rootCmd.AddCommand(gatewayCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(providerCmd)
	rootCmd.AddCommand(cronCmd)
}
