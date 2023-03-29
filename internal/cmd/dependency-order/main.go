package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/bufbuild/plugins/internal/plugin"
)

func main() {
	var (
		relative = flag.Bool("relative", false, "Output relative paths")
	)
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
	includedPlugins, err := plugin.FilterByPluginsEnv(plugins, os.Getenv("PLUGINS"))
	if err != nil {
		log.Fatalf("failed to filter plugins by PLUGINS env var: %v", err)
	}
	for _, includedPlugin := range includedPlugins {
		toOutput := includedPlugin.Path
		if *relative {
			toOutput = includedPlugin.Relpath
		}
		if _, err := fmt.Fprintln(os.Stdout, toOutput); err != nil {
			log.Fatalf("failed to print plugin: %v", err)
		}
	}
}
