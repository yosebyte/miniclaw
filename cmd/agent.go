// MIT License - Copyright (c) 2026 yosebyte
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yosebyte/miniclaw/internal/agent"
	"github.com/yosebyte/miniclaw/internal/config"
	"github.com/yosebyte/miniclaw/internal/provider"
)

var agentMessage string

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Chat with the agent (interactive or single message)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if !cfg.IsAuthenticated() {
			return fmt.Errorf("not authenticated; run: miniclaw provider login\n  or add apiKey to %s", config.ConfigPath())
		}

		claude := provider.New(cfg)
		loop := agent.NewLoop(cfg, claude)
		ctx := context.Background()

		if agentMessage != "" {
			return runOnce(ctx, loop, agentMessage)
		}
		return runInteractive(ctx, loop)
	},
}

func init() {
	agentCmd.Flags().StringVarP(&agentMessage, "message", "m", "", "Single message to send")
}

func runOnce(ctx context.Context, loop *agent.Loop, msg string) error {
	resp, err := loop.ProcessMessage(ctx, "cli:direct", msg)
	if err != nil {
		return err
	}
	fmt.Println(resp)
	return nil
}

func runInteractive(ctx context.Context, loop *agent.Loop) error {
	fmt.Println("🐾 miniclaw interactive mode. Type /help for commands, exit/quit to leave.")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\nYou: ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if lower == "exit" || lower == "quit" || lower == "/exit" || lower == "/quit" || lower == ":q" {
			fmt.Println("Goodbye!")
			break
		}

		resp, err := loop.ProcessMessage(ctx, "cli:direct", line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		fmt.Printf("\nAssistant: %s\n", resp)
	}
	return scanner.Err()
}
