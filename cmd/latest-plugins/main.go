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
	"slices"
	"strings"

	"aead.dev/minisign"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/spf13/pflag"
	"golang.org/x/mod/semver"

	"github.com/bufbuild/plugins/internal/release"
)

func main() {
	appcmd.Main(context.Background(), newRootCommand("latest-plugins"))
}

type flags struct {
	includedPlugins []string
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVar(&f.includedPlugins,
		"include",
		nil,
		"Additional plugins to include",
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
	client := release.NewClient(ctx)
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
	latestPlugins, err := getLatestPluginsAndDependencies(&pluginReleases, flags.includedPlugins)
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
	additionalPlugins []string,
) ([]release.PluginRelease, error) {
	additionalPluginsSet := slicesext.ToStructMap(additionalPlugins)
	versionToPlugin := make(map[string]release.PluginRelease, len(releases.Releases))
	latestVersions := make(map[string]release.PluginRelease)
	for _, pluginRelease := range releases.Releases {
		owner, pluginName, found := strings.Cut(pluginRelease.PluginName, "/")
		if !found {
			return nil, errors.New("failed to split plugin pluginName into owner/pluginName")
		}
		if _, ok := additionalPluginsSet[pluginRelease.PluginName]; !ok {
			// Don't include deprecated plugins.
			switch owner {
			case "community":
				if pluginName == "mitchellh-go-json" {
					continue
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
					continue
				}
			}
		}
		versionToPlugin[pluginRelease.PluginName+":"+pluginRelease.PluginVersion] = pluginRelease
		latestVersion, ok := latestVersions[pluginRelease.PluginName]
		if !ok || semver.Compare(latestVersion.PluginVersion, pluginRelease.PluginVersion) < 0 {
			latestVersions[pluginRelease.PluginName] = pluginRelease
		}
	}
	toInclude := make(map[string]struct{})
	deps := make(map[string]struct{})
	for _, pluginRelease := range latestVersions {
		toInclude[pluginRelease.PluginName+":"+pluginRelease.PluginVersion] = struct{}{}
		for _, d := range pluginRelease.Dependencies {
			deps[strings.TrimPrefix(d, "buf.build/")] = struct{}{}
		}
	}
	for len(deps) > 0 {
		nextDeps := make(map[string]struct{})
		for dep := range deps {
			if _, ok := toInclude[dep]; ok {
				continue
			}
			toInclude[dep] = struct{}{}
			for _, nextDep := range versionToPlugin[dep].Dependencies {
				nextDeps[strings.TrimPrefix(nextDep, "buf.build/")] = struct{}{}
			}
		}
		deps = nextDeps
	}
	var latestPluginsAndDeps []release.PluginRelease
	for _, pluginRelease := range releases.Releases {
		if _, ok := toInclude[pluginRelease.PluginName+":"+pluginRelease.PluginVersion]; ok {
			latestPluginsAndDeps = append(latestPluginsAndDeps, pluginRelease)
		}
	}
	return latestPluginsAndDeps, nil
}
