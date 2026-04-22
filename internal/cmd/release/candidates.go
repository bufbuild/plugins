package main

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/google/go-github/v72/github"
	"golang.org/x/mod/semver"

	"github.com/bufbuild/plugins/internal/git"
	"github.com/bufbuild/plugins/internal/plugin"
	"github.com/bufbuild/plugins/internal/release"
)

// collectCandidates returns the set of plugin (name, version) pairs that may
// have changed since latestRelease.
//
// A nil result means "treat every plugin as a candidate" and is returned when
// there is no prior release (initial release).
//
// Candidates are the union of three sources:
//   - buf.plugin.yaml files changed since the prior release tag
//   - GHCR container packages with versions updated since the prior release
//   - plugins selected via the PLUGINS env var (escape hatch)
func (c *command) collectCandidates(
	ctx context.Context,
	ghClient *release.Client,
	allPlugins []*plugin.Plugin,
	latestRelease *github.RepositoryRelease,
) (map[pluginNameVersion]struct{}, error) {
	if latestRelease == nil {
		return nil, nil
	}
	candidates := make(map[pluginNameVersion]struct{})
	if tag := latestRelease.GetTagName(); tag != "" {
		added, err := c.addGitCandidates(ctx, tag, candidates)
		if err != nil {
			return nil, fmt.Errorf("git candidates: %w", err)
		}
		c.logger.InfoContext(ctx, "candidates from git diff",
			slog.String("tag", tag),
			slog.Any("plugins", added),
		)
	}
	// Reach back 30 minutes before the prior run started to catch images pushed
	// concurrent with it.
	since := latestRelease.GetCreatedAt().Add(-30 * time.Minute)
	added, err := c.addGHCRCandidates(ctx, ghClient, allPlugins, since, candidates)
	if err != nil {
		return nil, fmt.Errorf("ghcr candidates: %w", err)
	}
	c.logger.InfoContext(ctx, "candidates from ghcr",
		slog.Time("since", since),
		slog.Any("plugins", added),
	)
	added, err = c.addPluginsEnvCandidates(allPlugins, candidates)
	if err != nil {
		return nil, fmt.Errorf("plugins env candidates: %w", err)
	}
	c.logger.InfoContext(ctx, "candidates from PLUGINS env var",
		slog.Any("plugins", added),
	)
	return candidates, nil
}

// sortedKeys returns a copy of keys sorted by name then version for stable
// log output.
func sortedKeys(keys []pluginNameVersion) []pluginNameVersion {
	out := slices.Clone(keys)
	slices.SortFunc(out, func(a, b pluginNameVersion) int {
		if c := cmp.Compare(a.name, b.name); c != 0 {
			return c
		}
		return cmp.Compare(a.version, b.version)
	})
	return out
}

// addGitCandidates adds (name, version) pairs for every plugin whose
// buf.plugin.yaml changed since ref.
//
// Only buf.plugin.yaml is scanned: changes to Dockerfile/patches/etc. rebuild
// the image and are picked up by the GHCR pass, while changes to unreferenced
// files (README, etc.) affect neither yaml_digest nor image_id and wouldn't
// trigger a republish even if flagged.
func (c *command) addGitCandidates(ctx context.Context, ref string, candidates map[pluginNameVersion]struct{}) ([]pluginNameVersion, error) {
	changedFiles, err := git.ChangedFilesFrom(ctx, ref)
	if err != nil {
		return nil, err
	}
	var added []pluginNameVersion
	for _, file := range changedFiles {
		key, ok := pluginKeyFromPath(file)
		if !ok {
			continue
		}
		if _, exists := candidates[key]; exists {
			continue
		}
		candidates[key] = struct{}{}
		added = append(added, key)
	}
	return sortedKeys(added), nil
}

// pluginKeyFromPath parses "plugins/<owner>/<name>/<semver>/buf.plugin.yaml"
// into {name: "<owner>/<name>", version: "<semver>"}. Any other path returns
// (_, false).
func pluginKeyFromPath(path string) (pluginNameVersion, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(path), "plugins/")
	if !ok {
		return pluginNameVersion{}, false
	}
	parts := strings.Split(rest, "/")
	if len(parts) != 4 || parts[3] != "buf.plugin.yaml" {
		return pluginNameVersion{}, false
	}
	if !semver.IsValid(parts[2]) {
		return pluginNameVersion{}, false
	}
	return pluginNameVersion{
		name:    parts[0] + "/" + parts[1],
		version: parts[2],
	}, true
}

// addGHCRCandidates adds (name, version) pairs for every container package
// version whose image was updated after since.
//
// The list-packages endpoint returns every container package owned by the org
// (a few dozen), and per-package version listings are only fetched for packages
// that were touched after since.
func (c *command) addGHCRCandidates(
	ctx context.Context,
	ghClient *release.Client,
	allPlugins []*plugin.Plugin,
	since time.Time,
	candidates map[pluginNameVersion]struct{},
) ([]pluginNameVersion, error) {
	packageToPlugin := make(map[string]string, len(allPlugins))
	for _, p := range allPlugins {
		pkg := fmt.Sprintf("plugins-%s-%s", p.Identity.Owner(), p.Identity.Plugin())
		packageToPlugin[pkg] = p.Identity.Owner() + "/" + p.Identity.Plugin()
	}
	var added []pluginNameVersion
	opts := &github.PackageListOptions{
		PackageType: new("container"),
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		pkgs, resp, err := ghClient.GitHub.Organizations.ListPackages(ctx, string(release.GithubOwnerBufbuild), opts)
		if err != nil {
			return nil, fmt.Errorf("list packages: %w", err)
		}
		for _, pkg := range pkgs {
			if pkg.GetUpdatedAt().Before(since) {
				continue
			}
			pluginName, ok := packageToPlugin[pkg.GetName()]
			if !ok {
				continue
			}
			pkgAdded, err := c.addPackageVersionCandidates(ctx, ghClient, pkg.GetName(), pluginName, since, candidates)
			if err != nil {
				return nil, err
			}
			added = append(added, pkgAdded...)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return sortedKeys(added), nil
}

// addPackageVersionCandidates adds one entry per semver tag found on any
// package version updated after since. It returns the keys newly added to the
// candidate set by this pass.
func (c *command) addPackageVersionCandidates(
	ctx context.Context,
	ghClient *release.Client,
	pkgName, pluginName string,
	since time.Time,
	candidates map[pluginNameVersion]struct{},
) ([]pluginNameVersion, error) {
	var added []pluginNameVersion
	opts := &github.PackageListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		versions, resp, err := ghClient.GitHub.Organizations.PackageGetAllVersions(
			ctx, string(release.GithubOwnerBufbuild), "container", pkgName, opts,
		)
		if err != nil {
			return nil, fmt.Errorf("list %q versions: %w", pkgName, err)
		}
		for _, v := range versions {
			if v.GetUpdatedAt().Before(since) {
				continue
			}
			meta, ok := v.GetMetadata()
			if !ok || meta.Container == nil {
				continue
			}
			for _, tag := range meta.Container.Tags {
				// Skip moving tags (latest, v1, v1.2); only canonical semver
				// corresponds to a plugin version directory.
				if semver.Canonical(tag) != tag {
					continue
				}
				key := pluginNameVersion{name: pluginName, version: tag}
				if _, exists := candidates[key]; exists {
					continue
				}
				candidates[key] = struct{}{}
				added = append(added, key)
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return added, nil
}

// addPluginsEnvCandidates applies the PLUGINS env var as an escape hatch. The
// selected plugins are added to the candidate set; they still must have a
// differing yaml or image digest to be republished.
func (c *command) addPluginsEnvCandidates(allPlugins []*plugin.Plugin, candidates map[pluginNameVersion]struct{}) ([]pluginNameVersion, error) {
	pluginsEnv := os.Getenv("PLUGINS")
	if pluginsEnv == "" {
		return nil, nil
	}
	selected, err := plugin.FilterByPluginsEnv(allPlugins, pluginsEnv)
	if err != nil {
		return nil, err
	}
	var added []pluginNameVersion
	for _, p := range selected {
		key := pluginNameVersion{
			name:    p.Identity.Owner() + "/" + p.Identity.Plugin(),
			version: p.PluginVersion,
		}
		if _, exists := candidates[key]; exists {
			continue
		}
		candidates[key] = struct{}{}
		added = append(added, key)
	}
	return sortedKeys(added), nil
}
