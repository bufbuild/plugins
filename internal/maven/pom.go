package maven

import (
	"bytes"
	_ "embed"
	"encoding/xml"
	"fmt"
	"strings"
	"text/template"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
)

const (
	// CompilerPluginVersion is the maven-compiler-plugin version used by the
	// Java compile service. Plugin images must cache this version.
	CompilerPluginVersion = "3.12.1"
	// SourcePluginVersion is the maven-source-plugin version used by the
	// Java compile service. Plugin images must cache this version.
	SourcePluginVersion = "3.3.0"
)

var (
	//go:embed pom.xml.gotext
	pomTemplateContents string

	pomTemplate = template.Must(template.New("pom.xml").Funcs(template.FuncMap{
		"xml": xmlEscape,
	}).Parse(pomTemplateContents))
)

// RenderPOM generates a Maven POM XML from a parsed plugin config.
// The POM includes all runtime dependencies, additional runtimes, and
// build plugins (maven-compiler-plugin, maven-source-plugin, and
// kotlin-maven-plugin for Kotlin plugins).
func RenderPOM(pluginConfig *bufremotepluginconfig.Config) (string, error) {
	if pluginConfig.Registry == nil || pluginConfig.Registry.Maven == nil {
		return "", fmt.Errorf("no Maven registry configured for %q", pluginConfig.Name)
	}
	var buf bytes.Buffer
	if err := pomTemplate.Execute(&buf, pluginConfig); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderDockerfilePOM generates a simplified POM suitable for inlining in a
// Dockerfile heredoc. It strips XML namespace attributes, uses temp
// groupId/artifactId, removes properties, and injects build plugins for
// caching compiler dependencies.
func RenderDockerfilePOM(pluginConfig *bufremotepluginconfig.Config) (string, error) {
	raw, err := RenderPOM(pluginConfig)
	if err != nil {
		return "", err
	}
	return addBuildPlugins(simplifyPOM(raw)), nil
}

func xmlEscape(raw string) (string, error) {
	var b strings.Builder
	if err := xml.EscapeText(&b, []byte(raw)); err != nil {
		return "", err
	}
	return b.String(), nil
}

// simplifyPOM converts the render-pom template output to a simpler format
// matching existing Dockerfile conventions.
func simplifyPOM(pom string) string {
	pom = strings.Replace(pom,
		`<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">`,
		"<project>", 1)
	pom = strings.Replace(pom, "  <groupId>build.buf.gensdk</groupId>", "  <groupId>temp</groupId>", 1)
	pom = strings.Replace(pom, "  <artifactId>example</artifactId>", "  <artifactId>temp</artifactId>", 1)
	pom = strings.Replace(pom, "  <version>1.0.0-SNAPSHOT</version>", "  <version>1.0</version>", 1)
	pom = strings.Replace(pom,
		"  <properties>\n    <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>\n  </properties>\n", "", 1)
	// Remove runtime name comments (e.g., <!-- lite -->).
	lines := strings.Split(pom, "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<!--") && strings.HasSuffix(trimmed, "-->") {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n")
}

const compilerSourcePlugins = `      <plugin>
        <groupId>org.apache.maven.plugins</groupId>
        <artifactId>maven-compiler-plugin</artifactId>
        <version>` + CompilerPluginVersion + `</version>
      </plugin>
      <plugin>
        <groupId>org.apache.maven.plugins</groupId>
        <artifactId>maven-source-plugin</artifactId>
        <version>` + SourcePluginVersion + `</version>
      </plugin>`

// addBuildPlugins injects maven-compiler-plugin and maven-source-plugin into
// the POM. If a <build> section already exists (Kotlin plugins with
// kotlin-maven-plugin), the plugins are appended. Otherwise a new <build>
// section is inserted before </project>.
func addBuildPlugins(pom string) string {
	if strings.Contains(pom, "<build>") {
		return strings.Replace(pom, "    </plugins>", compilerSourcePlugins+"\n    </plugins>", 1)
	}
	buildSection := "  <build>\n    <plugins>\n" + compilerSourcePlugins + "\n    </plugins>\n  </build>\n"
	return strings.Replace(pom, "</project>", buildSection+"</project>", 1)
}
