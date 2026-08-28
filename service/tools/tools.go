package tools

import (
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
