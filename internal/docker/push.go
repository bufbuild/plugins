package docker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"

	"github.com/bufbuild/plugins/internal/plugin"
)

// Push pushes a docker image for the given plugin to the Docker organization.
// It assumes it has already been built in a previous step.
//
// Images are saved from the local Docker daemon via "docker save" and pushed
// using go-containerregistry to preserve Docker distribution manifest v2 format.
func Push(ctx context.Context, pluginToPush *plugin.Plugin, dockerOrg string) (retErr error) {
	imageName := ImageName(pluginToPush, dockerOrg)
	tmpFile, err := os.CreateTemp("", "plugin-image-*.tar")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, os.Remove(tmpFile.Name()))
	}()
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	cmd := exec.CommandContext(ctx, "docker", "save", imageName, "-o", tmpFile.Name())
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker save %q: %w\noutput: %s", imageName, err, output)
	}
	image, err := tarball.ImageFromPath(tmpFile.Name(), nil)
	if err != nil {
		return fmt.Errorf("load image from tarball: %w", err)
	}
	tag, err := name.NewTag(imageName)
	if err != nil {
		return fmt.Errorf("parse image reference %q: %w", imageName, err)
	}
	if err := remote.Write(tag, image, remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithContext(ctx)); err != nil {
		return fmt.Errorf("push image %q: %w", imageName, err)
	}
	return nil
}
