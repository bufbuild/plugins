package main

import (
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdatePluginDeps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		input           string
		latestVersions  map[string]string
		expectedUpdates map[string]string // plugin name -> expected version
		wantErr         bool
	}{
		{
			name: "updates single plugin dependency",
			input: `version: v1
name: buf.build/test/plugin
plugin_version: v1.0.0
deps:
  - plugin: buf.build/protocolbuffers/go:v1.30.0
`,
			latestVersions: map[string]string{
				"buf.build/protocolbuffers/go": "v1.36.11",
			},
			expectedUpdates: map[string]string{
				"buf.build/protocolbuffers/go": "v1.36.11",
			},
		},
		{
			name: "updates multiple plugin dependencies",
			input: `version: v1
name: buf.build/test/plugin
plugin_version: v1.0.0
deps:
  - plugin: buf.build/protocolbuffers/go:v1.30.0
  - plugin: buf.build/protocolbuffers/python:v30.0
`,
			latestVersions: map[string]string{
				"buf.build/protocolbuffers/go":     "v1.36.11",
				"buf.build/protocolbuffers/python": "v33.5",
			},
			expectedUpdates: map[string]string{
				"buf.build/protocolbuffers/go":     "v1.36.11",
				"buf.build/protocolbuffers/python": "v33.5",
			},
		},
		{
			name: "no updates when already at latest version",
			input: `version: v1
name: buf.build/test/plugin
plugin_version: v1.0.0
deps:
  - plugin: buf.build/protocolbuffers/go:v1.36.11
`,
			latestVersions: map[string]string{
				"buf.build/protocolbuffers/go": "v1.36.11",
			},
			expectedUpdates: map[string]string{
				"buf.build/protocolbuffers/go": "v1.36.11",
			},
		},
		{
			name: "handles missing plugin in latestVersions map",
			input: `version: v1
name: buf.build/test/plugin
plugin_version: v1.0.0
deps:
  - plugin: buf.build/unknown/plugin:v1.0.0
`,
			latestVersions: map[string]string{
				"buf.build/protocolbuffers/go": "v1.36.11",
			},
			expectedUpdates: map[string]string{
				"buf.build/unknown/plugin": "v1.0.0", // unchanged
			},
		},
		{
			name: "handles yaml without deps",
			input: `version: v1
name: buf.build/test/plugin
plugin_version: v1.0.0
`,
			latestVersions: map[string]string{
				"buf.build/protocolbuffers/go": "v1.36.11",
			},
			expectedUpdates: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := updatePluginDeps([]byte(tt.input), tt.latestVersions)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Parse the result to check if dependencies were updated correctly
			var config bufremotepluginconfig.ExternalConfig
			err = encoding.UnmarshalJSONOrYAMLStrict(result, &config)
			require.NoError(t, err, "failed to parse result YAML")

			if tt.expectedUpdates == nil {
				// No deps expected in result
				assert.Empty(t, config.Deps, "expected no deps field, but got one")
				return
			}

			require.NotEmpty(t, config.Deps, "expected deps field in result")

			// Check each dependency
			foundDeps := make(map[string]string)
			for _, dep := range config.Deps {
				if dep.Plugin == "" {
					continue
				}
				pluginName, version, ok := strings.Cut(dep.Plugin, ":")
				require.True(t, ok, "invalid plugin reference format: %s", dep.Plugin)
				foundDeps[pluginName] = version
			}

			// Verify all expected updates
			for pluginName, expectedVersion := range tt.expectedUpdates {
				foundVersion, ok := foundDeps[pluginName]
				assert.True(t, ok, "missing plugin dependency: %s", pluginName)
				assert.Equal(t, expectedVersion, foundVersion, "plugin %s version mismatch", pluginName)
			}
		})
	}
}
