// Package protocol defines the Baize API protocol — shared types, error codes,
// provider interfaces, and response structures used by both the server and SDK clients.
package protocol

import (
	"context"
	"encoding/json"
	"time"
)

// ── Provider interfaces ─────────────────────────────────────────────────────

// ToolProvider supplies tool metadata and execution capability.
type ToolProvider interface {
	ToolInfos() []ToolInfo
	Execute(ctx context.Context, name string, args map[string]any) (string, error)
}

// MemoryProvider supplies long-term memory operations.
type MemoryProvider interface {
	Search(ctx context.Context, query string, topK int) ([]MemoryResult, error)
	Save(ctx context.Context, content string, metadata map[string]any) error
}

// ── Version ─────────────────────────────────────────────────────────────────

const Version = "v1"

// HealthResponse returns server status.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// ── Response envelope ───────────────────────────────────────────────────────

// Response is the standard API response envelope.
type Response struct {
	Code      int             `json:"code"`
	Data      json.RawMessage `json:"data,omitempty"`
	Message   string          `json:"message,omitempty"`
	RequestID string          `json:"request_id"`
}

// ── Error codes ─────────────────────────────────────────────────────────────

const (
	CodeOK            = 0
	CodeBadRequest    = 1001
	CodeUnauthorized  = 1002
	CodeNotFound      = 1003
	CodeConflict      = 1004
	CodeRateLimit     = 1005
	CodeInternalError = 2001
	CodeTimeout       = 2002
	CodeAgentError    = 3001
	CodeToolError     = 3002
	CodeLLMError      = 3003
)

// ── Chat ────────────────────────────────────────────────────────────────────

// ChatRequest is sent to POST /api/v1/chat to start a streaming agent run.
type ChatRequest struct {
	SessionID string   `json:"session_id,omitempty"` // empty = auto-create
	Message   string   `json:"message"`
	Images    []string `json:"images,omitempty"` // base64-encoded images ("data:image/png;base64,..." or raw base64)
	Provider  string   `json:"provider,omitempty"`  // override llmgate provider
	Model     string   `json:"model,omitempty"`     // override model
	MaxSteps  int      `json:"max_steps,omitempty"` // default 30
}

// ContentBlock is a structured unit of rich content in a chat event.
// Clients render by type; unknown types should fall back to Meta["fallback_text"].
type ContentBlock struct {
	Type    string         `json:"type"`              // "text" | "image" | "code" | "html" | custom
	Content string         `json:"content,omitempty"` // for text/code/html
	Lang    string         `json:"lang,omitempty"`    // for type=code
	Data    string         `json:"data,omitempty"`    // for type=image, base64 data URL
	Meta    map[string]any `json:"meta,omitempty"`    // skill-defined metadata / fallback_text
}

// ChatEvent is streamed via SSE from POST /api/v1/chat.
type ChatEvent struct {
	Type     string         `json:"type"` // "thought", "tool_call", "tool_result", "answer", "done", "error"
	Content  string         `json:"content,omitempty"` // plain text / markdown; empty when Blocks is set
	Blocks   []ContentBlock `json:"blocks,omitempty"`  // rich content from MCP/skills; takes priority over Content
	ToolName string         `json:"tool_name,omitempty"`
	ToolArgs map[string]any `json:"tool_args,omitempty"`
	Tokens   int            `json:"tokens,omitempty"`
}

// SSE event type constants.
const (
	EventThought     = "thought"
	EventToolCall    = "tool_call"
	EventToolResult  = "tool_result"
	EventAnswer      = "answer"
	EventDone        = "done"
	EventError       = "error"
)

// ── Tools ───────────────────────────────────────────────────────────────────

// ToolInfo describes a registered tool (used by both API responses and ToolProvider).
type ToolInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
	ReadOnly    bool           `json:"read_only"`
	Source      string         `json:"source"` // "builtin" | "mcp:<server>"
}

// ToolInfo describes a registered tool.
type ListToolsResponse struct {
	Tools []ToolInfo `json:"tools"`
}

// CallToolRequest is sent to POST /api/v1/tools/call.
type CallToolRequest struct {
	SessionID string         `json:"session_id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// CallToolResponse is the response from POST /api/v1/tools/call.
type CallToolResponse struct {
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// ── Messages ────────────────────────────────────────────────────────────────

// Message represents a single conversation turn in the API.
type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	ToolName  string    `json:"tool_name,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ToolCall is a tool invocation within an assistant message.
type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ── Sessions ────────────────────────────────────────────────────────────────

// SessionInfo is a summary returned in list endpoints.
type SessionInfo struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	WorkspaceRoot string    `json:"workspace_root,omitempty"`
	Model         string    `json:"model,omitempty"`
	StepCount     int       `json:"step_count"`
	TotalTokens   int       `json:"total_tokens"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SessionDetail includes messages.
type SessionDetail struct {
	SessionInfo
	Messages []Message `json:"messages"`
}

// CreateSessionRequest is sent to POST /api/v1/sessions.
type CreateSessionRequest struct {
	Title         string `json:"title"`
	WorkspaceRoot string `json:"workspace_root,omitempty"`
}

// ── Memory ──────────────────────────────────────────────────────────────────

// MemoryResult is one search hit from long-term memory (used by both API responses and MemoryProvider).
type MemoryResult struct {
	Content  string         `json:"content"`
	Score    float64        `json:"score"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// MemorySearchRequest is sent to POST /api/v1/memory/search.
type MemorySearchRequest struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k,omitempty"` // default 5
}

// MemorySearchResult is one result from memory search.
type MemorySearchResult struct {
	ID       string  `json:"id"`
	Content  string  `json:"content"`
	Score    float64 `json:"score"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// MemorySearchResponse is the response from POST /api/v1/memory/search.
type MemorySearchResponse struct {
	Results []MemorySearchResult `json:"results"`
}

// MemorySaveRequest is sent to POST /api/v1/memory/save.
type MemorySaveRequest struct {
	Content  string         `json:"content"`
	Metadata map[string]any `json:"metadata,omitempty"`
}
