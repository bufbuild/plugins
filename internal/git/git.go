package git

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"

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
