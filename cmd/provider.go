// MIT License - Copyright (c) 2026 yosebyte
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yosebyte/miniclaw/internal/config"
	"github.com/yosebyte/miniclaw/internal/provider"
)

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage the Claude provider",
}

var providerLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Claude via OAuth",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ctx := context.Background()
		accessToken, refreshToken, err := provider.Login(ctx)
		if err != nil {
			return fmt.Errorf("OAuth login failed: %w", err)
		}

		cfg.Provider.AccessToken = accessToken
		cfg.Provider.RefreshToken = refreshToken
		// Clear API key since we now use OAuth
		cfg.Provider.APIKey = ""

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving tokens: %w", err)
		}

		fmt.Println("\n✅ Authenticated with Claude! Tokens saved.")
		fmt.Printf("Config: %s\n", config.ConfigPath())
		return nil
	},
}

func init() {
	providerCmd.AddCommand(providerLoginCmd)
}
