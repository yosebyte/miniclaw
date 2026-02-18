// MIT License - Copyright (c) 2026 yosebyte
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yosebyte/miniclaw/internal/config"
	"github.com/yosebyte/miniclaw/internal/cron"
)

var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Manage scheduled cron jobs",
}

var cronListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all scheduled jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := loadCronSvc()
		fmt.Println(svc.ListFormatted())
		return nil
	},
}

var (
	cronName     string
	cronSchedule string
	cronMessage  string
	cronChatID   string
)

var cronAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a scheduled job",
	Example: `  miniclaw cron add --name "morning" --schedule "0 9 * * *" --message "Good morning!" --chat-id 123456
  miniclaw cron add --name "hourly" --schedule "@every 1h" --message "Check for updates"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cronName == "" || cronSchedule == "" || cronMessage == "" {
			return fmt.Errorf("--name, --schedule, and --message are required")
		}
		svc := loadCronSvc()
		job, err := svc.Add(cronName, cronSchedule, cronMessage, cronChatID)
		if err != nil {
			return err
		}
		fmt.Printf("✅ Job added: [%s] %s — %s\n", job.ID, job.Name, job.Schedule)
		return nil
	},
}

var cronRemoveCmd = &cobra.Command{
	Use:     "remove <id>",
	Aliases: []string{"rm"},
	Short:   "Remove a scheduled job by ID",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := loadCronSvc()
		if err := svc.Remove(args[0]); err != nil {
			return err
		}
		fmt.Printf("✅ Job %s removed.\n", args[0])
		return nil
	},
}

func init() {
	cronAddCmd.Flags().StringVar(&cronName, "name", "", "Job name")
	cronAddCmd.Flags().StringVar(&cronSchedule, "schedule", "", "Cron expression or shorthand (@every 1h, @daily, etc.)")
	cronAddCmd.Flags().StringVar(&cronMessage, "message", "", "Message the agent receives when job fires")
	cronAddCmd.Flags().StringVar(&cronChatID, "chat-id", "", "Telegram chat ID to send response to")

	cronCmd.AddCommand(cronListCmd, cronAddCmd, cronRemoveCmd)
}

func loadCronSvc() *cron.Service {
	return cron.New(config.CronPath(), nil, nil)
}
