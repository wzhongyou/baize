// Package tui provides a Bubble Tea terminal UI for the Baize agent platform.
package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Configuration ──────────────────────────────────────────────────────────

// Config configures the TUI session.
type Config struct {
	SessionID    string
	Workspace    string
	Model        string
	Provider     string
	ServerURL    string // remote Baize server; empty = embedded
}

// StreamRunner is the interface for running an agent with streaming events.
type StreamRunner interface {
	RunStream(ctx context.Context, input string, onEvent func(StreamEvent))
}

// StreamEvent is emitted during agent execution.
type StreamEvent struct {
	Type     string // "thought", "tool_call", "tool_result", "answer", "done", "error"
	Content  string
	ToolName string
	ToolArgs string
	Tokens   int
}

// ── Model ──────────────────────────────────────────────────────────────────

// Model is the top-level Bubble Tea model.
type Model struct {
	cfg    Config
	width  int
	height int

	mode    uiMode
	ready   bool
	quitting bool

	// Chat state.
	messages   []ChatMsg
	thinkingBuf strings.Builder
	streaming  bool

	// Input.
	input    strings.Builder
	cursor   int
	history  []string
	histIdx  int

	// Permission.
	permPrompt   string
	permTool     string
	permConfirm  chan bool

	// Runner.
	runner StreamRunner
	ctx    context.Context
	cancel context.CancelFunc
}

// ── Constructor ────────────────────────────────────────────────────────────

// New creates a new TUI model.
func New(runner StreamRunner, cfg Config) *Model {
	m := &Model{
		cfg:      cfg,
		mode:     modeInput,
		messages: make([]ChatMsg, 0),
		history:  make([]string, 0),
		runner:   runner,
	}
	return m
}

// ── Bubble Tea Interface ───────────────────────────────────────────────────

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
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
			if m.mode == modeInput && m.input.Len() > 0 {
				return m.handleSubmit()
			}

		case "ctrl+u":
			m.input.Reset()
			m.cursor = 0

		case "backspace":
			if m.cursor > 0 {
				s := m.input.String()
				m.input.Reset()
				m.input.WriteString(s[:m.cursor-1] + s[m.cursor:])
				m.cursor--
			}

		case "ctrl+a":
			m.cursor = 0

		case "ctrl+e":
			m.cursor = m.input.Len()

		case "left":
			if m.cursor > 0 {
				m.cursor--
			}

		case "right":
			if m.cursor < m.input.Len() {
				m.cursor++
			}

		case "up":
			if m.histIdx < len(m.history) {
				m.histIdx++
				idx := len(m.history) - m.histIdx
				m.input.Reset()
				m.input.WriteString(m.history[idx])
				m.cursor = m.input.Len()
			}

		case "down":
			if m.histIdx > 0 {
				m.histIdx--
				if m.histIdx == 0 {
					m.input.Reset()
				} else {
					idx := len(m.history) - m.histIdx
					m.input.Reset()
					m.input.WriteString(m.history[idx])
				}
				m.cursor = m.input.Len()
			}

		default:
			if m.mode == modeInput && len(msg.Runes) == 1 {
				r := msg.Runes[0]
				if r >= 32 {
					s := m.input.String()
					m.input.Reset()
					m.input.WriteString(s[:m.cursor] + string(r) + s[m.cursor:])
					m.cursor++
				}
			}
		}

	case streamEvent:
		m.handleStreamEvent(msg)
		if msg.Type == "done" || msg.Type == "error" {
			m.streaming = false
			m.mode = modeInput
		}

	case streamDone:
		m.messages = append(m.messages, ChatMsg{
			Role:    "assistant",
			Content: msg.Content,
		})
		m.thinkingBuf.Reset()
		m.streaming = false
		m.mode = modeInput

		return m, nil

	case permissionMsg:
		m.mode = modeConfirm
		m.permTool = msg.Tool
		m.permPrompt = msg.Question
		m.permConfirm = msg.Confirmed

	case tickMsg:
		// No-op for now; used if we add a spinner.
	}

	return m, nil
}

func (m *Model) View() string {
	if m.quitting {
		return "Goodbye.\n"
	}
	if !m.ready {
		return "Starting Baize...\n"
	}

	status := m.statusView()
	chat := m.chatView()
	input := m.inputView()

	// Permission modal overlay.
	if m.mode == modeConfirm {
		return m.confirmView()
	}

	chatHeight := m.height - lipgloss.Height(status) - lipgloss.Height(input) - 1
	chatStyle := lipgloss.NewStyle().Height(chatHeight).Width(m.width)

	return lipgloss.JoinVertical(lipgloss.Left,
		status,
		chatStyle.Render(chat),
		input,
	)
}

// ── Handlers ───────────────────────────────────────────────────────────────

func (m *Model) handleSubmit() (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(m.input.String())
	if text == "" {
		return m, nil
	}

	// Slash commands.
	if strings.HasPrefix(text, "/") {
		return m.handleCommand(text)
	}

	// Add to history.
	m.history = append(m.history, text)
	m.histIdx = 0

	// Add user message.
	m.messages = append(m.messages, ChatMsg{Role: "user", Content: text})
	m.input.Reset()
	m.cursor = 0

	// Start agent with streaming events fed back to the Bubble Tea loop.
	m.mode = modeThinking
	m.streaming = true
	m.thinkingBuf.Reset()

	ctx, cancel := context.WithCancel(context.Background())
	m.ctx = ctx
	m.cancel = cancel

	return m, m.startAgent()
}

func (m *Model) handleStreamEvent(ev streamEvent) {
	switch ev.Type {
	case "thought":
		m.thinkingBuf.WriteString(ev.Content)
	case "tool_call":
		// Create a new message for the tool call.
		m.messages = append(m.messages, ChatMsg{
			Role: "tool",
			ToolCalls: []ToolCallMsg{{
				Name: ev.ToolName,
				Args: ev.Content,
			}},
		})
	case "tool_result":
		// Append result to last tool call message.
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
		// Flush any remaining thinking text as answer if needed.
		if m.thinkingBuf.Len() > 0 {
			m.messages = append(m.messages, ChatMsg{
				Role:    "assistant",
				Content: m.thinkingBuf.String(),
			})
			m.thinkingBuf.Reset()
		}
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
			Content: fmt.Sprintf("Current model: %s (%s)", m.cfg.Model, m.cfg.Provider),
		})
	default:
		m.messages = append(m.messages, ChatMsg{
			Role:    "system",
			Content: fmt.Sprintf("Unknown command: %s. Type /help for commands.", text),
		})
	}
	m.input.Reset()
	m.cursor = 0
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
		// "Always" — allow and remember.
		m.permConfirm <- true
		m.mode = modeThinking
	}
	return m, nil
}

const helpText = `Commands:
  /help           Show this help
  /quit, /exit    Exit Baize
  /clear          Clear chat history
  /model          Show current model
  Ctrl+C          Cancel current operation
  Ctrl+U          Clear input line
  Ctrl+A/E        Jump to start/end of line
  Up/Down         Navigate history`
