package permission

import (
	"context"
	"fmt"
	"strings"

	"github.com/wzhongyou/baize/agent"
	"github.com/wzhongyou/baize/tool"
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

func extractCommand(args map[string]any) string {
	if cmd, ok := args["command"].(string); ok {
		return cmd
	}
	if cmd, ok := args["cmd"].(string); ok {
		return cmd
	}
	return ""
}
