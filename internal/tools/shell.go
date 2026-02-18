// MIT License - Copyright (c) 2026 yosebyte
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/yosebyte/miniclaw/internal/provider"
)

// ExecTool runs shell commands.
type ExecTool struct {
	Timeout time.Duration
}

func (e ExecTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "exec",
		Description: "Execute a shell command and return combined stdout+stderr. Timeout: 60 seconds.",
		InputSchema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "command": {"type": "string", "description": "Shell command to execute."},
    "workdir": {"type": "string", "description": "Working directory (optional)."}
  },
  "required": ["command"]
}`),
	}
}

func (e ExecTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		Command string `json:"command"`
		Workdir string `json:"workdir"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}

	timeout := e.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", args.Command)
	if args.Workdir != "" {
		cmd.Dir = expandHome(args.Workdir)
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	output := out.String()

	if ctx.Err() == context.DeadlineExceeded {
		return output + "\n[command timed out]", nil
	}
	if err != nil {
		return fmt.Sprintf("exit code %d:\n%s", cmd.ProcessState.ExitCode(), output), nil
	}
	if output == "" {
		return "(no output)", nil
	}
	return output, nil
}
