package tools

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/er1cw00/claw.go/core/provider"
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
	// Start tool
	Start() error

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

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool, 0),
		// specs: make([]*provider.ToolFunction, 0),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(t Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := t.Start(); err != nil {
		return err
	}
	r.tools[t.Name()] = t
	return nil
}

func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *Registry) GetSpecs() []*provider.ToolFunction {
	r.mu.RLock()
	defer r.mu.RUnlock()

	specs := make([]*provider.ToolFunction, 0)

	for _, tool := range r.tools {
		spec := tool.Spec()
		schema := &provider.ToolFunction{
			Name:        spec.Name,
			Description: spec.Description,
			Parameters:  spec.Parameters,
		}
		specs = append(specs, schema)
	}
	return specs
}

// func (s *Registry) Start() error {

// 	logger.Info("[Tools] Registry Start")
// 	list := []Tool{
// 		NewBashTool(),
// 		NewCronTool(),
// 	}

// 	toolSpecs := make([]*provider.ToolFunction, len(list))
// 	toolMap := make(map[string]Tool, 0)

// 	for i, tool := range list {
// 		spec := tool.Spec()
// 		toolSpecs[i] = &provider.ToolFunction{
// 			Name:        spec.Name,
// 			Description: spec.Description,
// 			Parameters:  spec.Parameters,
// 		}
// 		toolMap[tool.Name()] = tool
// 	}
// 	s.specs = toolSpecs
// 	s.tools = toolMap
// 	return nil
// }
