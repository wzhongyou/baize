package context

import (
	"os"
	"path/filepath"
	"strings"
)

// InstructionLoader loads BAIZE.md instruction files from multiple levels.
//
// Load order (lowest to highest priority, all concatenated):
//  1. ~/.baize/BAIZE.md          — user global
//  2. <project>/BAIZE.md         — project
//  3. <project>/BAIZE.local.md   — local overrides (not committed)
type InstructionLoader struct {
	ProjectRoot string
}

// Load returns the combined content of all instruction files found.
// Missing files are silently skipped.
func (l *InstructionLoader) Load() string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".baize", "BAIZE.md"),
		filepath.Join(l.ProjectRoot, "BAIZE.md"),
		filepath.Join(l.ProjectRoot, "BAIZE.local.md"),
	}

	var parts []string
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := strings.TrimSpace(string(data))
		if content != "" {
			parts = append(parts, content)
		}
	}
	return strings.Join(parts, "\n\n")
}
