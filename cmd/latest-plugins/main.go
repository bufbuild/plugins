/*
latest-plugins outputs the latest plugins (and their dependencies) in JSON format to stdout.
To determine available plugins, it downloads the plugin-releases.json file from the latest bufbuild/plugins release.
Additionally, it verifies the contents of the file against the minisign signature.

This utility is used downstream by some other tooling to package up plugins to install in the BSR.
*/
package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"aead.dev/minisign"
	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"github.com/spf13/pflag"
	"golang.org/x/mod/semver"

	"github.com/bufbuild/plugins/internal/release"
)

func main() {
	appcmd.Main(context.Background(), newRootCommand("latest-plugins"))
}

type flags struct {
	includedPluginsFile string
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.includedPluginsFile,
		"include-file",
		"",
		"JSON file containing previous plugins to include",
	)
}

func newRootCommand(name string) *appcmd.Command {
	builder := appext.NewBuilder(name)
	flags := &flags{}
	return &appcmd.Command{
		Use:   name,
		Short: "Outputs the latest plugins in JSON format to stdout.",
		Run: builder.NewRunFunc(func(ctx context.Context, container appext.Container) error {
			return run(ctx, container, flags)
		}),
		BindFlags:           flags.Bind,
		BindPersistentFlags: builder.BindRoot,
	}
}

func run(ctx context.Context, container appext.Container, flags *flags) error {
	client := release.NewClient()
	latestRelease, err := client.GetLatestRelease(ctx, release.GithubOwnerBufbuild, release.GithubRepoPlugins)
	if err != nil {
		return fmt.Errorf("failed to determine latest %s/%s release: %w", release.GithubOwnerBufbuild, release.GithubRepoPlugins, err)
	}
	releasesBytes, _, err := client.DownloadAsset(ctx, latestRelease, release.PluginReleasesFile)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", release.PluginReleasesFile, err)
	}
	releasesMinisigBytes, _, err := client.DownloadAsset(ctx, latestRelease, release.PluginReleasesSignatureFile)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", release.PluginReleasesSignatureFile, err)
	}
	publicKey, err := release.DefaultPublicKey()
	if err != nil {
		return fmt.Errorf("failed to load minisign public key: %w", err)
	}
	if !minisign.Verify(publicKey, releasesBytes, releasesMinisigBytes) {
		return errors.New("failed to verify plugin-releases.json")
	}
	var pluginReleases release.PluginReleases
	if err := json.NewDecoder(bytes.NewReader(releasesBytes)).Decode(&pluginReleases); err != nil {
		return err
	}
	var includedPlugins []nameVersion
	if flags.includedPluginsFile != "" {
		includedPlugins, err = pluginsFromFile(flags.includedPluginsFile)
		if err != nil {
			return err
		}
	}
	latestPlugins, err := getLatestPluginsAndDependencies(&pluginReleases, includedPlugins)
	if err != nil {
		return fmt.Errorf("failed to determine latest plugins and dependencies: %w", err)
	}
	// sort first by plugin name + version
	slices.SortFunc(latestPlugins, func(a, b release.PluginRelease) int {
		if c := cmp.Compare(a.PluginName, b.PluginName); c != 0 {
			return c
		}
		return semver.Compare(a.PluginVersion, b.PluginVersion)
	})
	// sort by dependency order
	sortedPlugins, err := release.SortReleasesInDependencyOrder(latestPlugins)
	if err != nil {
		return fmt.Errorf("failed to sort plugins in dependency order: %w", err)
	}
	enc := json.NewEncoder(container.Stdout())
	enc.SetIndent("", "  ")
	return enc.Encode(&release.PluginReleases{Releases: sortedPlugins})
}

func getLatestPluginsAndDependencies(
	releases *release.PluginReleases,
	additionalPlugins []nameVersion,
) ([]release.PluginRelease, error) {
	nameVersionToRelease := make(map[nameVersion]release.PluginRelease, len(releases.Releases))
	for _, plugin := range releases.Releases {
		nameVersionToRelease[nameVersion{name: plugin.PluginName, version: plugin.PluginVersion}] = plugin
	}
	toInclude := make(map[nameVersion]struct{})
	latestVersions := latestNonDeprecatedPlugins(releases)
	deps := make(map[nameVersion]struct{})
	addDeps := func(pluginRelease release.PluginRelease) {
		for _, depNameVersion := range pluginRelease.Dependencies {
			depName, depVersion, _ := strings.Cut(strings.TrimPrefix(depNameVersion, "buf.build/"), ":")
			deps[nameVersion{name: depName, version: depVersion}] = struct{}{}
		}
	}
	for _, pluginRelease := range latestVersions {
		toInclude[nameVersion{name: pluginRelease.PluginName, version: pluginRelease.PluginVersion}] = struct{}{}
		addDeps(pluginRelease)
	}
	for _, additionalPlugin := range additionalPlugins {
		toInclude[additionalPlugin] = struct{}{}
		pluginRelease, ok := nameVersionToRelease[additionalPlugin]
		if !ok {
			return nil, fmt.Errorf("no plugin found for %s", additionalPlugin)
		}
		addDeps(pluginRelease)
	}
	for len(deps) > 0 {
		nextDeps := make(map[nameVersion]struct{})
		for dep := range deps {
			if _, ok := toInclude[dep]; ok {
				continue
			}
			toInclude[dep] = struct{}{}
			for _, nextDep := range nameVersionToRelease[dep].Dependencies {
				depName, depVersion, _ := strings.Cut(strings.TrimPrefix(nextDep, "buf.build/"), ":")
				nextDeps[nameVersion{name: depName, version: depVersion}] = struct{}{}
			}
		}
		deps = nextDeps
	}
	var latestPluginsAndDeps []release.PluginRelease
	for _, pluginRelease := range releases.Releases {
		nameVersion := nameVersion{name: pluginRelease.PluginName, version: pluginRelease.PluginVersion}
		if _, ok := toInclude[nameVersion]; ok {
			latestPluginsAndDeps = append(latestPluginsAndDeps, pluginRelease)
		}
	}
	return latestPluginsAndDeps, nil
}

func latestNonDeprecatedPlugins(releases *release.PluginReleases) []release.PluginRelease {
	latestPluginNameToRelease := make(map[string]release.PluginRelease)
	for _, pluginRelease := range releases.Releases {
		if isDeprecated(pluginRelease.PluginName) {
			continue
		}
		latestVersion, ok := latestPluginNameToRelease[pluginRelease.PluginName]
		if !ok || semver.Compare(latestVersion.PluginVersion, pluginRelease.PluginVersion) < 0 {
			latestPluginNameToRelease[pluginRelease.PluginName] = pluginRelease
		}
	}
	latestPlugins := slices.Collect(maps.Values(latestPluginNameToRelease))
	slices.SortFunc(latestPlugins, func(a, b release.PluginRelease) int {
		return cmp.Compare(a.PluginName, b.PluginName)
	})
	return latestPlugins
}

func isDeprecated(pluginName string) bool {
	owner, pluginName, _ := strings.Cut(pluginName, "/")
	// Don't include deprecated plugins.
	switch owner {
	case "community":
		if pluginName == "mitchellh-go-json" {
			return true
		}
	case "bufbuild":
		switch pluginName {
		case "connect-es",
			"connect-go",
			"connect-kotlin",
			"connect-query",
			"connect-swift",
			"connect-swift-mocks",
			"connect-web",
			"protoschema-bigquery":
			return true
		}
	}
	return false
}

func pluginsFromFile(filename string) (_ []nameVersion, retErr error) {
	var pluginReleases release.PluginReleases
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, f.Close())
	}()
	if err := json.NewDecoder(f).Decode(&pluginReleases); err != nil {
		return nil, err
	}
	plugins := make([]nameVersion, 0, len(pluginReleases.Releases))
	for _, pluginRelease := range pluginReleases.Releases {
		plugins = append(plugins, nameVersion{name: pluginRelease.PluginName, version: pluginRelease.PluginVersion})
	}
	return plugins, nil
}

type nameVersion struct {
	name    string
	version string
}

func (p nameVersion) String() string {
	return fmt.Sprintf("%s:%s", p.name, p.version)
}
