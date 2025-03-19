package git_test

import (
	"testing"

	"github.com/bufbuild/plugins/internal/git"
	"github.com/stretchr/testify/require"
)

func TestChangedFilesFrom(t *testing.T) {
	files, err := git.ChangedFilesFrom(t.Context(), "main")
	require.NoError(t, err)
	t.Log(files)
}
