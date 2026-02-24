package maven

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderPOM_BasicJavaPlugin(t *testing.T) {
	t.Parallel()

	// Create temporary config file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "buf.plugin.yaml")
	yamlContent := `version: v1
name: buf.build/test/plugin
plugin_version: v1.0.0
output_languages:
  - java
registry:
  maven:
    deps:
      - com.google.protobuf:protobuf-java:4.33.5
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

	config, err := bufremotepluginconfig.ParseConfig(yamlPath)
	require.NoError(t, err)

	pom, err := RenderPOM(config)
	require.NoError(t, err)

	assert.Contains(t, pom, "<groupId>com.google.protobuf</groupId>")
	assert.Contains(t, pom, "<artifactId>protobuf-java</artifactId>")
	assert.Contains(t, pom, "<version>4.33.5</version>")
}

func TestRenderPOM_XMLEscaping(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "buf.plugin.yaml")
	yamlContent := `version: v1
name: buf.build/test/plugin
plugin_version: v1.0.0
output_languages:
  - java
registry:
  maven:
    deps:
      - com.test:artifact<>&:1.0.0
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

	config, err := bufremotepluginconfig.ParseConfig(yamlPath)
	require.NoError(t, err)

	pom, err := RenderPOM(config)
	require.NoError(t, err)

	assert.Contains(t, pom, "artifact&lt;&gt;&amp;")
	assert.NotContains(t, pom, "artifact<>&")
}

func TestRenderPOM_KotlinCompiler(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "buf.plugin.yaml")
	yamlContent := `version: v1
name: buf.build/test/kotlin-plugin
plugin_version: v1.0.0
output_languages:
  - kotlin
registry:
  maven:
    compiler:
      kotlin:
        version: 1.8.22
        jvm_target: "1.8"
        language_version: "1.8"
        api_version: "1.8"
    deps: []
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

	config, err := bufremotepluginconfig.ParseConfig(yamlPath)
	require.NoError(t, err)

	pom, err := RenderPOM(config)
	require.NoError(t, err)

	assert.Contains(t, pom, "<artifactId>kotlin-maven-plugin</artifactId>")
	assert.Contains(t, pom, "<version>1.8.22</version>")
	assert.Contains(t, pom, "<jvmTarget>1.8</jvmTarget>")
	assert.Contains(t, pom, "<languageVersion>1.8</languageVersion>")
	assert.Contains(t, pom, "<apiVersion>1.8</apiVersion>")
}

func TestRenderPOM_AdditionalRuntimes(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "buf.plugin.yaml")
	yamlContent := `version: v1
name: buf.build/test/plugin
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
        opts: [lite]
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

	config, err := bufremotepluginconfig.ParseConfig(yamlPath)
	require.NoError(t, err)

	pom, err := RenderPOM(config)
	require.NoError(t, err)

	assert.Contains(t, pom, "<!-- lite -->")
	assert.Contains(t, pom, "protobuf-javalite")
}

func TestRenderPOM_ClassifierAndExtension(t *testing.T) {
	t.Parallel()
	t.Skip("Classifier/extension format in YAML unknown - no real-world examples in codebase")
}

func TestRenderPOM_NoMavenConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "buf.plugin.yaml")
	yamlContent := `version: v1
name: buf.build/test/plugin
plugin_version: v1.0.0
output_languages:
  - go
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

	config, err := bufremotepluginconfig.ParseConfig(yamlPath)
	require.NoError(t, err)

	_, err = RenderPOM(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Maven registry configured")
}

func TestRenderPOM_EmptyDeps(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "buf.plugin.yaml")
	yamlContent := `version: v1
name: buf.build/test/plugin
plugin_version: v1.0.0
output_languages:
  - java
registry:
  maven:
    deps: []
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

	config, err := bufremotepluginconfig.ParseConfig(yamlPath)
	require.NoError(t, err)

	pom, err := RenderPOM(config)
	require.NoError(t, err)

	// Should still generate valid POM structure
	assert.Contains(t, pom, "<modelVersion>4.0.0</modelVersion>")
	assert.Contains(t, pom, "<groupId>temp</groupId>")
	assert.Contains(t, pom, "<artifactId>temp</artifactId>")
}

func TestRenderPOM_MalformedXMLDetected(t *testing.T) {
	t.Parallel()

	// A runtime name containing "--" produces an invalid XML comment
	// (<!-- foo--bar --> is not well-formed XML).
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "buf.plugin.yaml")
	yamlContent := `version: v1
name: buf.build/test/plugin
plugin_version: v1.0.0
output_languages:
  - java
registry:
  maven:
    deps:
      - com.google.protobuf:protobuf-java:4.33.5
    additional_runtimes:
      - name: "bad--name"
        deps:
          - com.google.protobuf:protobuf-javalite:4.33.5
        opts: [lite]
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

	config, err := bufremotepluginconfig.ParseConfig(yamlPath)
	require.NoError(t, err)

	_, err = RenderPOM(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "generated POM is not well-formed XML")
}

func TestXMLEscape_SpecialCharacters(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{"normal-text", "normal-text"},
		{"<tag>", "&lt;tag&gt;"},
		{"a&b", "a&amp;b"},
		{`"quoted"`, "&#34;quoted&#34;"},
		{"'single'", "&#39;single&#39;"},
		{"<>&\"'", "&lt;&gt;&amp;&#34;&#39;"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := xmlEscape(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderPOM_KotlinDynamicDependencies(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "buf.plugin.yaml")
	yamlContent := `version: v1
name: buf.build/test/kotlin-plugin
plugin_version: v1.0.0
output_languages:
  - kotlin
registry:
  maven:
    compiler:
      kotlin:
        version: 1.9.0
    deps: []
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

	config, err := bufremotepluginconfig.ParseConfig(yamlPath)
	require.NoError(t, err)

	pom, err := RenderPOM(config)
	require.NoError(t, err)

	// Verify dynamic Kotlin dependencies are included
	assert.Contains(t, pom, "<!-- kotlin-maven-plugin dynamic dependencies")
	assert.Contains(t, pom, "kotlin-compiler-embeddable")
	assert.Contains(t, pom, "kotlin-scripting-compiler")
	assert.Contains(t, pom, "<version>1.9.0</version>")
}

func TestRenderPOM_ValidXMLStructure(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "buf.plugin.yaml")
	yamlContent := `version: v1
name: buf.build/test/plugin
plugin_version: v1.0.0
output_languages:
  - java
registry:
  maven:
    deps:
      - com.test:test:1.0.0
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

	config, err := bufremotepluginconfig.ParseConfig(yamlPath)
	require.NoError(t, err)

	pom, err := RenderPOM(config)
	require.NoError(t, err)

	// Verify well-formed XML structure
	assert.True(t, strings.HasPrefix(strings.TrimSpace(pom), "<project>"))
	assert.True(t, strings.HasSuffix(strings.TrimSpace(pom), "</project>"))
	assert.Contains(t, pom, "</dependencies>")
}
