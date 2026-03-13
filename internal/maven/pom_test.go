package maven

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pomProject mirrors the Maven POM structure for test assertions.
type pomProject struct {
	XMLName      xml.Name        `xml:"project"`
	ModelVersion string          `xml:"modelVersion"`
	GroupID      string          `xml:"groupId"`
	ArtifactID   string          `xml:"artifactId"`
	Version      string          `xml:"version"`
	Packaging    string          `xml:"packaging"`
	Dependencies []pomDependency `xml:"dependencies>dependency"`
	Build        *pomBuild       `xml:"build"`
}

type pomDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
	Classifier string `xml:"classifier"`
	Type       string `xml:"type"`
}

type pomBuild struct {
	Plugins []pomPlugin `xml:"plugins>plugin"`
}

type pomPlugin struct {
	GroupID       string            `xml:"groupId"`
	ArtifactID    string            `xml:"artifactId"`
	Version       string            `xml:"version"`
	Configuration *pomConfiguration `xml:"configuration"`
}

type pomConfiguration struct {
	APIVersion      string `xml:"apiVersion"`
	JVMTarget       string `xml:"jvmTarget"`
	LanguageVersion string `xml:"languageVersion"`
}

func TestRenderPOM(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		yaml    string
		wantErr string
		// check runs assertions against the parsed POM. XML comments
		// are not preserved by encoding/xml, so rawPOM is provided
		// for comment checks.
		check func(t *testing.T, p pomProject, rawPOM string)
	}{
		{
			name: "basic Java plugin",
			yaml: `version: v1
name: buf.build/test/plugin
plugin_version: v1.0.0
output_languages:
  - java
registry:
  maven:
    deps:
      - com.google.protobuf:protobuf-java:4.33.5
`,
			check: func(t *testing.T, p pomProject, _ string) { //nolint:thelper
				require.Len(t, p.Dependencies, 1)
				dep := p.Dependencies[0]
				assert.Equal(t, "com.google.protobuf", dep.GroupID)
				assert.Equal(t, "protobuf-java", dep.ArtifactID)
				assert.Equal(t, "4.33.5", dep.Version)
			},
		},
		{
			name: "Kotlin compiler adds plugin dependency",
			yaml: `version: v1
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
`,
			check: func(t *testing.T, p pomProject, _ string) { //nolint:thelper
				require.Len(t, p.Dependencies, 1)
				dep := p.Dependencies[0]
				assert.Equal(t, "org.jetbrains.kotlin", dep.GroupID)
				assert.Equal(t, "kotlin-maven-plugin", dep.ArtifactID)
				assert.Equal(t, "1.8.22", dep.Version)
				assert.Nil(t, p.Build)
			},
		},
		{
			name: "additional runtimes",
			yaml: `version: v1
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
`,
			check: func(t *testing.T, p pomProject, rawPOM string) { //nolint:thelper
				require.Len(t, p.Dependencies, 2)
				assert.Equal(t, "protobuf-java", p.Dependencies[0].ArtifactID)
				assert.Equal(t, "protobuf-javalite", p.Dependencies[1].ArtifactID)
				// XML comments are not preserved by encoding/xml.
				assert.Contains(t, rawPOM, "<!-- lite -->")
			},
		},
		{
			name: "no Maven config returns error",
			yaml: `version: v1
name: buf.build/test/plugin
plugin_version: v1.0.0
output_languages:
  - go
`,
			wantErr: "no Maven registry configured",
		},
		{
			name: "empty deps renders valid structure",
			yaml: `version: v1
name: buf.build/test/plugin
plugin_version: v1.0.0
output_languages:
  - java
registry:
  maven:
    deps: []
`,
			check: func(t *testing.T, p pomProject, _ string) { //nolint:thelper
				assert.Empty(t, p.Dependencies)
			},
		},
		{
			name: "malformed XML detected",
			yaml: `version: v1
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
`,
			wantErr: "generated POM is not well-formed XML",
		},
		{
			name: "Kotlin plugin with no explicit deps",
			yaml: `version: v1
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
`,
			check: func(t *testing.T, p pomProject, _ string) { //nolint:thelper
				require.Len(t, p.Dependencies, 1)
				assert.Equal(t, "org.jetbrains.kotlin", p.Dependencies[0].GroupID)
				assert.Equal(t, "kotlin-maven-plugin", p.Dependencies[0].ArtifactID)
				assert.Equal(t, "1.9.0", p.Dependencies[0].Version)
				assert.Nil(t, p.Build)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			yamlPath := filepath.Join(tmpDir, "buf.plugin.yaml")
			require.NoError(t, os.WriteFile(yamlPath, []byte(tt.yaml), 0644))
			config, err := bufremotepluginconfig.ParseConfig(yamlPath)
			require.NoError(t, err)
			pom, err := RenderPOM(config)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			var project pomProject
			require.NoError(t, xml.NewDecoder(strings.NewReader(pom)).Decode(&project))
			tt.check(t, project, pom)
		})
	}
}
