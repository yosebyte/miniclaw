// MIT License - Copyright (c) 2026 yosebyte
package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yosebyte/miniclaw/internal/provider"
)

// SessionMessage is a single message stored in session history.
type SessionMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	ToolsUsed []string  `json:"toolsUsed,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Session holds the conversation history for a single chat.
type Session struct {
	Key              string           `json:"key"`
	Messages         []SessionMessage `json:"messages"`
	LastConsolidated int              `json:"lastConsolidated"`
}

// Add appends a message to the session.
func (s *Session) Add(role, content string, tools ...string) {
	msg := SessionMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().UTC(),
	}
	if len(tools) > 0 {
		msg.ToolsUsed = tools
	}
	s.Messages = append(s.Messages, msg)
}

// RecentMessages returns up to n most recent messages as provider.Message slices.
func (s *Session) RecentMessages(n int) []provider.Message {
	msgs := s.Messages
	if len(msgs) > n {
		msgs = msgs[len(msgs)-n:]
	}
	result := make([]provider.Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, provider.Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return result
}

// Clear resets the session messages.
func (s *Session) Clear() {
	s.Messages = nil
	s.LastConsolidated = 0
}

// SessionManager manages per-chat sessions.
type SessionManager struct {
	dir string
}

// NewSessionManager creates a manager rooted at dir.
func NewSessionManager(dir string) *SessionManager {
	return &SessionManager{dir: dir}
}

func (m *SessionManager) path(key string) string {
	safe := strings.NewReplacer(":", "_", "/", "_", "\\", "_").Replace(key)
	return filepath.Join(m.dir, safe+".json")
}

// Get loads a session by key, returning an empty one if not found.
func (m *SessionManager) Get(key string) *Session {
	s := &Session{Key: key}
	data, err := os.ReadFile(m.path(key))
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, s)
	return s
}

// Save persists a session to disk.
func (m *SessionManager) Save(s *Session) error {
	if err := os.MkdirAll(m.dir, 0700); err != nil {
		return fmt.Errorf("creating sessions dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path(s.Key), data, 0600)
}
