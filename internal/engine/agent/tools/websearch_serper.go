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

const defaultSerperEndpoint = "https://google.serper.dev/search"

type SerperProvider struct{}

func (p *SerperProvider) Search(ctx context.Context, client *http.Client, endpoint, apiKey, query string, maxResults int) ([]SearchResult, error) {
	key := apiKey
	if key == "" {
		key = os.Getenv("SERPER_API_KEY")
	}
	if key == "" {
		return nil, fmt.Errorf("serper search requires an API key (configure in tools.websearch.api_key or set SERPER_API_KEY)")
	}

	targetEndpoint := endpoint
	if targetEndpoint == "" {
		targetEndpoint = defaultSerperEndpoint
	}

	payload := map[string]any{
		"q":   query,
		"num": maxResults,
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
	req.Header.Set("X-API-KEY", key)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("serper api returned status %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Organic []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode serper response: %w", err)
	}

	var results []SearchResult
	for _, item := range data.Organic {
		if item.Title == "" || item.Link == "" {
			continue
		}
		results = append(results, SearchResult{
			Title:   strings.TrimSpace(item.Title),
			URL:     strings.TrimSpace(item.Link),
			Snippet: strings.TrimSpace(item.Snippet),
		})
		if maxResults > 0 && len(results) >= maxResults {
			break
		}
	}
	return results, nil
}
