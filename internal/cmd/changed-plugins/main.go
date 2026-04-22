package main

import (
	"context"
	"fmt"
	"strings"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"

	"github.com/bufbuild/plugins/internal/plugin"
)

func main() {
	appcmd.Main(context.Background(), newRootCommand("changed-plugins"))
}

func newRootCommand(name string) *appcmd.Command {
	builder := appext.NewBuilder(name)
	return &appcmd.Command{
		Use:                 name + " <directory>",
		Short:               "Outputs plugins that changed relative to a base git ref.",
		Args:                appcmd.ExactArgs(1),
		Run:                 builder.NewRunFunc(run),
		BindPersistentFlags: builder.BindRoot,
	}
}

func run(ctx context.Context, container appext.Container) error {
	plugins, err := plugin.FindAll(container.Arg(0))
	if err != nil {
		return fmt.Errorf("find plugins: %w", err)
	}
	includedPlugins, err := plugin.FilterByBaseRefDiff(ctx, plugins)
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
