package main

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/bufbuild/buf/private/pkg/interrupt"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"

	"github.com/bufbuild/plugins/internal/docker"
	"github.com/bufbuild/plugins/internal/plugin"
)

// dockerbuild is a helper program used to build plugins from Dockerfiles in an optimized fashion.
// It replaces some clunky (and non-parallel) jobs in the Makefile.
// It knows about relationships between common containers (like protoc/grpc Bazel plugins),
// which enables optimized builds which build common code first before binaries.
// It also makes it easier to add new labels to images using existing code to parse buf.plugin.yaml.

const (
	// larger amount of parallelism lead to OOM errors in testing - clamp for now.
	maxLimit = 8
	// plugin group for all bazel builds (they are run serially on their own as they are very resource intensive).
	bazelPluginGroup = "bazel"
)

func main() {
	var (
		dir      = flag.String("dir", ".", "Directory path to plugins")
		org      = flag.String("org", "bufbuild", "Docker Organization")
		cacheDir = flag.String("cache-dir", "", "Cache directory")
	)
	flag.Parse()
	cmd := &command{
		pluginsDir:      *dir,
		dockerOrg:       *org,
		cacheDir:        *cacheDir,
		dockerBuildArgs: flag.Args(),
	}
	if err := cmd.run(); err != nil {
		log.Fatalf("failed to build: %v", err)
	}
}

type command struct {
	// pluginsDir specifies the directory where plugins are found (typically the root of the bufbuild/plugins repo).
	pluginsDir string
	// dockerOrg is the Docker organization to use in the tagged image.
	dockerOrg string
	// cacheDir is an optional setting to the root where Docker buildx local caches should be kept.
	cacheDir string
	// dockerBuildArgs contains additional arguments to pass to the Docker build.
	dockerBuildArgs []string
}

func (c *command) run() error {
	// Catch ctrl+c to kill the build process
	ctx, cancel := interrupt.NotifyContext(context.Background())
	defer cancel()
	allPlugins, err := plugin.FindAll(c.pluginsDir)
	if err != nil {
		return err
	}
	includedPlugins, err := plugin.FilterByPluginsEnv(allPlugins, os.Getenv("PLUGINS"))
	if err != nil {
		return err
	}
	if len(includedPlugins) == 0 {
		return nil // nothing to build
	}
	pluginGroups := getPluginGroups(includedPlugins)
	// Build bazel plugins on their own as they are very resource intensive.
	if bazelPlugins, ok := pluginGroups[bazelPluginGroup]; ok {
		delete(pluginGroups, bazelPluginGroup)
		// Sort plugins to build first by version, then by name.
		// This ensures the best use of the Docker build cache for expensive builds like protoc/grpc plugins.
		slices.SortFunc(bazelPlugins, func(a, b *plugin.Plugin) int {
			if v := semver.Compare(a.PluginVersion, b.PluginVersion); v != 0 {
				return v
			}
			return cmp.Compare(a.Name, b.Name)
		})
		if err := c.buildPluginGroup(ctx, bazelPluginGroup, bazelPlugins); err != nil {
			return err
		}
		if len(pluginGroups) == 0 {
			return nil
		}
	}
	limit := runtime.GOMAXPROCS(0)
	if limit > maxLimit {
		limit = maxLimit
	}
	var eg *errgroup.Group
	eg, ctx = errgroup.WithContext(ctx)
	eg.SetLimit(limit)
	for pluginGroup, plugins := range pluginGroups {
		eg.Go(func() error {
			return c.buildPluginGroup(ctx, pluginGroup, plugins)
		})
	}
	return eg.Wait()
}

func getPluginGroups(plugins []*plugin.Plugin) map[string][]*plugin.Plugin {
	pluginGroups := make(map[string][]*plugin.Plugin)
	for _, pluginToBuild := range plugins {
		var pluginKey string
		identity := pluginToBuild.Identity
		// Group grpc/protobuf builds together so one finishes to completion before running additional jobs.
		// This is important because builds can share the Docker cache to optimize longer Bazel builds.
		switch owner := identity.Owner(); owner {
		case "grpc":
			switch identity.Plugin() {
			case "cpp", "csharp", "objc", "php", "python", "ruby":
				pluginKey = bazelPluginGroup
			}
		case "protocolbuffers":
			switch identity.Plugin() {
			case "cpp", "csharp", "java", "kotlin", "objc", "php", "pyi", "python", "ruby", "rust":
				pluginKey = bazelPluginGroup
			}
		default:
			// Assume everything else can be built independently
			pluginKey = identity.IdentityString()
		}
		pluginGroups[pluginKey] = append(pluginGroups[pluginKey], pluginToBuild)
	}
	return pluginGroups
}

func (c *command) buildPluginGroup(ctx context.Context, pluginGroup string, plugins []*plugin.Plugin) error {
	for _, pluginToBuild := range plugins {
		pluginIdentity := pluginToBuild.Identity
		var pluginCacheDir string
		// If cache is enabled, create a unique cache directory per group of plugins.
		// This enables bazel builds to share a common cache while allowing parallel builds.
		if c.cacheDir != "" {
			if pluginGroup == bazelPluginGroup {
				pluginCacheDir = filepath.Join(c.cacheDir, pluginGroup, pluginIdentity.Owner(), pluginToBuild.PluginVersion)
			} else {
				pluginCacheDir = filepath.Join(c.cacheDir, pluginIdentity.Owner(), pluginIdentity.Plugin(), pluginToBuild.PluginVersion)
			}
		}
		if !c.shouldBuild(ctx, pluginToBuild) {
			log.Println("skipping (up-to-date):", pluginToBuild.Name, pluginToBuild.PluginVersion)
			continue
		}
		log.Println("building:", pluginToBuild.Name, pluginToBuild.PluginVersion)
		start := time.Now()
		imageName := docker.ImageName(pluginToBuild, c.dockerOrg)
		output, err := docker.Build(ctx, pluginToBuild, imageName, pluginCacheDir, c.dockerBuildArgs)
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

func (c *command) shouldBuild(ctx context.Context, plugin *plugin.Plugin) bool {
	gitCommit := plugin.GitCommit(ctx)
	// There are uncommitted changes to the plugin's directory - we should always rebuild.
	if gitCommit == "" {
		return true
	}
	imageName := docker.ImageName(plugin, c.dockerOrg)
	cmd := exec.CommandContext(
		ctx,
		"docker",
		"image",
		"inspect",
		imageName,
		"--format",
		`{{ index .Config.Labels "org.opencontainers.image.revision" }}`,
	)
	output, err := cmd.Output()
	if err != nil {
		// This could be because the image doesn't exist or any other error.
		// Err on the side of caution and assume that we should rebuild the image.
		return true
	}
	return strings.TrimSpace(string(output)) != gitCommit
}
