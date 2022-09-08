package plugin

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sethvargo/go-envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalk(t *testing.T) {
	t.Parallel()
	var plugins []*Plugin
	err := Walk("../..", func(plugin *Plugin) {
		plugins = append(plugins, plugin)
	})
	require.NoError(t, err)
	assert.NotEmpty(t, plugins)
}

func TestFilterByPluginsEnv(t *testing.T) {
	t.Parallel()
	var plugins []*Plugin
	err := Walk("../..", func(plugin *Plugin) {
		plugins = append(plugins, plugin)
	})
	require.NoError(t, err)
	assert.Empty(t, runFilterByPluginsEnv(t, plugins, "no-match"))
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "library/connect-go/", "library/connect-web/v0.2.1/"),
		runFilterByPluginsEnv(t, plugins, "connect-go connect-web:v0.2.1"))
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "library/connect-go/", "library/connect-web/v0.2.1/"),
		runFilterByPluginsEnv(t, plugins, "library/connect-go library/connect-web:v0.2.1"))
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "contrib/chrusty/jsonschema/"),
		runFilterByPluginsEnv(t, plugins, "chrusty-jsonschema"))
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "contrib/", "library/"), runFilterByPluginsEnv(t, plugins, ""))
	latestConnectWeb := getLatestPluginVersionsByName(plugins)["buf.build/library/connect-web"]
	require.NotEmpty(t, latestConnectWeb)
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "library/connect-web/"+latestConnectWeb+"/"),
		runFilterByPluginsEnv(t, plugins, "connect-web:latest"))
}

func TestFilterByChangedFiles(t *testing.T) {
	t.Parallel()
	var plugins []*Plugin
	err := Walk("../..", func(plugin *Plugin) {
		plugins = append(plugins, plugin)
	})
	require.NoError(t, err)
	assert.Empty(t, runFilterByChangedFiles(t, plugins, nil, false))
	assert.Len(t, runFilterByChangedFiles(t, plugins, []string{"Makefile"}, true), len(plugins))
	assert.Len(t, runFilterByChangedFiles(t, plugins, []string{"tests/plugins_test.go"}, true), len(plugins))
	assert.Len(t, runFilterByChangedFiles(t, plugins, []string{"tests/testdata/images/eliza.bin.gz"}, true), len(plugins))
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "library/protoc/"), runFilterByChangedFiles(t, plugins, []string{"library/protoc/base-build/Dockerfile"}, true))
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "library/protoc/v21.3/"), runFilterByChangedFiles(t, plugins, []string{"library/protoc/v21.3/base/Dockerfile"}, true))
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "library/protoc/v21.3/cpp/"), runFilterByChangedFiles(t, plugins, []string{"tests/testdata/buf.build/library/cpp/v21.3/eliza/plugin.sum"}, true))
	assert.Equal(t,
		filterPluginsByPathPrefixes(t, plugins,
			"library/grpc/v1.2.0/",
			"library/protoc/v21.3/cpp/",
			"library/protoc/v21.5/java/",
			"library/connect-go/v0.3.0/",
		), runFilterByChangedFiles(t, plugins,
			[]string{
				"library/connect-go/v0.3.0/buf.plugin.yaml",
				"library/grpc/v1.2.0/base/Dockerfile",
				"tests/testdata/buf.build/library/cpp/v21.3/eliza/plugin.sum",
				"tests/testdata/buf.build/library/java/v21.5/petapis/plugin.sum",
			}, true),
	)
}

func TestGetBaseDockerfiles(t *testing.T) {
	files, err := GetBaseDockerfiles("../..")
	require.NoError(t, err)
	assert.NotEmpty(t, files)
	for _, file := range files {
		assert.Containsf(t, []string{"base-build", "base"}, filepath.Base(filepath.Dir(file)), "not a base dockerfile: %s", file)
	}
}

func TestGetDockerfiles(t *testing.T) {
	t.Parallel()
	var plugins []*Plugin
	err := Walk("../..", func(plugin *Plugin) {
		plugins = append(plugins, plugin)
	})
	require.NoError(t, err)
	baseFiles, err := GetBaseDockerfiles("../..")
	require.NoError(t, err)
	require.NotEmpty(t, baseFiles)
	files, err := GetDockerfiles("../..", plugins)
	require.NotEmpty(t, files)
	require.NoError(t, err)
	for _, baseFile := range baseFiles {
		assert.Contains(t, files, baseFile)
	}
	// Verify protoc/base-build/Dockerfile comes before protoc/v21.3/base/Dockerfile
	assert.Less(t, indexOf(t, files, "library/protoc/base-build/Dockerfile"), indexOf(t, files, "library/protoc/v21.3/base/Dockerfile"))
	// Verify protoc/v21.3/base/Dockerfile comes before protoc/v21.3/cpp/Dockerfile
	assert.Less(t, indexOf(t, files, "library/protoc/v21.3/base/Dockerfile"), indexOf(t, files, "library/protoc/v21.3/cpp/Dockerfile"))
}

func indexOf(t *testing.T, haystack []string, needle string) int {
	t.Helper()
	for i, item := range haystack {
		if item == needle {
			return i
		}
	}
	t.Fatalf("failed to find %q in: %v", needle, haystack)
	panic("unreachable")
}

func runFilterByPluginsEnv(t *testing.T, plugins []*Plugin, pluginsEnv string) []string {
	t.Helper()
	filtered, err := FilterByPluginsEnv(plugins, pluginsEnv)
	require.NoError(t, err)
	paths := make([]string, 0, len(filtered))
	for _, plugin := range filtered {
		paths = append(paths, plugin.Relpath)
	}
	return paths
}

func runFilterByChangedFiles(t *testing.T, plugins []*Plugin, allModified []string, anyModified bool) []string {
	t.Helper()
	lookuper := envconfig.MapLookuper(map[string]string{
		"ALL_MODIFIED_FILES": strings.Join(allModified, ","),
		"ANY_MODIFIED":       strconv.FormatBool(anyModified),
	})
	filtered, err := FilterByChangedFiles(plugins, lookuper)
	require.NoError(t, err)
	paths := make([]string, 0, len(filtered))
	for _, plugin := range filtered {
		paths = append(paths, plugin.Relpath)
	}
	return paths
}

func filterPluginsByPathPrefixes(t *testing.T, plugins []*Plugin, prefixes ...string) []string {
	t.Helper()
	var filtered []string
	for _, plugin := range plugins {
		for _, prefix := range prefixes {
			if strings.HasPrefix(plugin.Relpath, prefix) {
				filtered = append(filtered, plugin.Relpath)
				break
			}
		}
	}
	return filtered
}
