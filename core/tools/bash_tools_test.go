package tools

import (
	"context"
	"testing"
)

func TestBashTool_ExecuteComplexCommand(t *testing.T) {
	ctx := context.Background()
	tool := NewBashTool()

	command := "cat /Users/wadahana/workspace/AI/stock.ai/claw.go/data/memory/MEMORY.md 2>/dev/null; echo \"---HISTORY---\"; tail -50 /Users/wadahana/workspace/AI/stock.ai/claw.go/data/memory/HISTORY.md 2>/dev/null"

	out, err := tool.Execute(ctx, map[string]interface{}{"command": command})
	t.Logf("tool output:\n%s", out)
	t.Logf("tool error: %v", err)

	if out == "" {
		t.Log("tool returned empty output (files may be missing)")
	}
}
