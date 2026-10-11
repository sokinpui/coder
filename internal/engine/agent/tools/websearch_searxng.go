package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type SearXNGProvider struct{}

func (p *SearXNGProvider) Search(ctx context.Context, client *http.Client, endpoint, apiKey, query string, maxResults int) ([]SearchResult, error) {
	targetEndpoint := endpoint
	if targetEndpoint == "" {
		targetEndpoint = os.Getenv("SEARXNG_URL")
	}
	if targetEndpoint == "" {
		return nil, fmt.Errorf("searxng search requires an endpoint (configure in tools.websearch.endpoint or set SEARXNG_URL)")
	}

	reqURL, err := url.Parse(targetEndpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid searxng endpoint: %w", err)
	}

	q := reqURL.Query()
	q.Set("q", query)
	q.Set("format", "json")
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	key := apiKey
	if key == "" {
		key = os.Getenv("SEARXNG_API_KEY")
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("searxng api returned status %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode searxng response: %w", err)
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
