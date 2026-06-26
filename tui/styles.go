package tui

import "github.com/charmbracelet/lipgloss"

// Theme colours.
var (
	primary   = lipgloss.Color("#7C3AED") // purple
	secondary = lipgloss.Color("#06B6D4") // cyan
	success   = lipgloss.Color("#10B981") // green
	warning   = lipgloss.Color("#F59E0B") // amber
	danger    = lipgloss.Color("#EF4444") // red
	mutedColor = lipgloss.Color("#6B7280") // gray-500
	dark      = lipgloss.Color("#1F2937")  // gray-800
	bg        = lipgloss.Color("#0F172A")  // slate-900

	// Styles built from colors.
	mutedStyle = lipgloss.NewStyle().Foreground(mutedColor)
)

// Shared styles.
var (
	containerStyle = lipgloss.NewStyle().
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Background(primary).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 2).
			Height(1)

	userBubble = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#93C5FD")). // blue-300
			Bold(true)

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E2E8F0")) // slate-200

	thinkingStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	toolCallStyle = lipgloss.NewStyle().
			Foreground(secondary).
			Bold(true)

	toolResultStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			PaddingLeft(4)

	diffAdd = lipgloss.NewStyle().
		Foreground(success)

	diffDel = lipgloss.NewStyle().
		Foreground(danger)

	diffHunk = lipgloss.NewStyle().
		Foreground(secondary)

	errorStyle = lipgloss.NewStyle().
			Foreground(danger).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(secondary)

	promptStyle = lipgloss.NewStyle().
			Foreground(warning).
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(warning).
			Padding(0, 1)

	inputPrompt = lipgloss.NewStyle().
			Foreground(primary).
			Bold(true)
)

// RenderAligned renders left and right aligned text in a given width.
func RenderAligned(width int, left, right string) string {
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := width - leftW - rightW - 2 // -2 for padding
	if gap < 1 {
		gap = 1
	}
	return left + lipgloss.NewStyle().Width(gap).Render(" ") + right
}
