package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	//	"github.com/er1cw00/claw.go/base/logger"
	"github.com/er1cw00/claw.go/base/logger"
	"github.com/er1cw00/claw.go/core/memory"
	"github.com/er1cw00/claw.go/core/prompts"
	"github.com/er1cw00/claw.go/core/provider"
)

// BootstrapFiles are loaded from the workspace and included in the system prompt.
var BootstrapFiles = []string{"AGENTS.md", "SOUL.md", "USER.md", "TOOLS.md", "IDENTITY.md"}

// ContextBuilder assembles the agent's system prompt and message list.
type ContextBuilder struct {
	workspace string
	memory    *memory.MemoryStore
	skills    *SkillsLoader
}

// NewContextBuilder creates a ContextBuilder for the given workspace.
func NewContextBuilder(workspace string) *ContextBuilder {
	return &ContextBuilder{
		workspace: workspace,
		memory:    memory.NewMemoryStore(workspace),
		skills:    NewSkillsLoader(workspace, []string{}),
	}
}

// BuildSystemPrompt builds the system prompt from bootstrap files, memory, and skills.
func (b *ContextBuilder) BuildSystemPrompt(channel, chatId string, skillNames []string) string {
	var parts []string

	//parts = append(parts, b.getIdentity(channel, chatID))
	workspacePath, _ := filepath.Abs(b.workspace)
	parts = append(parts, prompts.IdentifyPrompt(channel, chatId, workspacePath))
	//logger.Debugf("identify: %s", parts[0])

	bootstrap := b.loadBootstrapFiles()
	if bootstrap != "" {
		parts = append(parts, bootstrap)
	}

	parts = append(parts, "If you decide something should be remembered, call the tool 'write_memory' with JSON arguments: {\"target\": \"today\"|\"long\", \"content\": \"...\", \"append\": true|false}. Use a tool call rather than plain chat text when writing memory.")

	memory := b.memory.GetMemoryContext()
	if memory != "" {
		parts = append(parts, fmt.Sprintf("# Memory\n\n%s", memory))
	}

	alwaysSkills := b.skills.GetAlwaysSkills()
	if len(alwaysSkills) > 0 {
		alwaysContent := b.skills.LoadSkillsForContext(alwaysSkills)
		if alwaysContent != "" {
			parts = append(parts, fmt.Sprintf("# Active Skills\n\n%s", alwaysContent))
		}
	}

	skillsSummary := b.skills.BuildSkillsSummary(nil)
	if skillsSummary != "" {
		parts = append(parts, fmt.Sprintf(`# Skills

The following skills extend your capabilities. To use a skill, read its SKILL.md file using the read_file tool.
Skills with available="false" need dependencies installed first - you can try installing them with apt/brew.

%s`, skillsSummary))
	}

	return strings.Join(parts, "\n\n---\n\n")
}

// getIdentity returns the core identity section including current time and workspace info.
func (b *ContextBuilder) getIdentity(channel, chatId string) string {
	//now := time.Now().Format("2006-01-02 15:04 (Monday)")
	now := time.Now().Format(time.RFC1123)

	workspacePath, _ := filepath.Abs(b.workspace)

	return fmt.Sprintf(`# claw.go 🦧

You are clawbot, a helpful AI assistant. You are operating on channel=%q chatID=%q. You have access to tools that allow you to:
- Read, write, and edit files
- Execute shell commands
- Search the web and fetch web pages
- Send messages to users on chat channels

## Current Time
%s

## Workspace
Your workspace is at: %s
- Memory files: %s/memory/MEMORY.md
- History log: %s/memory/HISTORY.md (grep-searchable)
- Custom skills: %s/skills/{skill-name}/SKILL.md

IMPORTANT: When responding to direct questions or conversations, reply directly with your text response.
Only use the 'message' tool when you need to send a message to a specific chat channel (like WhatsApp).
For normal conversation, just respond with text - do not call the message tool.

Always be helpful, accurate, and concise. When using tools, explain what you're doing.
When remembering something, write to %s/memory/MEMORY.md
To recall past events, grep %s/memory/HISTORY.md`, now, channel, chatId, workspacePath, workspacePath, workspacePath, workspacePath, workspacePath, workspacePath)
}

// loadBootstrapFiles loads all bootstrap files from the workspace.
func (b *ContextBuilder) loadBootstrapFiles() string {
	var parts []string
	for _, filename := range BootstrapFiles {
		filePath := filepath.Join(b.workspace, filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("## %s\n\n%s", filename, string(content)))
	}
	return strings.Join(parts, "\n\n")
}

// BuildMessages builds the complete message list for an LLM call.
func (b *ContextBuilder) BuildMessages(history []provider.Message, channel, chatId, currentMessage string, skillNames []string) []provider.Message {
	messages := make([]provider.Message, 0, len(history)+2)
	systemPrompt := b.BuildSystemPrompt(channel, chatId, skillNames)
	messages = append(messages, provider.Message{
		Role:    provider.RoleSystem,
		Content: systemPrompt, //b.BuildSystemPrompt(skillNames),
	})
	logger.Debugf("system prompt: %s", systemPrompt)
	messages = append(messages, history...)
	messages = append(messages, provider.Message{
		Role:    provider.RoleUser,
		Content: currentMessage,
	})
	return messages
}

// AddToolResult appends a tool result message.
func (b *ContextBuilder) AddToolResult(messages []provider.Message, toolCallID, toolName, result string) []provider.Message {
	return append(messages, provider.Message{
		Role:       provider.RoleTool,
		ToolCallID: toolCallID,
		Name:       toolName,
		Content:    result,
	})
}

// AddAssistantMessage appends an assistant message, optionally with tool calls.
func (b *ContextBuilder) AddAssistantMessage(messages []provider.Message, content string, toolCalls []provider.ToolCall) []provider.Message {
	return append(messages, provider.Message{
		Role:      provider.RoleAssistant,
		Content:   content,
		ToolCalls: toolCalls,
	})
}
