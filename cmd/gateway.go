// MIT License - Copyright (c) 2026 yosebyte
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/yosebyte/miniclaw/internal/agent"
	"github.com/yosebyte/miniclaw/internal/config"
	"github.com/yosebyte/miniclaw/internal/cron"
	"github.com/yosebyte/miniclaw/internal/heartbeat"
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
		if cfg.Telegram.Token == "" {
			return fmt.Errorf("telegram.token not configured in %s", config.ConfigPath())
		}

		claude := provider.New(cfg)

		// 1. Create the agent loop (base tools only).
		loop := agent.NewLoop(cfg, claude)

		// 2. Create the Telegram bot — provides the Send function.
		bot := telegram.New(cfg, loop)

		// 3. Wire send_message tool into loop.
		loop.SetSendFunc(bot.Send)

		// 4. Create cron service — calls loop.ProcessMessage when jobs fire.
		cronSvc := cron.New(
			config.CronPath(),
			bot.Send,
			func(ctx context.Context, chatID, message string) (string, error) {
				return loop.ProcessMessage(ctx, "cron_"+chatID, chatID, message)
			},
		)

		// 5. Register cron tools into loop.
		loop.SetCronService(cronSvc)

		// 6. Heartbeat service.
		var hbService *heartbeat.Service
		if cfg.Heartbeat.Enabled {
			hbService = heartbeat.New(cfg, func(ctx context.Context, sessionKey, chatID, message string) (string, error) {
				return loop.ProcessMessage(ctx, sessionKey, chatID, message)
			}, bot.Send)
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		cronSvc.Start()
		defer cronSvc.Stop()

		if hbService != nil {
			go hbService.Run(ctx)
		}

		slog.Info("miniclaw gateway starting")
		return bot.Run(ctx)
	},
}
