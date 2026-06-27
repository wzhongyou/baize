package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ── Streaming integration ──────────────────────────────────────────────────

// startAgent returns a tea.Cmd that runs the agent and feeds streamEvent
// messages back into the Bubble Tea event loop.
func (m *Model) startAgent(text string) tea.Cmd {
	if text == "" {
		return nil
	}

	// Create a channel for events.
	ch := make(chan tea.Msg, 64)
	m.eventChan = ch

	go func() {
		defer close(ch)

		m.runner.RunStream(m.ctx, text, func(ev StreamEvent) {
			se := streamEvent{
				Type:     ev.Type,
				Content:  ev.Content,
				ToolName: ev.ToolName,
				Tokens:   ev.Tokens,
			}
			if ev.Type == "permission_ask" {
				se.PermissionResponse = ev.ConfirmChan
			}
			select {
			case ch <- se:
			case <-m.ctx.Done():
				return
			}
		})
	}()

	return waitForEvent(ch)
}

// waitForEvent returns a Cmd that reads the next message from a channel.
func waitForEvent(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return streamDone{Content: ""}
		}
		return msg
	}
}

// ── Permission integration ─────────────────────────────────────────────────

// requestPermission sends a permission confirmation request to the UI.
func (m *Model) requestPermission(tool, question string) tea.Cmd {
	ch := make(chan bool, 1)
	return func() tea.Msg {
		return permissionMsg{
			Tool:      tool,
			Question:  question,
			Confirmed: ch,
		}
	}
}

