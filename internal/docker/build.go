package docker

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"

	"github.com/bufbuild/plugins/internal/plugin"
)

// Build runs a Docker build command for the specified plugin tagging it with the given image name.
// The args parameter passes any additional arguments to be passed to the build.
// Returns the combined stdout/stderr of the build along with any error.
func Build(
	ctx context.Context,
	plugin *plugin.Plugin,
	imageName string,
	cachePath string,
	args []string,
) ([]byte, error) {
	identity := plugin.Identity
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
		fmt.Sprintf("org.opencontainers.image.created=%s", time.Now().UTC().Format(time.RFC3339)),
		"--label",
		fmt.Sprintf("org.opencontainers.image.description=%s", plugin.Description),
		"--label",
		fmt.Sprintf("org.opencontainers.image.licenses=%s", plugin.SPDXLicenseID),
		"--label",
		fmt.Sprintf("org.opencontainers.image.vendor=%s", "Buf Technologies, Inc."),
		"--progress",
		"plain",
	}
	if gitCommit := plugin.GitCommit(ctx); gitCommit != "" {
		commonArgs = append(commonArgs, "--label", fmt.Sprintf("org.opencontainers.image.revision=%s", gitCommit))
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
	buildArgs := slices.Concat(commonArgs, []string{
		"-t", imageName,
		filepath.Dir(plugin.Path),
	})
	cmd := exec.CommandContext(ctx, "docker", buildArgs...)
	// Set file modification times to bust Docker cache for local files
	if err := filepath.WalkDir(filepath.Dir(plugin.Path), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			if err := os.Chtimes(path, time.Time{}, time.Now().UTC()); err != nil {
				return fmt.Errorf("failed to set mtime for %q: %w", path, err)
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to change times: %w", err)
	}
	return cmd.CombinedOutput()
}
