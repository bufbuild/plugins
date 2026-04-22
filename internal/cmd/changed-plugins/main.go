package main

import (
	"context"
	"fmt"
	"strings"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"github.com/spf13/pflag"

	"github.com/bufbuild/plugins/internal/plugin"
)

func main() {
	appcmd.Main(context.Background(), newRootCommand("changed-plugins"))
}

func newRootCommand(name string) *appcmd.Command {
	builder := appext.NewBuilder(name)
	f := &flags{}
	return &appcmd.Command{
		Use:                 name,
		Short:               "Outputs plugins that changed relative to a base git ref.",
		Args:                appcmd.NoArgs,
		Run:                 builder.NewRunFunc(func(ctx context.Context, container appext.Container) error { return run(ctx, container, f) }),
		BindFlags:           f.Bind,
		BindPersistentFlags: builder.BindRoot,
	}
}

type flags struct {
	dir             string
	baseRef         string
	includeTestdata bool
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.dir, "dir", ".", "directory path to plugins")
	flagSet.StringVar(&f.baseRef, "base-ref", "", "base git ref to diff against")
	flagSet.BoolVar(&f.includeTestdata, "include-testdata", false, "include testdata plugin paths in the diff")
	_ = appcmd.MarkFlagRequired(flagSet, "base-ref")
}

func run(ctx context.Context, container appext.Container, f *flags) error {
	plugins, err := plugin.FindAll(f.dir)
	if err != nil {
		return fmt.Errorf("find plugins: %w", err)
	}
	includedPlugins, err := plugin.FilterByBaseRefDiff(ctx, plugins, f.baseRef, f.includeTestdata)
	if err != nil {
		return fmt.Errorf("filter plugins by changed files: %w", err)
	}
	var sb strings.Builder
	for _, p := range includedPlugins {
		sb.WriteString(strings.TrimPrefix(p.Name, "buf.build/"))
		sb.WriteByte(':')
		sb.WriteString(p.PluginVersion)
		sb.WriteByte(' ')
	}
	fmt.Fprintln(container.Stdout(), strings.TrimSpace(sb.String()))
	return nil
}
