package maven

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegenerateMavenDepsUpdatesStaleVersions(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Simulate the grpc/java scenario: the dep plugin (protocolbuffers/java)
	// was updated to v34.0 which uses protobuf-java:4.34.0, but the
	// grpc/java plugin still has stale pins at 4.33.5 from the previous
	// version's copy.
	baseDir := filepath.Join(tmpDir, "plugins", "protocolbuffers", "java", "v34.0")
	require.NoError(t, os.MkdirAll(baseDir, 0755))
	baseYAML := `version: v1
name: buf.build/protocolbuffers/java
plugin_version: v34.0
output_languages:
  - java
registry:
  maven:
    deps:
      - com.google.protobuf:protobuf-java:4.34.0
    additional_runtimes:
      - name: lite
        deps:
          - com.google.protobuf:protobuf-javalite:4.34.0
          - build.buf:protobuf-javalite:4.34.0
        opts: [lite]
`
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "buf.plugin.yaml"), []byte(baseYAML), 0644))

	consumerDir := filepath.Join(tmpDir, "plugins", "grpc", "java", "v1.80.0")
	require.NoError(t, os.MkdirAll(consumerDir, 0755))
	// This has the dep updated to v34.0 but Maven pins still at 4.33.5
	// (the old version from the copy step).
	consumerYAML := `version: v1
name: buf.build/grpc/java
plugin_version: v1.80.0
source_url: https://github.com/grpc/grpc-java
description: Generates Java client and server stubs for the gRPC framework.
deps:
  - plugin: buf.build/protocolbuffers/java:v34.0
output_languages:
  - java
spdx_license_id: Apache-2.0
license_url: https://github.com/grpc/grpc-java/blob/v1.80.0/LICENSE
registry:
  maven:
    deps:
      - io.grpc:grpc-core:1.80.0
      - io.grpc:grpc-protobuf:1.80.0
      - io.grpc:grpc-stub:1.80.0
      # Add direct dependency on newer protobuf as gRPC is still on 3.25.8
      - com.google.protobuf:protobuf-java:4.33.5
    additional_runtimes:
      - name: lite
        deps:
          - io.grpc:grpc-core:1.80.0
          - io.grpc:grpc-protobuf-lite:1.80.0
          - io.grpc:grpc-stub:1.80.0
          # Add direct dependency on newer protobuf as gRPC is still on 3.25.8
          - com.google.protobuf:protobuf-javalite:4.33.5
          - build.buf:protobuf-javalite:4.33.5
        opts: [lite]
`
	require.NoError(t, os.WriteFile(filepath.Join(consumerDir, "buf.plugin.yaml"), []byte(consumerYAML), 0644))

	pluginsDir := filepath.Join(tmpDir, "plugins")
	err := RegenerateMavenDeps(consumerDir, pluginsDir)
	require.NoError(t, err)

	// Verify buf.plugin.yaml was updated with the correct versions.
	updatedYAML, err := os.ReadFile(filepath.Join(consumerDir, "buf.plugin.yaml"))
	require.NoError(t, err)
	yamlStr := string(updatedYAML)
	assert.Contains(t, yamlStr, "com.google.protobuf:protobuf-java:4.34.0")
	assert.NotContains(t, yamlStr, "com.google.protobuf:protobuf-java:4.33.5")
	assert.Contains(t, yamlStr, "com.google.protobuf:protobuf-javalite:4.34.0")
	assert.NotContains(t, yamlStr, "com.google.protobuf:protobuf-javalite:4.33.5")
	assert.Contains(t, yamlStr, "build.buf:protobuf-javalite:4.34.0")
	assert.NotContains(t, yamlStr, "build.buf:protobuf-javalite:4.33.5")
	// grpc deps should be unchanged
	assert.Contains(t, yamlStr, "io.grpc:grpc-core:1.80.0")
	// Comments should be preserved
	assert.Contains(t, yamlStr, "# Add direct dependency on newer protobuf")

	// Verify pom.xml has the correct versions.
	pomBytes, err := os.ReadFile(filepath.Join(consumerDir, "pom.xml"))
	require.NoError(t, err)
	var pom pomProject
	require.NoError(t, xml.Unmarshal(pomBytes, &pom))
	var depVersions []string
	for _, dep := range pom.Dependencies {
		depVersions = append(depVersions, dep.GroupID+":"+dep.ArtifactID+":"+dep.Version)
	}
	assert.Contains(t, depVersions, "com.google.protobuf:protobuf-java:4.34.0")
	assert.Contains(t, depVersions, "io.grpc:grpc-core:1.80.0")
}

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
	// deps plus a lite runtime.
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

	// Run RegenerateMavenDeps on the consumer plugin.
	pluginsDir := filepath.Join(tmpDir, "plugins")
	err := RegenerateMavenDeps(consumerDir, pluginsDir)
	require.NoError(t, err)

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
