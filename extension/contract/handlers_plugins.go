package contract

import (
	"context"
	"sort"

	"github.com/xraph/forge/extensions/dashboard/contract"
)

// PluginSummary describes an installed Authsome plugin. Settings counts are
// taken from the live settings registry, not the dashboard contributors.
type PluginSummary struct {
	Name         string `json:"name"`
	SettingCount int    `json:"settingCount"`
}

type PluginsListResponse struct {
	Plugins []PluginSummary `json:"plugins"`
}

func pluginsListHandler(deps Deps) func(context.Context, struct{}, contract.Principal) (PluginsListResponse, error) {
	return func(_ context.Context, _ struct{}, _ contract.Principal) (PluginsListResponse, error) {
		if deps.Engine == nil || deps.Engine.Plugins() == nil {
			return PluginsListResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth plugin registry not configured"}
		}
		out := PluginsListResponse{Plugins: make([]PluginSummary, 0, len(deps.Engine.Plugins().Plugins()))}
		for _, installed := range deps.Engine.Plugins().Plugins() {
			name := installed.Name()
			summary := PluginSummary{Name: name}
			if mgr := deps.Engine.Settings(); mgr != nil {
				summary.SettingCount = len(mgr.DefinitionsForNamespace(name))
			}
			out.Plugins = append(out.Plugins, summary)
		}
		sort.Slice(out.Plugins, func(i, j int) bool { return out.Plugins[i].Name < out.Plugins[j].Name })
		return out, nil
	}
}
