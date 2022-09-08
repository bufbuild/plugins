package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/bufbuild/plugins/internal/plugin"
	"github.com/sethvargo/go-envconfig"
)

func main() {
	var (
		base       bool
		dockerFile bool
	)
	flag.BoolVar(&base, "base", false, "only include base Dockerfiles (must be used with -dockerfile)")
	flag.BoolVar(&dockerFile, "dockerfile", false, "outputs Dockerfiles in dependency order")
	flag.Parse()
	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(2)
	}
	if base && !dockerFile {
		_, _ = fmt.Fprintln(os.Stderr, "-base option must be used with -dockerfile option")
		flag.Usage()
		os.Exit(2)
	}

	basedir := flag.Args()[0]

	plugins := make([]*plugin.Plugin, 0)
	if err := plugin.Walk(basedir, func(plugin *plugin.Plugin) {
		plugins = append(plugins, plugin)
	}); err != nil {
		log.Fatalf("failed to walk directory: %v", err)
	}

	var includedPlugins []*plugin.Plugin
	var err error
	if pluginsMatch := os.Getenv("PLUGINS"); pluginsMatch != "" {
		includedPlugins, err = plugin.FilterByPluginsEnv(plugins, pluginsMatch)
		if err != nil {
			log.Fatalf("failed to filter plugins by PLUGINS env var: %v", err)
		}
	} else {
		// Filter by changed plugins (for PR builds)
		includedPlugins, err = plugin.FilterByChangedFiles(plugins, envconfig.OsLookuper())
		if err != nil {
			log.Fatalf("failed to filter plugins by changed files: %v", err)
		}
	}

	if dockerFile {
		dockerFiles, _ := plugin.GetDockerfiles(basedir, includedPlugins)
		for _, f := range dockerFiles {
			if !base || plugin.IsBaseDockerfile(f) {
				if _, err := fmt.Fprintln(os.Stdout, f); err != nil {
					log.Fatalf("failed to print dockerfile: %v", err)
				}
			}
		}
	} else {
		for _, includedPlugin := range includedPlugins {
			if _, err := fmt.Fprintln(os.Stdout, includedPlugin.Path); err != nil {
				log.Fatalf("failed to print plugin: %v", err)
			}
		}
	}
}
