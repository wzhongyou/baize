package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wzhongyou/baize/core/tool"
)

// WebSearchTool performs web searches via DuckDuckGo HTML (no API key needed).
type WebSearchTool struct{}

func (t *WebSearchTool) Name() string        { return "web_search" }
func (t *WebSearchTool) Description() string { return "Search the web using DuckDuckGo. Returns titles, URLs, and snippets." }
func (t *WebSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query.",
			},
		},
		"required": []string{"query"},
	}
}
func (t *WebSearchTool) IsReadOnly() bool                  { return true }
func (t *WebSearchTool) RequiredPermissions() []tool.Permission { return []tool.Permission{tool.PermNetworkOutbound} }
func (t *WebSearchTool) AffectedPaths(map[string]any) []string  { return nil }

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return "", fmt.Errorf("web_search: query is required")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://html.duckduckgo.com/html/?q="+url.QueryEscape(query), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Baize/0.3 (web-search)")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("web_search: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	results := parseDuckDuckGoHTML(string(body))
	if len(results) == 0 {
		return "No results found for: " + query, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for: %s\n\n", query))
	for i, r := range results {
		if i >= 10 {
			break
		}
		sb.WriteString(fmt.Sprintf("%d. **%s**\n   %s\n   %s\n\n", i+1, r.Title, r.URL, r.Snippet))
	}
	return sb.String(), nil
}

type searchResult struct{ Title, URL, Snippet string }

func parseDuckDuckGoHTML(html string) []searchResult {
	var results []searchResult
	for _, raw := range strings.Split(html, "class=\"result__body\"") {
		title := extractBetween(raw, "class=\"result__a\"", "</a>")
		title = stripTags(title)
		link := extractBetween(raw, "class=\"result__url\"", "</a>")
		link = stripTags(strings.TrimSpace(link))
		snippet := extractBetween(raw, "class=\"result__snippet\"", "</a>")
		snippet = stripTags(snippet)

		if title == "" {
			continue
		}
		results = append(results, searchResult{
			Title:   title,
			URL:     link,
			Snippet: snippet,
		})
	}
	if len(results) == 0 {
		// Fallback parse for different HTML structure.
		for _, raw := range strings.Split(html, "class=\"result\"") {
			title := extractBetween(raw, "class=\"result__title\"", "</h2>")
			title = stripTags(title)
			snippet := extractBetween(raw, "class=\"result__snippet\"", "</div>")
			snippet = stripTags(snippet)
			if title == "" {
				continue
			}
			results = append(results, searchResult{Title: title, Snippet: snippet})
		}
	}
	return results
}

func extractBetween(s, start, end string) string {
	idx := strings.Index(s, start)
	if idx < 0 {
		return ""
	}
	s = s[idx+len(start):]
	idx = strings.Index(s, end)
	if idx < 0 {
		return s
	}
	return s[:idx]
}

func stripTags(s string) string {
	var (
		b     strings.Builder
		inTag bool
	)
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		default:
			if !inTag {
				b.WriteRune(r)
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// WebFetchTool fetches a URL and returns its text content.
type WebFetchTool struct{}

func (t *WebFetchTool) Name() string { return "web_fetch" }
func (t *WebFetchTool) Description() string {
	return "Fetch a URL and return its text content. Use for reading documentation."
}
func (t *WebFetchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to fetch.",
			},
		},
		"required": []string{"url"},
	}
}
func (t *WebFetchTool) IsReadOnly() bool                  { return true }
func (t *WebFetchTool) RequiredPermissions() []tool.Permission { return []tool.Permission{tool.PermNetworkOutbound} }
func (t *WebFetchTool) AffectedPaths(map[string]any) []string  { return nil }

func (t *WebFetchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	rawURL, _ := args["url"].(string)
	if rawURL == "" {
		return "", fmt.Errorf("web_fetch: url is required")
	}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("web_fetch: %w", err)
	}
	req.Header.Set("User-Agent", "Baize/0.3 (web-fetch)")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("web_fetch: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))

	// Try to extract human-readable text (basic HTML-to-text).
	text := extractText(string(body))
	if len(text) > 8000 {
		text = text[:8000] + "\n\n... [truncated]"
	}

	return fmt.Sprintf("URL: %s\nStatus: %d\n\n%s", rawURL, resp.StatusCode, text), nil
}

func extractText(html string) string {
	// Very basic HTML-to-text: strip tags, decode entities, collapse whitespace.
	text := stripTags(html)
	// Collapse blank lines.
	lines := strings.Split(text, "\n")
	var out []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return strings.Join(out, "\n")
}

// Ensure these tools implement SafeTool.
var (
	_ tool.SafeTool = (*WebSearchTool)(nil)
	_ tool.SafeTool = (*WebFetchTool)(nil)
)

// RegisterWebTools is a helper that registers web tools into a registry.
func RegisterWebTools(reg *tool.ToolRegistry) {
	reg.Register(&WebSearchTool{})
	reg.Register(&WebFetchTool{})

	// Marshal their definitions for the JSON schema.
	_ = json.Marshal // keep json import
}
