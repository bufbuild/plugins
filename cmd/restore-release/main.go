package main

// restore-release takes a GitHub release name (e.g. yyyyMMdd.N)
// and restores the container registry to the state of the release.
// This can be used in case images are pushed by accident, to avoid
// unnecessary installation and revision bumps for images which
// haven't changed.
//
// It takes a -dry-run argument to do all tasks except the final
// docker push command, so it is safe to run prior to making changes.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"
	"github.com/bufbuild/plugins/internal/release"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "perform a dry-run (no GitHub modifications)")
	flag.Parse()

	if len(flag.Args()) != 1 {
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "usage: restore-release <releaseName>")
		flag.PrintDefaults()
		os.Exit(2)
	}
	releaseName := flag.Args()[0]
	cmd := &command{
		dryRun:  *dryRun,
		release: releaseName,
	}
	if err := cmd.run(); err != nil {
		log.Fatalln(err.Error())
	}
}

type command struct {
	dryRun  bool
	release string
}

func (c *command) run() error {
	ctx := context.Background()
	client := release.NewClient(ctx)
	githubRelease, err := client.GetReleaseByTag(ctx, release.GithubOwnerBufbuild, release.GithubRepoPlugins, c.release)
	if err != nil {
		return fmt.Errorf("failed to retrieve release %s: %w", c.release, err)
	}
	pluginReleasesBytes, _, err := client.DownloadAsset(ctx, githubRelease, release.PluginReleasesFile)
	if err != nil {
		return fmt.Errorf("failed to download plugin-releases.json: %w", err)
	}
	var pluginReleases release.PluginReleases
	if err := json.Unmarshal(pluginReleasesBytes, &pluginReleases); err != nil {
		return fmt.Errorf("invalid plugin-releases.json format: %w", err)
	}
	for _, pluginRelease := range pluginReleases.Releases {
		image, err := fetchRegistryImage(pluginRelease)
		if err != nil {
			return err
		}
		// Detect if the current registry image doesn't match the release's plugin-releases.json.
		if pluginRelease.RegistryImage != image {
			taggedImage, _, found := strings.Cut(image, "@")
			if !found {
				return fmt.Errorf("invalid image format: %s", image)
			}
			taggedImage += ":" + pluginRelease.PluginVersion
			log.Printf("updating image tag %q to point from %q to %q", taggedImage, image, pluginRelease.RegistryImage)
			if err := pullImage(pluginRelease.RegistryImage); err != nil {
				return fmt.Errorf("failed to pull %q: %w", pluginRelease.RegistryImage, err)
			}
			if err := tagImage(pluginRelease.RegistryImage, taggedImage); err != nil {
				return fmt.Errorf("failed to tag %q: %w", taggedImage, err)
			}
			if !c.dryRun {
				if err := pushImage(taggedImage); err != nil {
					return fmt.Errorf("failed to push %q: %w", taggedImage, err)
				}
			}
		}
	}
	return nil
}

func pullImage(name string) error {
	cmd, err := dockerCmd("pull", name)
	if err != nil {
		return err
	}
	log.Printf("pulling image: %s", name)
	return cmd.Run()
}

func tagImage(previousName, newName string) error {
	cmd, err := dockerCmd("tag", previousName, newName)
	if err != nil {
		return err
	}
	log.Printf("tagging image: %s => %s", previousName, newName)
	return cmd.Run()
}

func pushImage(name string) error {
	cmd, err := dockerCmd("push", name)
	if err != nil {
		return err
	}
	log.Printf("pushing image: %s", name)
	return cmd.Run()
}

func dockerCmd(command string, args ...string) (*exec.Cmd, error) {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return nil, err
	}
	cmd := &exec.Cmd{
		Path: dockerPath,
		Args: append([]string{
			dockerPath,
			command,
		}, args...),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	return cmd, nil
}

func fetchRegistryImage(pluginRelease release.PluginRelease) (string, error) {
	identity, err := bufpluginref.PluginIdentityForString("buf.build/" + pluginRelease.PluginName)
	if err != nil {
		return "", err
	}
	imageName := fmt.Sprintf("ghcr.io/%s/plugins-%s-%s", release.GithubOwnerBufbuild, identity.Owner(), identity.Plugin())
	cmd, err := dockerCmd("manifest", "inspect", "--verbose", imageName+":"+pluginRelease.PluginVersion)
	if err != nil {
		return "", err
	}
	var bb bytes.Buffer
	cmd.Stdout = &bb
	if err := cmd.Run(); err != nil {
		return "", err
	}
	type manifestJSON struct {
		Descriptor struct {
			Digest string `json:"digest"`
		} `json:"Descriptor"` //nolint:tagliatelle
		SchemaV2Manifest struct {
			Config struct {
				Digest string `json:"digest"`
			} `json:"config"`
		} `json:"SchemaV2Manifest"` //nolint:tagliatelle
	}
	var result manifestJSON
	if err := json.Unmarshal(bb.Bytes(), &result); err != nil {
		return "", fmt.Errorf("unable to parse docker manifest inspect output: %w", err)
	}
	descriptorDigest := result.Descriptor.Digest
	if descriptorDigest == "" {
		return "", errors.New("unable to parse descriptor digest from docker manifest inspect output")
	}
	imageDigest := result.SchemaV2Manifest.Config.Digest
	if imageDigest == "" {
		return "", errors.New("unable to parse image config digest from docker manifest inspect output")
	}
	return fmt.Sprintf("%s@%s", imageName, descriptorDigest), nil
}
