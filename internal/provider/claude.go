// MIT License - Copyright (c) 2026 yosebyte
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/yosebyte/miniclaw/internal/config"
)

const anthropicMessagesURL = "https://api.anthropic.com/v1/messages"
const anthropicVersion = "2023-06-01"
const oauthBeta = "oauth-2024-09-20"

// ToolDefinition describes a tool available to the model.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// Message is a single turn in the conversation.
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []ContentBlock
}

// ContentBlock is a typed content item (text or tool_use or tool_result).
type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// ChatRequest is the payload for the Messages API.
type ChatRequest struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	System    string           `json:"system,omitempty"`
	Messages  []Message        `json:"messages"`
	Tools     []ToolDefinition `json:"tools,omitempty"`
}

// ChatResponse is the response from the Messages API.
type ChatResponse struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Role       string         `json:"role"`
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Model      string         `json:"model"`
	Error      *APIError      `json:"error,omitempty"`
}

// APIError is an error returned by the API.
type APIError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Claude calls the Anthropic Messages API.
type Claude struct {
	cfg    *config.Config
	client *http.Client
}

// New creates a new Claude provider.
func New(cfg *config.Config) *Claude {
	return &Claude{
		cfg:    cfg,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

// Chat sends messages to the Claude API and returns the response.
func (c *Claude) Chat(ctx context.Context, system string, messages []Message, tools []ToolDefinition) (*ChatResponse, error) {
	model := c.cfg.Provider.Model
	if model == "" {
		model = "claude-opus-4-5"
	}
	maxTokens := c.cfg.Provider.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8192
	}

	req := ChatRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  messages,
		Tools:     tools,
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		if isUnauthorized(err) && c.cfg.Provider.RefreshToken != "" {
			log.Printf("[INFO] access token expired, refreshing")
			newAccess, newRefresh, rerr := RefreshAccessToken(ctx, c.cfg.Provider.RefreshToken)
			if rerr != nil {
				return nil, fmt.Errorf("token refresh failed: %w", rerr)
			}
			c.cfg.Provider.AccessToken = newAccess
			if newRefresh != "" {
				c.cfg.Provider.RefreshToken = newRefresh
			}
			if serr := config.Save(c.cfg); serr != nil {
				log.Printf("[WARN] could not persist refreshed token: %v", serr)
			}
			return c.doRequest(ctx, req)
		}
		return nil, err
	}
	return resp, nil
}

func (c *Claude) doRequest(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicMessagesURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	if c.cfg.Provider.APIKey != "" {
		httpReq.Header.Set("x-api-key", c.cfg.Provider.APIKey)
	} else if c.cfg.Provider.AccessToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.Provider.AccessToken)
		httpReq.Header.Set("anthropic-beta", oauthBeta)
	} else {
		return nil, fmt.Errorf("no credentials configured; run: miniclaw provider login")
	}

	res, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if res.StatusCode == http.StatusUnauthorized {
		return nil, &unauthorizedError{status: res.StatusCode, body: string(respBody)}
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", res.StatusCode, respBody)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	if chatResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", chatResp.Error.Message)
	}
	return &chatResp, nil
}

type unauthorizedError struct {
	status int
	body   string
}

func (e *unauthorizedError) Error() string {
	return fmt.Sprintf("unauthorized (%d): %s", e.status, e.body)
}

func isUnauthorized(err error) bool {
	_, ok := err.(*unauthorizedError)
	return ok
}
