package main

import (
	"context"
	"fmt"
	"os"

	"buf.build/go/app"
	"buf.build/go/app/appcmd"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/spf13/pflag"

	"github.com/bufbuild/plugins/internal/maven"
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
	pom, err := maven.RenderPOM(pluginConfig)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(os.Stdout, pom)
	return err
}
