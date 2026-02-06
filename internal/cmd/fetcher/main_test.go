package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bufbuild/plugins/internal/plugin"
	"github.com/bufbuild/plugins/internal/source"
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

// TestRunDependencyOrdering tests the end-to-end behavior of run() with dependency ordering.
// It verifies that when creating multiple plugin versions in one run, they are processed
// in dependency order and consumers reference the newly created dependency versions.
func TestRunDependencyOrdering(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Setup: Create complete repository structure
	setupTestRepository(t, tmpDir)

	// Mock fetcher that returns new versions for our test plugins
	// Cache keys are formatted as "github-owner-repository"
	fetcher := &mockFetcher{
		versions: map[string]string{
			"github-test-base-plugin":     "v2.0.0",
			"github-test-consumer-plugin": "v2.0.0",
		},
	}

	// Run the fetcher
	created, err := run(ctx, tmpDir, fetcher)
	require.NoError(t, err)

	// Verify plugins were created in dependency order
	require.Len(t, created, 2, "should create 2 new plugin versions")
	assert.Equal(t, "base-plugin", created[0].name, "base-plugin should be created first (no dependencies)")
	assert.Equal(t, "v2.0.0", created[0].newVersion)
	assert.Equal(t, "consumer-plugin", created[1].name, "consumer-plugin should be created second (depends on base-plugin)")
	assert.Equal(t, "v2.0.0", created[1].newVersion)

	// Verify consumer references the newly created base-plugin v2.0.0
	consumerYAMLPath := filepath.Join(tmpDir, "plugins", "test", "consumer-plugin", "v2.0.0", "buf.plugin.yaml")
	content, err := os.ReadFile(consumerYAMLPath)
	require.NoError(t, err, "should be able to read created consumer plugin config")

	var config bufremotepluginconfig.ExternalConfig
	err = encoding.UnmarshalJSONOrYAMLStrict(content, &config)
	require.NoError(t, err, "should be able to parse consumer plugin config")

	require.Len(t, config.Deps, 1, "consumer should have one dependency")
	assert.Equal(t, "buf.build/test/base-plugin:v2.0.0", config.Deps[0].Plugin,
		"consumer should reference newly created base-plugin v2.0.0, not the old v1.0.0")
}

// mockFetcher returns predetermined versions for testing.
type mockFetcher struct {
	versions map[string]string // maps cache key (e.g., "github-owner-repo") -> version to return
}

func (m *mockFetcher) Fetch(ctx context.Context, config *source.Config) (string, error) {
	key := config.CacheKey()
	if version, ok := m.versions[key]; ok {
		return version, nil
	}
	// Return a default version if not in map
	return "v1.0.0", nil
}

// setupTestRepository creates a complete test repository structure with:
// - plugins/ directory with base-plugin and consumer-plugin
// - source.yaml files for version detection
// - .github/docker/ directory with base images
func setupTestRepository(t *testing.T, tmpDir string) {
	t.Helper()

	// Create base Docker images directory (required by run())
	baseImageDir := filepath.Join(tmpDir, ".github", "docker")
	require.NoError(t, os.MkdirAll(baseImageDir, 0755))

	// Create required docker/dockerfile base image
	dockerfileImage := `FROM docker/dockerfile:1.19
`
	require.NoError(t, os.WriteFile(filepath.Join(baseImageDir, "Dockerfile.dockerfile"), []byte(dockerfileImage), 0644))

	// Create golang base image
	golangImage := `FROM golang:1.22.0-bookworm
`
	require.NoError(t, os.WriteFile(filepath.Join(baseImageDir, "Dockerfile.golang"), []byte(golangImage), 0644))

	// Setup base-plugin v1.0.0
	basePluginDir := filepath.Join(tmpDir, "plugins", "test", "base-plugin")
	require.NoError(t, os.MkdirAll(filepath.Join(basePluginDir, "v1.0.0"), 0755))

	// Create source.yaml for base-plugin
	baseSourceYAML := `source:
  github:
    owner: test
    repository: base-plugin
`
	require.NoError(t, os.WriteFile(filepath.Join(basePluginDir, "source.yaml"), []byte(baseSourceYAML), 0644))

	// Create buf.plugin.yaml for base-plugin v1.0.0
	basePluginYAML := `version: v1
name: buf.build/test/base-plugin
plugin_version: v1.0.0
output_languages:
  - go
`
	require.NoError(t, os.WriteFile(
		filepath.Join(basePluginDir, "v1.0.0", "buf.plugin.yaml"),
		[]byte(basePluginYAML),
		0644,
	))

	// Create Dockerfile for base-plugin v1.0.0
	baseDockerfile := `FROM golang:1.22.0-bookworm
COPY --from=base /binary /usr/local/bin/protoc-gen-base
`
	require.NoError(t, os.WriteFile(
		filepath.Join(basePluginDir, "v1.0.0", "Dockerfile"),
		[]byte(baseDockerfile),
		0644,
	))

	// Setup consumer-plugin v1.0.0
	consumerPluginDir := filepath.Join(tmpDir, "plugins", "test", "consumer-plugin")
	require.NoError(t, os.MkdirAll(filepath.Join(consumerPluginDir, "v1.0.0"), 0755))

	// Create source.yaml for consumer-plugin
	consumerSourceYAML := `source:
  github:
    owner: test
    repository: consumer-plugin
`
	require.NoError(t, os.WriteFile(filepath.Join(consumerPluginDir, "source.yaml"), []byte(consumerSourceYAML), 0644))

	// Create buf.plugin.yaml for consumer-plugin v1.0.0 that depends on base-plugin v1.0.0
	consumerPluginYAML := `version: v1
name: buf.build/test/consumer-plugin
plugin_version: v1.0.0
deps:
  - plugin: buf.build/test/base-plugin:v1.0.0
output_languages:
  - go
`
	require.NoError(t, os.WriteFile(
		filepath.Join(consumerPluginDir, "v1.0.0", "buf.plugin.yaml"),
		[]byte(consumerPluginYAML),
		0644,
	))

	// Create Dockerfile for consumer-plugin v1.0.0
	consumerDockerfile := `FROM golang:1.22.0-bookworm
COPY --from=consumer /binary /usr/local/bin/protoc-gen-consumer
`
	require.NoError(t, os.WriteFile(
		filepath.Join(consumerPluginDir, "v1.0.0", "Dockerfile"),
		[]byte(consumerDockerfile),
		0644,
	))
}
