// MIT License - Copyright (c) 2026 yosebyte
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config is the root configuration for miniclaw.
type Config struct {
	Provider ProviderConfig  `json:"provider"`
	Telegram TelegramConfig  `json:"telegram"`
	Workspace string         `json:"workspace"`
}

// ProviderConfig holds Claude provider settings.
type ProviderConfig struct {
	AccessToken   string `json:"accessToken"`
	RefreshToken  string `json:"refreshToken"`
	APIKey        string `json:"apiKey"`
	Model         string `json:"model"`
	MaxTokens     int    `json:"maxTokens"`
	MaxIterations int    `json:"maxIterations"`
	MemoryWindow  int    `json:"memoryWindow"`
}

// TelegramConfig holds Telegram bot settings.
type TelegramConfig struct {
	Token     string   `json:"token"`
	AllowFrom []string `json:"allowFrom"`
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Provider: ProviderConfig{
			Model:         "claude-opus-4-5",
			MaxTokens:     8192,
			MaxIterations: 20,
			MemoryWindow:  50,
		},
		Telegram:  TelegramConfig{AllowFrom: []string{}},
		Workspace: "~/.miniclaw/workspace",
	}
}

// ConfigPath returns the default config file path.
func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".miniclaw", "config.json")
}

// Load reads config from the default path, returning defaults if not found.
func Load() (*Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}

// Save writes config to the default path.
func Save(cfg *Config) error {
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// WorkspacePath returns the expanded workspace directory path.
func (c *Config) WorkspacePath() string {
	if c.Workspace == "" {
		c.Workspace = "~/.miniclaw/workspace"
	}
	if c.Workspace[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, c.Workspace[2:])
	}
	return c.Workspace
}

// IsAuthenticated reports whether a valid credential is present.
func (c *Config) IsAuthenticated() bool {
	return c.Provider.AccessToken != "" || c.Provider.APIKey != ""
}
