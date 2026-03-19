package maven

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"golang.org/x/mod/semver"
)

// RegenerateMavenDeps processes a Maven plugin version directory by
// merging transitive deps, deduplicating, and rendering POM to a
// pom.xml file. When transitive deps bring in newer versions of
// artifacts already pinned in the plugin's buf.plugin.yaml, the YAML
// file is updated first so that deduplication and POM generation see
// consistent versions. Returns nil without changes if the plugin has
// no Maven registry config.
func RegenerateMavenDeps(pluginVersionDir, pluginsDir string) error {
	yamlPath := filepath.Join(pluginVersionDir, "buf.plugin.yaml")
	pluginConfig, err := bufremotepluginconfig.ParseConfig(yamlPath)
	if err != nil {
		return err
	}
	if pluginConfig.Registry == nil || pluginConfig.Registry.Maven == nil {
		return nil
	}
	// Collect the versions declared by transitive deps so we can detect
	// stale pins in the plugin's own buf.plugin.yaml.
	transitiveDeps, err := collectTransitiveMavenDeps(pluginConfig, pluginsDir)
	if err != nil {
		return fmt.Errorf("collecting transitive deps: %w", err)
	}
	// Update buf.plugin.yaml if any direct dep versions are older than
	// what transitive deps declare.
	if err := updateBufPluginYAML(yamlPath, pluginConfig.Registry.Maven, transitiveDeps); err != nil {
		return fmt.Errorf("updating buf.plugin.yaml: %w", err)
	}
	// Re-parse the (potentially updated) YAML so the in-memory config
	// reflects the updated versions.
	pluginConfig, err = bufremotepluginconfig.ParseConfig(yamlPath)
	if err != nil {
		return err
	}
	if err := MergeTransitiveDeps(pluginConfig, pluginsDir); err != nil {
		return fmt.Errorf("merging transitive deps: %w", err)
	}
	if err := DeduplicateAllDeps(pluginConfig.Registry.Maven); err != nil {
		return fmt.Errorf("deduplicating deps: %w", err)
	}
	pom, err := RenderPOM(pluginConfig)
	if err != nil {
		return fmt.Errorf("rendering POM: %w", err)
	}
	pomPath := filepath.Join(pluginVersionDir, "pom.xml")
	if err := os.WriteFile(pomPath, []byte(pom), 0644); err != nil { //nolint:gosec
		return fmt.Errorf("writing pom.xml: %w", err)
	}
	return nil
}

// mavenDepKey returns the deduplication key for a Maven dependency
// (groupId:artifactId, optionally with classifier).
func mavenDepKey(dep bufremotepluginconfig.MavenDependencyConfig) string {
	key := dep.GroupID + ":" + dep.ArtifactID
	if dep.Classifier != "" {
		key += ":" + dep.Classifier
	}
	return key
}

// collectTransitiveMavenDeps walks the plugin's dependency tree and
// returns a map of artifact key -> version for every Maven dep found
// in transitive dependencies. This does not mutate pluginConfig.
func collectTransitiveMavenDeps(
	pluginConfig *bufremotepluginconfig.Config,
	pluginsDir string,
) (map[string]string, error) {
	versions := make(map[string]string)
	visited := make(map[string]bool)
	if err := collectTransitiveMavenDepsRecursive(pluginConfig, pluginsDir, visited, versions); err != nil {
		return nil, err
	}
	return versions, nil
}

func collectTransitiveMavenDepsRecursive(
	pluginConfig *bufremotepluginconfig.Config,
	pluginsDir string,
	visited map[string]bool,
	versions map[string]string,
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
		if err := collectTransitiveMavenDepsRecursive(depConfig, pluginsDir, visited, versions); err != nil {
			return err
		}
		if depConfig.Registry == nil || depConfig.Registry.Maven == nil {
			continue
		}
		for _, d := range depConfig.Registry.Maven.Deps {
			versions[mavenDepKey(d)] = d.Version
		}
		for _, rt := range depConfig.Registry.Maven.AdditionalRuntimes {
			for _, d := range rt.Deps {
				versions[mavenDepKey(d)] = d.Version
			}
		}
	}
	return nil
}

// updateBufPluginYAML rewrites buf.plugin.yaml when transitive deps
// declare newer versions of artifacts already pinned in the plugin's
// Maven config. It performs targeted text replacements of the Maven
// dep strings (group:artifact:oldVer -> group:artifact:newVer) so
// that comments and formatting are preserved.
func updateBufPluginYAML(
	yamlPath string,
	maven *bufremotepluginconfig.MavenRegistryConfig,
	transitiveDeps map[string]string,
) error {
	replacements := make(map[string]string) // "group:artifact:old" -> "group:artifact:new"
	collectReplacements := func(deps []bufremotepluginconfig.MavenDependencyConfig) {
		for _, dep := range deps {
			key := mavenDepKey(dep)
			transitiveVersion, ok := transitiveDeps[key]
			if !ok || transitiveVersion == dep.Version {
				continue
			}
			// Only upgrade if the transitive version is higher.
			oldSemver := "v" + dep.Version
			newSemver := "v" + transitiveVersion
			if !semver.IsValid(oldSemver) || !semver.IsValid(newSemver) {
				continue
			}
			if semver.Compare(newSemver, oldSemver) <= 0 {
				continue
			}
			ref := dep.GroupID + ":" + dep.ArtifactID
			if dep.Classifier != "" {
				ref += ":" + dep.Classifier
			}
			replacements[ref+":"+dep.Version] = ref + ":" + transitiveVersion
		}
	}
	collectReplacements(maven.Deps)
	for _, rt := range maven.AdditionalRuntimes {
		collectReplacements(rt.Deps)
	}
	if len(replacements) == 0 {
		return nil
	}
	content, err := os.ReadFile(yamlPath)
	if err != nil {
		return err
	}
	result := string(content)
	for oldRef, newRef := range replacements {
		result = strings.ReplaceAll(result, oldRef, newRef)
	}
	return os.WriteFile(yamlPath, []byte(result), 0644) //nolint:gosec
}
