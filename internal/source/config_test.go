package source

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigWithUpdateFrequency(t *testing.T) {
	t.Parallel()
	sourceData := `source:
  update_frequency: 30d
  github:
    owner: test
    repository: test-repo
`
	config, err := NewConfig(strings.NewReader(sourceData))
	require.NoError(t, err)
	require.NotNil(t, config.Source.UpdateFrequency)
	assert.Equal(t, Duration(30*24*time.Hour), *config.Source.UpdateFrequency)
	assert.Equal(t, "test", config.Source.GitHub.Owner)
}
