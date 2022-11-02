package tests

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"text/template"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginconfig"
	"github.com/bufbuild/plugins/internal/plugin"
	"github.com/sethvargo/go-envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/sumdb/dirhash"
)

var (
	bufGenYamlTemplate = template.Must(template.New("buf.gen.yaml").Parse(`version: v1
managed:
  enabled: true
  go_package_prefix:
    default: github.com/bufbuild/plugins/internal/gen
plugins:
  - name: {{.Name}}
    out: gen
    path: ./protoc-gen-plugin
    strategy: all
{{- if .DefaultOpts }}
    opt:
{{- range .DefaultOpts }}
      - {{ . }}
{{- end }}
{{- end -}}
`))
	protocGenPluginTemplate = template.Must(template.New("protoc-gen-plugin").Parse(`#!/bin/bash

exec docker run --log-driver=none --rm -i {{.ImageName}}:{{.Version}} "$@"
`))
	images = []string{
		"eliza",
		"petapis",
	}
)

func TestGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping code generation test")
	}
	t.Parallel()
	allowEmpty, _ := strconv.ParseBool(os.Getenv("ALLOW_EMPTY_PLUGIN_SUM"))
	testPluginWithImage := func(t *testing.T, pluginMeta *plugin.Plugin, image string) {
		imageDir, err := filepath.Abs(filepath.Join("testdata", "images"))
		require.NoError(t, err)
		t.Run(image, func(t *testing.T) {
			t.Parallel()
			pluginDir := filepath.Join("testdata", pluginMeta.Name, pluginMeta.Version, image)
			pluginGenDir := filepath.Join(pluginDir, "gen")
			require.NoError(t, os.RemoveAll(pluginGenDir))
			require.NoError(t, os.MkdirAll(pluginDir, 0755))
			require.NoError(t, createBufGenYaml(t, pluginDir, pluginMeta))
			require.NoError(t, createProtocGenPlugin(t, pluginDir, pluginMeta))
			bufCmd := exec.Command("buf", "generate", filepath.Join(imageDir, image+".bin.gz"))
			bufCmd.Dir = pluginDir
			output, err := bufCmd.CombinedOutput()
			assert.NoErrorf(t, err, "buf generate output: %s", string(output))
			genDirHash, err := dirhash.HashDir(pluginGenDir, "", dirhash.Hash1)
			require.NoError(t, err)
			pluginImageSumFile := filepath.Join(pluginDir, "plugin.sum")
			existingPluginSumBytes, err := os.ReadFile(pluginImageSumFile)
			if err != nil {
				if os.IsNotExist(err) {
					t.Logf("plugin sum file does not exist: %s", pluginImageSumFile)
				} else {
					t.Error(err)
				}
			}
			existingPluginSum := strings.TrimSpace(string(existingPluginSumBytes))
			if allowEmpty && existingPluginSum == "" {
				t.Log("allowing empty plugin.sum file (used by fetcher command)")
			} else {
				assert.Equal(t, genDirHash, existingPluginSum)
			}
			require.NoError(t, os.WriteFile(pluginImageSumFile, []byte(genDirHash+"\n"), 0644))
		})
	}

	plugins := loadFilteredPlugins(t)
	for _, toTest := range plugins {
		toTest := toTest
		t.Run(strings.TrimSuffix(toTest.Relpath, "/buf.plugin.yaml"), func(t *testing.T) {
			t.Parallel()
			for _, image := range images {
				image := image
				testPluginWithImage(t, toTest, image)
			}
		})
	}
}

func TestPluginVersionMatchesDirectory(t *testing.T) {
	t.Parallel()
	// Verify that buf.plugin.yaml plugin_version matches the directory name
	plugins := loadAllPlugins(t)
	for _, toTest := range plugins {
		dirPath := filepath.Dir(toTest.Path)
		dirVersion := filepath.Base(dirPath)
		assert.Equal(t, dirVersion, toTest.Version)
		st, err := os.Stat(filepath.Join(filepath.Dir(toTest.Path), ".dockerignore"))
		if err != nil {
			t.Errorf("failed to stat .dockerignore for %s:%s", toTest.Name, toTest.Version)
		} else {
			assert.False(t, st.IsDir())
		}
	}
}

func TestBufPluginConfig(t *testing.T) {
	t.Parallel()
	plugins := loadAllPlugins(t)
	for _, p := range plugins {
		yamlBytes, err := os.ReadFile(p.Path)
		require.NoError(t, err)
		config, err := bufpluginconfig.GetConfigForData(context.Background(), yamlBytes)
		require.NoErrorf(t, err, "invalid plugin config: %q", p.Path)
		assert.NotEmpty(t, config.Name)
		assert.NotEmpty(t, config.PluginVersion)
		assert.NotEmpty(t, config.SPDXLicenseID)
		assert.NotEmpty(t, config.LicenseURL)
	}
}

func createBufGenYaml(t *testing.T, basedir string, plugin *plugin.Plugin) error {
	t.Helper()
	bufGenYaml, err := os.Create(filepath.Join(basedir, "buf.gen.yaml"))
	if err != nil {
		return err
	}
	defer func() {
		require.NoError(t, bufGenYaml.Close())
	}()
	return bufGenYamlTemplate.Execute(bufGenYaml, map[string]any{
		"Name":        filepath.Base(plugin.Name),
		"DefaultOpts": plugin.DefaultOpts,
	})
}

func loadAllPlugins(t *testing.T) []*plugin.Plugin {
	t.Helper()
	var plugins []*plugin.Plugin
	if err := plugin.Walk("..", func(plugin *plugin.Plugin) {
		plugins = append(plugins, plugin)
	}); err != nil {
		t.Fatalf("failed to find plugins: %v", err)
	}
	return plugins
}

func loadFilteredPlugins(t *testing.T) []*plugin.Plugin {
	t.Helper()
	plugins := loadAllPlugins(t)
	var filtered []*plugin.Plugin
	var err error
	if pluginsEnv := os.Getenv("PLUGINS"); pluginsEnv != "" {
		filtered, err = plugin.FilterByPluginsEnv(plugins, pluginsEnv)
	} else {
		filtered, err = plugin.FilterByChangedFiles(plugins, envconfig.OsLookuper())
	}
	require.NoError(t, err)
	return filtered
}

func createProtocGenPlugin(t *testing.T, basedir string, plugin *plugin.Plugin) error {
	t.Helper()
	protocGenPlugin, err := os.OpenFile(filepath.Join(basedir, "protoc-gen-plugin"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer func() {
		require.NoError(t, protocGenPlugin.Close())
	}()
	fields := strings.SplitN(plugin.Name, "/", 3)
	if len(fields) != 3 {
		return fmt.Errorf("invalid plugin name: %v", plugin.Name)
	}
	dockerOrg := os.Getenv("DOCKER_ORG")
	if len(dockerOrg) == 0 {
		dockerOrg = "bufbuild"
	}
	return protocGenPluginTemplate.Execute(protocGenPlugin, map[string]any{
		"ImageName": fmt.Sprintf("%s/plugins-%s-%s", dockerOrg, fields[1], fields[2]),
		"Version":   plugin.Version,
	})
}
