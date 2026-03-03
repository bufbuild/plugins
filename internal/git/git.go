package git

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"buf.build/go/standard/xos/xexec"
)

// ChangedFilesFrom returns the list of file paths that changed, comparing the current git repo
// against a base Git ref (commit SHA, tag, branch).
func ChangedFilesFrom(ctx context.Context, ref string) ([]string, error) {
	changedFiles, err := execGitCommand(ctx, "--no-pager", "diff", "--name-only", ref)
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	log.Printf("git diff against %s:\n%s\n", ref, changedFiles)
	return strings.Split(changedFiles, "\n"), nil
}

// FirstCommitTime returns the author time of the commit that first added files
// at the given path. The path should be relative to the git repository root.
// Returns a zero time.Time if no commits are found (e.g. the path is uncommitted).
func FirstCommitTime(ctx context.Context, path string) (time.Time, error) {
	output, err := execGitCommand(ctx, "log", "--diff-filter=A", "--format=%aI", "--", path)
	if err != nil {
		return time.Time{}, fmt.Errorf("git log: %w", err)
	}
	output = strings.TrimSpace(output)
	if output == "" {
		return time.Time{}, nil
	}
	// --diff-filter=A may return multiple lines if files were added in
	// separate commits. Take the earliest (last line after sorting by
	// default git log order, which is newest-first).
	lines := strings.Split(output, "\n")
	lastLine := lines[len(lines)-1]
	t, err := time.Parse(time.RFC3339, lastLine)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing time %q: %w", lastLine, err)
	}
	return t, nil
}

func execGitCommand(ctx context.Context, args ...string) (string, error) {
	var (
		stdout = bytes.NewBuffer(nil)
		stderr = bytes.NewBuffer(nil)
	)
	if err := xexec.Run(
		ctx,
		"git",
		xexec.WithArgs(args...),
		xexec.WithStdout(stdout),
		xexec.WithStderr(stderr),
	); err != nil {
		return "", fmt.Errorf(
			"run git %v: %w\nstdout: %s\nstderr: %s",
			args, err, stdout.String(), stderr.String(),
		)
	}
	return stdout.String(), nil
}
