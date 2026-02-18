// MIT License - Copyright (c) 2026 yosebyte
package heartbeat

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yosebyte/miniclaw/internal/config"
)

const heartbeatFile = "HEARTBEAT.md"
const heartbeatOK = "HEARTBEAT_OK"

// AgentFunc processes a message in a given session and returns the response.
type AgentFunc func(ctx context.Context, sessionKey, chatID, message string) (string, error)

// SendFunc delivers a message to a Telegram chat.
type SendFunc func(chatID, text string) error

// Service runs periodic heartbeat checks.
type Service struct {
	cfg     *config.Config
	agent   AgentFunc
	send    SendFunc
}

// New creates a heartbeat Service.
func New(cfg *config.Config, agent AgentFunc, send SendFunc) *Service {
	return &Service{cfg: cfg, agent: agent, send: send}
}

// Run starts the heartbeat loop, blocking until ctx is cancelled.
func (s *Service) Run(ctx context.Context) {
	interval := time.Duration(s.cfg.Heartbeat.IntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 30 * time.Minute
	}

	slog.Info("heartbeat service started", "interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Service) tick(ctx context.Context) {
	content := s.readHeartbeatFile()
	if content == "" {
		slog.Debug("heartbeat: HEARTBEAT.md empty, skipping")
		return
	}

	slog.Info("heartbeat firing", "content_len", len(content))

	chatID := s.cfg.Heartbeat.ChatID
	sessionKey := "heartbeat:main"
	if chatID != "" {
		sessionKey = "telegram_" + chatID
	}

	hbCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	result, err := s.agent(hbCtx, sessionKey, chatID, content)
	if err != nil {
		slog.Error("heartbeat agent error", "err", err)
		return
	}

	// HEARTBEAT_OK means nothing to report
	if strings.TrimSpace(result) == heartbeatOK || result == "" {
		slog.Debug("heartbeat: nothing to report")
		return
	}

	if s.send != nil && chatID != "" {
		if err := s.send(chatID, result); err != nil {
			slog.Error("heartbeat send error", "err", err)
		}
	}
}

func (s *Service) readHeartbeatFile() string {
	path := filepath.Join(s.cfg.WorkspacePath(), heartbeatFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	// Treat comment-only files as empty
	content := strings.TrimSpace(string(data))
	var lines []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			lines = append(lines, line)
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
