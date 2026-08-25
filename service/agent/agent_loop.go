package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/er1cw00/claw.go/base"
	"github.com/er1cw00/claw.go/base/logger"
	"github.com/er1cw00/claw.go/core/provider"
	"github.com/er1cw00/claw.go/core/tools"
	"github.com/er1cw00/claw.go/model"
	bus "github.com/er1cw00/claw.go/service/bus"
)

type Agent struct {
	llm       provider.LLMProvider
	running   atomic.Bool
	config    *base.AgentConfig
	tools     map[string]tools.Tool
	toolSpecs []*provider.ToolFunction
}

func NewAgent() *Agent {
	agent := &Agent{
		config: &base.GetSettings().Agent,
	}
	return agent
}

func (agent *Agent) Start() error {
	var (
		err            error                = nil
		llm            provider.LLMProvider = nil
		agentConfig                         = agent.config
		providerConfig                      = base.GetProviderConfig(agentConfig.Provider)
	)
	llm, err = provider.NewProvider(agentConfig.Provider, providerConfig.APIKey, providerConfig.APIBase)
	if err != nil {
		logger.Errorf("failed to create llm provider: %v", err)
		return err
	}
	agent.llm = llm

	tools := []tools.Tool{
		tools.NewBashTool(),
	}

	toolSpecs := make([]*provider.ToolFunction, len(tools))
	toolMap := make(map[string]tools.Tool, 0)

	for i, tool := range tools {
		spec := tool.Spec()
		toolSpecs[i] = &provider.ToolFunction{
			Name:        spec.Name,
			Description: spec.Description,
			Parameters:  spec.Parameters,
		}
		toolMap[tool.Name()] = tool
	}
	agent.toolSpecs = toolSpecs
	agent.tools = toolMap
	return nil
}

func (agent *Agent) Run(ctx context.Context, wg *sync.WaitGroup) error {
	agent.running.Store(true)

	idleTicker := time.NewTicker(100 * time.Millisecond)
	defer idleTicker.Stop()
	msgBus := bus.GetService().GetMessageBus()
	for {
		select {
		case <-ctx.Done():
			wg.Done()
			return nil
		case <-idleTicker.C:
			if !agent.running.Load() {
				return nil
			}
		case msg, ok := <-msgBus.ConsumeInbound():
			if !ok {
				return nil
			}
			fmt.Printf("msg: %v", msg)
			agent.processInboundMessage(ctx, msg)
			continue
		}
	}
	return nil
}

func (agent *Agent) Stop() {
	agent.running.Store(false)
}

func (agent *Agent) processInboundMessage(ctx context.Context, inboundMessage model.InboundMessage) error {
	var (
		err          error                        = nil
		req          *provider.CompletionRequest  = nil
		resp         *provider.CompletionResponse = nil
		MaxIteration                              = 10
		msgBus                                    = bus.GetService().GetMessageBus()
	)
	logger.Infof("[Agent] process in msg [%s-%s]", inboundMessage.Channel, inboundMessage.ChatID)
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: SYSTEM_PROMPT},
		{Role: provider.RoleUser, Content: inboundMessage.Content},
	}

	for iteration := 0; iteration < MaxIteration; iteration++ {
		req = agent.buildRequest(messages)
		if resp, err = agent.llm.Complete(ctx, req); err != nil {
			logger.Errorf("[Agent] llm complete fail; err: %v", err)
			break
		}
		logger.Debugf("resp: %v", resp)
		logger.Debugf("Finish Reason: %s; toolcall: %d", resp.FinishReason, len(resp.ToolCalls))
		if resp.FinishReason == "stop" {
			logger.Debugf("[Agent] llm complete stop")
			break
		}
		var results []provider.ToolCall
		if len(resp.ToolCalls) > 0 {
			for _, tc := range resp.ToolCalls {
				if tool, ok := agent.tools[tc.Function.Name]; ok {
					var args map[string]interface{}
					var content string = ""
					if err = json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
						logger.Errorf("[Agent] unmarshal toolcall args fail, err: %v", err)
						content = err.Error()
					} else {
						content, err = tool.Execute(ctx, args)
					}
					logger.Debugf("[Agent] Tool(%s) content: %s", tc.Function.Name, content)
				}
			}
		}
	}

	outboundMessage := model.OutboundMessage{
		Channel: inboundMessage.Channel,
		ChatID:  inboundMessage.ChatID,
		Content: resp.Content,
	}
	msgBus.PublishOutbound(outboundMessage)
	return nil
}

func (agent *Agent) buildRequest(messages []provider.Message) *provider.CompletionRequest {

	tools := make([]provider.Tool, 0)
	for _, toolSpec := range agent.toolSpecs {
		tool := provider.Tool{
			Type:     "function",
			Function: *toolSpec,
		}
		tools = append(tools, tool)
	}
	return &provider.CompletionRequest{
		Model:       agent.config.Model,
		MaxTokens:   agent.config.MaxTokens,
		Temperature: agent.config.Temperature,
		Messages:    messages,
		Tools:       tools,
		ToolChoice:  "auto",
	}
}

// func (agent *Agent) runTurn(ctx context.Context, t *AgentTurn) error {
// 	turnCtx, turnCancel := context.WithCancel(ctx)
// 	defer turnCancel()
// 	t.setTurnCancel(turnCancel)

// 	for t.currentIteration() < 10 {
// 		iteration := t.currentIteration() + 1
// 		t.setIteration(iteration)
// 		t.setState(TurnStateRunning)
// 		req
// 		agent.llm.Complete(ctx, buildRequest)
// 	}
// }
