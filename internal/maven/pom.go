package maven

import (
	"bytes"
	_ "embed"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
)

type templateData struct {
	*bufremotepluginconfig.Config
}

var (
	//go:embed pom.xml.gotext
	pomTemplateContents string

	pomTemplate = template.Must(template.New("pom.xml").Funcs(template.FuncMap{
		"xml": xmlEscape,
	}).Parse(pomTemplateContents))
)

// RenderPOM generates a Maven POM XML from a parsed plugin config.
// The POM includes all runtime dependencies, additional runtimes, and
// kotlin-maven-plugin for Kotlin plugins. maven-compiler-plugin and
// maven-source-plugin are bundled in the maven-jdk base image.
func RenderPOM(pluginConfig *bufremotepluginconfig.Config) (string, error) {
	if pluginConfig.Registry == nil || pluginConfig.Registry.Maven == nil {
		return "", fmt.Errorf("no Maven registry configured for %q", pluginConfig.Name)
	}
	data := templateData{
		Config: pluginConfig,
	}
	var buf bytes.Buffer
	if err := pomTemplate.Execute(&buf, data); err != nil {
		return "", err
	}
	pom := buf.String()
	decoder := xml.NewDecoder(strings.NewReader(pom))
	for {
		if _, err := decoder.Token(); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", fmt.Errorf("generated POM is not well-formed XML: %w", err)
		}
	}
	return pom, nil
}

func xmlEscape(raw string) (string, error) {
	var b strings.Builder
	if err := xml.EscapeText(&b, []byte(raw)); err != nil {
		return "", err
	}
	return b.String(), nil
}
