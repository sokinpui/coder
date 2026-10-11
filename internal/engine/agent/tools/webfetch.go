package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sokinpui/coder/internal/config"
	agentprompt "github.com/sokinpui/coder/internal/engine/agent/prompt"
	"github.com/sokinpui/coder/internal/engine/generation"
	"github.com/sokinpui/coder/internal/types"
	"golang.org/x/net/html"
)

const (
	defaultFetchTimeout = 30 * time.Second
	maxFetchSizeBytes   = 1 * 1024 * 1024
	defaultMaxLines     = 1000
	defaultUserAgent    = "Mozilla/5.0 (compatible; CoderBot/1.0; +https://github.com/sokinpui/coder)"
)

type WebFetchTool struct {
	generator       *generation.Generator
	config          *config.ModelConfig
	client          *http.Client
	Timeout         time.Duration
	MaxSizeBytes    int64
	DefaultMaxLines int
	UserAgent       string
}

var _ ConfigurableTool = (*WebFetchTool)(nil)
var _ TargetExtractorTool = (*WebFetchTool)(nil)
var _ GeneratorConsumer = (*WebFetchTool)(nil)
var _ ClonableTool = (*WebFetchTool)(nil)

func NewWebFetchTool(generator *generation.Generator, cfg *config.ModelConfig, client ...*http.Client) *WebFetchTool {
	httpClient := &http.Client{Timeout: defaultFetchTimeout}
	if len(client) > 0 && client[0] != nil {
		httpClient = client[0]
	}
	return &WebFetchTool{
		generator:       generator,
		config:          cfg,
		client:          httpClient,
		Timeout:         defaultFetchTimeout,
		MaxSizeBytes:    maxFetchSizeBytes,
		DefaultMaxLines: defaultMaxLines,
		UserAgent:       defaultUserAgent,
	}
}

func (t *WebFetchTool) SetGenerator(generator *generation.Generator, cfg *config.ModelConfig) {
	t.generator = generator
	t.config = cfg
}

func (t *WebFetchTool) Configure(raw any) error {
	opts, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	if timeout := parseDuration(opts["timeout"]); timeout > 0 {
		t.Timeout = timeout
	}
	if maxBytes := parseInt64(opts["max_size_bytes"]); maxBytes > 0 {
		t.MaxSizeBytes = maxBytes
	} else if maxMb := parseInt64(opts["max_size_mb"]); maxMb > 0 {
		t.MaxSizeBytes = maxMb * 1024 * 1024
	}
	if maxLines := parseInt(opts["default_max_lines"]); maxLines > 0 {
		t.DefaultMaxLines = maxLines
	} else if maxLines := parseInt(opts["max_lines"]); maxLines > 0 {
		t.DefaultMaxLines = maxLines
	}
	if ua := parseString(opts["user_agent"]); ua != "" {
		t.UserAgent = ua
	}
	return nil
}

func (t *WebFetchTool) ExtractTarget(arguments string) string {
	return extractJSONField(arguments, "url")
}

func (t *WebFetchTool) Clone() Tool {
	return &WebFetchTool{
		generator:       t.generator,
		config:          t.config,
		client:          t.client,
		Timeout:         t.Timeout,
		MaxSizeBytes:    t.MaxSizeBytes,
		DefaultMaxLines: t.DefaultMaxLines,
		UserAgent:       t.UserAgent,
	}
}

func (t *WebFetchTool) Declaration() ToolDeclaration {
	return ToolDeclaration{
		Type:        "function",
		Name:        "webfetch",
		Description: "Fetch content from a URL and ask another AI to summarize for you.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "The URL of the webpage to fetch",
				},
				"prompt": map[string]any{
					"type":        "string",
					"description": "The question, instruction, or topic to extract from the webpage",
				},
				"start_line": map[string]any{
					"type":        "integer",
					"description": "Optional starting line number (1-based, inclusive). Defaults to 1.",
				},
				"end_line": map[string]any{
					"type":        "integer",
					"description": "Optional ending line number (1-based, inclusive). Defaults to start_line + 999.",
				},
			},
			"required": []string{"url", "prompt"},
		},
	}
}

func (t *WebFetchTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	var params struct {
		URL       string `json:"url"`
		Prompt    string `json:"prompt"`
		StartLine *int   `json:"start_line"`
		EndLine   *int   `json:"end_line"`
	}

	if err := json.Unmarshal([]byte(arguments), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	params.URL = strings.TrimSpace(params.URL)
	params.Prompt = strings.TrimSpace(params.Prompt)

	if params.URL == "" {
		return "", fmt.Errorf("url is required")
	}
	if params.Prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}

	if !strings.HasPrefix(params.URL, "http://") && !strings.HasPrefix(params.URL, "https://") {
		return "", fmt.Errorf("invalid url: must start with http:// or https://")
	}

	markdownContent, err := t.fetchContentAsMarkdown(ctx, params.URL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch url: %w", err)
	}

	if strings.TrimSpace(markdownContent) == "" {
		return "Failed to extract readable content from URL.", nil
	}

	lines := strings.Split(markdownContent, "\n")
	totalLines := len(lines)

	start := 1
	if params.StartLine != nil && *params.StartLine > 0 {
		start = *params.StartLine
	}

	maxLines := t.DefaultMaxLines
	if maxLines <= 0 {
		maxLines = defaultMaxLines
	}
	end := start + maxLines - 1
	if params.EndLine != nil && *params.EndLine > 0 {
		end = *params.EndLine
	}

	if start > totalLines {
		return fmt.Sprintf("[webfetch: total_lines=%d, requested lines %d-%d out of range]", totalLines, start, end), nil
	}

	if end > totalLines {
		end = totalLines
	}

	if end < start {
		end = start
	}

	slicedLines := lines[start-1 : end]
	boundedContent := strings.Join(slicedLines, "\n")

	metaLine := fmt.Sprintf("[webfetch: total_lines=%d, showing lines %d-%d]\n", totalLines, start, end)
	notice := ""
	if end < totalLines {
		notice = fmt.Sprintf("[Note: Content has %d lines. Showing lines %d-%d. Use start_line and end_line parameters to read further.]\n\n", totalLines, start, end)
	}

	if t.generator == nil {
		return metaLine + notice + boundedContent, nil
	}

	extracted, err := t.extract(ctx, params.Prompt, boundedContent)
	if err != nil {
		return "", err
	}
	if notice != "" {
		return metaLine + notice + extracted, nil
	}
	return metaLine + extracted, nil
}

func (t *WebFetchTool) extract(ctx context.Context, userPrompt, content string) (string, error) {
	promptText := strings.Replace(agentprompt.WebFetchExtractPrompt, "{{PROMPT}}", userPrompt, 1)
	promptText = strings.Replace(promptText, "{{CONTENT}}", content, 1)

	chatMsgs := []types.ChatMessage{
		{
			Role:    types.RoleUser,
			Content: promptText,
		},
	}

	genChan := make(chan types.StreamChunk, 100)
	go t.generator.GenerateTask(ctx, "", chatMsgs, nil, genChan, t.config)

	var sb strings.Builder
	for chunk := range genChan {
		if chunk.Error != nil {
			return "", chunk.Error
		}
		sb.WriteString(chunk.Content)
	}

	result := strings.TrimSpace(sb.String())
	if result == "" {
		return "The requested information was not found in the document.", nil
	}
	return result, nil
}

func (t *WebFetchTool) fetchContentAsMarkdown(ctx context.Context, targetURL string) (string, error) {
	runCtx := ctx
	timeout := t.Timeout
	if timeout <= 0 {
		timeout = defaultFetchTimeout
	}
	var cancel context.CancelFunc
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		runCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(runCtx, http.MethodGet, targetURL, nil)
	if err != nil {
		return "", err
	}

	ua := t.UserAgent
	if ua == "" {
		ua = defaultUserAgent
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain;q=0.9,*/*;q=0.8")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("server responded with status code %d", resp.StatusCode)
	}

	maxBytes := t.MaxSizeBytes
	if maxBytes <= 0 {
		maxBytes = maxFetchSizeBytes
	}
	limitedReader := io.LimitReader(resp.Body, maxBytes)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", err
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if !strings.Contains(contentType, "text/html") && !strings.Contains(contentType, "application/xhtml+xml") {
		return string(bodyBytes), nil
	}

	return htmlToMarkdown(string(bodyBytes))
}

func htmlToMarkdown(rawHTML string) (string, error) {
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return "", err
	}

	cleanNode(doc)

	var sb strings.Builder
	renderNodeToMarkdown(doc, &sb)

	lines := strings.Split(sb.String(), "\n")
	var cleanedLines []string
	consecutiveBlank := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			consecutiveBlank++
			if consecutiveBlank <= 1 {
				cleanedLines = append(cleanedLines, "")
			}
			continue
		}
		consecutiveBlank = 0
		cleanedLines = append(cleanedLines, line)
	}

	return strings.TrimSpace(strings.Join(cleanedLines, "\n")), nil
}

func cleanNode(n *html.Node) {
	var toRemove []*html.Node

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			tag := strings.ToLower(node.Data)
			switch tag {
			case "script", "style", "noscript", "svg", "iframe", "header", "footer", "nav", "aside":
				toRemove = append(toRemove, node)
				return
			}
		}

		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(n)

	for _, node := range toRemove {
		if node.Parent != nil {
			node.Parent.RemoveChild(node)
		}
	}
}

func renderNodeToMarkdown(n *html.Node, sb *strings.Builder) {
	switch n.Type {
	case html.TextNode:
		text := strings.TrimSpace(n.Data)
		if text != "" {
			sb.WriteString(n.Data)
		}
		return
	case html.ElementNode:
		tag := strings.ToLower(n.Data)
		switch tag {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			level := int(tag[1] - '0')
			sb.WriteString("\n\n" + strings.Repeat("#", level) + " ")
			renderChildren(n, sb)
			sb.WriteString("\n\n")
			return
		case "p", "div", "section", "article":
			sb.WriteString("\n\n")
			renderChildren(n, sb)
			sb.WriteString("\n\n")
			return
		case "br":
			sb.WriteString("\n")
			return
		case "hr":
			sb.WriteString("\n\n---\n\n")
			return
		case "li":
			sb.WriteString("\n- ")
			renderChildren(n, sb)
			return
		case "blockquote":
			sb.WriteString("\n\n> ")
			renderChildren(n, sb)
			sb.WriteString("\n\n")
			return
		case "pre":
			sb.WriteString("\n\n```\n")
			renderChildren(n, sb)
			sb.WriteString("\n```\n\n")
			return
		case "code":
			if n.Parent != nil && strings.ToLower(n.Parent.Data) == "pre" {
				renderChildren(n, sb)
				return
			}
			sb.WriteString("`")
			renderChildren(n, sb)
			sb.WriteString("`")
			return
		case "strong", "b":
			sb.WriteString("**")
			renderChildren(n, sb)
			sb.WriteString("**")
			return
		case "em", "i":
			sb.WriteString("*")
			renderChildren(n, sb)
			sb.WriteString("*")
			return
		case "a":
			href := getAttr(n, "href")
			if href != "" && !strings.HasPrefix(href, "javascript:") {
				sb.WriteString("[")
				renderChildren(n, sb)
				sb.WriteString("](")
				sb.WriteString(href)
				sb.WriteString(")")
				return
			}
		}
	}

	renderChildren(n, sb)
}

func renderChildren(n *html.Node, sb *strings.Builder) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		renderNodeToMarkdown(c, sb)
	}
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val
		}
	}
	return ""
}

func init() {
	DefaultRegistry.Register(NewWebFetchTool(nil, nil))
}
