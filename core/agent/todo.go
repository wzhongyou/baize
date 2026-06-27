package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TodoManager maintains a .baize/todo.md file that tracks multi-step task progress.
// The current todo content is injected near the end of context on each LLM call
// to keep goals in the model's recent attention window (Manus pattern).
type TodoManager struct {
	WorkspaceRoot string
}

func (t *TodoManager) path() string {
	return filepath.Join(t.WorkspaceRoot, ".baize", "todo.md")
}

// Load returns the current todo content, or empty string if none exists.
func (t *TodoManager) Load() string {
	data, err := os.ReadFile(t.path())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// Save writes new todo content.
func (t *TodoManager) Save(content string) error {
	if err := os.MkdirAll(filepath.Dir(t.path()), 0755); err != nil {
		return err
	}
	return os.WriteFile(t.path(), []byte(content), 0644)
}

// InjectIntoState appends current todo as a system message near the end of
// MessageState.Messages if it is non-empty. Called before each LLM invocation.
func (t *TodoManager) InjectIntoState(s *MessageState) {
	todo := t.Load()
	if todo == "" {
		return
	}
	// Remove any previously injected todo block to avoid duplication.
	filtered := s.Messages[:0]
	for _, m := range s.Messages {
		if m.Role == RoleSystem && strings.HasPrefix(m.Content, "== 当前任务进度") {
			continue
		}
		filtered = append(filtered, m)
	}
	s.Messages = append(filtered, Message{
		Role:    RoleSystem,
		Content: fmt.Sprintf("== 当前任务进度 ==\n%s", todo),
	})
}
