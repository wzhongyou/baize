package agent

// ContextBudget trims message history to stay within a token budget.
// It uses a simple heuristic: 1 token ≈ 4 chars.
//
// Strategy:
//   - Always preserve system messages.
//   - Keep the most recent messages up to the budget.
//   - When trimming, drop oldest non-system messages first.
//   - If even recent messages exceed the budget, truncate the oldest
//     kept message's content to fit.
type ContextBudget struct {
	// MaxHistoryTokens is the token budget for conversation history.
	// Defaults to 60_000 (safe for most 128K context models).
	MaxHistoryTokens int
}

func DefaultContextBudget() *ContextBudget {
	return &ContextBudget{MaxHistoryTokens: 60_000}
}

// Trim returns a trimmed copy of msgs that fits within the budget.
// System messages are always kept. The most recent messages are preferred.
func (b *ContextBudget) Trim(msgs []Message) []Message {
	limit := b.MaxHistoryTokens
	if limit <= 0 {
		limit = 60_000
	}

	// Separate system messages from the rest.
	var system, rest []Message
	for _, m := range msgs {
		if m.Role == RoleSystem {
			system = append(system, m)
		} else {
			rest = append(rest, m)
		}
	}

	// Walk from newest to oldest, accumulating within budget.
	budget := limit
	for _, m := range system {
		budget -= estimateTokens(m.Content)
	}
	if budget <= 0 {
		return system
	}

	kept := make([]Message, 0, len(rest))
	for i := len(rest) - 1; i >= 0; i-- {
		cost := estimateTokens(rest[i].Content)
		if budget-cost < 0 {
			break
		}
		budget -= cost
		kept = append([]Message{rest[i]}, kept...)
	}

	return append(system, kept...)
}

// estimateTokens estimates token count from char length (1 token ≈ 4 chars).
func estimateTokens(s string) int {
	if len(s) == 0 {
		return 0
	}
	t := len(s) / 4
	if t == 0 {
		return 1
	}
	return t
}
