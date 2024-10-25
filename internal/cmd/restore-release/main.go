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

	"github.com/bufbuild/buf/private/pkg/interrupt"

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
	ctx := interrupt.Handle(context.Background())
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
		image, err := fetchRegistryImage(ctx, pluginRelease)
		if err != nil {
			return err
		}
		if image == pluginRelease.RegistryImage {
			continue
		}
		// The current registry image doesn't match the release's plugin-releases.json.
		taggedImage, _, found := strings.Cut(image, "@")
		if !found {
			return fmt.Errorf("invalid image format: %s", image)
		}
		taggedImage += ":" + pluginRelease.PluginVersion
		log.Printf("updating image tag %q to point from %q to %q", taggedImage, image, pluginRelease.RegistryImage)
		if err := pullImage(ctx, pluginRelease.RegistryImage); err != nil {
			return fmt.Errorf("failed to pull %q: %w", pluginRelease.RegistryImage, err)
		}
		if err := tagImage(ctx, pluginRelease.RegistryImage, taggedImage); err != nil {
			return fmt.Errorf("failed to tag %q: %w", taggedImage, err)
		}
		if !c.dryRun {
			if err := pushImage(ctx, taggedImage); err != nil {
				return fmt.Errorf("failed to push %q: %w", taggedImage, err)
			}
		}
	}
	return nil
}

func pullImage(ctx context.Context, name string) error {
	log.Printf("pulling image: %s", name)
	return dockerCmd(ctx, "pull", name).Run()
}

func tagImage(ctx context.Context, previousName, newName string) error {
	log.Printf("tagging image: %s => %s", previousName, newName)
	return dockerCmd(ctx, "tag", previousName, newName).Run()
}

func pushImage(ctx context.Context, name string) error {
	log.Printf("pushing image: %s", name)
	return dockerCmd(ctx, "push", name).Run()
}

func dockerCmd(ctx context.Context, command string, args ...string) *exec.Cmd {
	commandArgs := make([]string, 0, len(args)+1)
	commandArgs = append(commandArgs, command)
	commandArgs = append(commandArgs, args...)
	cmd := exec.CommandContext(ctx, "docker", commandArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func fetchRegistryImage(ctx context.Context, pluginRelease release.PluginRelease) (string, error) {
	owner, pluginName, found := strings.Cut(pluginRelease.PluginName, "/")
	if !found {
		return "", fmt.Errorf("invalid plugin name: %q", pluginRelease.PluginName)
	}
	imageName := fmt.Sprintf("ghcr.io/%s/plugins-%s-%s", release.GithubOwnerBufbuild, owner, pluginName)
	cmd := dockerCmd(ctx, "manifest", "inspect", "--verbose", imageName+":"+pluginRelease.PluginVersion)
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
	return fmt.Sprintf("%s@%s", imageName, descriptorDigest), nil
}
