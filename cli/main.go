// Command baize is the CLI entry point for the Baize agent platform.
//
// Usage:
//
//	baize                          Launch interactive mode.
//	baize [flags] "question"       Run a single-shot agent query.
//	baize server [flags]           Start the API server.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhongyou/baize/cli/tui"
	"github.com/wzhongyou/baize/core/agent"
	llmgateadapter "github.com/wzhongyou/baize/core/agent/llmgate"
	baizecontext "github.com/wzhongyou/baize/core/context"
	"github.com/wzhongyou/baize/core/memory"
	"github.com/wzhongyou/baize/core/permission"
	"github.com/wzhongyou/baize/core/skill"
	"github.com/wzhongyou/baize/core/tool"
	"github.com/wzhongyou/baize/core/tool/builtin"
	"github.com/wzhongyou/baize/internal/version"
	"github.com/wzhongyou/baize/protocol"
	"github.com/wzhongyou/baize/server"
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
	mode       = flag.String("mode", "auto-edit", "Permission mode: suggest | auto-edit | full-auto")

	// TUI / Server flags.
	noTui      = flag.Bool("no-tui", false, "Disable Bubble Tea TUI (use simple REPL)")
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

	sm := buildSkillManager(ctx)
	defer sm.Close()

	reg := buildToolRegistry(wsRoot, sm)
	tools := reg.List()
	permChecker, policyEngine := buildPermChecker(reg, wsRoot)
	question := flag.Arg(0)

	if *verbose {
		fmt.Fprintf(os.Stderr, "workspace: %s  provider: %s  model: %s\n",
			wsRoot, orDefault(*provider, "auto"), orDefault(*modelName, "default"))
	}

	sysPrompt := buildSystemPrompt(wsRoot, sm)
	if question != "" {
		runSingleShot(ctx, llm, tools, permChecker, wsRoot, question)
	} else if *noTui {
		runSimpleREPL(ctx, llm, tools, permChecker, wsRoot)
	} else {
		runTUI(llm, tools, permChecker, policyEngine, wsRoot)
	}
	_ = sysPrompt
}

// ── Permission ──────────────────────────────────────────────────────────────

func buildPermChecker(reg *tool.ToolRegistry, wsRoot string) (agent.PermissionChecker, *permission.PolicyEngine) {
	pe := permission.NewPolicyEngine(permission.DefaultPolicy(wsRoot))
	switch *mode {
	case "suggest":
		// Read-only: deny all writes and shell execution.
		return permission.ReadOnlyChecker(), pe
	case "full-auto":
		// Allow everything; policy engine still enforces hard denies.
		return pe.AsAgentCheckerFullAuto(reg), pe
	default: // "auto-edit"
		return pe.AsAgentChecker(reg), pe
	}
}

// ── Server mode ─────────────────────────────────────────────────────────────

func runServer() {
	llm := buildLLM()
	if llm == nil {
		log.Fatal("No LLM configured. Set up conf/llmgate.toml.")
	}
	ctx := context.Background()
	wsRoot, _ := filepath.Abs(*workspace)

	sm := buildSkillManager(ctx)
	defer sm.Close()

	reg := buildToolRegistry(wsRoot, sm)
	tools := reg.List()

	runner := &agentRunner{
		llm:         llm,
		tools:       tools,
		sysPrompt:   buildSystemPrompt(wsRoot, sm),
		maxSteps:    *maxSteps,
		permChecker: func() agent.PermissionChecker { pc, _ := buildPermChecker(reg, wsRoot); return pc }(),
	}

	// Build server with tools and optional memory.
	opts := []server.Option{
		server.WithTools(reg.AsToolProvider()),
	}
	if memDir := os.Getenv("BAIZE_MEMORY_DIR"); memDir != "" {
		if ms, err := memory.NewMarkdownStore(memDir); err == nil {
			opts = append(opts, server.WithMemory(ms))
			log.Printf("Memory: %s", memDir)
		}
	}

	srv, err := server.New(runner, server.Config{
		Port:    *serverPort,
		Host:    *serverHost,
		DataDir: "./data",
	}, opts...)
	if err != nil {
		log.Fatalf("Server: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Baize Server\n")
	fmt.Fprintf(os.Stderr, "  API:  http://%s:%d/api/v1/health\n", *serverHost, *serverPort)
	fmt.Fprintf(os.Stderr, "  Tools: %d registered\n", len(tools))

	if err := srv.Start(); err != nil {
		log.Fatalf("Server: %v", err)
	}
}

// ── Agent runner (bridges server.AgentRunner to ReAct agent) ────────────────

type agentRunner struct {
	llm         agent.LLMModel
	tools       []agent.Tool
	sysPrompt   string
	maxSteps    int
	permChecker agent.PermissionChecker
}

func (r *agentRunner) Run(ctx context.Context, req server.AgentRunRequest) (*server.AgentRunResult, error) {
	maxSteps := req.MaxSteps
	if maxSteps <= 0 {
		maxSteps = r.maxSteps
	}
	ag := agent.NewReActAgent(agent.ReActAgentConfig{
		Name:         "baize-server",
		LLM:          r.llm,
		SystemPrompt: r.sysPrompt,
		Tools:        r.tools,
		MaxSteps:     maxSteps,
		PermChecker:  r.permChecker,
	})
	g, err := ag.BuildGraph()
	if err != nil {
		return nil, err
	}
	engine := graph.NewEngine(g)
	messages := append(req.History, agent.Message{Role: agent.RoleUser, Content: req.Message, Images: req.Images})
	result, err := engine.Run(ctx, &agent.MessageState{
		Messages: messages,
		MaxSteps: maxSteps,
	})
	if err != nil {
		return nil, err
	}
	return &server.AgentRunResult{
		Content: lastAssistantContent(result.FinalState),
		Tokens:  result.FinalState.TotalTokens,
		Steps:   result.TotalSteps,
	}, nil
}

func (r *agentRunner) RunStream(ctx context.Context, req server.AgentRunRequest, onEvent func(server.StreamEvent)) {
	maxSteps := req.MaxSteps
	if maxSteps <= 0 {
		maxSteps = r.maxSteps
	}
	ag := agent.NewReActAgent(agent.ReActAgentConfig{
		Name:         "baize-server",
		LLM:          r.llm,
		SystemPrompt: r.sysPrompt,
		Tools:        r.tools,
		MaxSteps:     maxSteps,
		PermChecker:  r.permChecker,
		Stream:       true,
		OnChunk: func(chunk *agent.StreamChunk) {
			if chunk.ReasoningContent != "" {
				onEvent(server.StreamEvent{Type: "thought", Content: chunk.ReasoningContent})
			}
			if chunk.Content != "" {
				onEvent(server.StreamEvent{Type: "answer", Content: chunk.Content})
			}
		},
	})
	g, err := ag.BuildGraph()
	if err != nil {
		onEvent(server.StreamEvent{Type: "error", Content: err.Error()})
		return
	}

	engine := graph.NewEngine(g)
	hook := &streamHook{onEvent: onEvent}
	messages := append(req.History, agent.Message{Role: agent.RoleUser, Content: req.Message, Images: req.Images})
	state := &agent.MessageState{
		Messages: messages,
		MaxSteps: maxSteps,
	}
	result, err := engine.Run(ctx, state, graph.WithHook(hook))
	if err != nil {
		onEvent(server.StreamEvent{Type: "error", Content: err.Error()})
		return
	}

	onEvent(server.StreamEvent{Type: "done", Tokens: result.FinalState.TotalTokens})
}

func lastAssistantContent(state *agent.MessageState) string {
	if state == nil || len(state.Messages) == 0 {
		return ""
	}
	last := state.Messages[len(state.Messages)-1]
	if last.Role == agent.RoleAssistant && last.Content != "" {
		return last.Content
	}
	return ""
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
		if blocks := parseRichBlocks(last.Content); blocks != nil {
			h.onEvent(server.StreamEvent{Type: "tool_result", Blocks: blocks})
		} else {
			h.onEvent(server.StreamEvent{Type: "tool_result", Content: last.Content})
		}
	}
}
func (h *streamHook) OnRetry(_ context.Context, _ string, _ int, _ error) {}

// parseRichBlocks detects the __baize_blocks envelope from MCP/skill tool results.
// Returns nil for plain-text results so callers can fall back to Content.
func parseRichBlocks(content string) []protocol.ContentBlock {
	if len(content) == 0 || content[0] != '{' {
		return nil
	}
	var env struct {
		Blocks []protocol.ContentBlock `json:"__baize_blocks"`
	}
	if err := json.Unmarshal([]byte(content), &env); err != nil || len(env.Blocks) == 0 {
		return nil
	}
	return env.Blocks
}

// ── Core execution ──────────────────────────────────────────────────────────

func runSingleShot(ctx context.Context, llm agent.LLMModel, tools []agent.Tool, permChecker agent.PermissionChecker, wsRoot, question string) {
	startTime := time.Now()
	ag := agent.NewReActAgent(agent.ReActAgentConfig{
		Name:         "baize",
		LLM:          llm,
		SystemPrompt: buildSystemPrompt(wsRoot, nil),
		Tools:        tools,
		MaxSteps:     *maxSteps,
		PermChecker:  permChecker,
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

func runSimpleREPL(ctx context.Context, llm agent.LLMModel, tools []agent.Tool, permChecker agent.PermissionChecker, wsRoot string) {
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
		runSingleShot(ctx, llm, tools, permChecker, wsRoot, line)
		fmt.Println()
	}
}

func runTUI(llm agent.LLMModel, tools []agent.Tool, permChecker agent.PermissionChecker, policyEngine *permission.PolicyEngine, wsRoot string) {
	runner := &tuiAgentRunner{
		llm:         llm,
		tools:       tools,
		permChecker: permChecker,
		sysPrompt:   buildSystemPrompt(wsRoot, nil),
		maxSteps:    *maxSteps,
	}

	cfg := tui.Config{
		Workspace: wsRoot,
		Model:     *modelName,
		Provider:  *provider,
		MaxSteps:  *maxSteps,
	}

	projectInfo := fmt.Sprintf("Workspace: %s", wsRoot)
	if proj, err := baizecontext.Analyze(wsRoot, baizecontext.DefaultAnalysisOptions()); err == nil && len(proj.Languages) > 0 {
		projectInfo = proj.Summary()
	}

	model := tui.New(runner, cfg, projectInfo)
	model.SetOnAlwaysAllow(func(toolName string) {
		policyEngine.Learn(permission.DecisionRecord{
			Decision: permission.DecisionAllow,
			Scope:    permission.ScopeAlways,
			Reason:   "user chose always allow for " + toolName,
		})
	})
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("TUI: %v", err)
	}
}

// tuiAgentRunner adapts the agent to tui.StreamRunner.
type tuiAgentRunner struct {
	llm         agent.LLMModel
	tools       []agent.Tool
	permChecker agent.PermissionChecker
	sysPrompt   string
	maxSteps    int
}

func (r *tuiAgentRunner) RunStream(ctx context.Context, input string, history []tui.ChatMsg, onEvent func(tui.StreamEvent)) {
	// Build an AskFunc that bridges ToolNode "ask" decisions to the TUI confirm modal.
	askFunc := func(ctx context.Context, toolName string, args map[string]any, reason string) bool {
		respCh := make(chan bool, 1)
		onEvent(tui.StreamEvent{
			Type:        "permission_ask",
			Content:     reason,
			ToolName:    toolName,
			ToolArgs:    fmt.Sprintf("%v", args),
			ConfirmChan: respCh,
		})
		select {
		case allowed := <-respCh:
			return allowed
		case <-ctx.Done():
			return false
		}
	}

	// Convert TUI chat history to agent messages.
	var messages []agent.Message
	for _, h := range history {
		role := agent.RoleUser
		if h.Role == "assistant" {
			role = agent.RoleAssistant
		}
		if h.Content != "" {
			messages = append(messages, agent.Message{Role: role, Content: h.Content})
		}
	}
	messages = append(messages, agent.Message{Role: agent.RoleUser, Content: input})

	ag := agent.NewReActAgent(agent.ReActAgentConfig{
		Name:         "baize-tui",
		LLM:          r.llm,
		SystemPrompt: r.sysPrompt,
		Tools:        r.tools,
		MaxSteps:     r.maxSteps,
		PermChecker:  r.permChecker,
		AskFunc:      askFunc,
		Stream:       true,
		OnChunk: func(chunk *agent.StreamChunk) {
			if chunk.Content != "" {
				onEvent(tui.StreamEvent{Type: "thought", Content: chunk.Content})
			}
		},
	})
	g, err := ag.BuildGraph()
	if err != nil {
		onEvent(tui.StreamEvent{Type: "error", Content: err.Error()})
		return
	}

	engine := graph.NewEngine(g)
	hook := &tuiStreamHook{onEvent: onEvent}
	state := &agent.MessageState{
		Messages: messages,
		MaxSteps: r.maxSteps,
	}
	result, err := engine.Run(ctx, state, graph.WithHook(hook))
	if err != nil {
		onEvent(tui.StreamEvent{Type: "error", Content: err.Error()})
		return
	}

	if content := lastAssistantContent(result.FinalState); content != "" {
		onEvent(tui.StreamEvent{Type: "answer", Content: content})
	}
	onEvent(tui.StreamEvent{Type: "done", Tokens: result.FinalState.TotalTokens})
}

type tuiStreamHook struct{ onEvent func(tui.StreamEvent) }

func (h *tuiStreamHook) OnGraphStart(_ context.Context, _ string, _ *agent.MessageState)        {}
func (h *tuiStreamHook) OnGraphEnd(_ context.Context, _ string, _ *agent.MessageState, _ error) {}
func (h *tuiStreamHook) OnNodeStart(_ context.Context, _ string, _ *agent.MessageState)         {}
func (h *tuiStreamHook) OnNodeEnd(_ context.Context, _ string, s *agent.MessageState, _ error, _ time.Duration) {
	if len(s.Messages) == 0 {
		return
	}
	last := s.Messages[len(s.Messages)-1]
	switch {
	case len(last.ToolCalls) > 0:
		for _, tc := range last.ToolCalls {
			h.onEvent(tui.StreamEvent{
				Type:     "tool_call",
				ToolName: tc.Name,
				ToolArgs: fmt.Sprintf("%v", tc.Arguments),
			})
		}
	case last.Role == agent.RoleTool:
		h.onEvent(tui.StreamEvent{Type: "tool_result", Content: last.Content})
	}
}
func (h *tuiStreamHook) OnRetry(_ context.Context, _ string, _ int, _ error) {}

// ── LLM ─────────────────────────────────────────────────────────────────────

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

// ── Tools ───────────────────────────────────────────────────────────────────

func buildSkillManager(ctx context.Context) *skill.Manager {
	home, _ := os.UserHomeDir()
	skillsDir := filepath.Join(home, ".baize", "skills")
	sm := skill.NewManager(skillsDir)
	_ = sm.Load(skillsDir)
	_ = sm.Start(ctx)
	return sm
}

func buildToolRegistry(wsRoot string, sm *skill.Manager) *tool.ToolRegistry {
	reg := tool.NewToolRegistry()
	reg.Register(&builtin.CalculatorTool{})
	reg.Register(&builtin.FileTool{WorkspaceRoot: wsRoot})
	reg.Register(&builtin.GrepTool{WorkspaceRoot: wsRoot})
	reg.Register(&builtin.ShellTool{WorkspaceRoot: wsRoot, MaxRuntime: 120 * time.Second})
	reg.Register(&builtin.GitTool{WorkspaceRoot: wsRoot})
	reg.Register(&builtin.WebSearchTool{})
	reg.Register(&builtin.WebFetchTool{})

	// Agent auto-memory tool.
	home, _ := os.UserHomeDir()
	memDir := filepath.Join(home, ".baize", "projects", slugifyPath(wsRoot), "memory")
	if am, err := memory.NewAutoMemory(memDir); err == nil {
		reg.Register(&builtin.MemorySaveTool{AutoMemory: am})
	}

	// Register tools from skill MCP servers + the activate_skill tool.
	for _, t := range sm.Tools() {
		reg.Register(t)
	}

	return reg
}

func slugifyPath(p string) string {
	p = strings.ReplaceAll(p, "/", "-")
	p = strings.ReplaceAll(p, "\\", "-")
	if len(p) > 60 {
		p = p[len(p)-60:]
	}
	return strings.Trim(p, "-")
}

func buildTools(wsRoot string) []agent.Tool {
	sm := buildSkillManager(context.Background())
	return buildToolRegistry(wsRoot, sm).List()
}

func buildSystemPrompt(wsRoot string, sm *skill.Manager) string {
	hostname, _ := os.Hostname()
	base := fmt.Sprintf(`你是白泽（Baize），一名专为软件工程师打造的 AI 编程助手。

你运行在思考-行动循环中：分析问题，选择工具，执行操作，观察结果，迭代直到任务完成。

== 可用工具 ==
- file: 文件读写、编辑（字符串替换）、目录列表、glob 搜索。所有路径相对于工作区。
- grep: 正则/字符串搜索文件内容，返回文件名:行号:匹配内容。支持 --include 过滤文件类型。
- shell: 执行 Shell 命令（工作区 %s）。超时 120s。危险命令已屏蔽。
- git: status、diff、log、add、commit、branch、checkout。
- web_search: 通过 DuckDuckGo 搜索网页。返回前 10 条结果（标题+URL+摘要）。
- web_fetch: 抓取 URL 并提取可读文本（HTML 转文本，8KB 限制）。
- calculator: 计算算术表达式（+、-、*、/、括号）。

== 行为准则 ==
- 复杂任务拆分成清晰步骤，逐步执行。
- 编辑之前先阅读文件，理解代码再修改。
- 精准修改，优先选择字符串替换而非全文重写。
- 执行 Shell 命令前说明目的。
- 需要最新信息或文档时使用 web_search。
- 简洁为上，代码胜于文字。
- 不知道就说不知道，不要猜测 API 或语法。

当前工作区：%s
主机：%s`, wsRoot, wsRoot, hostname)

	// Append project-level instructions from BAIZE.md files.
	loader := &baizecontext.InstructionLoader{ProjectRoot: wsRoot}
	if instructions := loader.Load(); instructions != "" {
		base += "\n\n== 项目指令 ==\n" + instructions
	}

	// Append skill index (name+description only) from the passed manager.
	if sm != nil {
		if idx := sm.SystemPromptIndex(); idx != "" {
			base += "\n\n" + idx
		}
	}

	return base
}

// ── CLI Hook ────────────────────────────────────────────────────────────────

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

// ── Helpers ─────────────────────────────────────────────────────────────────

func usage() {
	fmt.Fprintf(os.Stderr, `Baize — Unified AI Agent Platform  (v%s, %s)

Usage:
  baize [flags] "question"     Single-shot agent query.
  baize [flags]                 Interactive mode.
  baize server [flags]          API server.

Flags:
`, version.Version, version.Commit)
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
