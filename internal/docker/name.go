package docker

import (
	"fmt"

	"github.com/bufbuild/plugins/internal/plugin"
)

// ImageName returns the name of the plugin's tagged image in the given organization.
func ImageName(plugin *plugin.Plugin, org string) string {
	identity := plugin.Identity
	return fmt.Sprintf("%s/plugins-%s-%s:%s", org, identity.Owner(), identity.Plugin(), plugin.PluginVersion)
}
