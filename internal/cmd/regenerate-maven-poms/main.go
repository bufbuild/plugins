package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/bufbuild/plugins/internal/maven"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <plugin-dir> [<plugin-dir>...]\n", os.Args[0])
		os.Exit(1)
	}

	for _, pluginDir := range os.Args[1:] {
		if err := regenerateMavenDeps(pluginDir); err != nil {
			fmt.Fprintf(os.Stderr, "failed to regenerate %s: %v\n", pluginDir, err)
			os.Exit(1)
		}
		fmt.Printf("regenerated: %s\n", pluginDir)
	}
}

func regenerateMavenDeps(pluginDir string) error {
	yamlPath := filepath.Join(pluginDir, "buf.plugin.yaml")
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
	maven.DeduplicateAllDeps(pluginConfig.Registry.Maven)

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
		// Update the existing maven-deps stage POM.
		updated, err = replacePOM(dockerfile, pom)
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

	return os.WriteFile(dockerfilePath, []byte(updated), 0644)
}

// replacePOM replaces the POM XML between COPY <<EOF and EOF in the Dockerfile.
func replacePOM(dockerfile, newPOM string) (string, error) {
	// Match from "COPY <<EOF /tmp/pom.xml" to "EOF"
	re := regexp.MustCompile(`(?s)(COPY <<EOF /tmp/pom\.xml\n).*?\n(EOF)`)
	if !re.MatchString(dockerfile) {
		return "", fmt.Errorf("could not find POM heredoc in Dockerfile")
	}
	return re.ReplaceAllString(dockerfile, "${1}"+newPOM+"\n${2}"), nil
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
		return "", fmt.Errorf("no FROM line found in Dockerfile")
	}

	// Build the maven-deps stage lines. The POM ends with "\n\n" from the
	// template; remove one trailing empty string so there is exactly one
	// blank line before EOF in the heredoc.
	pomLines := strings.Split(pom, "\n")
	if len(pomLines) > 0 && pomLines[len(pomLines)-1] == "" {
		pomLines = pomLines[:len(pomLines)-1]
	}
	mavenDepsLines := []string{
		"FROM maven:3.9.11-eclipse-temurin-25 AS maven-deps",
		"COPY <<EOF /tmp/pom.xml",
	}
	mavenDepsLines = append(mavenDepsLines, pomLines...)
	mavenDepsLines = append(mavenDepsLines,
		"EOF",
		"RUN cd /tmp && mvn -f pom.xml dependency:go-offline",
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
	var finalLines []string
	if copyInsertAt < 0 {
		finalLines = append(newLines, copyLine)
	} else {
		finalLines = make([]string, 0, len(newLines)+1)
		finalLines = append(finalLines, newLines[:copyInsertAt]...)
		finalLines = append(finalLines, copyLine)
		finalLines = append(finalLines, newLines[copyInsertAt:]...)
	}

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
