package tools

import (
	"context"
	"encoding/json"
	"errors"
)

type BaseTool struct {
	name        string
	description string
	parameters  json.RawMessage
}

func NewBaseTool(name, description string, parameters json.RawMessage) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		parameters:  parameters,
	}
}

func (t *BaseTool) Name() string {
	return t.name
}

func (t *BaseTool) Description() string {
	return t.description
}

func (t *BaseTool) ParametersSchema() json.RawMessage {
	return t.parameters
}

func (t *BaseTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.parameters,
	}
}

func (t *BaseTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	return "", errors.New("method not implemented")
}
