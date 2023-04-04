package docker

import (
	"context"
	"os/exec"

	"github.com/bufbuild/plugins/internal/plugin"
)

// Push pushes a docker image for the given plugin to the Docker organization.
// It assumes it has already been built in a previous step.
func Push(ctx context.Context, plugin *plugin.Plugin, dockerOrg string) ([]byte, error) {
	dockerCmd, err := exec.LookPath("docker")
	if err != nil {
		return nil, err
	}
	imageName, err := ImageName(plugin, dockerOrg)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(
		ctx,
		dockerCmd,
		"push",
		imageName,
	)
	return cmd.CombinedOutput()
}
