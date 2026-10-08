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

func TestNewConfigNormalizesVersions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		sourceYAML         string
		wantMaxVersion     string
		wantIgnoreVersions []string
		wantErr            string
	}{
		{
			name: "adds v prefix",
			sourceYAML: `source:
  github:
    owner: test
    repository: test-repo
  max_version: 2.0.0
  ignore_versions:
    - 1.2.3
    - v1.2.4
`,
			wantMaxVersion:     "v2.0.0",
			wantIgnoreVersions: []string{"v1.2.3", "v1.2.4"},
		},
		{
			name: "no constraints",
			sourceYAML: `source:
  github:
    owner: test
    repository: test-repo
`,
		},
		{
			name: "invalid max version",
			sourceYAML: `source:
  github:
    owner: test
    repository: test-repo
  max_version: two
`,
			wantErr: "max_version is not a valid semver: two",
		},
		{
			name: "invalid ignore version",
			sourceYAML: `source:
  github:
    owner: test
    repository: test-repo
  ignore_versions:
    - v1.2.x
`,
			wantErr: `ignore_versions entry is not a valid semver: "v1.2.x"`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			config, err := NewConfig(strings.NewReader(test.sourceYAML))
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantMaxVersion, config.Source.MaxVersion)
			assert.Equal(t, test.wantIgnoreVersions, config.Source.IgnoreVersions)
		})
	}
}

func TestSourceLatestVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		constraints string
		versions    []string
		want        string
		wantErr     string
	}{
		{
			name:     "returns highest version",
			versions: []string{"v1.10.0", "v1.9.0", "v2.0.0", "v1.2"},
			want:     "v2.0.0",
		},
		{
			name: "skips ignored versions",
			constraints: `  ignore_versions:
    - v2.0.0
`,
			versions: []string{"v1.0.0", "v1.1.0", "v2.0.0"},
			want:     "v1.1.0",
		},
		{
			name: "ignored versions match without patch",
			constraints: `  ignore_versions:
    - "3.0"
`,
			versions: []string{"v2.0.0", "v3.0.0"},
			want:     "v2.0.0",
		},
		{
			name: "max version is exclusive",
			constraints: `  max_version: 2.0.0
`,
			versions: []string{"v1.21.0", "v2.0.0", "v2.1.0"},
			want:     "v1.21.0",
		},
		{
			name: "ignore and max version combined",
			constraints: `  max_version: v2.0.0
  ignore_versions:
    - v1.1.0
`,
			versions: []string{"v1.0.0", "v1.1.0", "v2.0.0"},
			want:     "v1.0.0",
		},
		{
			name: "no versions satisfy constraints",
			constraints: `  max_version: v2.0.0
`,
			versions: []string{"v2.0.0"},
			wantErr:  "no versions satisfy",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			sourceYAML := `source:
  github:
    owner: test
    repository: test-repo
` + test.constraints
			config, err := NewConfig(strings.NewReader(sourceYAML))
			require.NoError(t, err)
			got, err := config.Source.LatestVersion(test.versions)
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}
