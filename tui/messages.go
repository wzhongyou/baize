package tui

import "time"

// Bubble Tea messages flowing through Update.

type tickMsg time.Time

// streamEvent is sent when the agent produces an event.
type streamEvent struct {
	Type     string
	Content  string
	ToolName string
	Tokens   int
	Error    string
}

// streamDone signals the end of agent streaming.
type streamDone struct {
	Content string
	Steps   int
	Tokens  int
}

// permissionMsg asks the user to confirm an action.
type permissionMsg struct {
	Tool      string
	Question  string
	Confirmed chan bool
}

// ── Chat message for display ──────────────────────────────────────────────

// ChatMsg is a rendered message in the chat viewport.
type ChatMsg struct {
	Role      string
	Content   string
	ToolCalls []ToolCallMsg
}

// ToolCallMsg records a tool invocation and its result.
type ToolCallMsg struct {
	Name   string
	Args   string
	Result string
	Error  string
}

// ── UI mode ────────────────────────────────────────────────────────────────

type uiMode int

const (
	modeInput    uiMode = iota
	modeThinking
	modeConfirm
)

func (m uiMode) String() string {
	switch m {
	case modeInput:
		return "INPUT"
	case modeThinking:
		return "THINKING"
	case modeConfirm:
		return "CONFIRM"
	}
	return "UNKNOWN"
}
