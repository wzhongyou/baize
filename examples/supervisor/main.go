// supervisor_demo demonstrates multi-agent orchestration — a SupervisorAgent
// that routes tasks to specialized sub-agents.
//
// Usage:
//
//	go run ./examples/supervisor
//
// Architecture: supervisor ─[route]─→ sub-agent ─→ collect ─→ supervisor (loop)
//
// Runs in mock mode, no API key required.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/wzhongyou/baize/agent"
	"github.com/wzhongyou/weave/graph"
)

func main() {
	ctx := context.Background()

	// ── 1. Define sub-agents ────────────────────────────────────────────────
	calculatorLLM := &roundRobinMockLLM{
		responses: []mockResponse{
			{content: "", toolName: "calculator", toolArgs: map[string]any{"expression": "123 * 456"}},
			{content: "Result: 123 * 456 = 56088"},
		},
	}
	calculatorAgent := agent.NewReActAgent(agent.ReActAgentConfig{
		Name:         "calculator",
		LLM:          calculatorLLM,
		SystemPrompt: "You are a calculator agent. Use the calculator tool for math.",
		Tools:        []agent.Tool{&agent.CalculatorTool{}},
		MaxSteps:     5,
	})

	echoLLM := &roundRobinMockLLM{
		responses: []mockResponse{
			{content: "You said: Hello World. I received it!"},
		},
	}
	echoAgent := agent.NewReActAgent(agent.ReActAgentConfig{
		Name:         "echo",
		LLM:          echoLLM,
		SystemPrompt: "You are an echo agent. Reply to the user's message directly.",
		MaxSteps:     3,
	})

	// ── 2. Create the supervisor agent ──────────────────────────────────────
	supervisorLLM := &roundRobinMockLLM{
		responses: []mockResponse{
			{content: "", toolName: "route", toolArgs: map[string]any{"agent": "calculator"}},
			{content: "", toolName: "route", toolArgs: map[string]any{"agent": "echo"}},
			{content: "All sub-agents have completed their tasks."},
		},
	}

	supAgent := agent.NewSupervisorAgent(agent.SupervisorAgentConfig{
		Name: "supervisor-demo",
		LLM:  supervisorLLM,
		SubAgents: map[string]agent.SubAgent{
			"calculator": calculatorAgent,
			"echo":       echoAgent,
		},
		MaxRounds: 10,
	})

	g, err := supAgent.BuildGraph()
	if err != nil {
		panic(err)
	}

	// ── 3. Run ──────────────────────────────────────────────────────────────
	engine := graph.NewEngine(g)
	state := &agent.MessageState{
		Messages: []agent.Message{{
			Role:    agent.RoleUser,
			Content: "Calculate 123*456, then echo 'Hello World'",
		}},
	}

	fmt.Println("=== Multi-Agent Orchestration Demo ===")
	fmt.Printf("Task: %s\n\n", state.Messages[0].Content)

	result, err := engine.Run(ctx, state, graph.WithHook(&traceHook{}))
	if err != nil {
		panic(err)
	}

	fmt.Println("\n=== Results ===")
	fmt.Printf("Completed agents: %v\n", result.FinalState.CompletedAgents)
	if len(result.FinalState.Messages) > 0 {
		last := result.FinalState.Messages[len(result.FinalState.Messages)-1]
		fmt.Printf("Final output: %s\n", last.Content)
	}
	fmt.Printf("Total steps: %d\n", result.TotalSteps)
	fmt.Printf("Duration: %v\n", result.TotalDuration.Round(time.Millisecond))
}

// ── Mock LLM ────────────────────────────────────────────────────────────────

type mockResponse struct {
	content  string
	toolName string
	toolArgs map[string]any
}

// roundRobinMockLLM returns a predefined sequence of responses in order.
type roundRobinMockLLM struct {
	responses []mockResponse
	index     int
}

func (m *roundRobinMockLLM) Chat(_ context.Context, _ *agent.ChatRequest) (*agent.ChatResponse, error) {
	if m.index >= len(m.responses) {
		m.index = len(m.responses) - 1
	}
	resp := m.responses[m.index]
	m.index++

	cr := &agent.ChatResponse{
		Content:      resp.content,
		FinishReason: "stop",
	}
	if resp.toolName != "" {
		cr.ToolCalls = []agent.ToolCall{
			{ID: fmt.Sprintf("call-%d", m.index), Name: resp.toolName, Arguments: resp.toolArgs},
		}
		cr.FinishReason = "tool_calls"
	}
	return cr, nil
}

func (m *roundRobinMockLLM) ChatStream(ctx context.Context, req *agent.ChatRequest) (<-chan *agent.StreamChunk, error) {
	resp, err := m.Chat(ctx, req)
	if err != nil {
		return nil, err
	}
	ch := make(chan *agent.StreamChunk, 1)
	ch <- &agent.StreamChunk{
		Content:      resp.Content,
		ToolCalls:    resp.ToolCalls,
		FinishReason: resp.FinishReason,
	}
	close(ch)
	return ch, nil
}

// ── Trace hook ──────────────────────────────────────────────────────────────

type traceHook struct{}

func (h *traceHook) OnNodeStart(_ context.Context, name string, s *agent.MessageState) {
	fmt.Printf("  > [%s]\n", name)
}
func (h *traceHook) OnNodeEnd(_ context.Context, name string, s *agent.MessageState, err error, dur time.Duration) {
	if err != nil {
		fmt.Printf("  x [%s] error: %v (%v)\n", name, err, dur)
		return
	}
	if len(s.Messages) > 0 {
		last := s.Messages[len(s.Messages)-1]
		if last.Role == agent.RoleTool {
			fmt.Printf("  < [%s] → Tool result: %s (%v)\n", name, last.Content, dur)
		} else if len(last.ToolCalls) > 0 {
			fmt.Printf("  < [%s] → Call: %s (%v)\n", name, last.ToolCalls[0].Name, dur)
		} else if last.Content != "" {
			fmt.Printf("  < [%s] → %s (%v)\n", name, last.Content, dur)
		}
	}
}
func (h *traceHook) OnGraphStart(_ context.Context, name string, _ *agent.MessageState) {
	fmt.Printf("◆ Graph start: %s\n", name)
}
func (h *traceHook) OnGraphEnd(_ context.Context, _ string, _ *agent.MessageState, err error) {
	if err != nil {
		fmt.Printf("◆ Graph failed: %v\n", err)
	}
}
func (h *traceHook) OnRetry(_ context.Context, _ string, _ int, _ error) {}
