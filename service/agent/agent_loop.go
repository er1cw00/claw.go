package agent

import (
	"context"
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
	ss "github.com/er1cw00/claw.go/service/session"
)

type Agent struct {
	llm        provider.LLMProvider
	running    atomic.Bool
	config     *base.AgentConfig
	sessionKey string
	name       string
}

func NewAgent() *Agent {
	cfg := &base.GetSettings().Agent
	agent := &Agent{
		llm:    nil,
		config: cfg,
		name:   cfg.Name,
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

	skey := ss.SessionKey(agent.name, inboundMessage.Channel, inboundMessage.ChatID)
	session := ss.GetService().LoadSession(skey)

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
		if len(resp.ToolCalls) == 0 {
			logger.Debugf("[Agent] no tool calls, stop")
			break
		}

		// Append assistant message that requested the tool calls.
		messages = append(messages, provider.Message{
			Role:      provider.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// Execute each tool call and append the result as a tool message.
		for _, tc := range resp.ToolCalls {
			toolResult, err := tools.ExecuteToolCall(ctx, tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				logger.Warnf("[Agent] execute tool(%s) fail, err: %v", tc.Function.Name, err)
			}
			messages = append(messages, provider.Message{
				Role:       provider.RoleTool,
				Content:    toolResult,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			})
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
	specs := tools.GetToolSpecs()
	tools := make([]provider.Tool, 0)
	for _, toolSpec := range specs {
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
