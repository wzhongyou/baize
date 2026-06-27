package builtin

import (
	"context"
	"fmt"

	"github.com/wzhongyou/baize/core/memory"
	"github.com/wzhongyou/baize/core/tool"
)

// MemorySaveTool lets the agent persist a named memory entry.
type MemorySaveTool struct {
	AutoMemory *memory.AutoMemory
}

var _ tool.Tool = (*MemorySaveTool)(nil)

func (t *MemorySaveTool) Name() string { return "memory_save" }
func (t *MemorySaveTool) Description() string {
	return "Save a memory entry for future sessions. Use to remember project conventions, user preferences, key decisions, or important context."
}
func (t *MemorySaveTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Short slug name for the memory (e.g. 'user-prefs', 'project-context').",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "One-line summary of what this memory contains.",
			},
			"type": map[string]any{
				"type":        "string",
				"enum":        []string{"user", "feedback", "project", "reference"},
				"description": "Memory type: user (about the user), feedback (guidance), project (project context), reference (external pointers).",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The memory content in markdown.",
			},
		},
		"required": []string{"name", "description", "type", "content"},
	}
}

func (t *MemorySaveTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	name, _ := args["name"].(string)
	desc, _ := args["description"].(string)
	memType, _ := args["type"].(string)
	content, _ := args["content"].(string)

	if content == "" {
		return "", fmt.Errorf("memory_save: content is required")
	}
	if err := t.AutoMemory.Save(ctx, name, desc, memType, content); err != nil {
		return "", fmt.Errorf("memory_save: %w", err)
	}
	return fmt.Sprintf("Memory saved: %s", name), nil
}
