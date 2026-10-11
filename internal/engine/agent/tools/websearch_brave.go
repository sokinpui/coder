package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const defaultBraveEndpoint = "https://api.search.brave.com/res/v1/web/search"

type BraveProvider struct{}

func (p *BraveProvider) Search(ctx context.Context, client *http.Client, endpoint, apiKey, query string, maxResults int) ([]SearchResult, error) {
	key := apiKey
	if key == "" {
		key = os.Getenv("BRAVE_API_KEY")
	}
	if key == "" {
		return nil, fmt.Errorf("brave search requires an API key (configure in tools.websearch.api_key or set BRAVE_API_KEY)")
	}

	targetEndpoint := endpoint
	if targetEndpoint == "" {
		targetEndpoint = defaultBraveEndpoint
	}

	reqURL, err := url.Parse(targetEndpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid brave endpoint: %w", err)
	}

	q := reqURL.Query()
	q.Set("q", query)
	if maxResults > 0 {
		q.Set("count", strconv.Itoa(min(maxResults, 20)))
	}
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", key)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("brave api returned status %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode brave response: %w", err)
	}

	var results []SearchResult
	for _, item := range data.Web.Results {
		if item.Title == "" || item.URL == "" {
			continue
		}
		results = append(results, SearchResult{
			Title:   strings.TrimSpace(item.Title),
			URL:     strings.TrimSpace(item.URL),
			Snippet: strings.TrimSpace(item.Description),
		})
		if maxResults > 0 && len(results) >= maxResults {
			break
		}
	}
	return results, nil
}
