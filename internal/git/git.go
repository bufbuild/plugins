package git

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/pkg/execext"
)

// ChangedFilesFrom returns the list of file paths that changed, comparing the current git repo
// against a base Git ref (commit SHA, tag, branch).
func ChangedFilesFrom(ctx context.Context, ref string) ([]string, error) {
	var (
		stdout = bytes.NewBuffer(nil)
		stderr = bytes.NewBuffer(nil)
	)
	if err := execext.Run(
		ctx,
		"git",
		execext.WithArgs("--no-pager", "diff", "--name-only", ref),
		execext.WithStdout(stdout),
		execext.WithStderr(stderr),
	); err != nil {
		return nil, fmt.Errorf(
			"run git diff (stdout: %s, stderr: %s): %w",
			stdout.String(), stderr.String(), err,
		)
	}
	return strings.Split(stdout.String(), "\n"), nil
}
