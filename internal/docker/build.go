package docker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/bufbuild/plugins/internal/plugin"
)

// Build runs a Docker build command for the specified plugin tagging it with the given organization.
// The args parameter passes any additional arguments to be passed to the build.
// Returns the combined stdout/stderr of the build along with any error.
func Build(
	ctx context.Context,
	plugin *plugin.Plugin,
	dockerOrg string,
	cachePath string,
	args []string,
) (_ []byte, retErr error) {
	dockerCmd, err := exec.LookPath("docker")
	if err != nil {
		return nil, err
	}
	identity := plugin.Identity
	imageName := ImageName(plugin, dockerOrg)
	commonArgs := []string{
		"buildx",
		"build",
		"--load",
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
	if cachePath != "" {
		cacheDir, err := filepath.Abs(cachePath)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return nil, err
		}
		commonArgs = append(commonArgs, []string{
			// These require building with the docker-container buildx driver
			// The Makefile sets this up for us with 'docker buildx create --use ...'
			"--cache-to",
			fmt.Sprintf("type=local,dest=%s,mode=max,compression=zstd", cacheDir),
			"--cache-from",
			fmt.Sprintf("type=local,src=%s", cacheDir),
		}...)
	}
	commonArgs = append(commonArgs, args...)
	f, err := os.Open(filepath.Join(filepath.Dir(plugin.Path), "Dockerfile"))
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, f.Close())
	}()
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
