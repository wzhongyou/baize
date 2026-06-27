package permission

import (
	"context"
	"fmt"
	"strings"

	"github.com/wzhongyou/baize/core/agent"
	"github.com/wzhongyou/baize/core/tool"
)

// AsAgentChecker wraps a PolicyEngine as an agent.PermissionChecker.
// It requires access to the tool registry to extract SafeTool metadata.
func (pe *PolicyEngine) AsAgentChecker(reg *tool.ToolRegistry) agent.PermissionChecker {
	return &agentPermAdapter{pe: pe, reg: reg}
}

type agentPermAdapter struct {
	pe  *PolicyEngine
	reg *tool.ToolRegistry
}

func (a *agentPermAdapter) CheckPermission(ctx context.Context, toolName string, args map[string]any) (string, string) {
	t, ok := a.reg.Get(toolName)
	if !ok {
		return "deny", fmt.Sprintf("unknown tool: %s", toolName)
	}

	st, isSafe := t.(tool.SafeTool)

	// For file tool, derive actual permissions from the action argument
	// so that read/list/search don't trigger write-permission checks.
	if isSafe {
		if action, hasAction := args["action"].(string); hasAction {
			st = actionScopedSafeTool{SafeTool: st, action: action}
		}
	}

	// If the tool doesn't declare permissions, allow by default.
	if !isSafe {
		return "allow", ""
	}

	// Collect the required permissions.
	perms := st.RequiredPermissions()
	if len(perms) == 0 {
		return "allow", ""
	}

	// Check each permission.
	for _, p := range perms {
		op := buildOperation(Permission(p), toolName, st, args)
		decision := a.pe.Check(ctx, op)
		switch decision {
		case DecisionDeny:
			return "deny", fmt.Sprintf("%s: %s not allowed", toolName, p)
		case DecisionAsk:
			return "ask", fmt.Sprintf("%s requires %s — confirmation needed", toolName, p)
		}
	}

	return "allow", ""
}

func buildOperation(p Permission, toolName string, st tool.SafeTool, args map[string]any) Operation {
	op := Operation{
		Permission: p,
		ToolName:   toolName,
		Args:       args,
	}

	// Extract path/command info from AffectedPaths.
	paths := st.AffectedPaths(args)
	switch p {
	case PermFileRead, PermFileWrite:
		if len(paths) > 0 {
			op.Path = paths[0]
		}
	case PermShellExec:
		cmd := extractCommand(args)
		op.Command = cmd
		if cmd != "" {
			// For compound commands, take the first word.
			if idx := strings.Index(cmd, " "); idx > 0 {
				op.Command = cmd[:idx]
			}
		}
	case PermNetworkOutbound:
		if domain, ok := args["url"].(string); ok {
			op.Domain = domain
		}
	}

	return op
}

// actionScopedSafeTool narrows SafeTool permissions based on the action argument.
// For file tools, read/list/search only need PermFileRead.
type actionScopedSafeTool struct {
	tool.SafeTool
	action string
}

var readOnlyFileActions = map[string]bool{"read": true, "list": true, "search": true}

func (a actionScopedSafeTool) IsReadOnly() bool { return readOnlyFileActions[a.action] }
func (a actionScopedSafeTool) RequiredPermissions() []tool.Permission {
	if readOnlyFileActions[a.action] {
		return []tool.Permission{tool.PermFileRead}
	}
	return a.SafeTool.RequiredPermissions()
}

// AsAgentCheckerFullAuto wraps PolicyEngine as a checker that never asks —
// only hard denies from policy rules are enforced.
func (pe *PolicyEngine) AsAgentCheckerFullAuto(reg *tool.ToolRegistry) agent.PermissionChecker {
	return &fullAutoAdapter{pe: pe, reg: reg}
}

type fullAutoAdapter struct {
	pe  *PolicyEngine
	reg *tool.ToolRegistry
}

func (a *fullAutoAdapter) CheckPermission(ctx context.Context, toolName string, args map[string]any) (string, string) {
	t, ok := a.reg.Get(toolName)
	if !ok {
		return "deny", fmt.Sprintf("unknown tool: %s", toolName)
	}
	st, isSafe := t.(tool.SafeTool)
	if !isSafe {
		return "allow", ""
	}
	for _, p := range st.RequiredPermissions() {
		op := buildOperation(Permission(p), toolName, st, args)
		if a.pe.Check(ctx, op) == DecisionDeny {
			return "deny", fmt.Sprintf("%s: %s not allowed", toolName, p)
		}
	}
	return "allow", ""
}

// ReadOnlyChecker returns a checker that denies all write/exec operations.
func ReadOnlyChecker() agent.PermissionChecker { return readOnlyChecker{} }

type readOnlyChecker struct{}

func (readOnlyChecker) CheckPermission(_ context.Context, _ string, _ map[string]any) (string, string) {
	return "deny", "suggest mode: no writes or execution allowed"
}

func extractCommand(args map[string]any) string {
	if cmd, ok := args["command"].(string); ok {
		return cmd
	}
	if cmd, ok := args["cmd"].(string); ok {
		return cmd
	}
	return ""
}
