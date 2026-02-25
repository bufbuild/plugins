package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"buf.build/go/app"
	"buf.build/go/app/appcmd"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"

	"github.com/bufbuild/plugins/internal/maven"
)

func main() {
	appcmd.Main(context.Background(), newCommand("regenerate-maven-poms"))
}

func newCommand(name string) *appcmd.Command {
	return &appcmd.Command{
		Use:   name + " <plugin-dir> [<plugin-dir>...]",
		Short: "Regenerates maven-deps POM and Dockerfile stage for Java/Kotlin plugins",
		Args:  appcmd.MinimumNArgs(1),
		Run: func(_ context.Context, container app.Container) error {
			for i := range container.NumArgs() {
				pluginDir := container.Arg(i)
				if err := regenerateMavenDeps(pluginDir); err != nil {
					return fmt.Errorf("failed to regenerate %s: %w", pluginDir, err)
				}
				fmt.Fprintf(container.Stdout(), "regenerated: %s\n", pluginDir)
			}
			return nil
		},
	}
}

func regenerateMavenDeps(pluginDir string) error {
	yamlPath := filepath.Join(pluginDir, "buf.plugin.yaml")
	if !fileExists(yamlPath) {
		return nil // no buf.plugin.yaml, skip
	}
	pluginConfig, err := bufremotepluginconfig.ParseConfig(yamlPath)
	if err != nil {
		return err
	}
	if pluginConfig.Registry == nil || pluginConfig.Registry.Maven == nil {
		return nil // not a Maven plugin
	}

	// Merge transitive Maven deps from plugin dependencies and deduplicate.
	// pluginDir is e.g. plugins/org/name/version, so plugins root is 3
	// levels up.
	pluginsDir := filepath.Dir(filepath.Dir(filepath.Dir(pluginDir)))
	if err := maven.MergeTransitiveDeps(pluginConfig, pluginsDir); err != nil {
		return fmt.Errorf("merging transitive deps: %w", err)
	}
	if err := maven.DeduplicateAllDeps(pluginConfig.Registry.Maven); err != nil {
		return fmt.Errorf("deduplicating deps: %w", err)
	}

	pom, err := maven.RenderPOM(pluginConfig)
	if err != nil {
		return fmt.Errorf("rendering POM: %w", err)
	}

	dockerfilePath := filepath.Join(pluginDir, "Dockerfile")
	dockerfileBytes, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return err
	}
	dockerfile := string(dockerfileBytes)

	var updated string
	if strings.Contains(dockerfile, "maven-deps") {
		updated, err = maven.ReplacePOMInDockerfile(dockerfile, pom)
		if err != nil {
			return fmt.Errorf("replacing POM: %w", err)
		}
	} else {
		// Insert a new maven-deps stage.
		updated, err = insertMavenDepsStage(dockerfile, pom)
		if err != nil {
			return fmt.Errorf("inserting maven-deps stage: %w", err)
		}
	}

	return os.WriteFile(dockerfilePath, []byte(updated), 0644) //nolint:gosec // Dockerfiles should be world-readable.
}

// insertMavenDepsStage inserts a new maven-deps stage into a Dockerfile that
// does not have one. The stage is inserted before the final FROM line, and a
// COPY --from=maven-deps line is added in the final stage before the first
// USER, CMD, or ENTRYPOINT directive.
func insertMavenDepsStage(dockerfile, pom string) (string, error) {
	lines := strings.Split(dockerfile, "\n")

	// Find the index of the last FROM line (the final stage).
	lastFromIdx := -1
	for i, line := range lines {
		if isFromLine(line) {
			lastFromIdx = i
		}
	}
	if lastFromIdx < 0 {
		return "", errors.New("no FROM line found in Dockerfile")
	}

	// Build the maven-deps stage lines. The POM ends with "\n\n" from the
	// template; remove one trailing empty string so there is exactly one
	// blank line before EOF in the heredoc.
	pomLines := strings.Split(pom, "\n")
	if len(pomLines) > 0 && pomLines[len(pomLines)-1] == "" {
		pomLines = pomLines[:len(pomLines)-1]
	}
	mavenDepsLines := slices.Concat(
		[]string{
			"FROM maven:3.9.11-eclipse-temurin-25 AS maven-deps",
			"COPY <<EOF /tmp/pom.xml",
		},
		pomLines,
		[]string{
			"EOF",
			"RUN cd /tmp && mvn -f pom.xml dependency:go-offline",
		},
	)

	// Find the insertion point: strip trailing blank lines before the last FROM
	// so we can place exactly two blank lines before the new stage.
	insertAt := lastFromIdx
	for insertAt > 0 && strings.TrimSpace(lines[insertAt-1]) == "" {
		insertAt--
	}

	// Assemble: [build stage content] + 2 blank lines + maven-deps stage +
	// 1 blank line + final stage.
	var newLines []string
	newLines = append(newLines, lines[:insertAt]...)
	newLines = append(newLines, "", "")
	newLines = append(newLines, mavenDepsLines...)
	newLines = append(newLines, "")
	newLines = append(newLines, lines[lastFromIdx:]...)

	// Find the last FROM in the new lines array (the final stage).
	finalFromIdx := -1
	for i, line := range newLines {
		if isFromLine(line) {
			finalFromIdx = i
		}
	}

	// Insert COPY --from=maven-deps before the first USER/CMD/ENTRYPOINT in
	// the final stage.
	copyInsertAt := -1
	for i := finalFromIdx + 1; i < len(newLines); i++ {
		if isCopyInsertTarget(newLines[i]) {
			copyInsertAt = i
			break
		}
	}

	copyLine := "COPY --from=maven-deps /root/.m2/repository /maven-repository"
	if copyInsertAt < 0 {
		newLines = append(newLines, copyLine)
		return strings.Join(newLines, "\n"), nil
	}
	finalLines := slices.Concat(newLines[:copyInsertAt], []string{copyLine}, newLines[copyInsertAt:])
	return strings.Join(finalLines, "\n"), nil
}

func isFromLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(strings.ToUpper(trimmed), "FROM ")
}

func isCopyInsertTarget(line string) bool {
	upper := strings.ToUpper(strings.TrimSpace(line))
	return strings.HasPrefix(upper, "USER ") ||
		strings.HasPrefix(upper, "CMD ") ||
		strings.HasPrefix(upper, "CMD[") ||
		strings.HasPrefix(upper, "ENTRYPOINT ") ||
		strings.HasPrefix(upper, "ENTRYPOINT[")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
