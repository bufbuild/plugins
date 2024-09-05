package tests

import (
	"bufio"
	"cmp"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"text/template"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
	"golang.org/x/mod/sumdb/dirhash"

	"github.com/bufbuild/plugins/internal/plugin"
)

const (
	defaultGoModVersion = "1.16"
	defaultGoPkgPrefix  = "github.com/bufbuild/plugins/internal/gen"
)

var (
	bufGenYamlTemplate = template.Must(template.New("buf.gen.yaml").Parse(`version: v1
managed:
  enabled: true
  go_package_prefix:
    default: {{.GoPkgPrefix}}
plugins:
  - name: {{.Name}}
    out: gen
    path: ./protoc-gen-plugin
    strategy: all
{{- if .Opts }}
    opt:
{{- range .Opts }}
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
	imageOverrides = map[string][]string{
		// betterproto (at least at v1.2.5) doesn't support eliza since it uses client streaming
		"buf.build/community/danielgtaylor-betterproto": {"petapis"},
	}
	// Options to pass to the plugin during tests. The prost plugins depend on insertion points by default, which
	// breaks our current test strategy which is to run each plugin in isolation. Override the test options for
	// these plugins until the tests are updated to support running all plugin dependencies in sequence.
	testOverrideOptions = map[string][]string{
		"buf.build/community/neoeinstein-prost-crate": {"no_features"},
		"buf.build/community/neoeinstein-prost-serde": {"no_include"},
		"buf.build/community/neoeinstein-tonic":       {"no_include"},
		"buf.build/community/mercari-grpc-federation": {"paths=source_relative"},
	}
	// Some plugins do not generate any code for the test protos, so we allow an empty plugin.sum file for these
	// plugins. The format of the map is map[pluginName]map[image]bool, where the bool indicates whether an empty
	// plugin.sum file is allowed for the given image.
	allowedEmptyPluginSums = map[string]map[string]bool{
		"buf.build/bufbuild/validate-java":            {"eliza": true, "petapis": true},
		"buf.build/grpc-ecosystem/gateway":            {"eliza": true, "petapis": true},
		"buf.build/community/mercari-grpc-federation": {"eliza": true, "petapis": true},
		"buf.build/googlecloudplatform/bq-schema":     {"eliza": true, "petapis": true},
	}
)

func TestGeneration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping code generation test")
	}
	ctx := context.Background()
	allowEmpty, _ := strconv.ParseBool(os.Getenv("ALLOW_EMPTY_PLUGIN_SUM"))
	testPluginWithImage := func(t *testing.T, pluginMeta *plugin.Plugin, image string) {
		t.Helper()
		t.Run(image, func(t *testing.T) {
			t.Parallel()
			pluginDir := filepath.Join("testdata", pluginMeta.Name, pluginMeta.PluginVersion, image)
			genDir := runPluginWithImage(ctx, t, pluginDir, pluginMeta, image, defaultGoPkgPrefix)
			genDirHash, err := dirhash.HashDir(genDir, "", dirhash.Hash1)
			require.NoError(t, err, "failed to calculate directory hash of generated code")
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
				assert.Equal(t, existingPluginSum, genDirHash)
			}
			require.NoError(t, os.WriteFile(pluginImageSumFile, []byte(genDirHash+"\n"), 0o644))
		})
	}

	plugins := loadFilteredPlugins(t)
	for _, toTest := range plugins {
		testName := strings.TrimSuffix(toTest.Relpath, "/buf.plugin.yaml")
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			images := images
			if imageOverrides, ok := imageOverrides[toTest.Name]; ok {
				images = imageOverrides
			}
			for _, image := range images {
				testPluginWithImage(t, toTest, image)
			}
			switch strings.TrimPrefix(toTest.Name, "buf.build/") {
			case "bufbuild/knit-ts":
				testPluginWithImage(t, toTest, "knit-demo")
			case "grpc-ecosystem/gateway":
				if semver.Compare(toTest.PluginVersion, "v2.16.0") >= 0 {
					testPluginWithImage(t, toTest, "grpc-gateway")
				}
			case "community/mercari-grpc-federation":
				switch {
				case semver.Compare(toTest.PluginVersion, "v1.4.1") >= 0:
					testPluginWithImage(t, toTest, "grpc-federation-v1.4.1")
				case semver.Compare(toTest.PluginVersion, "v0.13.6") >= 0:
					testPluginWithImage(t, toTest, "grpc-federation-v0.13.6")
				case semver.Compare(toTest.PluginVersion, "v0.11.0") >= 0:
					testPluginWithImage(t, toTest, "grpc-federation-v0.11.0")
				default:
					// There was a breaking change in v0.11.0, so we need to test the old version separately
					// https://github.com/mercari/grpc-federation/commit/baca78bf2421322c97e6977a06931fed29e4058a
					testPluginWithImage(t, toTest, "grpc-federation")
				}
			case "googlecloudplatform/bq-schema":
				testPluginWithImage(t, toTest, "bq-schema")
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
		config, err := bufremotepluginconfig.GetConfigForData(context.Background(), yamlBytes)
		require.NoErrorf(t, err, "invalid plugin config: %q", p.Path)
		assert.NotEmpty(t, config.Name)
		assert.NotEmpty(t, config.PluginVersion)
		assert.NotEmpty(t, config.SPDXLicenseID)
		assert.NotEmpty(t, config.LicenseURL)
		// Don't allow underscore in plugin names - this would cause issues in remote packages
		assert.NotContains(t, config.Name.IdentityString(), "_")
	}
}

func TestGoMinVersion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	client := &http.Client{}
	plugins := loadFilteredPlugins(t)
	for _, p := range plugins {
		if p.Registry.Go == nil {
			continue
		}
		if len(p.Registry.Go.Deps) == 0 {
			continue
		}
		minVersion := cmp.Or(p.Registry.Go.MinVersion, defaultGoModVersion)
		// Verify that we only store major.minor versions in buf.plugin.yaml.
		require.Equal(t, minVersion, strings.TrimPrefix(semver.MajorMinor("v"+minVersion), "v"))
		t.Run(fmt.Sprintf("%s/%s@%s", p.Identity.Owner(), p.Identity.Plugin(), p.PluginVersion), func(t *testing.T) {
			t.Parallel()
			maxMajorMinor := minVersion
			maxDep := ""
			for _, dep := range p.Registry.Go.Deps {
				depGoModVersion := getGoModVersion(ctx, t, client, dep.Module, dep.Version)
				depGoModMajorMinor := strings.TrimPrefix(semver.MajorMinor("v"+depGoModVersion), "v")
				require.NotEmptyf(t, depGoModMajorMinor, "invalid dep go version: %q", depGoModMajorMinor)
				if semver.Compare("v"+maxMajorMinor, "v"+depGoModMajorMinor) < 0 {
					maxMajorMinor = depGoModMajorMinor
					maxDep = fmt.Sprintf("%s@%s", dep.Module, dep.Version)
				}
			}
			assert.Equalf(
				t,
				maxMajorMinor,
				minVersion,
				"expected go plugin registry.go.min_version %q to be greater or equal to the max version of its dependencies %q (%s)",
				minVersion,
				maxMajorMinor,
				maxDep,
			)
		})
	}
}

func TestGrpcGatewayDeprecationMessage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	genSDKGoPkg := "buf.build/gen/go/someorg/somemodule/protocolbuffers/go"
	plugins := loadFilteredPlugins(t)
	for _, p := range plugins {
		if p.Identity.IdentityString() != "buf.build/grpc-ecosystem/gateway" {
			continue
		}
		if semver.Compare(p.PluginVersion, "v2.16.0") < 0 {
			continue
		}
		genDir := runPluginWithImage(ctx, t, t.TempDir(), p, "grpc-gateway", genSDKGoPkg)
		err := filepath.WalkDir(genDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.Contains(line, "Deprecated:") && strings.Contains(line, genSDKGoPkg) {
					return fmt.Errorf("line %q should not contain %q", line, genSDKGoPkg)
				}
			}
			return scanner.Err()
		})
		require.NoError(t, err)
	}
}

func TestMavenDependencies(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	client := &http.Client{}
	plugins := loadFilteredPlugins(t)
	for _, p := range plugins {
		if p.Registry.Maven == nil || len(p.Registry.Maven.Deps) == 0 {
			continue
		}
		t.Run(fmt.Sprintf("%s/%s@%s", p.Identity.Owner(), p.Identity.Plugin(), p.PluginVersion), func(t *testing.T) {
			t.Parallel()
			for _, dep := range p.Registry.Maven.Deps {
				fields := strings.Split(dep, ":")
				require.Len(t, fields, 3)
				groupID, artifactID, version := fields[0], fields[1], fields[2]
				url := fmt.Sprintf(
					"https://repo.maven.apache.org/maven2/%[1]s/%[2]s/%[3]s/%[2]s-%[3]s.pom",
					strings.ReplaceAll(groupID, ".", "/"),
					artifactID,
					version,
				)
				req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
				require.NoError(t, err)
				resp, err := client.Do(req)
				require.NoError(t, err)
				require.NoError(t, resp.Body.Close())
				assert.Equalf(t, http.StatusOK, resp.StatusCode, "failed to find maven dependency %s", dep)
			}
		})
	}
}

func TestNugetDependencies(t *testing.T) {
	t.Parallel()

	type packageReference struct {
		XMLName   xml.Name `xml:"PackageReference"`
		Include   string   `xml:"Include,attr"`
		Version   string   `xml:"Version,attr"`
		Condition string   `xml:"Condition,attr,omitempty"`
	}
	type propertyGroup struct {
		XMLName          xml.Name `xml:"PropertyGroup"`
		TargetFramework  string   `xml:"TargetFramework,omitempty"`
		TargetFrameworks string   `xml:"TargetFrameworks,omitempty"`
	}
	type csharpProject struct {
		XMLName           xml.Name           `xml:"Project"`
		SDK               string             `xml:"Sdk,attr"`
		PropertyGroup     propertyGroup      `xml:"PropertyGroup"`
		PackageReferences []packageReference `xml:"ItemGroup>PackageReference"`
	}

	allPlugins := loadAllPlugins(t)
	plugins := loadFilteredPlugins(t)
	for _, p := range plugins {
		if p.Registry.Nuget == nil {
			continue
		}
		t.Run(p.String(), func(t *testing.T) {
			t.Parallel()
			// We require all NuGet-enabled plugins to have a build.csproj file to load plugin dependencies.
			buildCsprojBytes, err := os.ReadFile(filepath.Join(filepath.Dir(p.Path), "build.csproj"))
			require.NoError(t, err)
			var project csharpProject
			require.NoError(t, xml.Unmarshal(buildCsprojBytes, &project))

			nugetConfig := p.Registry.Nuget
			if len(nugetConfig.TargetFrameworks) == 1 {
				require.Equal(t, project.PropertyGroup.TargetFramework, nugetConfig.TargetFrameworks[0])
			} else {
				require.EqualValues(t, strings.Split(project.PropertyGroup.TargetFrameworks, ";"), nugetConfig.TargetFrameworks)
			}

			// name -> version
			allDependencies := map[string]string{}
			populateNugetDeps(t, allDependencies, p, allPlugins)

			// Include -> Version
			packageReferences := map[string]string{}
			for _, packageReference := range project.PackageReferences {
				// TODO: Support conditions in this test.
				require.Empty(t, packageReference.Condition)
				_, exists := packageReferences[packageReference.Include]
				// Should not have duplicate Include values in the build.csproj.
				require.False(t, exists)
				packageReferences[packageReference.Include] = packageReference.Version
			}

			require.Equal(t, allDependencies, packageReferences)
		})
	}
}

func runPluginWithImage(ctx context.Context, t *testing.T, basedir string, pluginMeta *plugin.Plugin, image string, goPkgPrefix string) string {
	t.Helper()
	gendir := filepath.Join(basedir, "gen")
	require.NoError(t, createBufGenYaml(t, basedir, pluginMeta, goPkgPrefix))
	require.NoError(t, createProtocGenPlugin(t, basedir, pluginMeta))
	imageDir, err := filepath.Abs(filepath.Join("testdata", "images"))
	require.NoError(t, err)
	bufCmd := exec.CommandContext(ctx, "buf", "generate", filepath.Join(imageDir, image+".bin.gz"))
	bufCmd.Dir = basedir
	output, err := bufCmd.CombinedOutput()
	require.NoErrorf(t, err, "buf generate failed - output: %s", string(output))
	// Ensure the gen directory is not empty, otherwise we'll get a sum of an empty directory.
	// This is either a problem with the plugin itself, or the input. Some plugins require
	// input protos that contain custom options to generate code. We should craft a test proto
	// for these plugins. See grpc-ecosystem/gateway for an example.
	genDirFiles, err := os.ReadDir(gendir)
	require.NoError(t, err, "failed to read generated code directory")
	if len(genDirFiles) == 0 {
		allowedEmptyImages, ok := allowedEmptyPluginSums[pluginMeta.Name]
		if !ok || !allowedEmptyImages[image] {
			t.Fatalf("generated code directory is empty for %s", pluginMeta)
		}
	}
	return gendir
}

func getGoModVersion(ctx context.Context, t *testing.T, client *http.Client, module string, version string) string {
	t.Helper()
	goModURL := fmt.Sprintf("https://proxy.golang.org/%s/@v/%s.mod", module, version)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, goModURL, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	modFileContents, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	modFile, err := modfile.ParseLax("go.mod", modFileContents, nil)
	require.NoError(t, err)
	if modFile.Go == nil {
		return defaultGoModVersion
	}
	return cmp.Or(modFile.Go.Version, defaultGoModVersion)
}

func createBufGenYaml(t *testing.T, basedir string, plugin *plugin.Plugin, goPkgPrefix string) error {
	t.Helper()
	require.NoError(t, os.MkdirAll(basedir, 0755))
	bufGenYAMLPath := filepath.Join(basedir, "buf.gen.yaml")
	bufGenYaml, err := os.Create(bufGenYAMLPath)
	require.NoErrorf(t, err, "failed to create %s: %s", bufGenYAMLPath, err)
	defer func() {
		require.NoError(t, bufGenYaml.Close())
	}()
	opts := plugin.Registry.Opts
	opts = append(opts, testOverrideOptions[plugin.Name]...)
	return bufGenYamlTemplate.Execute(bufGenYaml, map[string]any{
		"GoPkgPrefix": goPkgPrefix,
		"Name":        filepath.Base(plugin.Name),
		"Opts":        opts,
	})
}

func loadAllPlugins(t *testing.T) []*plugin.Plugin {
	t.Helper()
	plugins, err := plugin.FindAll("..")
	require.NoError(t, err)
	return plugins
}

func loadFilteredPlugins(t *testing.T) []*plugin.Plugin {
	t.Helper()
	plugins := loadAllPlugins(t)
	filtered, err := plugin.FilterByPluginsEnv(plugins, os.Getenv("PLUGINS"))
	require.NoError(t, err)
	return filtered
}

func createProtocGenPlugin(t *testing.T, basedir string, plugin *plugin.Plugin) error {
	t.Helper()
	protocGenPlugin, err := os.OpenFile(filepath.Join(basedir, "protoc-gen-plugin"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
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
		"Version":   plugin.PluginVersion,
	})
}

func populateNugetDeps(t *testing.T, dependencies map[string]string, nugetPlugin *plugin.Plugin, plugins []*plugin.Plugin) {
	t.Helper()
	for _, pluginDependency := range nugetPlugin.Deps {
		var pluginDep *plugin.Plugin
		for _, otherPlugin := range plugins {
			if otherPlugin.String() == pluginDependency.Plugin {
				pluginDep = otherPlugin
				break
			}
		}
		require.NotNil(t, pluginDep)
		// Recurse into deps.
		populateNugetDeps(t, dependencies, pluginDep, plugins)
	}

	for _, nugetDependency := range nugetPlugin.Registry.Nuget.Deps {
		// TODO: Handle target frameworks in dependencies.
		require.Empty(t, nugetDependency.TargetFrameworks)

		if existingVersion, exists := dependencies[nugetDependency.Name]; exists {
			// Versions cannot conflict amongst deps.
			require.Equal(t, existingVersion, nugetDependency.Version)
		}
		dependencies[nugetDependency.Name] = nugetDependency.Version
	}
}
