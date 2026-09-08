package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/er1cw00/claw.go/base"
	wbs "github.com/er1cw00/claw.go/core/tools/web_search"
)

type WebSearchTool struct {
	BaseTool
	ddg *wbs.DuckDuckGo
}

func NewWebSearchTool() *WebSearchTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": { "type": "string", "description": "Search query" }
		},
		"required": ["query"]
	}`)
	proxy := base.GetSettings().Proxy
	return &WebSearchTool{
		BaseTool: BaseTool{
			name:        "web_search",
			description: "Search the web return relevant results",
			parameters:  schema,
		},
		ddg: wbs.NewDuckDuckGo(proxy),
	}

}

func (t *WebSearchTool) Name() string {
	return t.name
}

func (t *WebSearchTool) Description() string {
	return t.description
}

func (t *WebSearchTool) ParametersSchema() json.RawMessage {
	return t.parameters
}

func (t *WebSearchTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.parameters,
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	q, ok := args["query"].(string)
	if !ok || strings.TrimSpace(q) == "" {
		return "", fmt.Errorf("web_search: 'query' argument required")
	}
	return t.ddg.Query(ctx, q)
}

type WebFetchTool struct {
	BaseTool
}

func NewWebFetchTool() *WebFetchTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"url": { "type": "string", "description": "The URL to fetch (must be http or https)" }
		},
		"required": ["url"]
	}`)
	return &WebFetchTool{
		BaseTool: BaseTool{
			name:        "web_fetch",
			description: "Fetch web content from a URL",
			parameters:  schema,
		},
	}

}

func (t *WebFetchTool) Name() string {
	return t.name
}

func (t *WebFetchTool) Description() string {
	return t.description
}

func (t *WebFetchTool) ParametersSchema() json.RawMessage {
	return t.parameters
}

func (t *WebFetchTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.parameters,
	}
}

func (t *WebFetchTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	u, ok := args["url"].(string)
	if !ok || u == "" {
		return "", fmt.Errorf("web: 'url' argument required")
	}
	proxy := base.GetSettings().Proxy
	client := base.NewHttpClient(proxy, 20*time.Second)
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil

}
