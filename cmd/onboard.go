// MIT License - Copyright (c) 2026 yosebyte
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yosebyte/miniclaw/internal/config"
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Initialise miniclaw config and workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.ConfigPath()

		if _, err := os.Stat(path); err == nil {
			fmt.Printf("Config already exists at %s\n", path)
			printNextSteps()
			return nil
		}

		cfg := config.DefaultConfig()
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		workspace := cfg.WorkspacePath()
		if err := os.MkdirAll(workspace, 0755); err != nil {
			return fmt.Errorf("creating workspace: %w", err)
		}

		// Create starter workspace files
		writeIfAbsent(filepath.Join(workspace, "SOUL.md"),
			"# SOUL.md — Who You Are\n\nYou are a helpful personal AI assistant. Be concise, direct, and useful.\n")
		writeIfAbsent(filepath.Join(workspace, "USER.md"),
			"# USER.md — About Your User\n\n- Name: (fill in)\n- Timezone: (fill in)\n")
		writeIfAbsent(filepath.Join(workspace, "AGENTS.md"),
			"# AGENTS.md — Behaviour Notes\n\n<!-- Add instructions for the agent here. -->\n")
		writeIfAbsent(filepath.Join(workspace, "HEARTBEAT.md"),
			"# HEARTBEAT.md\n\n# Add tasks below. Comment-only files are skipped.\n# Example: Check my top unread emails and summarise any urgent ones.\n")

		fmt.Printf("✅ Config created at %s\n", path)
		fmt.Printf("✅ Workspace created at %s\n", workspace)
		fmt.Println("✅ Starter files created: SOUL.md, USER.md, AGENTS.md, HEARTBEAT.md")
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

  4. (Optional) Edit workspace files to personalise the agent:
       ~/.miniclaw/workspace/SOUL.md    — persona
       ~/.miniclaw/workspace/USER.md    — about you
       ~/.miniclaw/workspace/AGENTS.md  — behaviour notes
       ~/.miniclaw/workspace/HEARTBEAT.md — periodic tasks

  5. Start the gateway:
       miniclaw gateway
`)
}

func writeIfAbsent(path, content string) {
	if _, err := os.Stat(path); err == nil {
		return // already exists
	}
	_ = os.WriteFile(path, []byte(content), 0644)
}
