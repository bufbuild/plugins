package maven

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeTransitiveDeps(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Setup: grandparent -> parent -> child chain
	grandparentDir := filepath.Join(tmpDir, "plugins", "test", "grandparent", "v1.0.0")
	require.NoError(t, os.MkdirAll(grandparentDir, 0755))
	grandparentYAML := `version: v1
name: buf.build/test/grandparent
plugin_version: v1.0.0
output_languages:
  - java
registry:
  maven:
    deps:
      - org.example:grandparent-dep:1.0.0
`
	require.NoError(t, os.WriteFile(filepath.Join(grandparentDir, "buf.plugin.yaml"), []byte(grandparentYAML), 0644))

	parentDir := filepath.Join(tmpDir, "plugins", "test", "parent", "v1.0.0")
	require.NoError(t, os.MkdirAll(parentDir, 0755))
	parentYAML := `version: v1
name: buf.build/test/parent
plugin_version: v1.0.0
deps:
  - plugin: buf.build/test/grandparent:v1.0.0
output_languages:
  - java
registry:
  maven:
    deps:
      - org.example:parent-dep:1.0.0
`
	require.NoError(t, os.WriteFile(filepath.Join(parentDir, "buf.plugin.yaml"), []byte(parentYAML), 0644))

	childDir := filepath.Join(tmpDir, "plugins", "test", "child", "v1.0.0")
	require.NoError(t, os.MkdirAll(childDir, 0755))
	childYAML := `version: v1
name: buf.build/test/child
plugin_version: v1.0.0
deps:
  - plugin: buf.build/test/parent:v1.0.0
output_languages:
  - java
registry:
  maven:
    deps:
      - org.example:child-dep:1.0.0
`
	require.NoError(t, os.WriteFile(filepath.Join(childDir, "buf.plugin.yaml"), []byte(childYAML), 0644))

	// Parse the child plugin config
	childConfig, err := bufremotepluginconfig.ParseConfig(
		filepath.Join(childDir, "buf.plugin.yaml"),
	)
	require.NoError(t, err)

	// Merge transitive deps
	pluginsDir := filepath.Join(tmpDir, "plugins")
	err = MergeTransitiveDeps(childConfig, pluginsDir)
	require.NoError(t, err)

	// Child should now have all three deps: child-dep, parent-dep,
	// grandparent-dep (transitive through parent)
	var artifactIDs []string
	for _, dep := range childConfig.Registry.Maven.Deps {
		artifactIDs = append(artifactIDs, dep.ArtifactID)
	}
	assert.Contains(t, artifactIDs, "child-dep")
	assert.Contains(t, artifactIDs, "parent-dep")
	assert.Contains(t, artifactIDs, "grandparent-dep")
}

func TestDeduplicateAllDeps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		deps    []bufremotepluginconfig.MavenDependencyConfig
		want    []bufremotepluginconfig.MavenDependencyConfig
		wantErr string
	}{
		{
			name: "no duplicates",
			deps: []bufremotepluginconfig.MavenDependencyConfig{
				{GroupID: "com.example", ArtifactID: "a", Version: "1.0"},
				{GroupID: "com.example", ArtifactID: "b", Version: "1.0"},
			},
			want: []bufremotepluginconfig.MavenDependencyConfig{
				{GroupID: "com.example", ArtifactID: "a", Version: "1.0"},
				{GroupID: "com.example", ArtifactID: "b", Version: "1.0"},
			},
		},
		{
			name: "exact duplicate removed",
			deps: []bufremotepluginconfig.MavenDependencyConfig{
				{GroupID: "com.example", ArtifactID: "a", Version: "1.0"},
				{GroupID: "com.example", ArtifactID: "a", Version: "1.0"},
			},
			want: []bufremotepluginconfig.MavenDependencyConfig{
				{GroupID: "com.example", ArtifactID: "a", Version: "1.0"},
			},
		},
		{
			name: "version conflict returns error",
			deps: []bufremotepluginconfig.MavenDependencyConfig{
				{GroupID: "com.example", ArtifactID: "a", Version: "1.0"},
				{GroupID: "com.example", ArtifactID: "a", Version: "2.0"},
			},
			wantErr: "duplicate Maven dependency com.example:a with conflicting versions",
		},
		{
			name: "different classifiers are distinct",
			deps: []bufremotepluginconfig.MavenDependencyConfig{
				{GroupID: "com.example", ArtifactID: "a", Version: "1.0", Classifier: "sources"},
				{GroupID: "com.example", ArtifactID: "a", Version: "1.0"},
			},
			want: []bufremotepluginconfig.MavenDependencyConfig{
				{GroupID: "com.example", ArtifactID: "a", Version: "1.0", Classifier: "sources"},
				{GroupID: "com.example", ArtifactID: "a", Version: "1.0"},
			},
		},
		{
			name: "nil input",
			deps: nil,
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := &bufremotepluginconfig.MavenRegistryConfig{
				Deps: tt.deps,
			}
			err := DeduplicateAllDeps(config)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, config.Deps)
		})
	}
}
