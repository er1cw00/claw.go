package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ReadFileTool struct {
	BaseTool
}

func NewReadFileTool() *ReadFileTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": { "type": "string", "description": "The file path to read" }
		},
		"required": ["path"]
	}`)
	return &ReadFileTool{
		BaseTool: *NewBaseTool(
			"read_file",
			"Read the contents of a file at the given path.",
			schema,
		),
	}
}

// Execute executes the file read tool.
func (t *ReadFileTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "path is required", errors.New("path is missing or invalid")
	}

	// Expand path
	path = filepath.Clean(path)
	if state, err := os.Stat(path); err != nil {
		return err.Error(), err
	} else {
		if state.IsDir() {
			return fmt.Sprintf("%s is not a file", path), errors.New("not a file")
		}
	}
	// Read file
	output, err := os.ReadFile(path)
	if err != nil {
		return err.Error(), err
	}

	return string(output), nil
}

// FileWriteTool writes content to a file.
type WriteFileTool struct {
	BaseTool
}

// NewFileWriteTool creates a new FileWriteTool.
func NewWriteFileTool() *WriteFileTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": { "type": "string", "description": "The file path to write to" },
			"content": { "type": "string", "description": "Content to write to the file" },
			"append": { "type": "boolean", "description": "Whether to append to the file instead of overwriting" }
		},
		"required": ["path", "content"]
	}`)
	return &WriteFileTool{
		BaseTool: *NewBaseTool(
			"write_file",
			"Write content to a file at the given path. Creates parent directories if needed",
			schema,
		),
	}
}

// Execute executes the file write tool.
func (t *WriteFileTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	var (
		err     error
		path    string
		content string
		append  bool = false
		ok      bool = false
	)
	if path, ok = args["path"].(string); !ok {
		return "path is required", errors.New("path is missing or invalid")
	}

	if content, ok = args["content"].(string); !ok {
		return "content is required", errors.New("content is missing or invalid")
	}

	append, _ = args["append"].(bool)

	// Expand path
	path = filepath.Clean(path)

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return fmt.Sprintf("Error: create directory fail: %v", err), err
	}

	if append {
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err.Error(), err
		}
		defer file.Close()
		_, err = file.WriteString(content)
	} else {
		err = os.WriteFile(path, []byte(content), 0644)
	}

	if err != nil {
		return err.Error(), err
	}
	return fmt.Sprintf("Successfully wrote to %s", path), nil
}

type EditFileTool struct {
	BaseTool
}

func NewEditFileTool() *EditFileTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": { "type": "string", "description": "The file path to edit" },
			"old_text": { "type": "string", "description": "The exact text to find and replace" },
			"new_text": { "type": "string", "description": "The text to replace with" }
		},
		"required": ["path", "old_text", "new_text"]
	}`)
	return &EditFileTool{
		BaseTool: *NewBaseTool(
			"edit_file",
			"Edit a file by replacing old_text with new_text. The old_text must exist exactly in the file.",
			schema,
		),
	}
}

// Execute executes the file edit tool.
func (t *EditFileTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	var (
		err     error = nil
		path    string
		oldText string
		newText string
		ok      bool = false
	)
	if path, ok = args["path"].(string); !ok {
		return "path is required", errors.New("path is missing or invalid")
	}

	if oldText, ok = args["old_text"].(string); !ok {
		return "old_text is required", errors.New("old_text is missing or invalid")
	}

	if newText, ok = args["new_text"].(string); !ok {
		return "path is required", errors.New("new_text is missing or invalid")
	}

	// Expand path
	path = filepath.Clean(path)

	var content []byte
	if content, err = os.ReadFile(path); err != nil {
		return err.Error(), err
	}

	// Replace content
	var newContent string
	if newContent, err = replaceEditContent(string(content), oldText, newText); err != nil {
		return err.Error(), nil
	}

	// Write file
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return err.Error(), err
	}

	return fmt.Sprintf("Successfully edited %s", path), nil
}

func replaceEditContent(content, oldText, newText string) (string, error) {

	if !strings.Contains(content, oldText) {
		return "", fmt.Errorf("old_text not found in file. Make sure it matches exactly")
	}

	count := strings.Count(content, oldText)
	if count > 1 {
		return "", fmt.Errorf("old_text appears %d times. Please provide more context to make it unique", count)
	}

	newContent := strings.Replace(content, oldText, newText, 1)
	return newContent, nil
}
