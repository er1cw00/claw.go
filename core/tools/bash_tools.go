package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
)

type BashTool struct {
	BaseTool
}

func NewBashTool() *BashTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": { "type": "string", "description": "bash command to execute" }
		},
		"required": ["command"]
	}`)
	return &BashTool{
		BaseTool: BaseTool{
			name:        "bash",
			description: "execute bash",
			parameters:  schema,
		},
	}

}

func (t *BashTool) Name() string {
	return t.name
}

func (t *BashTool) Description() string {
	return t.description
}

func (t *BashTool) ParametersSchema() json.RawMessage {
	return t.parameters
}

func (t *BashTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.parameters,
	}
}

func (t *BashTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	command, ok := args["command"].(string)
	if !ok {
		return "command is required", errors.New("command parameter is missing or invalid")
	}
	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", command)
	output, err := cmd.CombinedOutput()

	if err != nil {
		if len(output) == 0 {
			return fmt.Sprintf("Command failed: %s", err.Error()), err
		}
	}
	return string(output), nil
}
