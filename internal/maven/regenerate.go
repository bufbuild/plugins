package maven

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
)

// RegenerateMavenDeps processes a Maven plugin version directory by
// merging transitive deps, deduplicating, rendering POM to a pom.xml
// file, and ensuring the Dockerfile has an up-to-date maven-deps
// stage. Returns nil without changes if the plugin has no Maven
// registry config.
func RegenerateMavenDeps(pluginVersionDir, pluginsDir string) error {
	yamlPath := filepath.Join(pluginVersionDir, "buf.plugin.yaml")
	pluginConfig, err := bufremotepluginconfig.ParseConfig(yamlPath)
	if err != nil {
		return err
	}
	if pluginConfig.Registry == nil || pluginConfig.Registry.Maven == nil {
		return nil
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
	dockerfilePath := filepath.Join(pluginVersionDir, "Dockerfile")
	dockerfileBytes, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return err
	}
	updated, err := EnsureMavenDepsStage(string(dockerfileBytes))
	if err != nil {
		return fmt.Errorf("ensuring maven-deps stage: %w", err)
	}
	if err := os.WriteFile(dockerfilePath, []byte(updated), 0644); err != nil { //nolint:gosec
		return fmt.Errorf("writing Dockerfile: %w", err)
	}
	dockerignorePath := filepath.Join(pluginVersionDir, ".dockerignore")
	if err := ensureDockerignoreAllowsPOM(dockerignorePath); err != nil {
		return fmt.Errorf("updating .dockerignore: %w", err)
	}
	return nil
}

// ensureDockerignoreAllowsPOM adds "!pom.xml" to the .dockerignore if it
// exists and doesn't already allow pom.xml. The pom.xml must be present in
// the build context for the maven-deps COPY instruction to succeed.
func ensureDockerignoreAllowsPOM(path string) error {
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	const pomRule = "!pom.xml"
	for line := range strings.SplitSeq(string(content), "\n") {
		if strings.TrimSpace(line) == pomRule {
			return nil
		}
	}
	updated := strings.TrimRight(string(content), "\n") + "\n" + pomRule + "\n"
	return os.WriteFile(path, []byte(updated), 0644) //nolint:gosec
}
