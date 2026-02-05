package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"buf.build/go/interrupt"

	"github.com/bufbuild/plugins/internal/docker"
	"github.com/bufbuild/plugins/internal/plugin"
	"github.com/bufbuild/plugins/internal/release"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("failed to create plugin zips: %v", err)
	}
}

func run() error {
	var (
		dir    = flag.String("dir", ".", "Directory path to plugins")
		org    = flag.String("org", "bufbuild", "Docker Organization (without registry)")
		outDir = flag.String("out", "downloads", "Output directory for plugin zips")
	)
	flag.Parse()

	ctx := interrupt.Handle(context.Background())
	basedir := *dir

	plugins, err := plugin.FindAll(basedir)
	if err != nil {
		return err
	}
	includedPlugins, err := plugin.FilterByPluginsEnv(plugins, os.Getenv("PLUGINS"))
	if err != nil {
		return err
	}
	if len(includedPlugins) == 0 {
		log.Printf("no plugins to process")
		return nil
	}

	// Create output directory
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, includedPlugin := range includedPlugins {
		if err := createPluginZip(ctx, includedPlugin, *org, *outDir); err != nil {
			log.Printf(
				"failed to process plugin %s:%s: %v",
				includedPlugin.Name,
				includedPlugin.PluginVersion,
				err,
			)
			return err
		}
		log.Printf("created zip for plugin %s:%s", includedPlugin.Name, includedPlugin.PluginVersion)
	}
	return nil
}

func createPluginZip(ctx context.Context, plugin *plugin.Plugin, dockerOrg string, outDir string) error {
	// Get image name from already-built local image
	imageName := docker.ImageName(plugin, dockerOrg)

	// Get the image ID
	imageID, err := getImageID(ctx, imageName)
	if err != nil {
		return fmt.Errorf("failed to get image ID for %s: %w (image must be built first)", imageName, err)
	}

	// Create zip file
	_, err = release.CreatePluginZip(ctx, outDir, plugin, imageID)
	if err != nil {
		return err
	}

	return nil
}

func getImageID(ctx context.Context, imageName string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format={{.Id}}", imageName)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output[:len(output)-1]), nil // trim newline
}
