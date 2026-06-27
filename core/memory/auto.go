package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AutoMemory lets the agent write named memory entries to a project-scoped
// memory directory (~/.baize/projects/<hash>/memory/ or .baize/memory/).
// It also maintains MEMORY.md as an index of all entries.
type AutoMemory struct {
	Dir string // e.g. ~/.baize/projects/<repo>/memory/
}

// NewAutoMemory creates an AutoMemory backed by dir.
func NewAutoMemory(dir string) (*AutoMemory, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &AutoMemory{Dir: dir}, nil
}

// Save writes a named memory file and updates MEMORY.md index.
// name is a short slug (e.g. "user-prefs"), content is the markdown body.
func (m *AutoMemory) Save(_ context.Context, name, description, memType, content string) error {
	if name == "" {
		name = fmt.Sprintf("memory-%d", time.Now().UnixMilli())
	}
	name = slugify(name)

	body := fmt.Sprintf("---\nname: %s\ndescription: %s\nmetadata:\n  type: %s\n---\n\n%s\n",
		name, description, memType, content)

	path := filepath.Join(m.Dir, name+".md")
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		return err
	}
	return m.updateIndex(name, description)
}

// LoadIndex returns the content of MEMORY.md (up to 200 lines).
func (m *AutoMemory) LoadIndex() string {
	data, err := os.ReadFile(filepath.Join(m.Dir, "MEMORY.md"))
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 200 {
		lines = lines[:200]
	}
	return strings.Join(lines, "\n")
}

func (m *AutoMemory) updateIndex(name, description string) error {
	indexPath := filepath.Join(m.Dir, "MEMORY.md")
	data, _ := os.ReadFile(indexPath)
	existing := string(data)

	entry := fmt.Sprintf("- [%s](%s.md) — %s\n", name, name, description)

	// Replace existing entry for same name, or append.
	if strings.Contains(existing, fmt.Sprintf("[%s]", name)) {
		lines := strings.Split(existing, "\n")
		for i, l := range lines {
			if strings.Contains(l, fmt.Sprintf("[%s]", name)) {
				lines[i] = strings.TrimRight(entry, "\n")
				existing = strings.Join(lines, "\n")
				return os.WriteFile(indexPath, []byte(existing), 0644)
			}
		}
	}

	return os.WriteFile(indexPath, []byte(existing+entry), 0644)
}
