// Package server provides the HTTP + SSE API server for the Baize agent platform.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/wzhongyou/baize/core/agent"
	"github.com/wzhongyou/baize/core/session"
	"github.com/wzhongyou/baize/protocol"
)

// Server is the Baize API server.
type Server struct {
	http       *http.Server
	agent      AgentRunner
	sessions   *session.Store
	tools      protocol.ToolProvider
	memory     protocol.MemoryProvider
}

// AgentRunner executes agent tasks. Implementations wrap the orchestrator.
type AgentRunner interface {
	Run(ctx context.Context, req AgentRunRequest) (*AgentRunResult, error)
	RunStream(ctx context.Context, req AgentRunRequest, onEvent func(StreamEvent))
}

// AgentRunRequest is the input for an agent run.
type AgentRunRequest struct {
	SessionID string
	Message   string
	History   []agent.Message // prior conversation messages, injected into MessageState
	Provider  string
	Model     string
	MaxSteps  int
}

var defaultBudget = agent.DefaultContextBudget()

// AgentRunResult is the final output of an agent run.
type AgentRunResult struct {
	Content string
	Tokens  int
	Steps   int
}

// StreamEvent is a single event emitted during agent execution.
type StreamEvent struct {
	Type     string         `json:"type"` // "thought", "tool_call", "tool_result", "answer", "done"
	Content  string         `json:"content,omitempty"`
	ToolName string         `json:"tool_name,omitempty"`
	ToolArgs map[string]any `json:"tool_args,omitempty"`
	Tokens   int            `json:"tokens,omitempty"`
}

// Config holds server configuration.
type Config struct {
	Port    int
	Host    string
	DataDir string
}

// DefaultConfig returns reasonable defaults.
func DefaultConfig() Config {
	return Config{
		Port:    9779,
		Host:    "127.0.0.1",
		DataDir: "./data",
	}
}

// New creates a new Baize API server.
func New(runner AgentRunner, cfg Config, opts ...Option) (*Server, error) {
	if err := os.MkdirAll(cfg.DataDir, 0700); err != nil {
		return nil, fmt.Errorf("server: data dir: %w", err)
	}

	store, err := session.NewStore(cfg.DataDir + "/baize.db")
	if err != nil {
		return nil, fmt.Errorf("server: session store: %w", err)
	}

	s := &Server{
		agent:    runner,
		sessions: store,
	}

	for _, opt := range opts {
		opt(s)
	}

	s.http = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      s.routes(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	return s, nil
}

// Option is a functional option for Server configuration.
type Option func(*Server)

// WithTools injects a tool provider.
func WithTools(tp protocol.ToolProvider) Option {
	return func(s *Server) { s.tools = tp }
}

// WithMemory injects a memory provider.
func WithMemory(mp protocol.MemoryProvider) Option {
	return func(s *Server) { s.memory = mp }
}

// Start begins listening and blocks.
func (s *Server) Start() error {
	log.Printf("Baize API: http://%s", s.http.Addr)
	log.Printf("Health:   http://%s/api/v1/health", s.http.Addr)
	return s.http.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
