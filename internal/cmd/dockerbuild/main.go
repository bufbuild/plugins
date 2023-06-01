package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"
	"github.com/bufbuild/buf/private/pkg/interrupt"
	"github.com/bufbuild/plugins/internal/docker"
	"github.com/bufbuild/plugins/internal/plugin"
	"golang.org/x/sync/errgroup"
)

// dockerbuild is a helper program used to build plugins from Dockerfiles in an optimized fashion.
// It replaces some clunky (and non-parallel) jobs in the Makefile.
// It knows about relationships between common containers (like protoc/grpc Bazel plugins),
// which enables optimized builds which build common code first before binaries.
// It also makes it easier to add new labels to images using existing code to parse buf.plugin.yaml.

// larger amount of parallelism lead to OOM errors in testing - clamp for now.
const maxLimit = 8

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
	pluginGroups := make(map[string][]*plugin.Plugin)
	for _, pluginToBuild := range includedPlugins {
		identity, err := bufpluginref.PluginIdentityForString(pluginToBuild.Name)
		if err != nil {
			return err
		}
		var pluginKey string
		// Group grpc/protobuf builds together so one finishes to completion before running additional jobs.
		// This is important because builds can share the Docker cache to optimize longer Bazel builds.
		switch owner := identity.Owner(); owner {
		case "grpc":
			switch identity.Plugin() {
			case "cpp", "csharp", "objc", "php", "python", "ruby":
				pluginKey = owner + "/" + pluginToBuild.PluginVersion
			}
		case "protocolbuffers":
			switch identity.Plugin() {
			case "cpp", "csharp", "java", "kotlin", "objc", "php", "pyi", "python", "ruby":
				pluginKey = owner + "/" + pluginToBuild.PluginVersion
			}
		default:
			// Assume everything else can be built independently
			pluginKey = identity.IdentityString()
		}
		pluginGroups[pluginKey] = append(pluginGroups[pluginKey], pluginToBuild)
	}
	limit := runtime.GOMAXPROCS(0)
	if limit > maxLimit {
		limit = maxLimit
	}
	var eg *errgroup.Group
	eg, ctx = errgroup.WithContext(ctx)
	eg.SetLimit(limit)
	for _, plugins := range pluginGroups {
		plugins := plugins
		eg.Go(func() error {
			for _, pluginToBuild := range plugins {
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
		})
	}
	return eg.Wait()
}
