package git

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bufbuild/buf/private/pkg/execext"
)

// ChangedFilesFrom returns the list of file paths that changed, comparing the current git repo
// against a base Git ref (commit SHA, tag, branch).
func ChangedFilesFrom(ctx context.Context, ref string) ([]string, error) {
	// Check what's the atual base_ref hash we are going to diff against
	//
	// FIXME: remove this debug
	stdoutRefs, err := execGitCommand(ctx, "show-ref", ref)
	if err != nil {
		return nil, fmt.Errorf("git show-ref: %w", err)
	}
	log.Printf("git ref %s resolves to:\n%s\n", ref, stdoutRefs)
	stdoutChangedFiles, err := execGitCommand(ctx, "--no-pager", "diff", "--name-only", ref)
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	return strings.Split(stdoutChangedFiles, "\n"), nil
}

func execGitCommand(ctx context.Context, args ...string) (string, error) {
	var (
		stdout = bytes.NewBuffer(nil)
		stderr = bytes.NewBuffer(nil)
	)
	if err := execext.Run(
		ctx,
		"git",
		execext.WithArgs(args...),
		execext.WithStdout(stdout),
		execext.WithStderr(stderr),
	); err != nil {
		return "", fmt.Errorf(
			"run git %v: %w\nstdout: %s\nstderr: %s",
			args, err, stdout.String(), stderr.String(),
		)
	}
	return stdout.String(), nil
}
