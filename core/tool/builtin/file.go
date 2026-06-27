package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wzhongyou/baize/core/tool"
)

const (
	maxReadLines   = 500
	maxToolOutputB = 40 * 1024 // 40KB raw output cap before truncation
)

// FileTool provides file system operations: read, write, edit, list, and search.
type FileTool struct {
	WorkspaceRoot string // All paths are resolved relative to this root.
}

// Ensure FileTool implements tool.Tool and tool.SafeTool.
var _ tool.Tool = (*FileTool)(nil)
var _ tool.SafeTool = (*FileTool)(nil)

func (f *FileTool) Name() string { return "file" }
func (f *FileTool) Description() string {
	return "Perform file operations: read, write, edit, list directory, and search for files. All paths are relative to the workspace root."
}
func (f *FileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"read", "write", "edit", "list", "search"},
				"description": "Operation to perform: read (read file), write (create/overwrite), edit (replace string in file), list (list directory), search (glob pattern search)",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "File or directory path relative to workspace root.",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Content for write/edit operations.",
			},
			"old_string": map[string]any{
				"type":        "string",
				"description": "String to replace (edit action only).",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "Replacement string (edit action only).",
			},
			"pattern": map[string]any{
				"type":        "string",
				"description": "Glob pattern for search action, e.g. '*.go'.",
			},
		},
		"required": []string{"action", "path"},
	}
}

func (f *FileTool) IsReadOnly() bool { return false }
func (f *FileTool) RequiredPermissions() []tool.Permission {
	return []tool.Permission{tool.PermFileRead, tool.PermFileWrite}
}
func (f *FileTool) AffectedPaths(args map[string]any) []string {
	if path, ok := args["path"].(string); ok {
		return []string{f.absPath(path)}
	}
	return nil
}

// readOnlyActions are file operations that don't modify the filesystem.
var readOnlyActions = map[string]bool{"read": true, "list": true, "search": true}

// safeFileTool wraps FileTool to expose per-action SafeTool metadata.
type safeFileToolAction struct {
	*FileTool
	action string
}

func (s *safeFileToolAction) IsReadOnly() bool { return readOnlyActions[s.action] }
func (s *safeFileToolAction) RequiredPermissions() []tool.Permission {
	if readOnlyActions[s.action] {
		return []tool.Permission{tool.PermFileRead}
	}
	return []tool.Permission{tool.PermFileWrite}
}

func (f *FileTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	action, _ := args["action"].(string)
	path, _ := args["path"].(string)

	absPath := f.absPath(path)

	switch action {
	case "read":
		return f.read(absPath)
	case "write":
		content, _ := args["content"].(string)
		return f.write(absPath, content)
	case "edit":
		oldStr, _ := args["old_string"].(string)
		newStr, _ := args["new_string"].(string)
		return f.edit(absPath, oldStr, newStr)
	case "list":
		return f.list(absPath)
	case "search":
		pattern, _ := args["pattern"].(string)
		return f.search(absPath, pattern)
	default:
		return "", fmt.Errorf("file: unknown action %q", action)
	}
}

func (f *FileTool) absPath(rel string) string {
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel)
	}
	if f.WorkspaceRoot == "" {
		return filepath.Clean(rel)
	}
	return filepath.Join(f.WorkspaceRoot, rel)
}

func (f *FileTool) read(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("file read %s: %w", path, err)
	}
	content := string(data)
	lines := strings.Split(content, "\n")
	if len(lines) > maxReadLines {
		truncated := strings.Join(lines[:maxReadLines], "\n")
		return truncated + fmt.Sprintf("\n...[truncated: showing %d of %d lines, use start_line/end_line to read more]", maxReadLines, len(lines)), nil
	}
	return content, nil
}

func (f *FileTool) write(path, content string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("file write %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("file write %s: %w", path, err)
	}
	return fmt.Sprintf("Wrote %d bytes to %s", len(content), path), nil
}

func (f *FileTool) edit(path, oldStr, newStr string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("file edit %s: %w", path, err)
	}
	content := string(data)
	count := strings.Count(content, oldStr)
	if count == 0 {
		return "", fmt.Errorf("file edit: string not found in %s", path)
	}
	if count > 1 {
		// Report line numbers of all occurrences to help the LLM provide more context.
		var lineNums []string
		for i, line := range strings.Split(content, "\n") {
			if strings.Contains(line, oldStr) {
				lineNums = append(lineNums, fmt.Sprintf("%d", i+1))
			}
		}
		return "", fmt.Errorf("file edit: old_string matches %d times in %s (lines: %s) — provide more context to make it unique",
			count, path, strings.Join(lineNums, ", "))
	}
	updated := strings.Replace(content, oldStr, newStr, 1)
	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		return "", fmt.Errorf("file edit %s: %w", path, err)
	}
	return fmt.Sprintf("Edited %s", path), nil
}

func (f *FileTool) list(path string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("file list %s: %w", path, err)
	}
	var b strings.Builder
	for _, e := range entries {
		if e.IsDir() {
			b.WriteString(e.Name() + "/\n")
		} else {
			info, _ := e.Info()
			size := ""
			if info != nil {
				size = fmt.Sprintf(" (%d bytes)", info.Size())
			}
			b.WriteString(e.Name() + size + "\n")
		}
	}
	return b.String(), nil
}

func (f *FileTool) search(dir, pattern string) (string, error) {
	var results []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "node_modules" || base == ".baize" {
				return filepath.SkipDir
			}
			return nil
		}
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if matched {
			results = append(results, path)
		}
		return nil
	})
	return strings.Join(results, "\n"), nil
}
