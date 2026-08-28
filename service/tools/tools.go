package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/er1cw00/claw.go/base/logger"
	"github.com/er1cw00/claw.go/core/provider"
)

type Service struct {
	tools map[string]Tool
	specs []*provider.ToolFunction
}

var tcService *Service = &Service{
	tools: make(map[string]Tool, 0),
	specs: make([]*provider.ToolFunction, 0),
}

func GetService() *Service {
	return tcService
}

func (s *Service) Start() error {

	logger.Info("[Tools] Service Start")
	list := []Tool{
		NewBashTool(),
		NewCronTool(),
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
	s.specs = toolSpecs
	s.tools = toolMap
	return nil
}
func (s *Service) Stop() {
	logger.Info("[Tools] Service Stop")
}

func (s *Service) Name() string {
	return "Tools"
}
func (s *Service) GetTool(name string) (Tool, bool) {
	tool, ok := s.tools[name]
	return tool, ok
}

func (s *Service) GetToolSpecs() []*provider.ToolFunction {
	return s.specs
}

func (s *Service) ExecuteToolCall(ctx context.Context, name, arguments string) (string, error) {
	var (
		err  error = nil
		ok   bool  = false
		tool Tool  = nil
		args map[string]interface{}
	)
	tool, ok = s.tools[name]
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
