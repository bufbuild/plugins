package docker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"

	"github.com/bufbuild/plugins/internal/plugin"
)

// Build runs a Docker build command for the specified plugin tagging it with the given organization.
// The args parameter passes any additional arguments to be passed to the build.
// Returns the combined stdout/stderr of the build along with any error.
func Build(
	ctx context.Context,
	plugin *plugin.Plugin,
	dockerOrg string,
	args []string,
) (_ []byte, retErr error) {
	dockerCmd, err := exec.LookPath("docker")
	if err != nil {
		return nil, err
	}
	identity, err := bufpluginref.PluginIdentityForString(plugin.Name)
	if err != nil {
		return nil, err
	}
	imageName, err := ImageName(plugin, dockerOrg)
	if err != nil {
		return nil, err
	}
	commonArgs := []string{
		"buildx",
		"build",
		"--label",
		fmt.Sprintf("build.buf.plugins.config.owner=%s", identity.Owner()),
		"--label",
		fmt.Sprintf("build.buf.plugins.config.name=%s", identity.Plugin()),
		"--label",
		fmt.Sprintf("build.buf.plugins.config.version=%s", plugin.PluginVersion),
		"--label",
		"org.opencontainers.image.source=https://github.com/bufbuild/plugins",
		"--label",
		fmt.Sprintf("org.opencontainers.image.description=%s", plugin.Description),
		"--label",
		fmt.Sprintf("org.opencontainers.image.licenses=%s", plugin.SPDXLicenseID),
		"--progress",
		"plain",
	}
	commonArgs = append(commonArgs, args...)
	f, err := os.Open(filepath.Join(filepath.Dir(plugin.Path), "Dockerfile"))
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, f.Close())
	}()
	buildStages, err := ParseDockerfileBuildStages(f)
	if err != nil {
		return nil, err
	}
	for _, stage := range buildStages {
		// Build each stage of multi-stage build (to improve caching)
		cmd := exec.CommandContext(
			ctx,
			dockerCmd,
			commonArgs...,
		)
		cmd.Args = append(
			cmd.Args,
			"--target",
			stage,
			"-t",
			imageName+"-"+stage,
		)
		cmd.Args = append(cmd.Args, filepath.Dir(plugin.Path))
		output, err := cmd.CombinedOutput()
		if err != nil {
			return output, err
		}
	}
	// Build the final stage of multi-stage build
	cmd := exec.CommandContext(
		ctx,
		dockerCmd,
		commonArgs...,
	)
	cmd.Args = append(
		cmd.Args,
		"-t",
		imageName,
	)
	cmd.Args = append(cmd.Args, filepath.Dir(plugin.Path))
	return cmd.CombinedOutput()
}
