package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

const ddgEndpoint = "https://html.duckduckgo.com/html/"

type DuckDuckGoProvider struct{}

func (p *DuckDuckGoProvider) Search(ctx context.Context, client *http.Client, endpoint, apiKey, query string, maxResults int) ([]SearchResult, error) {
	targetEndpoint := endpoint
	if targetEndpoint == "" {
		targetEndpoint = ddgEndpoint
	}

	data := url.Values{}
	data.Set("q", query)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse html: %w", err)
	}

	return extractResults(doc, maxResults), nil
}

func extractResults(doc *html.Node, maxResults int) []SearchResult {
	resultNodes := findElementsByClass(doc, "web-result")
	var results []SearchResult

	for _, node := range resultNodes {
		if maxResults > 0 && len(results) >= maxResults {
			break
		}

		titleNode := findElementByClass(node, "result__title")
		if titleNode == nil {
			continue
		}

		linkNode := findFirstElement(titleNode, "a")
		if linkNode == nil {
			continue
		}

		title := strings.TrimSpace(getNodeText(linkNode))
		rawHref := getElementAttr(linkNode, "href")
		targetURL := parseActualURL(rawHref)
		if title == "" || targetURL == "" {
			continue
		}

		var snippet string
		if snippetNode := findElementByClass(node, "result__snippet"); snippetNode != nil {
			snippet = strings.TrimSpace(getNodeText(snippetNode))
			if len(snippet) > maxSnippetLength {
				snippet = snippet[:maxSnippetLength] + "..."
			}
		}

		results = append(results, SearchResult{
			Title:   title,
			URL:     targetURL,
			Snippet: snippet,
		})
	}

	return results
}

func parseActualURL(raw string) string {
	if !strings.Contains(raw, "uddg=") {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	uddg := u.Query().Get("uddg")
	if uddg != "" {
		return uddg
	}
	return raw
}

func findElementsByClass(n *html.Node, className string) []*html.Node {
	var matches []*html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && hasClass(node, className) {
			matches = append(matches, node)
			return
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return matches
}

func findElementByClass(n *html.Node, className string) *html.Node {
	if n.Type == html.ElementNode && hasClass(n, className) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if res := findElementByClass(c, className); res != nil {
			return res
		}
	}
	return nil
}

func findFirstElement(n *html.Node, tagName string) *html.Node {
	if n.Type == html.ElementNode && strings.EqualFold(n.Data, tagName) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if res := findFirstElement(c, tagName); res != nil {
			return res
		}
	}
	return nil
}

func hasClass(n *html.Node, className string) bool {
	for _, attr := range n.Attr {
		if strings.EqualFold(attr.Key, "class") {
			for c := range strings.FieldsSeq(attr.Val) {
				if strings.EqualFold(c, className) {
					return true
				}
			}
		}
	}
	return false
}

func getElementAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val
		}
	}
	return ""
}

func getNodeText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}
