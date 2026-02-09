package main

import (
	"context"
	_ "embed"
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"text/template"

	"buf.build/go/app"
	"buf.build/go/app/appcmd"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/spf13/pflag"
)

var (
	//go:embed pom.xml.gotext
	pomTemplateContents string

	pomTemplate = template.Must(template.New("pom.xml").Funcs(map[string]any{
		"xml": xmlEscape,
	}).Parse(pomTemplateContents))
)

func main() {
	appcmd.Main(context.Background(), newCommand("render-pom"))
}

func newCommand(name string) *appcmd.Command {
	cmdFlags := &flags{}
	return &appcmd.Command{
		Use:       name,
		Short:     "Renders a pom.xml template to test building Java/Kotlin code using Maven",
		BindFlags: cmdFlags.Bind,
		Run: func(ctx context.Context, container app.Container) error {
			return run(ctx, container, cmdFlags)
		},
	}
}

type flags struct {
	pluginPath string
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.pluginPath, "plugin", "", "path to plugin YAML file (passed as context to render template)")
	_ = appcmd.MarkFlagRequired(flagSet, "plugin")
}

func run(_ context.Context, _ app.Container, cmdFlags *flags) error {
	pluginConfig, err := bufremotepluginconfig.ParseConfig(cmdFlags.pluginPath)
	if err != nil {
		return fmt.Errorf("failed to parse config %s: %w", cmdFlags.pluginPath, err)
	}
	if pluginConfig.Registry == nil || pluginConfig.Registry.Maven == nil {
		return fmt.Errorf("no Maven registry configured for %q", cmdFlags.pluginPath)
	}
	return pomTemplate.Execute(os.Stdout, pluginConfig)
}

func xmlEscape(raw string) (string, error) {
	var b strings.Builder
	if err := xml.EscapeText(&b, []byte(raw)); err != nil {
		return "", err
	}
	return b.String(), nil
}
