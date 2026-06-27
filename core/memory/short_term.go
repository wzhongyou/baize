// Package memory provides short-term and long-term memory implementations
// for the Baize agent platform.
package memory

// Message is a minimal copy of agent.Message used to avoid an import cycle.
// Callers should convert agent.Message → memory.Message before calling Add.
type Message struct {
	Role    string
	Content string
}

const RoleSystem = "system"

// ShortTermMemory keeps the recent conversation window in memory.
type ShortTermMemory struct {
	maxMessages int
	messages    []Message
}

// NewShortTermMemory creates a memory buffer capped at maxMessages.
func NewShortTermMemory(maxMessages int) *ShortTermMemory {
	return &ShortTermMemory{maxMessages: maxMessages}
}

// Add appends a message, evicting the oldest non-system messages when over limit.
func (m *ShortTermMemory) Add(msg Message) {
	m.messages = append(m.messages, msg)
	if len(m.messages) <= m.maxMessages {
		return
	}
	idx := 0
	for idx < len(m.messages) && len(m.messages) > m.maxMessages {
		if m.messages[idx].Role != RoleSystem {
			m.messages = append(m.messages[:idx], m.messages[idx+1:]...)
		} else {
			idx++
		}
	}
	if len(m.messages) > m.maxMessages {
		excess := len(m.messages) - m.maxMessages
		m.messages = m.messages[excess:]
	}
}

// Messages returns the current message window.
func (m *ShortTermMemory) Messages() []Message { return m.messages }
