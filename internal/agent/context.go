// MIT License - Copyright (c) 2026 yosebyte
package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yosebyte/miniclaw/internal/provider"
)

// BuildSystemPrompt constructs the system prompt, reading workspace persona files
// (SOUL.md, AGENTS.md, USER.md) and memory files (MEMORY.md, HISTORY.md).
func BuildSystemPrompt(workspace, memory, history string) string {
	var parts []string

	// Persona and behavioural files
	if soul := readWorkspaceFile(workspace, "SOUL.md"); soul != "" {
		parts = append(parts, soul)
	}
	if agents := readWorkspaceFile(workspace, "AGENTS.md"); agents != "" {
		parts = append(parts, agents)
	}

	// Runtime context
	ctx := fmt.Sprintf("Current time: %s\nWorkspace: %s",
		time.Now().Format("2006-01-02 15:04 MST"), workspace)
	parts = append(parts, ctx)

	// User profile
	if user := readWorkspaceFile(workspace, "USER.md"); user != "" {
		parts = append(parts, "## About the User\n"+user)
	}

	// Long-term memory
	if memory != "" {
		parts = append(parts, "## Long-term Memory\n"+memory)
	}

	// Conversation history archive
	if history != "" {
		h := history
		if len(h) > 2000 {
			h = "...(truncated)\n" + h[len(h)-2000:]
		}
		parts = append(parts, "## Conversation History\n"+h)
	}

	return strings.Join(parts, "\n\n")
}

// BuildMessages creates the full messages list for a chat request.
func BuildMessages(history []provider.Message, currentContent string) []provider.Message {
	msgs := make([]provider.Message, len(history))
	copy(msgs, history)
	msgs = append(msgs, provider.Message{
		Role:    "user",
		Content: currentContent,
	})
	return msgs
}

func readWorkspaceFile(workspace, name string) string {
	data, err := os.ReadFile(filepath.Join(workspace, name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
