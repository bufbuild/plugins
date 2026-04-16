// Package main implements the "package" helper: for each plugin selected via
// the PLUGINS env var, it produces a distributable zip archive at
// <output-dir>/<owner>-<name>-<version>.zip. The plugin's Docker image must
// already be available to the local Docker daemon (typically via "make build").
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"github.com/spf13/pflag"

	"github.com/bufbuild/plugins/internal/docker"
	"github.com/bufbuild/plugins/internal/plugin"
	"github.com/bufbuild/plugins/internal/pluginzip"
)

func main() {
	appcmd.Main(context.Background(), newRootCommand("package"))
}

func newRootCommand(name string) *appcmd.Command {
	builder := appext.NewBuilder(name)
	f := &flags{}
	return &appcmd.Command{
		Use:   name,
		Short: "Creates distributable plugin zips from locally-built Docker images.",
		Args:  appcmd.NoArgs,
		Run: builder.NewRunFunc(func(ctx context.Context, container appext.Container) error {
			return run(ctx, container.Logger(), f)
		}),
		BindFlags:           f.Bind,
		BindPersistentFlags: builder.BindRoot,
	}
}

type flags struct {
	pluginsDir string
	dockerOrg  string
	outputDir  string
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.pluginsDir, "dir", ".", "directory path to plugins")
	flagSet.StringVar(&f.dockerOrg, "org", "bufbuild", "Docker organization used to name locally-built images")
	flagSet.StringVar(&f.outputDir, "output-dir", "", "directory to write plugin zip files")
	_ = appcmd.MarkFlagRequired(flagSet, "output-dir")
}

func run(ctx context.Context, logger *slog.Logger, f *flags) error {
	if err := os.MkdirAll(f.outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir %q: %w", f.outputDir, err)
	}
	plugins, err := plugin.FindAll(f.pluginsDir)
	if err != nil {
		return fmt.Errorf("find plugins: %w", err)
	}
	selected, err := plugin.FilterByPluginsEnv(plugins, os.Getenv("PLUGINS"))
	if err != nil {
		return fmt.Errorf("filter plugins by PLUGINS env var: %w", err)
	}
	if len(selected) == 0 {
		logger.InfoContext(ctx, "no plugins selected")
		return nil
	}
	for _, p := range selected {
		imageRef := docker.ImageName(p, f.dockerOrg)
		logger.InfoContext(ctx, "packaging plugin",
			slog.String("plugin", p.String()),
			slog.String("image", imageRef),
		)
		if _, err := pluginzip.Create(ctx, logger, p, imageRef, f.outputDir); err != nil {
			return fmt.Errorf("package %s: %w", p, err)
		}
	}
	return nil
}
