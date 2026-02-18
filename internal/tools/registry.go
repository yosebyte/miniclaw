// MIT License - Copyright (c) 2026 yosebyte
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yosebyte/miniclaw/internal/provider"
)

// Tool is implemented by every built-in tool.
type Tool interface {
	Definition() provider.ToolDefinition
	Execute(ctx context.Context, input json.RawMessage) (string, error)
}

// Registry holds all registered tools.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool.
func (r *Registry) Register(t Tool) {
	r.tools[t.Definition().Name] = t
}

// Definitions returns all tool definitions for the API request.
func (r *Registry) Definitions() []provider.ToolDefinition {
	defs := make([]provider.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, t.Definition())
	}
	return defs
}

// Execute runs the named tool with the given JSON input.
func (r *Registry) Execute(ctx context.Context, name string, input json.RawMessage) (string, error) {
	t, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return t.Execute(ctx, input)
}
