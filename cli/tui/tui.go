package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Configuration ──────────────────────────────────────────────────────────

type Config struct {
	SessionID string
	Workspace string
	Model     string
	Provider  string
	MaxSteps  int
	ServerURL string
}

type StreamRunner interface {
	RunStream(ctx context.Context, input string, onEvent func(StreamEvent))
}

type StreamEvent struct {
	Type        string
	Content     string
	ToolName    string
	ToolArgs    string
	Tokens      int
	ConfirmChan chan bool
}

// ── Model ──────────────────────────────────────────────────────────────────

type Model struct {
	cfg         Config
	width       int
	height      int
	projectInfo string

	mode             uiMode
	startupSelection int
	ready            bool
	quitting         bool

	// Chat
	messages    []ChatMsg
	thinkingBuf strings.Builder
	streaming   bool
	eventChan   chan tea.Msg
	totalSteps  int
	maxSteps    int
	totalTokens int
	viewport    viewport.Model
	textarea    textarea.Model

	// Input history
	history []string
	histIdx int

	// Permission
	permPrompt    string
	permTool      string
	permConfirm   chan bool
	onAlwaysAllow func(toolName string)

	// Runner
	runner StreamRunner
	ctx    context.Context
	cancel context.CancelFunc
}

// ── Constructor ────────────────────────────────────────────────────────────

func New(runner StreamRunner, cfg Config, projectInfo string) *Model {
	ta := textarea.New()
	ta.Placeholder = "输入问题，或 / 查看命令..."
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.MaxHeight = 4
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()
	ta.KeyMap.InsertNewline.SetEnabled(false)
	ta.Focus()

	m := &Model{
		cfg:         cfg,
		mode:        modeStartup,
		messages:    make([]ChatMsg, 0),
		history:     make([]string, 0),
		runner:      runner,
		maxSteps:    cfg.MaxSteps,
		textarea:    ta,
		projectInfo: projectInfo,
	}
	if m.maxSteps <= 0 {
		m.maxSteps = 30
	}
	m.viewport = viewport.New(80, 24)
	return m
}

func (m *Model) SetOnAlwaysAllow(fn func(toolName string)) {
	m.onAlwaysAllow = fn
}

// ── Bubble Tea Interface ───────────────────────────────────────────────────

func (m *Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 3
		m.textarea.SetWidth(msg.Width - 2)
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		if m.mode == modeStartup {
			return m.startupUpdate(msg)
		}
		if m.mode == modeConfirm {
			return m.confirmUpdate(msg)
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			if m.streaming {
				m.cancel()
				m.streaming = false
				m.mode = modeInput
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if m.mode == modeInput {
				text := strings.TrimSpace(m.textarea.Value())
				if text != "" {
					return m.handleSubmit(text)
				}
			}
			return m, nil

		case "pgup":
			m.viewport.PageUp()
			return m, nil
		case "pgdown":
			m.viewport.PageDown()
			return m, nil
		case "home":
			m.viewport.GotoTop()
			return m, nil
		case "end":
			m.viewport.GotoBottom()
			return m, nil

		case "up":
			if m.mode == modeInput && m.histIdx < len(m.history) {
				m.histIdx++
				idx := len(m.history) - m.histIdx
				m.textarea.SetValue(m.history[idx])
				m.textarea.CursorEnd()
			}
			return m, nil
		case "down":
			if m.mode == modeInput && m.histIdx > 0 {
				m.histIdx--
				if m.histIdx == 0 {
					m.textarea.SetValue("")
				} else {
					idx := len(m.history) - m.histIdx
					m.textarea.SetValue(m.history[idx])
					m.textarea.CursorEnd()
				}
			}
			return m, nil
		}

		// Other keys: forward to textarea
		if m.mode == modeInput {
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}

	case streamEvent:
		m.handleStreamEvent(msg)
		if msg.Type == "done" || msg.Type == "error" {
			m.streaming = false
			m.mode = modeInput
		} else {
			// Continue reading from the event channel
			return m, waitForEvent(m.eventChan)
		}

	case streamDone:
		m.thinkingBuf.Reset()
		m.streaming = false
		m.mode = modeInput

	case permissionMsg:
		m.mode = modeConfirm
		m.permTool = msg.Tool
		m.permPrompt = msg.Question
		m.permConfirm = msg.Confirmed

	case tickMsg:
	}

	return m, nil
}

func (m *Model) View() string {
	if m.quitting {
		return "再见。\n"
	}
	if !m.ready {
		return "启动中...\n"
	}

	status := m.statusView()

	if m.mode == modeStartup {
		return lipgloss.JoinVertical(lipgloss.Left,
			status,
			"",
			m.startupView(),
		)
	}

	if m.mode == modeConfirm {
		return m.confirmView()
	}

	chat := m.chatView()
	m.viewport.SetContent(chat)
	if m.streaming {
		m.viewport.GotoBottom()
	}

	input := ""
	if m.mode == modeInput {
		input = m.textarea.View()
	} else {
		input = mutedStyle.Render("  思考中...")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		status,
		m.viewport.View(),
		input,
	)
}

// ── Handlers ───────────────────────────────────────────────────────────────

func (m *Model) handleSubmit(text string) (tea.Model, tea.Cmd) {
	if strings.HasPrefix(text, "/") {
		return m.handleCommand(text)
	}

	m.history = append(m.history, text)
	m.histIdx = 0

	m.messages = append(m.messages, ChatMsg{Role: "user", Content: text})
	m.textarea.Reset()

	m.mode = modeThinking
	m.streaming = true
	m.thinkingBuf.Reset()
	m.totalSteps = 0
	m.totalTokens = 0

	ctx, cancel := context.WithCancel(context.Background())
	m.ctx = ctx
	m.cancel = cancel

	return m, m.startAgent(text)
}

func (m *Model) handleStreamEvent(ev streamEvent) {
	switch ev.Type {
	case "thought":
		m.thinkingBuf.WriteString(ev.Content)
	case "tool_call":
		m.totalSteps++
		m.messages = append(m.messages, ChatMsg{
			Role: "tool",
			ToolCalls: []ToolCallMsg{{
				Name: ev.ToolName,
				Args: ev.Content,
			}},
		})
	case "tool_result":
		if len(m.messages) > 0 {
			last := &m.messages[len(m.messages)-1]
			if last.Role == "tool" && len(last.ToolCalls) > 0 {
				if ev.Error != "" {
					last.ToolCalls[len(last.ToolCalls)-1].Error = ev.Error
				} else {
					last.ToolCalls[len(last.ToolCalls)-1].Result = ev.Content
				}
			}
		}
	case "answer":
		m.thinkingBuf.Reset()
		m.messages = append(m.messages, ChatMsg{
			Role:    "assistant",
			Content: ev.Content,
		})
	case "done":
		m.totalTokens = ev.Tokens
		if m.thinkingBuf.Len() > 0 {
			m.messages = append(m.messages, ChatMsg{
				Role:    "assistant",
				Content: m.thinkingBuf.String(),
			})
			m.thinkingBuf.Reset()
		}
	case "permission_ask":
		m.mode = modeConfirm
		m.permTool = ev.ToolName
		m.permPrompt = ev.Content
		m.permConfirm = ev.PermissionResponse
	case "error":
		m.messages = append(m.messages, ChatMsg{
			Role:    "system",
			Content: "Error: " + ev.Error,
		})
	}
}

func (m *Model) handleCommand(text string) (tea.Model, tea.Cmd) {
	switch {
	case text == "/quit" || text == "/exit":
		m.quitting = true
		return m, tea.Quit
	case text == "/help":
		m.messages = append(m.messages, ChatMsg{
			Role:    "system",
			Content: helpText,
		})
	case text == "/clear":
		m.messages = make([]ChatMsg, 0)
	case text == "/model" && m.cfg.Model != "":
		m.messages = append(m.messages, ChatMsg{
			Role:    "system",
			Content: fmt.Sprintf("当前模型: %s (%s)", m.cfg.Model, m.cfg.Provider),
		})
	case text == "/workspace":
		m.messages = append(m.messages, ChatMsg{
			Role:    "system",
			Content: m.projectInfo,
		})
	default:
		m.messages = append(m.messages, ChatMsg{
			Role:    "system",
			Content: fmt.Sprintf("未知命令: %s。输入 /help 查看帮助。", text),
		})
	}
	m.textarea.Reset()
	return m, nil
}

func (m *Model) startupUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "up", "down", "k", "j":
		m.startupSelection = 1 - m.startupSelection
		return m, nil
	case "enter":
		if m.startupSelection == 0 {
			m.mode = modeInput
			m.messages = append(m.messages, ChatMsg{
				Role:    "assistant",
				Content: fmt.Sprintf("你好！工作区 %s 就绪。\n输入问题开始，或 / 查看命令。", m.cfg.Workspace),
			})
		} else {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	case "esc", "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) confirmUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y":
		m.permConfirm <- true
		m.mode = modeThinking
	case "n", "esc", "ctrl+c":
		m.permConfirm <- false
		m.mode = modeThinking
	case "a":
		m.permConfirm <- true
		if m.onAlwaysAllow != nil {
			m.onAlwaysAllow(m.permTool)
		}
		m.mode = modeThinking
	}
	return m, nil
}

const helpText = `命令:
  /help           显示帮助
  /quit, /exit    退出
  /clear          清空对话
  /model          显示当前模型
  /workspace      显示工作区信息
  Ctrl+C          取消当前操作
  Up/Down         浏览历史
  PageUp/Down     滚动对话
  Home/End        跳到顶部/底部`

var availableCommands = []string{
	"/help", "/quit", "/exit", "/clear", "/model", "/workspace",
}

func filteredCommands(prefix string) string {
	var matches []string
	for _, cmd := range availableCommands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}
	return strings.Join(matches, "  ")
}
