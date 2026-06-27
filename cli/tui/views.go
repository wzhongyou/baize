package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

	// Build right side: steps/tokens + mode indicator
	stats := ""
	if m.totalTokens > 0 || m.totalSteps > 0 {
		stats = fmt.Sprintf("steps: %d/%d  tokens: %d  ", m.totalSteps, m.maxSteps, m.totalTokens)
	}
	mode := m.mode.String()
	switch m.mode {
	case modeStartup:
		mode = "▶ START"
	case modeInput:
		mode = "◉ READY"
	case modeThinking:
		mode = "⋯ THINKING"
	case modeConfirm:
		mode = "? CONFIRM"
	}

	right := stats + mode
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
					sb.WriteString(" " + mutedStyle.Render(truncate(tc.Args, 120)))
				}
				sb.WriteString("\n")
				if tc.Result != "" {
					sb.WriteString(toolResultStyle.Render(truncate(tc.Result, 500)) + "\n")
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

// ── Startup View ───────────────────────────────────────────────────────────

func (m *Model) startupView() string {
	ws := m.cfg.Workspace

	itemStyle := func(idx int) lipgloss.Style {
		if m.startupSelection == idx {
			return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#10B981"))
		}
		return mutedStyle
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("Baize（白泽）"),
		lipgloss.NewStyle().Faint(true).Render("你的终端 AI 编程助手"),
		"",
		fmt.Sprintf("当前目录  %s", ws),
		mutedStyle.Render("检测到这是一个项目目录。"),
		"",
		itemStyle(0).Render("  1. 开始对话"),
		itemStyle(1).Render("  2. 退出"),
		"",
		mutedStyle.Render("↑↓ 选择  Enter 确认  Esc 退出"),
	)
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
