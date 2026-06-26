package tui

import (
	"fmt"
	"strings"
)

// ── Status Bar ─────────────────────────────────────────────────────────────

func (m *Model) statusView() string {
	left := fmt.Sprintf("Baize v0.4")
	mid := ""
	if m.cfg.SessionID != "" {
		mid = m.cfg.SessionID
	}
	if m.cfg.Model != "" {
		mid += " • " + m.cfg.Model
	}

	mode := m.mode.String()
	switch m.mode {
	case modeInput:
		mode = "◉ READY"
	case modeThinking:
		mode = "⋯ THINKING"
	case modeConfirm:
		mode = "? CONFIRM"
	}

	right := mode
	// Render with aligned left/mid/right.
	content := RenderAligned(m.width, left+"  "+mid, right)
	return statusBarStyle.Width(m.width).Render(content)
}

// ── Chat View ──────────────────────────────────────────────────────────────

func (m *Model) chatView() string {
	var sb strings.Builder

	// Render all messages.
	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			sb.WriteString(userBubble.Render("▸ "+msg.Content) + "\n")
		case "assistant":
			sb.WriteString(assistantStyle.Render(msg.Content) + "\n")
		case "tool":
			for _, tc := range msg.ToolCalls {
				sb.WriteString(toolCallStyle.Render(fmt.Sprintf("  → %s", tc.Name)))
				if tc.Args != "" {
					sb.WriteString(" " + mutedStyle.Render(truncate(tc.Args, 60)))
				}
				sb.WriteString("\n")
				if tc.Result != "" {
					sb.WriteString(toolResultStyle.Render(truncate(tc.Result, 120)) + "\n")
				}
				if tc.Error != "" {
					sb.WriteString(errorStyle.Render("  ✗ "+tc.Error) + "\n")
				}
			}
		case "system":
			sb.WriteString(helpStyle.Render("  "+msg.Content) + "\n")
		}
	}

	// Live thinking indicator.
	if m.streaming && m.thinkingBuf.Len() > 0 {
		sb.WriteString(thinkingStyle.Render(m.thinkingBuf.String()))
	}

	return sb.String()
}

// ── Input View ─────────────────────────────────────────────────────────────

func (m *Model) inputView() string {
	if m.mode != modeInput {
		// Show a dimmed input while thinking.
		return mutedStyle.Render("  Waiting for response...")
	}

	prompt := inputPrompt.Render("> ")
	text := m.input.String()
	if text == "" {
		return prompt + mutedStyle.Render("Type your message...")
	}

	// Render cursor.
	cursor := m.cursor
	if cursor > len(text) {
		cursor = len(text)
	}
	before := text[:cursor]
	after := text[cursor:]
	cursorChar := "█"

	return prompt + before + inputPrompt.Render(cursorChar) + after
}

// ── Confirm View ───────────────────────────────────────────────────────────

func (m *Model) confirmView() string {
	content := fmt.Sprintf("⚠ %s\n\n%s",
		promptStyle.Render("Permission Required"),
		m.permPrompt,
	)

	controls := "\n\n" +
		"[Y] Allow  [N] Deny  [A] Always Allow  [Esc] Cancel"

	return promptStyle.Render(content + controls)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		return s[:maxLen] + "…"
	}
	return s
}
