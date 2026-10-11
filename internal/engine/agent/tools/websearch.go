package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	defaultSearchTimeout = 15 * time.Second
	maxSearchResults     = 10
	maxSnippetLength     = 250
)

type SearchResult struct {
	Title   string
	URL     string
	Snippet string
}

type SearchProvider interface {
	Search(ctx context.Context, client *http.Client, endpoint, apiKey, query string, maxResults int) ([]SearchResult, error)
}

type WebSearchTool struct {
	client     *http.Client
	Timeout    time.Duration
	MaxResults int
	Endpoint   string
	Provider   string
	APIKey     string
	provider   SearchProvider
}

var _ ConfigurableTool = (*WebSearchTool)(nil)
var _ TargetExtractorTool = (*WebSearchTool)(nil)
var _ ClonableTool = (*WebSearchTool)(nil)

func NewWebSearchTool(client ...*http.Client) *WebSearchTool {
	httpClient := &http.Client{Timeout: defaultSearchTimeout}
	if len(client) > 0 && client[0] != nil {
		httpClient = client[0]
	}
	provider, _ := resolveProvider("duckduckgo", "")
	return &WebSearchTool{
		client:     httpClient,
		Timeout:    defaultSearchTimeout,
		MaxResults: maxSearchResults,
		Endpoint:   ddgEndpoint,
		Provider:   "duckduckgo",
		provider:   provider,
	}
}

func (t *WebSearchTool) Configure(raw any) error {
	opts, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	if timeout := parseDuration(opts["timeout"]); timeout > 0 {
		t.Timeout = timeout
	}
	if maxRes := parseInt(opts["max_results"]); maxRes > 0 {
		t.MaxResults = maxRes
	}
	if endpoint := parseString(opts["endpoint"]); endpoint != "" {
		t.Endpoint = endpoint
	}
	if provider := parseString(opts["provider"]); provider != "" {
		t.Provider = provider
	}
	if apiKey := parseString(opts["api_key"]); apiKey != "" {
		t.APIKey = apiKey
	} else if apiKey := parseString(opts["apikey"]); apiKey != "" {
		t.APIKey = apiKey
	}

	p, err := resolveProvider(t.Provider, t.Endpoint)
	if err != nil {
		return err
	}
	t.provider = p
	return nil
}

func (t *WebSearchTool) ExtractTarget(arguments string) string {
	return extractJSONField(arguments, "query")
}

func (t *WebSearchTool) Clone() Tool {
	p, _ := resolveProvider(t.Provider, t.Endpoint)
	return &WebSearchTool{
		client:     t.client,
		Timeout:    t.Timeout,
		MaxResults: t.MaxResults,
		Endpoint:   t.Endpoint,
		Provider:   t.Provider,
		APIKey:     t.APIKey,
		provider:   p,
	}
}

func (t *WebSearchTool) Declaration() ToolDeclaration {
	return ToolDeclaration{
		Type:        "function",
		Name:        "websearch",
		Description: "Search the web. Returns top search results with titles, links, and text snippets.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "The search query to look up on the web",
				},
			},
			"required": []string{"query"},
		},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	var params struct {
		Query string `json:"query"`
	}

	if err := json.Unmarshal([]byte(arguments), &params); err != nil || params.Query == "" {
		trimmed := strings.TrimSpace(arguments)
		if !strings.HasPrefix(trimmed, "{") && trimmed != "" {
			params.Query = trimmed
		} else if err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}

	params.Query = strings.TrimSpace(params.Query)
	if params.Query == "" {
		return "", fmt.Errorf("query is required")
	}

	maxRes := t.MaxResults
	if maxRes <= 0 {
		maxRes = maxSearchResults
	}
	results, err := t.search(ctx, params.Query, maxRes)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	return formatSearchResults(results), nil
}

func (t *WebSearchTool) search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	provider := t.provider
	if provider == nil {
		p, err := resolveProvider(t.Provider, t.Endpoint)
		if err != nil {
			return nil, err
		}
		provider = p
	}

	runCtx := ctx
	timeout := t.Timeout
	if timeout <= 0 {
		timeout = defaultSearchTimeout
	}
	var cancel context.CancelFunc
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		runCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	httpClient := t.client
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return provider.Search(runCtx, httpClient, t.Endpoint, t.APIKey, query, maxResults)
}

func formatSearchResults(results []SearchResult) string {
	if len(results) == 0 {
		return "No results found."
	}
	var sb strings.Builder
	for i, r := range results {
		snippet := r.Snippet
		if len(snippet) > maxSnippetLength {
			snippet = snippet[:maxSnippetLength] + "..."
		}
		sb.WriteString(fmt.Sprintf("%d. [%s](%s)\n   Snippet: %s\n\n", i+1, r.Title, r.URL, snippet))
	}
	return strings.TrimSpace(sb.String())
}

func init() {
	DefaultRegistry.Register(NewWebSearchTool())
}
