package tools

import (
	"fmt"
	"strings"
)

func resolveProvider(providerName, endpoint string) (SearchProvider, error) {
	name := strings.ToLower(strings.TrimSpace(providerName))
	if name == "" && endpoint != "" {
		lowerEndpoint := strings.ToLower(endpoint)
		switch {
		case strings.Contains(lowerEndpoint, "brave"):
			name = "brave"
		case strings.Contains(lowerEndpoint, "tavily"):
			name = "tavily"
		case strings.Contains(lowerEndpoint, "serper"):
			name = "serper"
		case strings.Contains(lowerEndpoint, "searx"):
			name = "searxng"
		case strings.Contains(lowerEndpoint, "duckduckgo"):
			name = "duckduckgo"
		}
	}

	switch name {
	case "brave":
		return &BraveProvider{}, nil
	case "tavily":
		return &TavilyProvider{}, nil
	case "serper", "google":
		return &SerperProvider{}, nil
	case "searxng", "searx":
		return &SearXNGProvider{}, nil
	case "duckduckgo", "ddg", "":
		return &DuckDuckGoProvider{}, nil
	default:
		return nil, fmt.Errorf("unsupported search provider '%s', supported: duckduckgo, brave, tavily, serper, searxng", providerName)
	}
}
