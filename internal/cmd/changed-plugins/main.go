package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/sethvargo/go-envconfig"

	"github.com/bufbuild/plugins/internal/plugin"
)

func main() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(2)
	}
	basedir := flag.Args()[0]

	plugins, err := plugin.FindAll(basedir)
	if err != nil {
		log.Fatalf("failed to find plugins: %v", err)
	}
	// Filter by changed plugins (for PR builds)
	includedPlugins, err := plugin.FilterByBaseRefDiff(context.Background(), plugins, envconfig.OsLookuper())
	if err != nil {
		log.Fatalf("failed to filter plugins by changed files: %v", err)
	}
	var sb strings.Builder
	for _, includedPlugin := range includedPlugins {
		sb.WriteString(strings.TrimPrefix(includedPlugin.Name, "buf.build/"))
		sb.WriteByte(':')
		sb.WriteString(includedPlugin.PluginVersion)
		sb.WriteByte(' ')
	}
	fmt.Println(strings.TrimSpace(sb.String())) //nolint:forbidigo
}
