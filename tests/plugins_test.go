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

	"github.com/sethvargo/go-envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/sumdb/dirhash"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginconfig"
	"github.com/bufbuild/plugins/internal/plugin"
)

var (
	bufGenYamlTemplate = template.Must(template.New("buf.gen.yaml").Parse(`version: v1
managed:
  enabled: true
  go_package_prefix:
    default: github.com/bufbuild/plugins/internal/gen
plugins:
{{- range . }}
    - plugin: {{.Name}}
      out: {{.Out}}
      path: {{.Path}}
      strategy: {{.Strategy}}
{{- if .Opts }}
      opt:
	{{- range .Opts }}
        - {{ . }}
	{{- end }}
{{- end }}
{{- end }}
`))
	protocGenPluginTemplate = template.Must(template.New("protoc-gen-plugin").Parse(`#!/bin/bash

exec docker run --log-driver=none --label=buf-plugins-test -i {{.ImageName}} "$@"
`))
	images = []string{
		"eliza",
		"petapis",
	}
)

type pluginConfig struct {
	Name     string
	Out      string
	Path     string
	Opts     []string
	Strategy string
}

func TestGeneration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping code generation test")
	}
	plugins := loadFilteredPlugins(t)
	allPlugins := loadAllPlugins(t)
	allowEmpty, _ := strconv.ParseBool(os.Getenv("ALLOW_EMPTY_PLUGIN_SUM"))
	testPluginWithImage := func(t *testing.T, pluginMeta *plugin.Plugin, image string) {
		t.Helper()
		imageDir, err := filepath.Abs(filepath.Join("testdata", "images"))
		require.NoError(t, err)
		t.Run(image, func(t *testing.T) {
			pluginDir := filepath.Join("testdata", pluginMeta.Name, pluginMeta.PluginVersion, image)
			pluginGenDir := filepath.Join(pluginDir, "gen")
			require.NoError(t, os.RemoveAll(pluginGenDir))
			require.NoError(t, os.MkdirAll(pluginDir, 0755))
			// We prepare the dependencies first to ensure they are run in the correct order
			// when passing pluginConfigs to the buf.gen.yaml. This ensures plugins that
			// depened on insertion points are captured in the correct order.
			var pluginConfigs []pluginConfig
			for _, dep := range pluginMeta.Deps {
				// Lookup the plugin dependency.
				found := lookupPlugin(allPlugins, dep.Plugin)
				require.NotNil(t, found)
				pluginRef, err := newDockerPluginRef(found.NameWithVersion())
				require.NoError(t, err)
				pluginConfigs = append(pluginConfigs, pluginConfig{
					Name:     pluginRef.fileName(),
					Out:      "gen",
					Path:     "./" + pluginRef.fileName(),
					Opts:     found.Registry.Opts,
					Strategy: "all",
				})
				err = buildDockerImage(t, pluginRef, filepath.Dir(found.Path))
				require.NoError(t, err)
				err = createProtocGenPlugin(t, pluginDir, pluginRef)
				require.NoError(t, err)
			}

			pluginRef, err := newDockerPluginRef(pluginMeta.NameWithVersion())
			require.NoError(t, err)
			err = buildDockerImage(t, pluginRef, filepath.Dir(pluginMeta.Path))
			require.NoError(t, err)
			err = createProtocGenPlugin(t, pluginDir, pluginRef)
			require.NoError(t, err)
			pluginConfigs = append(pluginConfigs, pluginConfig{
				Name:     pluginRef.fileName(),
				Out:      "gen",
				Path:     "./" + pluginRef.fileName(),
				Opts:     pluginMeta.Registry.Opts,
				Strategy: "all",
			})
			// Now that we have prepared the main plugin and its dependencies, we can create
			// a buf.gen.yaml file the combines the plugin configs in the correct order.
			err = createBufGenYaml(t, pluginDir, pluginConfigs)
			require.NoError(t, err)

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
		assert.Equal(t, dirVersion, toTest.PluginVersion)
		st, err := os.Stat(filepath.Join(filepath.Dir(toTest.Path), ".dockerignore"))
		if err != nil {
			t.Errorf("failed to stat .dockerignore for %s:%s", toTest.Name, toTest.PluginVersion)
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

func createBufGenYaml(t *testing.T, basedir string, pluginConfigs []pluginConfig) error {
	t.Helper()
	bufGenYaml, err := os.Create(filepath.Join(basedir, "buf.gen.yaml"))
	if err != nil {
		return err
	}
	defer func() {
		require.NoError(t, bufGenYaml.Close())
	}()
	return bufGenYamlTemplate.Execute(bufGenYaml, pluginConfigs)
}

func loadAllPlugins(t *testing.T) []*plugin.Plugin {
	t.Helper()
	var plugins []*plugin.Plugin
	if err := plugin.Walk("..", func(plugin *plugin.Plugin) error {
		plugins = append(plugins, plugin)
		return nil
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

func lookupPlugin(allPlugins []*plugin.Plugin, name string) *plugin.Plugin {
	for _, plugin := range allPlugins {
		if plugin.NameWithVersion() == name {
			return plugin
		}
	}
	return nil
}

func createProtocGenPlugin(t *testing.T, basedir string, plugin *dockerPluginRef) error {
	t.Helper()
	protocGenPlugin, err := os.OpenFile(filepath.Join(basedir, plugin.fileName()), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer func() {
		require.NoError(t, protocGenPlugin.Close())
	}()
	return protocGenPluginTemplate.Execute(protocGenPlugin, plugin)
}

type dockerPluginRef struct {
	dockerOrg string
	owner     string
	name      string
	version   string
}

// ImageName returns a unique name that is used to name a docker image.
func (d *dockerPluginRef) ImageName() string {
	return fmt.Sprintf("%s/plugins-%s-%s:%s", d.dockerOrg, d.owner, d.name, d.version)
}

func (d *dockerPluginRef) fileName() string {
	return fmt.Sprintf("%s_%s_%s.plugin", d.owner, d.name, d.version)
}

// newDockerPluginRef parses a plugin name of the format: remote/owner/name:version.
//
// If the DOCKER_ORG env variable is not set, then the default is bufbuild.
func newDockerPluginRef(input string) (*dockerPluginRef, error) {
	dockerOrg := os.Getenv("DOCKER_ORG")
	if len(dockerOrg) == 0 {
		dockerOrg = "bufbuild"
	}
	fields := strings.SplitN(input, "/", 3)
	if len(fields) != 3 {
		return nil, fmt.Errorf("invalid plugin name: %v", input)
	}
	name, version, ok := strings.Cut(fields[2], ":")
	if !ok {
		return nil, fmt.Errorf("failed to get version from %q", fields[2])
	}
	return &dockerPluginRef{
		dockerOrg: dockerOrg,
		owner:     fields[1],
		name:      name,
		version:   version,
	}, nil
}

func buildDockerImage(t *testing.T, ref *dockerPluginRef, path string) error {
	t.Helper()
	docker, err := exec.LookPath("docker")
	if err != nil {
		return err
	}
	args := fmt.Sprintf("buildx build --label=buf-plugins-test -t %s .", ref.ImageName())
	cmd := exec.Cmd{
		Path:   docker,
		Args:   strings.Split(args, " "),
		Dir:    path,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
