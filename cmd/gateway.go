// MIT License - Copyright (c) 2026 yosebyte
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/yosebyte/miniclaw/internal/agent"
	"github.com/yosebyte/miniclaw/internal/config"
	"github.com/yosebyte/miniclaw/internal/provider"
	"github.com/yosebyte/miniclaw/internal/telegram"
)

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Start the Telegram gateway (long polling)",
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
		bot := telegram.New(cfg, loop)

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		log.Printf("[INFO] miniclaw gateway starting")
		return bot.Run(ctx)
	},
}
