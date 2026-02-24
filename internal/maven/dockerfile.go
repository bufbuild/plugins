package maven

import (
	"errors"
	"fmt"
	"strings"
)

// ReplacePOMInDockerfile replaces the POM heredoc content between
// "COPY <<EOF /tmp/pom.xml" and "EOF" in a Dockerfile's maven-deps
// stage.
func ReplacePOMInDockerfile(dockerfile, newPOM string) (string, error) {
	const pomStart = "COPY <<EOF /tmp/pom.xml"
	const pomEnd = "EOF"
	startIdx := strings.Index(dockerfile, pomStart)
	if startIdx < 0 {
		return "", fmt.Errorf(
			"could not find %q in Dockerfile", pomStart,
		)
	}
	// Find the content start (after the COPY line).
	contentStart := startIdx + len(pomStart) + 1 // +1 for newline
	// Scan line-by-line from contentStart to find the first
	// standalone EOF line that closes the POM heredoc.
	remaining := dockerfile[contentStart:]
	lineStart := 0
	eofIdx := -1
	for i, ch := range remaining {
		if ch == '\n' || i == len(remaining)-1 {
			line := remaining[lineStart:i]
			if strings.TrimRight(line, "\r") == pomEnd {
				eofIdx = contentStart + lineStart
				break
			}
			lineStart = i + 1
		}
	}
	if eofIdx < 0 {
		return "", errors.New(
			"could not find closing EOF for POM heredoc in Dockerfile",
		)
	}
	var sb strings.Builder
	sb.WriteString(dockerfile[:contentStart])
	sb.WriteString(newPOM)
	if !strings.HasSuffix(newPOM, "\n") {
		sb.WriteByte('\n')
	}
	sb.WriteString(dockerfile[eofIdx:])
	return sb.String(), nil
}
