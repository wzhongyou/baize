package builtin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wzhongyou/baize/core/tool"
)

const maxGrepLines = 200

// GrepTool searches file contents using ripgrep (rg) or grep as fallback.
type GrepTool struct {
	WorkspaceRoot string
}

var _ tool.Tool = (*GrepTool)(nil)
var _ tool.SafeTool = (*GrepTool)(nil)

func (g *GrepTool) Name() string { return "grep" }
func (g *GrepTool) Description() string {
	return "Search file contents by pattern. Returns matching lines with file path and line number. Use for finding code, symbols, strings across the codebase."
}
func (g *GrepTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Search pattern (regular expression or literal string).",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Directory or file to search. Defaults to workspace root.",
			},
			"include": map[string]any{
				"type":        "string",
				"description": "File glob filter, e.g. '*.go', '*.ts'. Optional.",
			},
			"case_insensitive": map[string]any{
				"type":        "boolean",
				"description": "Case-insensitive search. Default false.",
			},
		},
		"required": []string{"pattern"},
	}
}

func (g *GrepTool) IsReadOnly() bool                       { return true }
func (g *GrepTool) RequiredPermissions() []tool.Permission { return []tool.Permission{tool.PermFileRead} }
func (g *GrepTool) AffectedPaths(_ map[string]any) []string { return nil }

func (g *GrepTool) Execute(_ context.Context, args map[string]any) (string, error) {
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return "", fmt.Errorf("grep: pattern is required")
	}

	searchPath := g.WorkspaceRoot
	if p, ok := args["path"].(string); ok && p != "" {
		if filepath.IsAbs(p) {
			searchPath = p
		} else {
			searchPath = filepath.Join(g.WorkspaceRoot, p)
		}
	}

	include, _ := args["include"].(string)
	caseInsensitive, _ := args["case_insensitive"].(bool)

	output, err := g.runRipgrep(pattern, searchPath, include, caseInsensitive)
	if err != nil {
		// fallback to system grep
		output, err = g.runGrep(pattern, searchPath, include, caseInsensitive)
		if err != nil {
			return "", fmt.Errorf("grep: %w", err)
		}
	}

	return truncateGrepOutput(output, maxGrepLines), nil
}

func (g *GrepTool) runRipgrep(pattern, path, include string, caseInsensitive bool) (string, error) {
	if _, err := exec.LookPath("rg"); err != nil {
		return "", fmt.Errorf("rg not found")
	}
	args := []string{"--line-number", "--no-heading", "--color=never", pattern, path}
	if include != "" {
		args = append([]string{"--glob", include}, args...)
	}
	if caseInsensitive {
		args = append([]string{"-i"}, args...)
	}
	cmd := exec.Command("rg", args...)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "(no matches)", nil // exit 1 = no matches, not an error
		}
		return "", err
	}
	return string(out), nil
}

func (g *GrepTool) runGrep(pattern, path, include string, caseInsensitive bool) (string, error) {
	args := []string{"-rn", "--color=never", pattern, path}
	if include != "" {
		args = append([]string{"--include=" + include}, args...)
	}
	if caseInsensitive {
		args = append([]string{"-i"}, args...)
	}
	cmd := exec.Command("grep", args...)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "(no matches)", nil
		}
		return "", err
	}
	return string(out), nil
}

func truncateGrepOutput(output string, maxLines int) string {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) <= maxLines {
		return output
	}
	return strings.Join(lines[:maxLines], "\n") +
		fmt.Sprintf("\n...[truncated: %d more lines, narrow your pattern or use --include]", len(lines)-maxLines)
}
