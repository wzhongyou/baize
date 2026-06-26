package tool

import (
	"context"

	"github.com/wzhongyou/baize/api"
)

// AsToolProvider returns an api.ToolProvider backed by this registry.
func (r *ToolRegistry) AsToolProvider() api.ToolProvider {
	return &toolRegistryAdapter{reg: r}
}

type toolRegistryAdapter struct {
	reg *ToolRegistry
}

func (a *toolRegistryAdapter) ToolInfos() []api.ToolInfo {
	tools := a.reg.List()
	infos := make([]api.ToolInfo, 0, len(tools))
	for _, t := range tools {
		readOnly := false
		if st, ok := t.(SafeTool); ok {
			readOnly = st.IsReadOnly()
		}
		infos = append(infos, api.ToolInfo{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
			ReadOnly:    readOnly,
			Source:      "builtin",
		})
	}
	return infos
}

func (a *toolRegistryAdapter) Execute(ctx context.Context, name string, args map[string]any) (string, error) {
	t, ok := a.reg.Get(name)
	if !ok {
		return "", api.Errorf(api.CodeNotFound, "tool not found: %s", name)
	}
	return t.Execute(ctx, args)
}
