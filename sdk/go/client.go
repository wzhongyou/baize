// Package sdk provides a Go client for the Baize Agent API.
//
// Usage:
//
//	client := sdk.NewClient("http://localhost:9779")
//	client.Chat(ctx, req, func(event ChatEvent) { ... })
package sdk

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/wzhongyou/baize/api"
)

// Client is a Go client for the Baize Agent API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Baize API client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Minute, // long timeout for streaming chat
		},
	}
}

// Chat starts a streaming agent chat. Events are delivered to onEvent.
func (c *Client) Chat(ctx context.Context, req api.ChatRequest, onEvent func(api.ChatEvent)) error {
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/chat", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		var event api.ChatEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		onEvent(event)
		if event.Type == api.EventDone || event.Type == api.EventError {
			break
		}
	}
	return scanner.Err()
}

// Health checks if the server is alive.
func (c *Client) Health(ctx context.Context) (*api.HealthResponse, error) {
	var resp api.Response
	if err := c.get(ctx, "/api/v1/health", &resp); err != nil {
		return nil, err
	}
	var hr api.HealthResponse
	if err := json.Unmarshal(resp.Data, &hr); err != nil {
		return nil, err
	}
	return &hr, nil
}

// ListTools returns all registered tools.
func (c *Client) ListTools(ctx context.Context) ([]api.ToolInfo, error) {
	var resp api.Response
	if err := c.post(ctx, "/api/v1/tools/list", nil, &resp); err != nil {
		return nil, err
	}
	var ltr api.ListToolsResponse
	if err := json.Unmarshal(resp.Data, &ltr); err != nil {
		return nil, err
	}
	return ltr.Tools, nil
}

// CallTool invokes a tool directly.
func (c *Client) CallTool(ctx context.Context, req api.CallToolRequest) (*api.CallToolResponse, error) {
	var resp api.Response
	if err := c.post(ctx, "/api/v1/tools/call", req, &resp); err != nil {
		return nil, err
	}
	var ctr api.CallToolResponse
	if err := json.Unmarshal(resp.Data, &ctr); err != nil {
		return nil, err
	}
	return &ctr, nil
}

// CreateSession creates a new agent session.
func (c *Client) CreateSession(ctx context.Context, title, workspace string) (string, error) {
	var resp api.Response
	if err := c.post(ctx, "/api/v1/sessions", api.CreateSessionRequest{Title: title, WorkspaceRoot: workspace}, &resp); err != nil {
		return "", err
	}
	var container struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp.Data, &container); err != nil {
		return "", err
	}
	return container.ID, nil
}

// ListSessions returns all sessions.
func (c *Client) ListSessions(ctx context.Context) ([]api.SessionInfo, error) {
	var resp api.Response
	if err := c.get(ctx, "/api/v1/sessions", &resp); err != nil {
		return nil, err
	}
	var container struct {
		Sessions []api.SessionInfo `json:"sessions"`
	}
	if err := json.Unmarshal(resp.Data, &container); err != nil {
		return nil, err
	}
	return container.Sessions, nil
}

// GetSession returns a session with messages.
func (c *Client) GetSession(ctx context.Context, id string) (*api.SessionDetail, error) {
	var resp api.Response
	if err := c.get(ctx, "/api/v1/sessions/"+id, &resp); err != nil {
		return nil, err
	}
	var sd api.SessionDetail
	if err := json.Unmarshal(resp.Data, &sd); err != nil {
		return nil, err
	}
	return &sd, nil
}

// DeleteSession deletes a session.
func (c *Client) DeleteSession(ctx context.Context, id string) error {
	return c.delete(ctx, "/api/v1/sessions/"+id)
}

// SearchMemory searches long-term memory.
func (c *Client) SearchMemory(ctx context.Context, query string, topK int) ([]api.MemorySearchResult, error) {
	var resp api.Response
	if err := c.post(ctx, "/api/v1/memory/search", api.MemorySearchRequest{Query: query, TopK: topK}, &resp); err != nil {
		return nil, err
	}
	var msr api.MemorySearchResponse
	if err := json.Unmarshal(resp.Data, &msr); err != nil {
		return nil, err
	}
	return msr.Results, nil
}

// SaveMemory saves to long-term memory.
func (c *Client) SaveMemory(ctx context.Context, content string, metadata map[string]any) error {
	var resp api.Response
	return c.post(ctx, "/api/v1/memory/save", api.MemorySaveRequest{Content: content, Metadata: metadata}, &resp)
}

// ── HTTP helpers ──────────────────────────────────────────────────────────

func (c *Client) get(ctx context.Context, path string, out *api.Response) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *Client) post(ctx context.Context, path string, body any, out *api.Response) error {
	var r io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		r = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, r)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *Client) delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return c.parseError(resp)
	}
	return nil
}

func (c *Client) do(req *http.Request, out *api.Response) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return c.parseError(resp)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) parseError(resp *http.Response) error {
	var apiResp api.Response
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Message != "" {
		return fmt.Errorf("[%d] %s", resp.StatusCode, apiResp.Message)
	}
	return fmt.Errorf("HTTP %d", resp.StatusCode)
}
