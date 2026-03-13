package maven

import (
	"errors"
	"slices"
	"strings"
)

// MavenImage is the Maven Docker image used in the maven-deps stage.
const MavenImage = "maven:3.9.11-eclipse-temurin-21"

// EnsureMavenDepsStage ensures a maven-deps stage exists in the
// Dockerfile with the correct Maven image. If the stage already
// exists, its FROM line is updated. If it does not exist, a new
// stage is inserted before the final FROM line and a
// COPY --from=maven-deps line is added in the final stage.
func EnsureMavenDepsStage(dockerfile string) (string, error) {
	lines := strings.Split(dockerfile, "\n")
	for i, line := range lines {
		if isMavenDepsFromLine(line) {
			lines[i] = "FROM " + MavenImage + " AS maven-deps"
			return strings.Join(lines, "\n"), nil
		}
	}
	return insertMavenDepsStage(lines)
}

func insertMavenDepsStage(lines []string) (string, error) {
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

	mavenDepsLines := []string{
		"FROM " + MavenImage + " AS maven-deps",
		"COPY pom.xml /tmp/pom.xml",
		"RUN cd /tmp && mvn -f pom.xml dependency:copy-dependencies " +
			"-Dmdep.useRepositoryLayout=true -Dmdep.copyPom=true " +
			"-DoutputDirectory=/maven-repository",
	}

	// Find the insertion point: strip trailing blank lines before
	// the last FROM so we can place exactly one blank line before
	// the new stage.
	insertAt := lastFromIdx
	for insertAt > 0 && strings.TrimSpace(lines[insertAt-1]) == "" {
		insertAt--
	}

	// Assemble: [build stage content] + blank line +
	// maven-deps stage + blank line + final stage.
	var newLines []string
	newLines = append(newLines, lines[:insertAt]...)
	newLines = append(newLines, "")
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

	// Insert COPY --from=maven-deps before the first
	// USER/CMD/ENTRYPOINT in the final stage.
	copyInsertAt := -1
	for i := finalFromIdx + 1; i < len(newLines); i++ {
		if isCopyInsertTarget(newLines[i]) {
			copyInsertAt = i
			break
		}
	}

	copyLine := "COPY --from=maven-deps /maven-repository /maven-repository"
	if copyInsertAt < 0 {
		newLines = append(newLines, copyLine)
		return strings.Join(newLines, "\n"), nil
	}
	finalLines := slices.Concat(
		newLines[:copyInsertAt],
		[]string{copyLine},
		newLines[copyInsertAt:],
	)
	return strings.Join(finalLines, "\n"), nil
}

func isFromLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(strings.ToUpper(trimmed), "FROM ")
}

func isMavenDepsFromLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	upper := strings.ToUpper(trimmed)
	return strings.HasPrefix(upper, "FROM ") && strings.Contains(strings.ToLower(trimmed), "as maven-deps")
}

func isCopyInsertTarget(line string) bool {
	upper := strings.ToUpper(strings.TrimSpace(line))
	return strings.HasPrefix(upper, "USER ") ||
		strings.HasPrefix(upper, "CMD ") ||
		strings.HasPrefix(upper, "CMD[") ||
		strings.HasPrefix(upper, "ENTRYPOINT ") ||
		strings.HasPrefix(upper, "ENTRYPOINT[")
}
