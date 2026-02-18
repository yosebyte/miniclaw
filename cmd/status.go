// MIT License - Copyright (c) 2026 yosebyte
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yosebyte/miniclaw/internal/config"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show miniclaw configuration and auth status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		fmt.Println("🐾 miniclaw status")
		fmt.Println("==================")
		fmt.Printf("Config:    %s\n", config.ConfigPath())
		fmt.Printf("Workspace: %s\n", cfg.WorkspacePath())
		fmt.Println()

		// Provider
		fmt.Println("Provider: Claude")
		if cfg.Provider.APIKey != "" {
			fmt.Println("  Auth:  API key ✅")
		} else if cfg.Provider.AccessToken != "" {
			fmt.Println("  Auth:  OAuth token ✅")
		} else {
			fmt.Println("  Auth:  ❌ Not authenticated (run: miniclaw provider login)")
		}
		fmt.Printf("  Model: %s\n", cfg.Provider.Model)
		fmt.Println()

		// Telegram
		fmt.Println("Telegram:")
		if cfg.Telegram.Token != "" {
			fmt.Println("  Token: configured ✅")
		} else {
			fmt.Println("  Token: ❌ Not configured")
		}
		if len(cfg.Telegram.AllowFrom) > 0 {
			fmt.Printf("  AllowFrom: %v\n", cfg.Telegram.AllowFrom)
		} else {
			fmt.Println("  AllowFrom: (allow all)")
		}
		return nil
	},
}
