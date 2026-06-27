// Package memory implements the long-term memory system using markdown files
// with YAML frontmatter metadata.
package memory

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/wzhongyou/baize/protocol"
)

// MarkdownStore is a file-based long-term memory provider.
// Each memory is stored as a markdown file with frontmatter metadata.
type MarkdownStore struct {
	mu   sync.RWMutex
	dir  string
}

// NewMarkdownStore creates a new file-based memory store rooted at dir.
func NewMarkdownStore(dir string) (*MarkdownStore, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &MarkdownStore{dir: dir}, nil
}

// Save writes a memory as a markdown file.
func (s *MarkdownStore) Save(ctx context.Context, content string, metadata map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use memory content first line as filename hint.
	title := firstLine(content)
	name := slugify(title)
	if name == "" {
		name = "memory"
	}

	path := filepath.Join(s.dir, name+".md")

	var sb strings.Builder
	sb.WriteString("---\n")
	if metadata != nil {
		for k, v := range metadata {
			sb.WriteString(k + ": " + formatValue(v) + "\n")
		}
	}
	sb.WriteString("---\n\n")
	sb.WriteString(content)
	sb.WriteString("\n")

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// Search performs keyword-based search across all memory files.
func (s *MarkdownStore) Search(ctx context.Context, query string, topK int) ([]protocol.MemoryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	type scored struct {
		result protocol.MemoryResult
		score  float64
	}
	var results []scored

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			continue
		}
		content := string(data)
		score := matchScore(strings.ToLower(content), strings.ToLower(query))

		if score > 0 {
			results = append(results, scored{
				result: protocol.MemoryResult{
					Content:  strings.TrimSpace(content),
					Score:    score,
					Metadata: map[string]any{"file": entry.Name()},
				},
				score: score,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i].score > results[j].score })

	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}

	out := make([]protocol.MemoryResult, len(results))
	for i, r := range results {
		out[i] = r.result
	}
	return out, nil
}

// ── Helpers ────────────────────────────────────────────────────────────────

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, s)
	s = strings.Trim(s, "-")
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}

func formatValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// matchScore computes a simple keyword overlap score.
func matchScore(content, query string) float64 {
	words := strings.Fields(query)
	if len(words) == 0 {
		return 0
	}
	hits := 0
	for _, w := range words {
		if strings.Contains(content, w) {
			hits++
		}
	}
	return float64(hits) / float64(len(words))
}
