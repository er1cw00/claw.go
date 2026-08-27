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
	"github.com/er1cw00/claw.go/model"
	bus "github.com/er1cw00/claw.go/service/bus"
	ss "github.com/er1cw00/claw.go/service/session"
	"github.com/er1cw00/claw.go/service/tools"
)

type Agent struct {
	llm            provider.LLMProvider
	running        atomic.Bool
	config         *base.AgentConfig
	sessionKey     string
	name           string
	contextBuilder *ContextBuilder
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
	agent.contextBuilder = NewContextBuilder(base.GetSettings().Workspace)

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

	messages := agent.contextBuilder.BuildMessages(
		session.GetHistory(40),
		inboundMessage.Content,
		nil,
	)

	// Append user message to session history.
	_ = session.AddMessage(provider.Message{
		Role:    provider.RoleUser,
		Content: inboundMessage.Content,
	})

	for iteration := 0; iteration < MaxIteration; iteration++ {

		req = agent.buildRequest(messages)
		if resp, err = agent.llm.Complete(ctx, req); err != nil {
			logger.Errorf("[Agent] llm complete fail; err: %v", err)
			break
		}
		logger.Debugf("resp: %v", resp)
		logger.Debugf("Finish Reason: %s; toolcall: %d", resp.FinishReason, len(resp.ToolCalls))

		assistantMsg := provider.Message{
			Role:      provider.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		_ = session.AddMessage(assistantMsg)
		messages = append(messages, assistantMsg)

		if resp.FinishReason == "stop" {
			logger.Debugf("[Agent] llm complete stop")
			break
		}
		if len(resp.ToolCalls) == 0 {
			logger.Debugf("[Agent] no tool calls, stop")
			break
		}

		// Execute each tool call and append the result as a tool message.
		for _, tc := range resp.ToolCalls {
			toolResult, err := tools.GetService().ExecuteToolCall(ctx, tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				logger.Warnf("[Agent] execute tool(%s) fail, err: %v", tc.Function.Name, err)
			}
			toolMsg := provider.Message{
				Role:       provider.RoleTool,
				Content:    toolResult,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			}
			_ = session.AddMessage(toolMsg)
			messages = append(messages, toolMsg)
		}
	}

	outboundMessage := model.OutboundMessage{
		Channel: inboundMessage.Channel,
		ChatID:  inboundMessage.ChatID,
		Content: resp.Content,
	}
	msgBus.PublishOutbound(outboundMessage)

	if err := ss.GetService().SaveSession(session); err != nil {
		logger.Warnf("[Agent] save session fail; err: %v", err)
	}
	return nil
}

func (agent *Agent) buildRequest(messages []provider.Message) *provider.CompletionRequest {
	specs := tools.GetService().GetToolSpecs()
	tools := make([]provider.Tool, 0)
	for _, toolSpec := range specs {
		tool := provider.Tool{
			Type:     "function",
			Function: *toolSpec,
		}
		tools = append(tools, tool)
	}

	for i, msg := range messages {
		str, _ := json.Marshal(msg)
		logger.Debugf("msg(%d): %s", i, string(str))
	}
	for i, tool := range tools {
		str, _ := json.Marshal(tool)
		logger.Debugf("tool(%d): %s", i, string(str))
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
