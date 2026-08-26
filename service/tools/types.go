package tools

import (
	"context"
	"encoding/json"
)

type ToolSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type Tool interface {
	// Name returns the name of the tool.
	Name() string

	// Description returns the description of the tool.
	Description() string

	// ParametersSchema returns the JSON schema for the tool's parameters.
	ParametersSchema() json.RawMessage

	// Execute executes the tool with the given arguments.
	Execute(ctx context.Context, args map[string]interface{}) (string, error)

	// Spec returns the tool specification.
	Spec() ToolSpec
}
