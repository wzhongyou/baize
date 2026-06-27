package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wzhongyou/baize/core/tool"
	"github.com/wzhongyou/baize/core/tool/mcp"
)

// Manager discovers, loads, and activates skills.
type Manager struct {
	skills     []*Skill
	mcpManager *mcp.Manager
}

// NewManager creates a Manager. Call Load then Start before using Tools/SystemPrompt.
func NewManager(dir string) *Manager {
	return &Manager{mcpManager: mcp.NewManager()}
}

// Load scans dir and loads all valid skills.
func (m *Manager) Load(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		s, err := Load(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		m.skills = append(m.skills, s)
	}
	return nil
}

// Start connects all MCP servers declared by loaded skills.
func (m *Manager) Start(ctx context.Context) error {
	for _, s := range m.skills {
		for _, srv := range s.MCPServers {
			name := s.Name + ":" + srv.Name
			if err := m.mcpManager.AddServer(ctx, name, srv.Command, srv.Args...); err != nil {
				return err
			}
		}
	}
	return nil
}

// Close shuts down all skill MCP servers.
func (m *Manager) Close() error {
	return m.mcpManager.Close()
}

// SystemPromptIndex returns a single compact line listing installed skills by
// name+description only. Injected into every session; does not grow with skill count.
func (m *Manager) SystemPromptIndex() string {
	if len(m.skills) == 0 {
		return ""
	}
	var parts []string
	for _, s := range m.skills {
		parts = append(parts, fmt.Sprintf("%s（%s）", s.Name, s.Description))
	}
	return "已安装 Skills（使用 activate_skill 工具按需加载完整指令）：" + strings.Join(parts, "、")
}

// Tools returns all tools contributed by skill MCP servers plus the activate_skill tool.
func (m *Manager) Tools() []tool.Tool {
	tools := m.mcpManager.Tools()
	tools = append(tools, &activateSkillTool{m: m})
	return tools
}

// Skills returns all loaded skills.
func (m *Manager) Skills() []*Skill {
	return m.skills
}

func (m *Manager) find(name string) *Skill {
	for _, s := range m.skills {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// ── activate_skill tool ──────────────────────────────────────────────────────

// activateSkillTool is a built-in tool the LLM calls to load a skill's full prompt.
// The tool result is injected into the conversation as a tool_result message, which
// the LLM sees as context — no MessageState surgery needed.
type activateSkillTool struct{ m *Manager }

func (t *activateSkillTool) Name() string { return "activate_skill" }
func (t *activateSkillTool) Description() string {
	return "Load the full instructions for an installed skill by name. Call this when the user's request matches a skill's purpose."
}
func (t *activateSkillTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "The skill name to activate (from the installed skills list)",
			},
		},
		"required": []string{"name"},
	}
}
func (t *activateSkillTool) Execute(_ context.Context, args map[string]any) (string, error) {
	name, _ := args["name"].(string)
	s := t.m.find(name)
	if s == nil {
		return "", fmt.Errorf("skill %q not found", name)
	}
	if s.Prompt == "" {
		return fmt.Sprintf("Skill %q has no additional instructions.", name), nil
	}
	return s.Prompt, nil
}
