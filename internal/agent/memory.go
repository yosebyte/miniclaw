// MIT License - Copyright (c) 2026 yosebyte
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yosebyte/miniclaw/internal/provider"
)

// MemoryStore manages MEMORY.md and HISTORY.md in the workspace.
type MemoryStore struct {
	workspace string
}

// NewMemoryStore creates a MemoryStore for the given workspace directory.
func NewMemoryStore(workspace string) *MemoryStore {
	return &MemoryStore{workspace: workspace}
}

// ReadMemory returns the content of MEMORY.md.
func (m *MemoryStore) ReadMemory() string {
	data, _ := os.ReadFile(filepath.Join(m.workspace, "MEMORY.md"))
	return string(data)
}

// WriteMemory overwrites MEMORY.md.
func (m *MemoryStore) WriteMemory(content string) error {
	return m.writeFile("MEMORY.md", content)
}

// AppendHistory appends an entry to HISTORY.md.
func (m *MemoryStore) AppendHistory(entry string) error {
	path := filepath.Join(m.workspace, "HISTORY.md")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, entry)
	return err
}

// ReadHistory returns the content of HISTORY.md.
func (m *MemoryStore) ReadHistory() string {
	data, _ := os.ReadFile(filepath.Join(m.workspace, "HISTORY.md"))
	return string(data)
}

func (m *MemoryStore) writeFile(name, content string) error {
	if err := os.MkdirAll(m.workspace, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.workspace, name), []byte(content), 0644)
}

// Consolidate summarises old messages into HISTORY.md and updates MEMORY.md using Claude.
func (m *MemoryStore) Consolidate(ctx context.Context, claude *provider.Claude, session *Session, memWindow int) {
	keepCount := memWindow / 2
	if len(session.Messages) <= keepCount {
		return
	}

	end := len(session.Messages) - keepCount
	if end <= session.LastConsolidated {
		return
	}
	oldMsgs := session.Messages[session.LastConsolidated:end]
	if len(oldMsgs) == 0 {
		return
	}

	var lines []string
	for _, msg := range oldMsgs {
		tools := ""
		if len(msg.ToolsUsed) > 0 {
			tools = " [tools: " + strings.Join(msg.ToolsUsed, ", ") + "]"
		}
		ts := msg.Timestamp.Format("2006-01-02 15:04")
		lines = append(lines, fmt.Sprintf("[%s] %s%s: %s", ts, strings.ToUpper(msg.Role), tools, msg.Content))
	}
	conversation := strings.Join(lines, "\n")
	currentMemory := m.ReadMemory()

	memSnippet := currentMemory
	if memSnippet == "" {
		memSnippet = "(empty)"
	}

	prompt := fmt.Sprintf(`You are a memory consolidation agent. Process this conversation and return a JSON object with exactly two keys:

1. "history_entry": A paragraph (2-5 sentences) summarizing the key events/decisions/topics. Start with a timestamp like [%s].

2. "memory_update": The updated long-term memory content. Add any new facts: user preferences, personal info, project context, technical decisions. If nothing new, return the existing content unchanged.

## Current Long-term Memory
%s

## Conversation to Process
%s

Respond with ONLY valid JSON, no markdown fences.`,
		time.Now().Format("2006-01-02 15:04"),
		memSnippet,
		conversation,
	)

	resp, err := claude.Chat(ctx, "You are a memory consolidation agent. Respond only with valid JSON.", []provider.Message{
		{Role: "user", Content: prompt},
	}, nil)
	if err != nil {
		slog.Error("memory consolidation failed", "err", err)
		return
	}

	text := ""
	for _, block := range resp.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}
	text = strings.TrimSpace(text)
	if text == "" {
		slog.Warn("memory consolidation: empty response")
		return
	}

	// Strip possible markdown fences
	if strings.HasPrefix(text, "```") {
		if _, after, ok := strings.Cut(text, "\n"); ok {
			text = after
		}
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}

	var result struct {
		HistoryEntry string `json:"history_entry"`
		MemoryUpdate string `json:"memory_update"`
	}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		preview := text
		if len(preview) > 200 {
			preview = preview[:200]
		}
		slog.Warn("memory consolidation: invalid JSON", "err", err, "text", preview)
		return
	}
	if result.HistoryEntry != "" {
		_ = m.AppendHistory(result.HistoryEntry)
	}
	if result.MemoryUpdate != "" && result.MemoryUpdate != currentMemory {
		_ = m.WriteMemory(result.MemoryUpdate)
	}
	session.LastConsolidated = end
	slog.Info("memory consolidation done", "messages", len(session.Messages), "last_consolidated", session.LastConsolidated)
}
