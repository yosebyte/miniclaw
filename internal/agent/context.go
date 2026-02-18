// MIT License - Copyright (c) 2026 yosebyte
package agent

import (
	"fmt"
	"time"

	"github.com/yosebyte/miniclaw/internal/provider"
)

const systemPromptTemplate = `You are a helpful personal AI assistant running as miniclaw.

Current time: %s
Workspace: %s

%s%s`

// BuildSystemPrompt constructs the system prompt with memory context.
func BuildSystemPrompt(workspace, memory, history string) string {
	memSection := ""
	if memory != "" {
		memSection = "## Long-term Memory\n" + memory + "\n\n"
	}
	histSection := ""
	if history != "" {
		// Trim history to last ~2000 chars to avoid bloat
		if len(history) > 2000 {
			history = "...(truncated)\n" + history[len(history)-2000:]
		}
		histSection = "## Conversation History\n" + history + "\n\n"
	}
	return fmt.Sprintf(systemPromptTemplate,
		time.Now().Format("2006-01-02 15:04 MST"),
		workspace,
		memSection,
		histSection,
	)
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
