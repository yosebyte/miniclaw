// MIT License - Copyright (c) 2026 yosebyte
package agent

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yosebyte/miniclaw/internal/config"
	"github.com/yosebyte/miniclaw/internal/provider"
	"github.com/yosebyte/miniclaw/internal/tools"
)

// SendFunc sends a message to a chat (Telegram).
type SendFunc func(chatID, text string) error

// CronService is the interface the loop needs from the cron service.
type CronService interface {
	AddJob(name, schedule, message, chatID string) error
	Remove(id string) error
	ListFormatted() string
}

// Loop is the core agent processing engine.
type Loop struct {
	cfg      *config.Config
	claude   *provider.Claude
	sessions *SessionManager
	memory   *MemoryStore
	reg      *tools.Registry

	// mutable context updated per-message so tools can route replies
	currentChatID string
	sendFn        SendFunc
	cronSvc       CronService
}

// NewLoop creates a Loop. Call SetSendFunc and SetCronService before starting.
func NewLoop(cfg *config.Config, claude *provider.Claude) *Loop {
	workspace := cfg.WorkspacePath()
	sessDir := filepath.Join(filepath.Dir(workspace), "sessions")

	l := &Loop{
		cfg:      cfg,
		claude:   claude,
		sessions: NewSessionManager(sessDir),
		memory:   NewMemoryStore(workspace),
		reg:      tools.NewRegistry(),
	}
	l.registerBaseTools()
	return l
}

// SetSendFunc sets the send callback and registers the send_message tool.
func (l *Loop) SetSendFunc(sendFn SendFunc) {
	l.sendFn = sendFn
	if sendFn != nil {
		l.reg.Register(tools.NewSendMessageTool(&l.currentChatID, sendFn))
	}
}

// SetCronService registers cron tools into the loop.
func (l *Loop) SetCronService(cronSvc CronService) {
	l.cronSvc = cronSvc
	if cronSvc != nil {
		addFn := func(name, schedule, message, chatID string) error {
			return cronSvc.AddJob(name, schedule, message, chatID)
		}
		l.reg.Register(tools.NewCronAddTool(&l.currentChatID, addFn))
		l.reg.Register(tools.NewCronListTool(cronSvc.ListFormatted))
		l.reg.Register(tools.NewCronRemoveTool(cronSvc.Remove))
	}
}

func (l *Loop) registerBaseTools() {
	l.reg.Register(tools.ReadFileTool{})
	l.reg.Register(tools.WriteFileTool{})
	l.reg.Register(tools.EditFileTool{})
	l.reg.Register(tools.ListDirTool{})
	l.reg.Register(tools.ExecTool{})
	l.reg.Register(tools.NewWebFetchTool())
}

// ProcessMessage handles one inbound message and returns the assistant reply.
// chatID is used for tool routing (send_message, cron_add).
func (l *Loop) ProcessMessage(ctx context.Context, sessionKey, chatID, userMsg string) (string, error) {
	l.currentChatID = chatID
	session := l.sessions.Get(sessionKey)

	switch strings.TrimSpace(strings.ToLower(userMsg)) {
	case "/new":
		old := *session
		session.Clear()
		_ = l.sessions.Save(session)
		go func() {
			l.memory.Consolidate(context.Background(), l.claude, &old, l.memWindow())
		}()
		return "New session started. Memory consolidation in progress.", nil
	case "/help":
		return "🐾 miniclaw commands:\n/new — Start a new conversation\n/help — Show available commands", nil
	}

	memWindow := l.memWindow()
	if len(session.Messages) > memWindow {
		go func() {
			snap := *session
			l.memory.Consolidate(context.Background(), l.claude, &snap, memWindow)
			session.LastConsolidated = snap.LastConsolidated
			_ = l.sessions.Save(session)
		}()
	}

	systemPrompt := BuildSystemPrompt(l.cfg.WorkspacePath(), l.memory.ReadMemory(), l.memory.ReadHistory())
	history := session.RecentMessages(memWindow)
	messages := BuildMessages(history, userMsg)

	finalContent, toolsUsed, err := l.runLoop(ctx, systemPrompt, messages)
	if err != nil {
		return "", err
	}

	session.Add("user", userMsg)
	session.Add("assistant", finalContent, toolsUsed...)
	_ = l.sessions.Save(session)

	return finalContent, nil
}

func (l *Loop) memWindow() int {
	if l.cfg.Provider.MemoryWindow > 0 {
		return l.cfg.Provider.MemoryWindow
	}
	return 50
}

func (l *Loop) runLoop(ctx context.Context, system string, messages []provider.Message) (string, []string, error) {
	maxIter := l.cfg.Provider.MaxIterations
	if maxIter == 0 {
		maxIter = 20
	}
	toolDefs := l.reg.Definitions()
	var toolsUsed []string

	for range maxIter {
		resp, err := l.claude.Chat(ctx, system, messages, toolDefs)
		if err != nil {
			return "", toolsUsed, fmt.Errorf("LLM error: %w", err)
		}

		var textContent string
		var toolCalls []provider.ContentBlock

		for _, block := range resp.Content {
			switch block.Type {
			case "text":
				textContent = block.Text
			case "tool_use":
				toolCalls = append(toolCalls, block)
			}
		}

		if resp.StopReason == "end_turn" || len(toolCalls) == 0 {
			return textContent, toolsUsed, nil
		}

		messages = append(messages, provider.Message{
			Role:    "assistant",
			Content: resp.Content,
		})

		var toolResults []provider.ContentBlock
		for _, tc := range toolCalls {
			toolsUsed = append(toolsUsed, tc.Name)
			input := string(tc.Input)
			if len(input) > 200 {
				input = input[:200] + "..."
			}
			slog.Info("tool call", "name", tc.Name, "input", input)

			result, execErr := l.reg.Execute(ctx, tc.Name, tc.Input)
			isError := false
			if execErr != nil {
				result = "Error: " + execErr.Error()
				isError = true
				slog.Warn("tool error", "name", tc.Name, "err", execErr)
			}

			toolResults = append(toolResults, provider.ContentBlock{
				Type:      "tool_result",
				ToolUseID: tc.ID,
				Content:   result,
				IsError:   isError,
			})
		}

		messages = append(messages, provider.Message{
			Role:    "user",
			Content: toolResults,
		})
	}

	return "I've completed processing but have no response to give.", toolsUsed, nil
}
