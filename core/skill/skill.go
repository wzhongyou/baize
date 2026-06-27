// Package skill manages installable capability packs for the Baize agent.
//
// A skill is a directory under ~/.baize/skills/<name>/ containing:
//
//	SKILL.md      — required; YAML frontmatter + markdown body (system prompt fragment)
//	mcp.json      — optional; MCP server definitions to start when skill is loaded
//
// SKILL.md frontmatter fields:
//
//	name:        string  (defaults to directory name)
//	description: string  (shown in /skills list; used for semantic activation)
//	slash:       bool    (expose as slash command, default false)
//	triggers:    []string  (keyword phrases that auto-activate this skill)
package skill

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Skill is a loaded, ready-to-use capability pack.
type Skill struct {
	Name        string
	Description string
	Slash       bool
	Triggers    []string
	Prompt      string // full markdown body (system prompt fragment)
	MCPServers  []MCPServerDef
	Dir         string
}

// MCPServerDef describes an MCP server bundled with a skill.
type MCPServerDef struct {
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// Load reads a skill from a directory. The directory must contain SKILL.md.
func Load(dir string) (*Skill, error) {
	mdPath := filepath.Join(dir, "SKILL.md")
	f, err := os.Open(mdPath)
	if err != nil {
		return nil, fmt.Errorf("skill: open %s: %w", mdPath, err)
	}
	defer f.Close()

	fm, body, err := parseFrontmatter(f)
	if err != nil {
		return nil, fmt.Errorf("skill: parse %s: %w", mdPath, err)
	}

	name := stringField(fm, "name")
	if name == "" {
		name = filepath.Base(dir)
	}

	s := &Skill{
		Name:        name,
		Description: stringField(fm, "description"),
		Slash:       boolField(fm, "slash"),
		Triggers:    stringSliceField(fm, "triggers"),
		Prompt:      strings.TrimSpace(body),
		Dir:         dir,
	}

	// Load optional mcp.json
	mcpPath := filepath.Join(dir, "mcp.json")
	if data, err := os.ReadFile(mcpPath); err == nil {
		if err := json.Unmarshal(data, &s.MCPServers); err != nil {
			return nil, fmt.Errorf("skill %s: parse mcp.json: %w", name, err)
		}
	}

	return s, nil
}

// parseFrontmatter splits "---\nkey: val\n---\nbody" into a map and body string.
// Files without frontmatter return an empty map and the full content as body.
func parseFrontmatter(f *os.File) (map[string]any, string, error) {
	scanner := bufio.NewScanner(f)
	fm := map[string]any{}
	var bodyLines []string
	inFront := false
	done := false

	for scanner.Scan() {
		line := scanner.Text()
		if !done {
			if line == "---" && !inFront {
				inFront = true
				continue
			}
			if line == "---" && inFront {
				done = true
				continue
			}
			if inFront {
				if k, v, ok := strings.Cut(line, ":"); ok {
					fm[strings.TrimSpace(k)] = strings.TrimSpace(v)
				}
				continue
			}
		}
		bodyLines = append(bodyLines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, "", err
	}
	return fm, strings.Join(bodyLines, "\n"), nil
}

func stringField(m map[string]any, k string) string {
	if v, ok := m[k]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func boolField(m map[string]any, k string) bool {
	if v, ok := m[k]; ok {
		s := strings.ToLower(fmt.Sprintf("%v", v))
		return s == "true" || s == "1" || s == "yes"
	}
	return false
}

func stringSliceField(m map[string]any, k string) []string {
	s := stringField(m, k)
	if s == "" {
		return nil
	}
	// simple comma-separated list
	var out []string
	for _, p := range strings.Split(s, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
