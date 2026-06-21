// Command baize is the CLI entry point for the Baize agent platform.
//
// Usage:
//
//	baize                          Launch interactive mode.
//	baize [flags] "question"       Run a single-shot agent query.
//	baize server [flags]           Start the API server with AGUI.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/wzhongyou/baize/agent"
	llmgateadapter "github.com/wzhongyou/baize/agent/llmgate"
	"github.com/wzhongyou/baize/server"
	"github.com/wzhongyou/baize/tool/builtin"
	"github.com/wzhongyou/weave/graph"
	"github.com/wzhongyou/llmgate/sdk"
)

var (
	configPath = flag.String("config", "", "Path to llmgate TOML config file")
	provider   = flag.String("provider", "", "Model provider (e.g. deepseek, openai)")
	modelName  = flag.String("model", "", "Specific model ID override")
	workspace  = flag.String("workspace", ".", "Workspace root directory")
	maxSteps   = flag.Int("max-steps", 30, "Maximum agent execution steps")
	verbose    = flag.Bool("verbose", false, "Enable verbose output")

	// Server flags.
	serverPort = flag.Int("port", 9779, "Server listen port")
	serverHost = flag.String("host", "127.0.0.1", "Server listen host")
)

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.Arg(0) == "server" {
		runServer()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; cancel() }()

	llm := buildLLM()
	if llm == nil {
		log.Fatal("No LLM configured. Set up conf/llmgate.toml or use environment variables.")
	}
	wsRoot, err := filepath.Abs(*workspace)
	if err != nil {
		log.Fatalf("Invalid workspace: %v", err)
	}

	tools := buildTools(wsRoot)
	question := flag.Arg(0)

	if *verbose {
		fmt.Fprintf(os.Stderr, "workspace: %s  provider: %s  model: %s\n",
			wsRoot, orDefault(*provider, "auto"), orDefault(*modelName, "default"))
	}

	if question != "" {
		runSingleShot(ctx, llm, tools, wsRoot, question)
	} else {
		runInteractive(ctx, llm, tools, wsRoot)
	}
}

// ── Server mode ────────────────────────────────────────────────────────────────

func runServer() {
	llm := buildLLM()
	if llm == nil {
		log.Fatal("No LLM configured. Set up conf/llmgate.toml.")
	}
	wsRoot, _ := filepath.Abs(*workspace)
	tools := buildTools(wsRoot)

	runner := &agentRunner{
		llm:       llm,
		tools:     tools,
		sysPrompt: buildSystemPrompt(wsRoot),
		maxSteps:  *maxSteps,
	}

	srv, err := server.New(runner, server.Config{
		Port:    *serverPort,
		Host:    *serverHost,
		DataDir: "./data",
	})
	if err != nil {
		log.Fatalf("Server: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Baize Server\n")
	fmt.Fprintf(os.Stderr, "  AGUI: http://%s:%d\n", *serverHost, *serverPort)
	fmt.Fprintf(os.Stderr, "  API:  http://%s:%d/api/health\n", *serverHost, *serverPort)

	if err := srv.Start(); err != nil {
		log.Fatalf("Server: %v", err)
	}
}

// ── Agent runner (bridges server.AgentRunner to ReAct agent) ───────────────────

type agentRunner struct {
	llm       agent.LLMModel
	tools     []agent.Tool
	sysPrompt string
	maxSteps  int
}

func (r *agentRunner) Run(ctx context.Context, state *agent.MessageState) (*agent.MessageState, error) {
	ag := agent.NewReActAgent(agent.ReActAgentConfig{
		Name:         "agui",
		LLM:          r.llm,
		SystemPrompt: r.sysPrompt,
		Tools:        r.tools,
		MaxSteps:     r.maxSteps,
	})
	g, err := ag.BuildGraph()
	if err != nil {
		return nil, err
	}
	engine := graph.NewEngine(g)
	result, err := engine.Run(ctx, state)
	if err != nil {
		return nil, err
	}
	return result.FinalState, nil
}

func (r *agentRunner) RunStream(ctx context.Context, state *agent.MessageState, onEvent func(server.StreamEvent)) {
	ag := agent.NewReActAgent(agent.ReActAgentConfig{
		Name:         "agui",
		LLM:          r.llm,
		SystemPrompt: r.sysPrompt,
		Tools:        r.tools,
		MaxSteps:     r.maxSteps,
	})
	g, err := ag.BuildGraph()
	if err != nil {
		onEvent(server.StreamEvent{Type: "error", Content: err.Error()})
		return
	}

	engine := graph.NewEngine(g)
	hook := &streamHook{onEvent: onEvent}
	result, err := engine.Run(ctx, state, graph.WithHook(hook))
	if err != nil {
		onEvent(server.StreamEvent{Type: "error", Content: err.Error()})
		return
	}

	if result != nil && len(result.FinalState.Messages) > 0 {
		last := result.FinalState.Messages[len(result.FinalState.Messages)-1]
		if last.Role == agent.RoleAssistant && last.Content != "" {
			onEvent(server.StreamEvent{Type: "answer", Content: last.Content})
		}
	}
	onEvent(server.StreamEvent{Type: "done", Tokens: result.FinalState.TotalTokens})
}

type streamHook struct{ onEvent func(server.StreamEvent) }

func (h *streamHook) OnGraphStart(_ context.Context, _ string, _ *agent.MessageState)        {}
func (h *streamHook) OnGraphEnd(_ context.Context, _ string, _ *agent.MessageState, _ error) {}
func (h *streamHook) OnNodeStart(_ context.Context, _ string, _ *agent.MessageState)         {}
func (h *streamHook) OnNodeEnd(_ context.Context, _ string, s *agent.MessageState, _ error, _ time.Duration) {
	if len(s.Messages) == 0 {
		return
	}
	last := s.Messages[len(s.Messages)-1]
	switch {
	case len(last.ToolCalls) > 0:
		for _, tc := range last.ToolCalls {
			h.onEvent(server.StreamEvent{Type: "tool_call", ToolName: tc.Name, Content: fmt.Sprintf("%v", tc.Arguments)})
		}
	case last.Role == agent.RoleTool:
		h.onEvent(server.StreamEvent{Type: "tool_result", Content: last.Content})
	case last.Role == agent.RoleAssistant && last.Content != "":
		h.onEvent(server.StreamEvent{Type: "answer", Content: last.Content})
	}
}
func (h *streamHook) OnRetry(_ context.Context, _ string, _ int, _ error) {}

// ── Core execution ─────────────────────────────────────────────────────────────

func runSingleShot(ctx context.Context, llm agent.LLMModel, tools []agent.Tool, wsRoot, question string) {
	startTime := time.Now()
	ag := agent.NewReActAgent(agent.ReActAgentConfig{
		Name:         "baize",
		LLM:          llm,
		SystemPrompt: buildSystemPrompt(wsRoot),
		Tools:        tools,
		MaxSteps:     *maxSteps,
	})
	g, err := ag.BuildGraph()
	if err != nil {
		log.Fatal(err)
	}

	engine := graph.NewEngine(g)
	result, err := engine.Run(ctx, &agent.MessageState{
		Messages: []agent.Message{{Role: agent.RoleUser, Content: question}},
		MaxSteps: *maxSteps,
	}, graph.WithHook(&cliHook{verbose: *verbose}))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n[%v]\n", err)
		os.Exit(1)
	}

	duration := time.Since(startTime).Round(time.Millisecond)
	last := result.FinalState.Messages[len(result.FinalState.Messages)-1]
	fmt.Printf("\n%s\n", last.Content)
	fmt.Fprintf(os.Stderr, "\n[%d steps | %v | %d tokens]\n",
		result.TotalSteps, duration, result.FinalState.TotalTokens)
}

func runInteractive(ctx context.Context, llm agent.LLMModel, tools []agent.Tool, wsRoot string) {
	fmt.Println("Baize Interactive Mode")
	fmt.Print("Type /help for commands, /quit to exit.\n\n")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			fmt.Println()
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		switch {
		case line == "/quit" || line == "/exit":
			fmt.Println("Goodbye.")
			return
		case line == "/help":
			printHelp()
			continue
		case strings.HasPrefix(line, "/"):
			fmt.Printf("Unknown: %s\n", line)
			continue
		}
		runSingleShot(ctx, llm, tools, wsRoot, line)
		fmt.Println()
	}
}

// ── LLM ────────────────────────────────────────────────────────────────────────

func buildLLM() agent.LLMModel {
	config := findConfig()
	gw, err := sdk.NewFromFile(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Trying env vars (no config found)...\n")
		gw = sdk.New()
	}
	if *provider != "" {
		return llmgateadapter.New(gw, llmgateadapter.Config{Provider: *provider, Model: *modelName})
	}
	if *modelName != "" {
		return llmgateadapter.New(gw, llmgateadapter.Config{Model: *modelName})
	}
	return llmgateadapter.NewWithStrategy(gw)
}

func findConfig() string {
	if *configPath != "" {
		return *configPath
	}
	for _, p := range []string{"conf/llmgate.toml", "llmgate.toml", filepath.Join(os.Getenv("HOME"), ".baize", "config.toml")} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "conf/llmgate.toml"
}

// ── Tools ──────────────────────────────────────────────────────────────────────

func buildTools(wsRoot string) []agent.Tool {
	return []agent.Tool{
		&builtin.CalculatorTool{},
		&builtin.FileTool{WorkspaceRoot: wsRoot},
		&builtin.ShellTool{WorkspaceRoot: wsRoot, MaxRuntime: 120 * time.Second},
		&builtin.GitTool{WorkspaceRoot: wsRoot},
	}
}

func buildSystemPrompt(wsRoot string) string {
	hostname, _ := os.Hostname()
	return fmt.Sprintf(`You are Baize（白泽）, a universal super-agent — an autonomous AI that handles any task across any domain.

You are named after the mythical beast Baize from Chinese mythology who knew all 11,520 creatures in the world — their names, forms, and how to master them. You inherit that same mission: understand any problem deeply, and solve it autonomously.

Your capabilities span:
- Research & analysis: web search, multi-source verification, synthesizing findings into structured reports
- Writing & creation: articles, reports, translations, editing, brainstorming, creative writing
- Software engineering: code generation, refactoring, debugging, testing, architecture design, deployment
- Data analysis: processing spreadsheets, querying databases, generating charts and insights
- Office productivity: drafting emails, summarizing documents, managing schedules, processing files
- General problem-solving: breaking down complex tasks, planning multi-step solutions, executing with tools

Current workspace: %s
Host: %s

Guidelines:
- Break complex tasks into clear steps. Execute them one at a time.
- Read files before editing them. Understand before acting.
- Use the shell tool to run tests, builds, scripts, and data processing.
- Make minimal, focused changes. Prefer surgical edits over full rewrites.
- When researching, verify facts from multiple sources before drawing conclusions.
- Think broadly — you are not limited to code. You can write, analyze, research, create, and compute.`, wsRoot, hostname)
}

// ── CLI Hook ───────────────────────────────────────────────────────────────────

type cliHook struct{ verbose bool }

func (h *cliHook) OnGraphStart(_ context.Context, _ string, _ *agent.MessageState)        {}
func (h *cliHook) OnGraphEnd(_ context.Context, _ string, _ *agent.MessageState, _ error) {}
func (h *cliHook) OnNodeStart(_ context.Context, _ string, _ *agent.MessageState)         {}
func (h *cliHook) OnNodeEnd(_ context.Context, _ string, s *agent.MessageState, _ error, _ time.Duration) {
	if len(s.Messages) == 0 {
		return
	}
	last := s.Messages[len(s.Messages)-1]
	switch {
	case len(last.ToolCalls) > 0:
		names := make([]string, len(last.ToolCalls))
		for i, tc := range last.ToolCalls {
			names[i] = tc.Name
		}
		fmt.Fprintf(os.Stderr, "  -> %s\n", strings.Join(names, ", "))
	case last.Role == agent.RoleTool && h.verbose:
		preview := last.Content
		if len(preview) > 120 {
			preview = preview[:120] + "..."
		}
		fmt.Fprintf(os.Stderr, "    %s\n", strings.ReplaceAll(preview, "\n", "\n    "))
	}
}
func (h *cliHook) OnRetry(_ context.Context, _ string, _ int, _ error) {}

// ── Helpers ────────────────────────────────────────────────────────────────────

func usage() {
	fmt.Fprintf(os.Stderr, `Baize — Unified AI Agent Platform

Usage:
  baize [flags] "question"     Single-shot agent query.
  baize [flags]                 Interactive mode.
  baize server [flags]          API server + AGUI.

Flags:
`)
	flag.PrintDefaults()
}

func printHelp() {
	fmt.Println(`Commands:
  /help          Show this help.
  /quit, /exit   Exit.
Just type your question.`)
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
