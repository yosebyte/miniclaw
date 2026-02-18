// MIT License - Copyright (c) 2026 yosebyte
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yosebyte/miniclaw/internal/provider"
)

// ---- read_file ----

type ReadFileTool struct{}

func (ReadFileTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "read_file",
		Description: "Read the full contents of a file from the filesystem.",
		InputSchema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "path": {"type": "string", "description": "File path to read."}
  },
  "required": ["path"]
}`),
	}
}

func (ReadFileTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	path := expandHome(args.Path)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read_file: %w", err)
	}
	return string(data), nil
}

// ---- write_file ----

type WriteFileTool struct{}

func (WriteFileTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "write_file",
		Description: "Write content to a file, creating it and any parent directories as needed.",
		InputSchema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "path":    {"type": "string", "description": "File path to write."},
    "content": {"type": "string", "description": "Content to write."}
  },
  "required": ["path", "content"]
}`),
	}
}

func (WriteFileTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	path := expandHome(args.Path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("write_file mkdir: %w", err)
	}
	if err := os.WriteFile(path, []byte(args.Content), 0644); err != nil {
		return "", fmt.Errorf("write_file: %w", err)
	}
	return fmt.Sprintf("Written %d bytes to %s", len(args.Content), args.Path), nil
}

// ---- edit_file ----

type EditFileTool struct{}

func (EditFileTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "edit_file",
		Description: "Replace the first occurrence of old_text with new_text in a file.",
		InputSchema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "path":     {"type": "string", "description": "File path to edit."},
    "old_text": {"type": "string", "description": "Exact text to find."},
    "new_text": {"type": "string", "description": "Replacement text."}
  },
  "required": ["path", "old_text", "new_text"]
}`),
	}
}

func (EditFileTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var args struct {
		Path    string `json:"path"`
		OldText string `json:"old_text"`
		NewText string `json:"new_text"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	path := expandHome(args.Path)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("edit_file read: %w", err)
	}
	content := string(data)
	if !strings.Contains(content, args.OldText) {
		return "", fmt.Errorf("edit_file: old_text not found in %s", args.Path)
	}
	updated := strings.Replace(content, args.OldText, args.NewText, 1)
	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		return "", fmt.Errorf("edit_file write: %w", err)
	}
	return "File edited successfully.", nil
}

// ---- list_dir ----

type ListDirTool struct{}

func (ListDirTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "list_dir",
		Description: "List files and directories in a given path.",
		InputSchema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "path": {"type": "string", "description": "Directory path to list."}
  },
  "required": ["path"]
}`),
	}
}

func (ListDirTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	path := expandHome(args.Path)
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("list_dir: %w", err)
	}
	var lines []string
	for _, e := range entries {
		kind := "file"
		if e.IsDir() {
			kind = "dir"
		}
		lines = append(lines, fmt.Sprintf("[%s] %s", kind, e.Name()))
	}
	if len(lines) == 0 {
		return "(empty directory)", nil
	}
	return strings.Join(lines, "\n"), nil
}

// expandHome expands a leading ~ in the path.
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
