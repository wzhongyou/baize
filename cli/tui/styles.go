package tui

import "github.com/charmbracelet/lipgloss"

// Terminal-native styles — no hardcoded colors, adapts to user's theme.
var (
	statusBarStyle = lipgloss.NewStyle().
			Reverse(true).
			Bold(true).
			Padding(0, 1).
			Height(1)

	userBubble = lipgloss.NewStyle().
			Bold(true)

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")) // bright black = gray in most themes

	thinkingStyle = lipgloss.NewStyle().
			Faint(true).
			Italic(true)

	toolCallStyle = lipgloss.NewStyle().
			Bold(true)

	toolResultStyle = lipgloss.NewStyle().
			Faint(true).
			PaddingLeft(4)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")). // red
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Faint(true)

	promptStyle = lipgloss.NewStyle().
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

	inputPrompt = lipgloss.NewStyle().
			Bold(true)

	mutedStyle = lipgloss.NewStyle().
			Faint(true)
)

// RenderAligned renders left and right aligned text in a given width.
func RenderAligned(width int, left, right string) string {
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := width - leftW - rightW - 2
	if gap < 1 {
		gap = 1
	}
	return left + lipgloss.NewStyle().Width(gap).Render(" ") + right
}
