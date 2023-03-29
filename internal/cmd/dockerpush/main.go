package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/bufbuild/buf/private/pkg/interrupt"

	"github.com/bufbuild/plugins/internal/docker"
	"github.com/bufbuild/plugins/internal/plugin"
)

// dockerpush automates pushing images build by dockerbuild to a remote registry.

func main() {
	var (
		dir = flag.String("dir", ".", "Directory path to plugins")
		org = flag.String("org", "bufbuild", "Docker Organization")
	)
	flag.Parse()
	if err := run(*dir, *org); err != nil {
		log.Fatalf("failed to build: %v", err)
	}
}

func run(basedir string, dockerOrg string) error {
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
	for _, includedPlugin := range includedPlugins {
		output, err := docker.Push(ctx, includedPlugin, dockerOrg)
		if err != nil {
			log.Printf(
				"docker push of plugin %s:%s failed with err %v:\noutput:\n%s",
				includedPlugin.Name,
				includedPlugin.PluginVersion,
				err,
				string(output),
			)
			return err
		}
		log.Printf("pushed plugin %s:%s", includedPlugin.Name, includedPlugin.PluginVersion)
	}
	return nil
}
