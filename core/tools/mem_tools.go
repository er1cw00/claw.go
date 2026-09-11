package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/er1cw00/claw.go/core/memory"
)

// resolveMemoryTarget maps a user-friendly target to a memory filename.
// target: "today", "long", or "YYYY-MM-DD".
func resolveMemoryTarget(target string) (string, error) {
	switch target {
	case "today":
		return time.Now().UTC().Format("2006-01-02") + ".md", nil
	case "long":
		return "MEMORY.md", nil
	default:
		if _, err := time.Parse("2006-01-02", target); err == nil {
			return target + ".md", nil
		}
		return "", fmt.Errorf("invalid target %q: use 'today', 'long', or 'YYYY-MM-DD'", target)
	}
}

// heartbeatPhrases contains lowercase substrings that identify heartbeat-status
// log entries. Any write_memory or edit_memory call whose content matches one
// of these is rejected — the heartbeat loop must never pollute memory files.
var heartbeatPhrases = []string{
	"heartbeat check",
	"heartbeat.md",
	"heartbeat log",
	"system status: healthy",
	"no pending tasks",
	"no actions required",
	"periodic tasks",
}

// isHeartbeatContent returns true if content looks like a heartbeat status log.
func isHeartbeatContent(content string) bool {
	l := strings.ToLower(content)
	for _, phrase := range heartbeatPhrases {
		if strings.Contains(l, phrase) {
			return true
		}
	}
	return false
}

// ListMemoryTool lists all files in the agent's memory directory.
type ListMemoryTool struct {
	BaseTool
	mem *memory.MemoryStore
}

func NewListMemoryTool(mem *memory.MemoryStore) *ListMemoryTool {
	return &ListMemoryTool{
		BaseTool: BaseTool{
			name:        "list_memory",
			description: "List all memory files (daily notes and long-term memory)",
			parameters:  nil,
		},
		mem: mem,
	}
}
func (t *ListMemoryTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	files, err := t.mem.ListFiles()
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "No memory files found.", nil
	}
	today := time.Now().UTC().Format("2006-01-02") + ".md"
	var sb strings.Builder
	fmt.Fprintf(&sb, "Memory files (%d):\n", len(files))
	for _, f := range files {
		switch f {
		case "MEMORY.md":
			fmt.Fprintf(&sb, "- %s (long-term)\n", f)
		case today:
			fmt.Fprintf(&sb, "- %s (today)\n", f)
		default:
			fmt.Fprintf(&sb, "- %s\n", f)
		}
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

// ReadMemoryTool reads the contents of a specific memory file.
type ReadMemoryTool struct {
	BaseTool
	mem *memory.MemoryStore
}

func NewReadMemoryTool(mem *memory.MemoryStore) *ReadMemoryTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"target": { "type": "string", "description": "'today' for today's note, 'long' for long-term memory, or a date 'YYYY-MM-DD'" },
		},
		"required": ["target"]
	}`)
	return &ReadMemoryTool{
		BaseTool: BaseTool{
			name:        "read_memory",
			description: "Read the contents of a memory file",
			parameters:  schema,
		},
		mem: mem,
	}
}

func (t *ReadMemoryTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	target, ok := args["target"].(string)
	if !ok || target == "" {
		return "", fmt.Errorf("read_memory: 'target' argument required (today|long|YYYY-MM-DD)")
	}
	name, err := resolveMemoryTarget(target)
	if err != nil {
		return "", err
	}
	content, err := t.mem.ReadFile(name)
	if err != nil {
		return "", err
	}
	if content == "" {
		return fmt.Sprintf("(%s is empty or does not exist)", name), nil
	}
	return content, nil
}

// WriteMemoryTool writes to the agent's memory (today's note or long-term MEMORY.md)
type WriteMemoryTool struct {
	BaseTool
	mem *memory.MemoryStore
}

func NewWriteMemoryTool(mem *memory.MemoryStore) *WriteMemoryTool {
	schema := json.RawMessage(`{
    "type": "object",
		"properties": {
			"target": { "type": "string", "description": "Memory target: 'today' for daily note or 'long' for long-term memory", "enum":  ["today", "long"]},
			"content": { "type": "string", "description": "The content to write or append", },
			"append": { "type": "boolean", "description": "If true, append to existing content; if false, overwrite", "default": true }
		},
		"required": ["target", "content"],
    }`)
	return &WriteMemoryTool{
		BaseTool: BaseTool{
			name:        "write_memory",
			description: "Write or append to memory (today's note or long-term MEMORY.md). NEVER store heartbeat status, health checks, or 'no pending tasks' results.",
			parameters:  schema,
		},
		mem: mem,
	}
}

// Expected args:
// {"target": "today"|"long", "content": "...", "append": true|false }
func (w *WriteMemoryTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	targetI, ok := args["target"]
	if !ok {
		return "", fmt.Errorf("write_memory: 'target' argument required (today|long)")
	}
	target, ok := targetI.(string)
	if !ok {
		return "", fmt.Errorf("write_memory: 'target' must be a string")
	}
	contentI, ok := args["content"]
	if !ok {
		return "", fmt.Errorf("write_memory: 'content' argument required")
	}
	content, ok := contentI.(string)
	if !ok {
		return "", fmt.Errorf("write_memory: 'content' must be a string")
	}
	if isHeartbeatContent(content) {
		return "", fmt.Errorf("write_memory: heartbeat status logs must not be stored in memory — skip this write")
	}
	appendFlag := true
	if a, ok := args["append"]; ok {
		if b, ok := a.(bool); ok {
			appendFlag = b
		}
	}

	switch target {
	case "today":
		if err := w.mem.AppendToday(content); err != nil {
			return "", err
		}
		return "appended to today", nil
	case "long":
		if appendFlag {
			prev, err := w.mem.ReadLongTerm()
			if err != nil {
				return "", err
			}
			new := prev + "\n" + content
			if err := w.mem.WriteLongTerm(new); err != nil {
				return "", err
			}
			return "appended to long-term memory", nil
		}
		if err := w.mem.WriteLongTerm(content); err != nil {
			return "", err
		}
		return "wrote long-term memory", nil
	default:
		return "", fmt.Errorf("write_memory: unknown target '%s'", target)
	}
}

// EditMemoryTool finds and replaces text within a memory file.
type EditMemoryTool struct {
	BaseTool
	mem *memory.MemoryStore
}

func NewEditMemoryTool(mem *memory.MemoryStore) *EditMemoryTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"target": {"type": "string", "description": "'today', 'long', or 'YYYY-MM-DD'", },
			"old_text": {"type": "string", "description": "Exact text to find and replace", },
			"new_text": {"type": "string", "description": "Replacement text (omit or set to empty string to delete the matched text)", },
		},
		"required": ["target", "old_text"],
    }`)
	return &EditMemoryTool{
		BaseTool: BaseTool{
			name:        "edit_memory",
			description: "Find and replace text within a memory file",
			parameters:  schema,
		},
		mem: mem,
	}
}

func (t *EditMemoryTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	target, ok := args["target"].(string)
	if !ok || target == "" {
		return "", fmt.Errorf("edit_memory: 'target' argument required (today|long|YYYY-MM-DD)")
	}
	oldText, ok := args["old_text"].(string)
	if !ok || oldText == "" {
		return "", fmt.Errorf("edit_memory: 'old_text' argument required")
	}
	newText, _ := args["new_text"].(string) // defaults to "" (deletion) if absent
	if isHeartbeatContent(newText) {
		return "", nil // skip sliently
	}

	name, err := resolveMemoryTarget(target)
	if err != nil {
		return "", err
	}
	content, err := t.mem.ReadFile(name)
	if err != nil {
		return "", err
	}
	if !strings.Contains(content, oldText) {
		return "", fmt.Errorf("edit_memory: text not found in %s", name)
	}
	updated := strings.ReplaceAll(content, oldText, newText)
	if err := t.mem.WriteFile(name, updated); err != nil {
		return "", err
	}
	return fmt.Sprintf("edited %s", name), nil
}

type DeleteMemoryTool struct {
	BaseTool
	mem *memory.MemoryStore
}

func NewDeleteMemoryTool(mem *memory.MemoryStore) *DeleteMemoryTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"target": { "type": "string", "description": "Date of the daily note to delete, in 'YYYY-MM-DD' format", },
		},
		"required": ["target"],
    }`)
	return &DeleteMemoryTool{
		BaseTool: BaseTool{
			name:        "delete_memory",
			description: "Delete a daily memory file (YYYY-MM-DD). Long-term memory (MEMORY.md) cannot be deleted this way.",
			parameters:  schema,
		},
		mem: mem,
	}
}

func (t *DeleteMemoryTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	target, ok := args["target"].(string)
	if !ok || target == "" {
		return "", fmt.Errorf("delete_memory: 'target' argument required (YYYY-MM-DD)")
	}
	// Only dated files are accepted — "long" / "today" are rejected here.
	if _, err := time.Parse("2006-01-02", target); err != nil {
		return "", fmt.Errorf("delete_memory: target must be a date in YYYY-MM-DD format, got %q", target)
	}
	if err := t.mem.DeleteFile(target + ".md"); err != nil {
		return "", err
	}
	return fmt.Sprintf("deleted %s.md", target), nil
}
