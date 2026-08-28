// Package tools provides AI tool-calling schemas and a dispatcher for saka.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	saka "github.com/you/saka"
)

// ---- JSON Schema definitions (shared by OpenAI and Anthropic) ----

const searchSchema = `{
  "type": "object",
  "properties": {
    "query": {
      "type": "string",
      "description": "The search query text."
    },
    "max_results": {
      "type": "integer",
      "description": "Maximum number of results to return (default 10).",
      "default": 10
    },
    "site": {
      "type": "string",
      "description": "Restrict results to this domain, e.g. 'arxiv.org'."
    }
  },
  "required": ["query"]
}`

const fetchSchema = `{
  "type": "object",
  "properties": {
    "url": {
      "type": "string",
      "description": "The URL of the page to fetch and extract readable text from."
    }
  },
  "required": ["url"]
}`

func SearchSchema() string { return searchSchema }
func FetchSchema() string  { return fetchSchema }

// ---- OpenAI function-calling format ----

func OpenAISchemas() []map[string]any {
	return []map[string]any{
		{
			"type": "function",
			"function": map[string]any{
				"name":        "web_search",
				"description": "Search the web for free. Returns structured results with title, URL, and snippet.",
				"parameters":  json.RawMessage(searchSchema),
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        "fetch_page",
				"description": "Fetch a web page and return its extracted readable article text.",
				"parameters":  json.RawMessage(fetchSchema),
			},
		},
	}
}

// ---- Anthropic tool-use format ----

func AnthropicSchemas() []map[string]any {
	return []map[string]any{
		{
			"name":         "web_search",
			"description":  "Search the web for free. Returns structured results with title, URL, and snippet.",
			"input_schema": json.RawMessage(searchSchema),
		},
		{
			"name":         "fetch_page",
			"description":  "Fetch a web page and return its extracted readable article text.",
			"input_schema": json.RawMessage(fetchSchema),
		},
	}
}

// ---- Dispatch ----

type searchArgs struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results"`
	Site       string `json:"site"`
}

type fetchArgs struct {
	URL string `json:"url"`
}

// ExecuteTool dispatches a tool call (name + raw JSON args) against the engine
// and returns plain-text output suitable for feeding back to the model.
func ExecuteTool(ctx context.Context, engine saka.Searcher, name string, args json.RawMessage) (string, error) {
	switch name {
	case "web_search":
		var a searchArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return "", fmt.Errorf("web_search: bad args: %w", err)
		}
		res, err := engine.Search(ctx, saka.Query{
			Text:       a.Query,
			MaxResults: a.MaxResults,
			Site:       a.Site,
		})
		if err != nil {
			return "", err
		}
		return renderSearch(res), nil

	case "fetch_page":
		var a fetchArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return "", fmt.Errorf("fetch_page: bad args: %w", err)
		}
		page, err := engine.Fetch(ctx, a.URL)
		if err != nil {
			return "", err
		}
		return renderPage(page), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func renderSearch(r *saka.Results) string {
	var out string
	for _, res := range r.Results {
		out += fmt.Sprintf("%d. %s\n   URL: %s\n   %s\n\n", res.Position, res.Title, res.URL, res.Snippet)
	}
	return out
}

func renderPage(p *saka.Page) string {
	header := p.Title
	if !p.PublishedAt.IsZero() {
		header += " (" + p.PublishedAt.Format("2006-01-02") + ")"
	}
	return header + "\n\n" + p.Text
}
