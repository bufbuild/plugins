package main

import (
	"context"
	"fmt"
	"path/filepath"

	"buf.build/go/app"
	"buf.build/go/app/appcmd"

	"github.com/bufbuild/plugins/internal/maven"
)

func main() {
	appcmd.Main(context.Background(), newCommand("regenerate-maven-poms"))
}

func newCommand(name string) *appcmd.Command {
	return &appcmd.Command{
		Use:   name + " <plugin-dir> [<plugin-dir>...]",
		Short: "Regenerates maven-deps POM and Dockerfile stage for Java/Kotlin plugins",
		Args:  appcmd.MinimumNArgs(1),
		Run: func(_ context.Context, container app.Container) error {
			for i := range container.NumArgs() {
				pluginDir := container.Arg(i)
				// pluginDir is e.g. plugins/org/name/version, so
				// plugins root is 3 levels up.
				pluginsDir := filepath.Dir(filepath.Dir(filepath.Dir(pluginDir)))
				if err := maven.RegenerateMavenDeps(pluginDir, pluginsDir); err != nil {
					return fmt.Errorf("failed to regenerate %s: %w", pluginDir, err)
				}
				fmt.Fprintf(container.Stdout(), "regenerated: %s\n", pluginDir)
			}
			return nil
		},
	}
}
