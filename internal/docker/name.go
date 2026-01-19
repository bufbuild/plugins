package docker

import (
	"fmt"
	"runtime"

	"github.com/bufbuild/plugins/internal/plugin"
)

// ImageName returns the name of the plugin's tagged image in the given organization.
func ImageName(plugin *plugin.Plugin, org string) string {
	return ImageNameForArch(plugin, org, runtime.GOARCH)
}

// ImageNameForArch returns the name of the plugin's tagged image in the given organization for a specific architecture.
func ImageNameForArch(plugin *plugin.Plugin, org string, arch string) string {
	identity := plugin.Identity
	prefix := "plugins"
	if arch == "arm64" {
		prefix = "plugins-arm64"
	}
	return fmt.Sprintf("%s/%s-%s-%s:%s", org, prefix, identity.Owner(), identity.Plugin(), plugin.PluginVersion)
}
