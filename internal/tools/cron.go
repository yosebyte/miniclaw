// MIT License - Copyright (c) 2026 yosebyte
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yosebyte/miniclaw/internal/provider"
)

// CronAdder is the interface the cron tool uses to schedule jobs.
type CronAdder interface {
	Add(name, schedule, message, chatID string) (interface{ GetID() string; GetSchedule() string }, error)
	Remove(id string) error
	List() []interface{ GetID() string; GetName() string; GetSchedule() string; GetMessage() string }
}

// --- cron_add ---

// CronAddTool schedules a new cron job.
type CronAddTool struct {
	chatID  *string // pointer so gateway can update it per-message
	addFunc func(name, schedule, message, chatID string) error
}

// NewCronAddTool creates a CronAddTool with a mutable chatID pointer.
func NewCronAddTool(chatID *string, addFunc func(name, schedule, message, chatID string) error) CronAddTool {
	return CronAddTool{chatID: chatID, addFunc: addFunc}
}

func (t CronAddTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "cron_add",
		Description: "Schedule a recurring or one-shot task. The agent will be called with the given message when the schedule fires, and the response is sent to the current chat. Use standard cron expressions (e.g. '0 9 * * *' for 9 AM daily) or '@every 30m', '@hourly', '@daily'.",
		InputSchema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "name":     {"type": "string",  "description": "Short human-readable name for the job."},
    "schedule": {"type": "string",  "description": "Cron expression (e.g. '0 9 * * *') or shorthand ('@every 30m', '@daily')."},
    "message":  {"type": "string",  "description": "Message the agent receives when the job fires."}
  },
  "required": ["name", "schedule", "message"]
}`),
	}
}

func (t CronAddTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var args struct {
		Name     string `json:"name"`
		Schedule string `json:"schedule"`
		Message  string `json:"message"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	chatID := ""
	if t.chatID != nil {
		chatID = *t.chatID
	}
	if err := t.addFunc(args.Name, args.Schedule, args.Message, chatID); err != nil {
		return "", err
	}
	return fmt.Sprintf("✅ Cron job %q scheduled (%s).", args.Name, args.Schedule), nil
}

// --- cron_list ---

// CronListTool lists all scheduled jobs.
type CronListTool struct {
	listFunc func() string
}

// NewCronListTool creates a CronListTool.
func NewCronListTool(listFunc func() string) CronListTool {
	return CronListTool{listFunc: listFunc}
}

func (t CronListTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "cron_list",
		Description: "List all scheduled cron jobs.",
		InputSchema: json.RawMessage(`{"type": "object", "properties": {}}`),
	}
}

func (t CronListTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	return t.listFunc(), nil
}

// --- cron_remove ---

// CronRemoveTool removes a scheduled job by ID.
type CronRemoveTool struct {
	removeFunc func(id string) error
}

// NewCronRemoveTool creates a CronRemoveTool.
func NewCronRemoveTool(removeFunc func(id string) error) CronRemoveTool {
	return CronRemoveTool{removeFunc: removeFunc}
}

func (t CronRemoveTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "cron_remove",
		Description: "Remove a scheduled cron job by its ID.",
		InputSchema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "id": {"type": "string", "description": "Job ID from cron_list."}
  },
  "required": ["id"]
}`),
	}
}

func (t CronRemoveTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var args struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	if err := t.removeFunc(args.ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("✅ Cron job %q removed.", args.ID), nil
}

// --- send_message ---

// SendMessageTool lets the agent proactively send a message to the current chat.
type SendMessageTool struct {
	chatID   *string
	sendFunc func(chatID, text string) error
}

// NewSendMessageTool creates a SendMessageTool.
func NewSendMessageTool(chatID *string, sendFunc func(chatID, text string) error) SendMessageTool {
	return SendMessageTool{chatID: chatID, sendFunc: sendFunc}
}

func (t SendMessageTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "send_message",
		Description: "Send a message to the current chat. Useful during heartbeat or cron execution to proactively notify the user.",
		InputSchema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "text":    {"type": "string", "description": "Message text to send."},
    "chat_id": {"type": "string", "description": "Optional chat ID override. Defaults to current chat."}
  },
  "required": ["text"]
}`),
	}
}

func (t SendMessageTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var args struct {
		Text   string `json:"text"`
		ChatID string `json:"chat_id"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	chatID := args.ChatID
	if chatID == "" && t.chatID != nil {
		chatID = *t.chatID
	}
	if chatID == "" {
		return "", fmt.Errorf("no chat_id available")
	}
	if err := t.sendFunc(chatID, args.Text); err != nil {
		return "", fmt.Errorf("send_message: %w", err)
	}
	return "Message sent.", nil
}

// FormatJobList formats a slice of cron jobs as a human-readable string.
func FormatJobList(jobs []struct {
	ID       string
	Name     string
	Schedule string
	Message  string
}) string {
	if len(jobs) == 0 {
		return "No cron jobs scheduled."
	}
	var sb strings.Builder
	for _, j := range jobs {
		fmt.Fprintf(&sb, "• [%s] %s — %s\n  Message: %s\n", j.ID, j.Name, j.Schedule, j.Message)
	}
	return strings.TrimRight(sb.String(), "\n")
}
