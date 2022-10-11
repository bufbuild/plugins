package plugin

import (
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
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "plugins/bufbuild/connect-go/", "plugins/bufbuild/connect-web/v0.2.1/"),
		runFilterByPluginsEnv(t, plugins, "connect-go connect-web:v0.2.1"))
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "plugins/bufbuild/connect-go/", "plugins/bufbuild/connect-web/v0.2.1/"),
		runFilterByPluginsEnv(t, plugins, "bufbuild/connect-go bufbuild/connect-web:v0.2.1"))
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "plugins/community/chrusty-jsonschema/"),
		runFilterByPluginsEnv(t, plugins, "chrusty-jsonschema"))
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "plugins/"), runFilterByPluginsEnv(t, plugins, ""))
	latestConnectWeb := getLatestPluginVersionsByName(plugins)["buf.build/bufbuild/connect-web"]
	require.NotEmpty(t, latestConnectWeb)
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "plugins/bufbuild/connect-web/"+latestConnectWeb+"/"),
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
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "plugins/protocolbuffers/cpp/v21.3/"), runFilterByChangedFiles(t, plugins, []string{"tests/testdata/buf.build/protocolbuffers/cpp/v21.3/eliza/plugin.sum"}, true))
	assert.Equal(t,
		filterPluginsByPathPrefixes(t, plugins,
			"plugins/protocolbuffers/cpp/v21.3/",
			"plugins/protocolbuffers/java/v21.5/",
			"plugins/bufbuild/connect-go/v0.3.0/",
		), runFilterByChangedFiles(t, plugins,
			[]string{
				"plugins/bufbuild/connect-go/v0.3.0/buf.plugin.yaml",
				"tests/testdata/buf.build/protocolbuffers/cpp/v21.3/eliza/plugin.sum",
				"tests/testdata/buf.build/protocolbuffers/java/v21.5/petapis/plugin.sum",
			}, true),
	)
}

func TestGetDockerfiles(t *testing.T) {
	t.Parallel()
	var plugins []*Plugin
	err := Walk("../..", func(plugin *Plugin) {
		plugins = append(plugins, plugin)
	})
	require.NoError(t, err)
	files, err := GetDockerfiles(plugins)
	require.NoError(t, err)
	require.NotEmpty(t, files)
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
