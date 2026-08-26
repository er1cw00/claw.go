package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/er1cw00/claw.go/core/provider"
)

var _tools map[string]Tool
var _specs []*provider.ToolFunction

func Init() {
	list := []Tool{
		NewBashTool(),
	}

	toolSpecs := make([]*provider.ToolFunction, len(list))
	toolMap := make(map[string]Tool, 0)

	for i, tool := range list {
		spec := tool.Spec()
		toolSpecs[i] = &provider.ToolFunction{
			Name:        spec.Name,
			Description: spec.Description,
			Parameters:  spec.Parameters,
		}
		toolMap[tool.Name()] = tool
	}
	_specs = toolSpecs
	_tools = toolMap
}

func GetTool(name string) (Tool, bool) {
	tool, ok := _tools[name]
	return tool, ok
}

func GetToolSpecs() []*provider.ToolFunction {
	return _specs
}

func ExecuteToolCall(ctx context.Context, name, arguments string) (string, error) {
	var (
		err  error = nil
		ok   bool  = false
		tool Tool  = nil
		args map[string]interface{}
	)
	tool, ok = _tools[name]
	if !ok {
		return fmt.Sprintf("tool %q not found", name), err
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return fmt.Sprintf("failed to parse arguments: %v", err), err
	}
	content, err := tool.Execute(ctx, args)
	if err != nil {
		return fmt.Sprintf("error: %v", err), err
	}
	return content, nil
}
