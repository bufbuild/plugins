package maven

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegenerateMavenDeps(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Setup: base-plugin has Maven deps including an additional lite runtime.
	baseDir := filepath.Join(tmpDir, "plugins", "test", "base-plugin", "v1.0.0")
	require.NoError(t, os.MkdirAll(baseDir, 0755))
	baseYAML := `version: v1
name: buf.build/test/base-plugin
plugin_version: v1.0.0
output_languages:
  - java
registry:
  maven:
    deps:
      - com.google.protobuf:protobuf-java:4.33.5
    additional_runtimes:
      - name: lite
        deps:
          - com.google.protobuf:protobuf-javalite:4.33.5
          - build.buf:protobuf-javalite:4.33.5
        opts: [lite]
`
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "buf.plugin.yaml"), []byte(baseYAML), 0644))

	// Setup: consumer-plugin depends on base-plugin and has its own Maven
	// deps plus a lite runtime. The Dockerfile has a maven-deps stage.
	consumerDir := filepath.Join(tmpDir, "plugins", "test", "consumer-plugin", "v1.0.0")
	require.NoError(t, os.MkdirAll(consumerDir, 0755))
	consumerYAML := `version: v1
name: buf.build/test/consumer-plugin
plugin_version: v1.0.0
deps:
  - plugin: buf.build/test/base-plugin:v1.0.0
output_languages:
  - kotlin
registry:
  maven:
    compiler:
      kotlin:
        version: 1.8.22
    deps:
      - com.google.protobuf:protobuf-kotlin:4.33.5
    additional_runtimes:
      - name: lite
        deps:
          - com.google.protobuf:protobuf-kotlin-lite:4.33.5
        opts: [lite]
`
	require.NoError(t, os.WriteFile(filepath.Join(consumerDir, "buf.plugin.yaml"), []byte(consumerYAML), 0644))

	dockerfile := `# syntax=docker/dockerfile:1.19
FROM debian:bookworm AS build
RUN echo hello

FROM scratch
COPY --from=build /app .
ENTRYPOINT ["/app"]
`
	require.NoError(t, os.WriteFile(filepath.Join(consumerDir, "Dockerfile"), []byte(dockerfile), 0644))

	// Run RegenerateMavenDeps on the consumer plugin.
	pluginsDir := filepath.Join(tmpDir, "plugins")
	err := RegenerateMavenDeps(consumerDir, pluginsDir)
	require.NoError(t, err)

	// Verify the maven-deps stage was inserted into the Dockerfile.
	dockerfileBytes, err := os.ReadFile(filepath.Join(consumerDir, "Dockerfile"))
	require.NoError(t, err)
	assert.Contains(t, string(dockerfileBytes), "FROM "+MavenImage+" AS maven-deps")
	assert.Contains(t, string(dockerfileBytes), "COPY --from=maven-deps /root/.m2/repository /maven-repository")

	// Read and parse pom.xml to verify deps include versions.
	pomBytes, err := os.ReadFile(filepath.Join(consumerDir, "pom.xml"))
	require.NoError(t, err)
	var pom pomProject
	require.NoError(t, xml.Unmarshal(pomBytes, &pom))
	var depVersions []string
	for _, dep := range pom.Dependencies {
		depVersions = append(depVersions, dep.GroupID+":"+dep.ArtifactID+":"+dep.Version)
	}
	// Consumer's own deps should be present.
	assert.Contains(t, depVersions, "com.google.protobuf:protobuf-kotlin:4.33.5")
	assert.Contains(t, depVersions, "com.google.protobuf:protobuf-kotlin-lite:4.33.5")
	// Base plugin's main deps should be merged in.
	assert.Contains(t, depVersions, "com.google.protobuf:protobuf-java:4.33.5")
	// Base plugin's lite runtime deps should be merged into the
	// matching lite runtime section.
	assert.Contains(t, depVersions, "com.google.protobuf:protobuf-javalite:4.33.5")
	assert.Contains(t, depVersions, "build.buf:protobuf-javalite:4.33.5")
}
