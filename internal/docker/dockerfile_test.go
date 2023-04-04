package docker

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDockerfileBuildStages(t *testing.T) {
	t.Parallel()
	multistage := `
FROM someimage AS build
RUN something

FROM --platform=linux/amd64 someotherimage:sometag AS next
RUN somethingelse

FROM scratch
CMD dosomething
`
	stages, err := ParseDockerfileBuildStages(strings.NewReader(multistage))
	require.NoError(t, err)
	assert.Equal(t, []string{"build", "next"}, stages)
}
