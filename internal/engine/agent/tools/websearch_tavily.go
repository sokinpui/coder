package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const defaultTavilyEndpoint = "https://api.tavily.com/search"

type TavilyProvider struct{}

func (p *TavilyProvider) Search(ctx context.Context, client *http.Client, endpoint, apiKey, query string, maxResults int) ([]SearchResult, error) {
	key := apiKey
	if key == "" {
		key = os.Getenv("TAVILY_API_KEY")
	}
	if key == "" {
		return nil, fmt.Errorf("tavily search requires an API key (configure in tools.websearch.api_key or set TAVILY_API_KEY)")
	}

	targetEndpoint := endpoint
	if targetEndpoint == "" {
		targetEndpoint = defaultTavilyEndpoint
	}

	payload := map[string]any{
		"query":       query,
		"max_results": maxResults,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("tavily api returned status %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode tavily response: %w", err)
	}

	var results []SearchResult
	for _, item := range data.Results {
		if item.Title == "" || item.URL == "" {
			continue
		}
		results = append(results, SearchResult{
			Title:   strings.TrimSpace(item.Title),
			URL:     strings.TrimSpace(item.URL),
			Snippet: strings.TrimSpace(item.Content),
		})
		if maxResults > 0 && len(results) >= maxResults {
			break
		}
	}
	return results, nil
}
