package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/er1cw00/claw.go/base"
)

// ddgResponse is the top-level DDG Instant Answer API JSON structure.
type ddgResponse struct {
	Heading       string     `json:"Heading"`
	AbstractText  string     `json:"AbstractText"`
	AbstractURL   string     `json:"AbstractURL"`
	Answer        string     `json:"Answer"`
	Definition    string     `json:"Definition"`
	DefinitionURL string     `json:"DefinitionURL"`
	RelatedTopics []ddgTopic `json:"RelatedTopics"`
	Results       []ddgTopic `json:"Results"`
}

// ddgTopic represents either a direct result or a grouped section.
// When grouped (e.g. "See also"), Name is set and Topics contains the items.
type ddgTopic struct {
	Text     string     `json:"Text"`
	FirstURL string     `json:"FirstURL"`
	Name     string     `json:"Name"`
	Topics   []ddgTopic `json:"Topics"`
}

type DuckDuckGo struct {
	client  *http.Client
	baseURL string // overridable in tests
}

func NewDuckDuckGo(proxy string) *DuckDuckGo {
	return &DuckDuckGo{
		client:  base.NewHttpClient(proxy, 20*time.Second),
		baseURL: "https://api.duckduckgo.com",
	}
}

func (c *DuckDuckGo) Query(ctx context.Context, query string) (string, error) {
	apiURL := c.baseURL + "/?q=" + url.QueryEscape(query) +
		"&format=json&no_html=1&skip_disambig=1&t=clawgo"
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "claw.go/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("web_search: DuckDuckGo returned HTTP %d", resp.StatusCode)
	}

	var ddg ddgResponse
	if err := json.NewDecoder(resp.Body).Decode(&ddg); err != nil {
		return "", err
	}

	return c.formatDDGResponse(query, &ddg), nil
}

// formatDDGResponse builds a clean, LLM-friendly text from the DDG API response.
func (c *DuckDuckGo) formatDDGResponse(query string, r *ddgResponse) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "DuckDuckGo search results for %q:\n", query)

	hasContent := false

	if r.Answer != "" {
		fmt.Fprintf(&sb, "\nAnswer: %s\n", r.Answer)
		hasContent = true
	}

	if r.AbstractText != "" {
		if r.Heading != "" {
			fmt.Fprintf(&sb, "\n%s\n", r.Heading)
		}
		fmt.Fprintf(&sb, "%s\n", r.AbstractText)
		if r.AbstractURL != "" {
			fmt.Fprintf(&sb, "Source: %s\n", r.AbstractURL)
		}
		hasContent = true
	}

	if r.Definition != "" {
		fmt.Fprintf(&sb, "\nDefinition: %s\n", r.Definition)
		if r.DefinitionURL != "" {
			fmt.Fprintf(&sb, "Source: %s\n", r.DefinitionURL)
		}
		hasContent = true
	}

	// Flatten related topics: accept direct items and one level of grouped sections.
	var topics []ddgTopic
	for _, rt := range r.RelatedTopics {
		if rt.Text != "" && rt.FirstURL != "" {
			topics = append(topics, rt)
		} else if len(rt.Topics) > 0 {
			for _, sub := range rt.Topics {
				if sub.Text != "" && sub.FirstURL != "" {
					topics = append(topics, sub)
				}
			}
		}
	}
	for _, res := range r.Results {
		if res.Text != "" && res.FirstURL != "" {
			topics = append(topics, res)
		}
	}

	const maxTopics = 5
	if len(topics) > maxTopics {
		topics = topics[:maxTopics]
	}
	if len(topics) > 0 {
		fmt.Fprintf(&sb, "\nRelated results:\n")
		for i, topic := range topics {
			fmt.Fprintf(&sb, "%d. %s\n   %s\n", i+1, topic.Text, topic.FirstURL)
		}
		hasContent = true
	}

	if !hasContent {
		fmt.Fprintf(&sb, "\nNo instant answer found. Try the 'web' tool to visit a specific URL.\n")
	}

	return sb.String()
}
