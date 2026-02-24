package maven

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
)

// MergeTransitiveDeps resolves Maven dependencies from the top-level deps
// stanza in the plugin config and merges them into the plugin's Maven
// registry config. Dependencies are resolved transitively so that all
// Maven artifacts needed for offline builds are included in the POM.
func MergeTransitiveDeps(
	pluginConfig *bufremotepluginconfig.Config,
	pluginsDir string,
) error {
	if pluginConfig.Registry == nil || pluginConfig.Registry.Maven == nil {
		return nil
	}
	visited := make(map[string]bool)
	return mergeTransitiveDepsRecursive(pluginConfig, pluginsDir, visited)
}

func mergeTransitiveDepsRecursive(
	pluginConfig *bufremotepluginconfig.Config,
	pluginsDir string,
	visited map[string]bool,
) error {
	for _, dep := range pluginConfig.Dependencies {
		depKey := dep.IdentityString() + ":" + dep.Version()
		if visited[depKey] {
			continue
		}
		visited[depKey] = true
		depPath := filepath.Join(
			pluginsDir, dep.Owner(), dep.Plugin(),
			dep.Version(), "buf.plugin.yaml",
		)
		depConfig, err := bufremotepluginconfig.ParseConfig(depPath)
		if err != nil {
			return fmt.Errorf("loading dep config %s from %s: %w", depKey, depPath, err)
		}
		// Recursively resolve transitive dependencies first so
		// that depConfig.Registry.Maven accumulates the full
		// transitive closure before we merge into pluginConfig.
		if err := mergeTransitiveDepsRecursive(depConfig, pluginsDir, visited); err != nil {
			return err
		}
		if depConfig.Registry == nil || depConfig.Registry.Maven == nil {
			continue
		}
		depMaven := depConfig.Registry.Maven
		pluginConfig.Registry.Maven.Deps = append(
			pluginConfig.Registry.Maven.Deps, depMaven.Deps...,
		)
		// Merge additional runtimes: for matching runtime names,
		// append deps; otherwise add the new runtime entry.
		for _, depRuntime := range depMaven.AdditionalRuntimes {
			merged := false
			for i, runtime := range pluginConfig.Registry.Maven.AdditionalRuntimes {
				if runtime.Name == depRuntime.Name {
					pluginConfig.Registry.Maven.AdditionalRuntimes[i].Deps = append(
						pluginConfig.Registry.Maven.AdditionalRuntimes[i].Deps,
						depRuntime.Deps...,
					)
					merged = true
					break
				}
			}
			if !merged {
				pluginConfig.Registry.Maven.AdditionalRuntimes = append(
					pluginConfig.Registry.Maven.AdditionalRuntimes,
					depRuntime,
				)
			}
		}
	}
	return nil
}

// DeduplicateAllDeps deduplicates across the main Deps and all
// AdditionalRuntimes Deps using a shared seen set. This ensures the
// flat <dependencies> block in the rendered POM contains no duplicates.
func DeduplicateAllDeps(
	ctx context.Context,
	mavenConfig *bufremotepluginconfig.MavenRegistryConfig,
) {
	if mavenConfig == nil {
		return
	}
	seen := make(map[string]string)
	mavenConfig.Deps = deduplicateWithSeen(ctx, mavenConfig.Deps, seen)
	for i := range mavenConfig.AdditionalRuntimes {
		mavenConfig.AdditionalRuntimes[i].Deps = deduplicateWithSeen(
			ctx, mavenConfig.AdditionalRuntimes[i].Deps, seen,
		)
	}
}

func deduplicateWithSeen(
	ctx context.Context,
	deps []bufremotepluginconfig.MavenDependencyConfig,
	seen map[string]string,
) []bufremotepluginconfig.MavenDependencyConfig {
	var result []bufremotepluginconfig.MavenDependencyConfig
	for _, dep := range deps {
		key := dep.GroupID + ":" + dep.ArtifactID
		if dep.Classifier != "" {
			key += ":" + dep.Classifier
		}
		if existingVersion, ok := seen[key]; ok {
			if existingVersion != dep.Version {
				slog.WarnContext(ctx, "duplicate Maven dependency",
					slog.String("key", key),
					slog.String("keeping", existingVersion),
					slog.String("discarding", dep.Version),
				)
			}
			continue
		}
		seen[key] = dep.Version
		result = append(result, dep)
	}
	return result
}
