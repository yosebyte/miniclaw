// MIT License - Copyright (c) 2026 yosebyte
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yosebyte/miniclaw/internal/config"
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Initialise miniclaw config and workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.ConfigPath()

		// Don't overwrite existing config
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("Config already exists at %s\n", path)
			printNextSteps()
			return nil
		}

		cfg := config.DefaultConfig()
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		// Create workspace directory
		if err := os.MkdirAll(cfg.WorkspacePath(), 0755); err != nil {
			return fmt.Errorf("creating workspace: %w", err)
		}

		fmt.Printf("✅ Config created at %s\n", path)
		fmt.Printf("✅ Workspace created at %s\n", cfg.WorkspacePath())
		printNextSteps()
		return nil
	},
}

func printNextSteps() {
	fmt.Print(`
Next steps:
  1. Authenticate with Claude:
       miniclaw provider login

     Or add your Anthropic API key to config.json:
       "provider": { "apiKey": "sk-ant-..." }

  2. Add your Telegram bot token to config.json:
       "telegram": { "token": "YOUR_BOT_TOKEN" }

     Get a token from @BotFather on Telegram.

  3. (Optional) Restrict access:
       "telegram": { "allowFrom": ["YOUR_USER_ID"] }

  4. Start the gateway:
       miniclaw gateway
`)
}
