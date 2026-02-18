// MIT License - Copyright (c) 2026 yosebyte
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yosebyte/miniclaw/internal/provider"
)

// WebFetchTool fetches a URL and returns its body.
type WebFetchTool struct {
	client *http.Client
}

// NewWebFetchTool creates a WebFetchTool with a sensible timeout.
func NewWebFetchTool() WebFetchTool {
	return WebFetchTool{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (w WebFetchTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "web_fetch",
		Description: "Fetch the content of a URL and return the response body (plain text). Useful for reading documentation, web pages, or API responses.",
		InputSchema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "url": {"type": "string", "description": "URL to fetch."}
  },
  "required": ["url"]
}`),
	}
}

func (w WebFetchTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	if args.URL == "" {
		return "", fmt.Errorf("url is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, args.URL, nil)
	if err != nil {
		return "", fmt.Errorf("web_fetch: %w", err)
	}
	req.Header.Set("User-Agent", "miniclaw/1.0 (+https://github.com/yosebyte/miniclaw)")

	res, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("web_fetch: %w", err)
	}
	defer res.Body.Close()

	const maxBytes = 512 * 1024 // 512 KB
	body, err := io.ReadAll(io.LimitReader(res.Body, maxBytes))
	if err != nil {
		return "", fmt.Errorf("web_fetch read: %w", err)
	}
	result := fmt.Sprintf("HTTP %d\n\n%s", res.StatusCode, body)
	if len(body) == maxBytes {
		result += "\n[response truncated at 512 KB]"
	}
	return result, nil
}
