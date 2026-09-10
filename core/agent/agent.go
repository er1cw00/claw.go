package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/er1cw00/claw.go/base"
	"github.com/er1cw00/claw.go/base/logger"
	"github.com/er1cw00/claw.go/core/cmd"
	"github.com/er1cw00/claw.go/core/provider"
	tl "github.com/er1cw00/claw.go/core/tools"
	"github.com/er1cw00/claw.go/model"
	bus "github.com/er1cw00/claw.go/service/bus"
	ss "github.com/er1cw00/claw.go/service/session"
)

type Agent struct {
	llm            provider.LLMProvider
	running        atomic.Bool
	config         *base.AgentConfig
	tools          *tl.Registry
	commands       *cmd.Registry
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
		tools          *tl.Registry         = nil
		agentConfig                         = agent.config
		providerConfig                      = base.GetProviderConfig(agentConfig.Provider)
	)

	llm, err = provider.NewProvider(agentConfig.Provider, providerConfig.APIKey, providerConfig.APIBase)
	if err != nil {
		logger.Errorf("[Agent] failed to create llm provider: %v", err)
		return err
	}

	if tools, err = agent.RegisterTools(); err != nil {
		logger.Errorf("[Agent] register tools fail, err: %v", err)
		return err
	}

	agent.llm = llm
	agent.tools = tools
	agent.commands = agent.RegisterCommands()
	agent.contextBuilder = NewContextBuilder(base.GetSettings().Workspace)

	return nil
}

func (agent *Agent) RegisterCommands() *cmd.Registry {
	list := []cmd.Command{
		cmd.NewCommandNew(),
		// TODO; register commands here
	}
	reg := cmd.NewRegistry()
	for _, tool := range list {
		reg.Register(tool)
	}
	logger.Infof("[Agent] register commands.")
	return reg
}
func (agent *Agent) RegisterTools() (*tl.Registry, error) {
	var err error = nil

	list := []tl.Tool{
		tl.NewBashTool(),
		tl.NewWebSearchTool(),
		tl.NewWebFetchTool(),
		tl.NewCronTool(),
		tl.NewReadFileTool(),
		tl.NewWriteFileTool(),
		tl.NewEditFileTool(),
		// TODO; register tools here
	}
	reg := tl.NewRegistry()
	for _, tool := range list {
		if err = reg.Register(tool); err != nil {
			logger.Errorf("[Agent] register tool(%s) fail, err: %v", tool.Name(), err)
			return nil, err
		}
	}
	logger.Infof("[Agent] register tools")
	return reg, nil
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
			agent.processInboundMessage(ctx, msg)
			continue
		}
	}
	return nil
}

func (agent *Agent) Stop() {
	agent.running.Store(false)
}
func (agent *Agent) printMessageContent(label, content string) {

	if len(content) > 256 {
		content = content[:256]
	}
	logger.Debugf("%s: %s", label, content)
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
	session := ss.GetService().GetOrCreate(skey)
	logger.Debugf("session key: %s", skey)

	if reply, ret := agent.commands.HandleCommand(inboundMessage.Sender(), inboundMessage.Content, session); ret {
		outboundMessage := model.OutboundMessage{
			Channel: inboundMessage.Channel,
			ChatID:  inboundMessage.ChatID,
			Content: reply,
		}
		msgBus.PublishOutbound(outboundMessage)
		return nil
	}

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
		agent.printMessageContent("last message", messages[len(messages)-1].Content)
		req = agent.buildRequest(messages)
		if agent.llm.Name() == "opencode" {
			req.SessionID = skey //session.SessionID()
		}
		if resp, err = agent.llm.Complete(ctx, req); err != nil {
			logger.Errorf("[Agent] llm complete fail; err: %v", err)
			break
		}
		agent.printMessageContent("llm response", resp.Content)
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
			toolResult, err := agent.executeToolCall(ctx, inboundMessage.Sender(), tc.Function.Name, tc.Function.Arguments)
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

	if err := ss.GetService().Save(session); err != nil {
		logger.Warnf("[Agent] save session fail; err: %v", err)
	}
	return nil
}

func (agent *Agent) buildRequest(messages []provider.Message) *provider.CompletionRequest {
	specs := agent.tools.GetSpecs()
	tools := make([]provider.Tool, 0)
	for _, toolSpec := range specs {
		tool := provider.Tool{
			Type:     "function",
			Function: *toolSpec,
		}
		tools = append(tools, tool)
	}

	// for i, msg := range messages {
	// 	str, _ := json.Marshal(msg)
	// 	logger.Debugf("msg(%d): %s", i, string(str))
	// }
	// for i, tool := range tools {
	// 	str, _ := json.Marshal(tool)
	// 	logger.Debugf("tool(%d): %s", i, string(str))
	// }
	return &provider.CompletionRequest{
		Model:       agent.config.Model,
		MaxTokens:   agent.config.MaxTokens,
		Temperature: agent.config.Temperature,
		Messages:    messages,
		Tools:       tools,
		ToolChoice:  "auto",
	}
}

func (agent *Agent) consolidateMemory(ctx context.Context, session *ss.Session) error {
	messages := session.Messages()
	memoryWindow := 50
	if memoryWindow <= 0 {
		memoryWindow = 20
	}
	keepCount := min(10, max(2, memoryWindow/2))
	if len(messages) <= keepCount {
		return nil
	}
	oldMessages := messages[:len(messages)-keepCount]

	var lines []string
	for _, m := range oldMessages {
		if strings.TrimSpace(m.Content) == "" {
			continue
		}
		var toolsUsed []string
		for _, tc := range m.ToolCalls {
			toolsUsed = append(toolsUsed, tc.Function.Name)
		}
		tools := ""
		if len(toolsUsed) > 0 {
			tools = fmt.Sprintf(" [tools: %s]", strings.Join(toolsUsed, ", "))
		}
		timestamp := "?"
		if m.Timestamp > 0 {
			timestamp = time.Unix(m.Timestamp, 0).Format("2006-01-02 15:04")
		}
		lines = append(lines, fmt.Sprintf("[%s] %s%s: %s", timestamp, strings.ToUpper(string(m.Role)), tools, m.Content))
	}
	if len(lines) == 0 {
		return nil
	}
	conversation := strings.Join(lines, "\n")
	memoryStore := agent.contextBuilder.memory
	currentMemory := memoryStore.ReadLongTerm()

	prompt := fmt.Sprintf(`You are a memory consolidation agent. Process this conversation and return a JSON object with exactly two keys:

1. "history_entry": A paragraph (2-5 sentences) summarizing the key events/decisions/topics. Start with a timestamp like [YYYY-MM-DD HH:MM]. Include enough detail to be useful when found by grep search later.

2. "memory_update": The updated long-term memory content. Add any new facts: user location, preferences, personal info, habits, project context, technical decisions, tools/services used. If nothing new, return the existing content unchanged.

## Current Long-term Memory
%s

## Conversation to Process
%s

Respond with ONLY valid JSON, no markdown fences.`, currentMemoryOrEmpty(currentMemory), conversation)

	systemMsg := provider.Message{
		Role:    provider.RoleSystem,
		Content: "You are a memory consolidation agent. Respond only with valid JSON.",
	}
	userMsg := provider.Message{
		Role:    provider.RoleUser,
		Content: prompt,
	}
	req := &provider.CompletionRequest{
		Model:       agent.config.Model,
		MaxTokens:   agent.config.MaxTokens,
		Temperature: agent.config.Temperature,
		Messages:    []provider.Message{systemMsg, userMsg},
	}

	resp, err := agent.llm.Complete(ctx, req)
	if err != nil {
		return fmt.Errorf("llm complete failed: %w", err)
	}

	text := strings.TrimSpace(resp.Content)
	if strings.HasPrefix(text, "```") {
		parts := strings.SplitN(text, "\n", 2)
		if len(parts) == 2 {
			text = strings.TrimSuffix(parts[1], "```")
			text = strings.TrimSpace(text)
		}
	}

	var result struct {
		HistoryEntry string `json:"history_entry"`
		MemoryUpdate string `json:"memory_update"`
	}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return fmt.Errorf("parse consolidation response failed: %w", err)
	}

	if result.HistoryEntry != "" {
		if err := memoryStore.AppendHistory(result.HistoryEntry); err != nil {
			return fmt.Errorf("append history failed: %w", err)
		}
	}
	if result.MemoryUpdate != "" && result.MemoryUpdate != currentMemory {
		if err := memoryStore.WriteLongTerm(result.MemoryUpdate); err != nil {
			return fmt.Errorf("write long-term memory failed: %w", err)
		}
	}

	newMessages := make([]provider.Message, keepCount)
	copy(newMessages, messages[len(messages)-keepCount:])
	session.SetMessages(newMessages)
	return nil
}

func (agent *Agent) executeToolCall(ctx context.Context, to *model.Participant, name, arguments string) (string, error) {
	var (
		err  error   = nil
		ok   bool    = false
		tool tl.Tool = nil
		args map[string]interface{}
	)
	tool, ok = agent.tools.Get(name)
	if !ok {
		return fmt.Sprintf("tool %q not found", name), err
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return fmt.Sprintf("failed to parse arguments: %v", err), err
	}
	if tool.Name() == "cron" {
		args["channel"] = to.Channel
		args["chat_id"] = to.ChatID
	}
	content, err := tool.Execute(ctx, args)
	if err != nil {
		return fmt.Sprintf("error: %v", err), err
	}
	return content, nil
}

func currentMemoryOrEmpty(content string) string {
	if content == "" {
		return "(empty)"
	}
	return content
}
