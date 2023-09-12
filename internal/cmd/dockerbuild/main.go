package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bufbuild/buf/private/pkg/interrupt"

	"github.com/bufbuild/plugins/internal/docker"
	"github.com/bufbuild/plugins/internal/plugin"
)

// dockerbuild is a helper program used to build plugins from Dockerfiles.
// It also makes it easier to add new labels to images using existing code to parse buf.plugin.yaml.

func main() {
	var (
		dir = flag.String("dir", ".", "Directory path to plugins")
		org = flag.String("org", "bufbuild", "Docker Organization")
	)
	flag.Parse()
	if err := run(*dir, *org, flag.Args()); err != nil {
		log.Fatalf("failed to build: %v", err)
	}
}

func run(basedir string, dockerOrg string, args []string) error {
	// Catch ctrl+c to kill the build process
	ctx, cancel := interrupt.WithCancel(context.Background())
	defer cancel()
	plugins, err := plugin.FindAll(basedir)
	if err != nil {
		return err
	}
	includedPlugins, err := plugin.FilterByPluginsEnv(plugins, os.Getenv("PLUGINS"))
	if err != nil {
		return err
	}
	if len(includedPlugins) == 0 {
		return nil // nothing to build
	}
	for _, pluginToBuild := range includedPlugins {
		log.Println("building:", pluginToBuild.Name, pluginToBuild.PluginVersion)
		start := time.Now()
		output, err := docker.Build(ctx, pluginToBuild, dockerOrg, args)
		if err != nil {
			if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "signal: killed") {
				return err
			}
			return fmt.Errorf(
				"failed to build %s:%s: %w\noutput:\n%s",
				pluginToBuild.Name,
				pluginToBuild.PluginVersion,
				err,
				string(output),
			)
		}
		elapsed := time.Since(start)
		log.Println("built:", pluginToBuild.Name, pluginToBuild.PluginVersion, "in", elapsed.Round(time.Second))
	}
	return nil
}
