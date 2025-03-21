package plugin

import (
	"strconv"
	"strings"
	"testing"

	"github.com/sethvargo/go-envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindAll(t *testing.T) {
	t.Parallel()
	plugins, err := FindAll("../..")
	require.NoError(t, err)
	assert.NotEmpty(t, plugins)
}

func TestWalk(t *testing.T) {
	t.Parallel()
	var plugins []*Plugin
	err := Walk("../..", func(plugin *Plugin) error {
		plugins = append(plugins, plugin)
		return nil
	})
	require.NoError(t, err)
	assert.NotEmpty(t, plugins)
}

func TestFilterByPluginsEnv(t *testing.T) {
	t.Parallel()
	plugins, err := FindAll("../..")
	require.NoError(t, err)
	assert.Empty(t, runFilterByPluginsEnv(t, plugins, "no-match"))
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "plugins/bufbuild/connect-go/", "plugins/bufbuild/connect-web/v0.2.1/"),
		runFilterByPluginsEnv(t, plugins, "connect-go connect-web:v0.2.1"))
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "plugins/bufbuild/connect-go/", "plugins/bufbuild/connect-web/v0.2.1/"),
		runFilterByPluginsEnv(t, plugins, "connect-go,connect-web:v0.2.1"))
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "plugins/bufbuild/connect-go/", "plugins/bufbuild/connect-web/v0.2.1/"),
		runFilterByPluginsEnv(t, plugins, "bufbuild/connect-go bufbuild/connect-web:v0.2.1"))
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "plugins/community/chrusty-jsonschema/"),
		runFilterByPluginsEnv(t, plugins, "chrusty-jsonschema"))
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "plugins/"), runFilterByPluginsEnv(t, plugins, "all"))
	latestConnectWeb := getLatestPluginVersionsByName(plugins)["buf.build/bufbuild/connect-web"]
	require.NotEmpty(t, latestConnectWeb)
	assert.Equal(t, filterPluginsByPathPrefixes(t, plugins, "plugins/bufbuild/connect-web/"+latestConnectWeb+"/"),
		runFilterByPluginsEnv(t, plugins, "connect-web:latest"))
}

func TestFilterByChangedFiles(t *testing.T) {
	t.Parallel()
	allPlugins, err := FindAll("../..")
	require.NoError(t, err)
	assert.Empty(t, runFilterByChangedFiles(t, allPlugins, nil, false))
	expected := filterPluginsByPathPrefixes(t, allPlugins, "plugins/protocolbuffers/cpp/v21.7/")
	got := runFilterByChangedFiles(
		t,
		allPlugins,
		[]string{"tests/testdata/buf.build/protocolbuffers/cpp/v21.7/eliza/plugin.sum"},
		true,
	)
	assert.Equal(t, expected, got)
	assert.Equal(t,
		filterPluginsByPathPrefixes(t, allPlugins,
			"plugins/protocolbuffers/cpp/v21.7/",
			"plugins/protocolbuffers/java/v21.7/",
			"plugins/bufbuild/connect-go/v1.0.0/",
		), runFilterByChangedFiles(t, allPlugins,
			[]string{
				"plugins/bufbuild/connect-go/v1.0.0/buf.plugin.yaml",
				"tests/testdata/buf.build/protocolbuffers/cpp/v21.7/eliza/plugin.sum",
				"tests/testdata/buf.build/protocolbuffers/java/v21.7/petapis/plugin.sum",
			}, true),
	)
}

func TestFilterPluginPaths(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name            string
		includeTestdata bool
		expected        []string
		changedFiles    []string
	}
	testCases := []testCase{
		{
			name:     "no_changed_files",
			expected: []string{},
		},
		{
			name: "changed_files",
			expected: []string{
				"plugins/owner1/plugin1/v1.2.3/Dockerfile",
				"plugins/owner2/plugin2/v4.5.6/some_other.file",
				"plugins/owner3/plugin3/v7.8.9/some/deeper.file",
			},
			changedFiles: []string{
				"some/unrelated/file.txt",
				"plugins/owner1/plugin1/v1.2.3/Dockerfile",
				"plugins/owner2/plugin2/v4.5.6/some_other.file",
				"plugins/owner3/plugin3/v7.8.9/some/deeper.file",
				"plugins/owner4/plugin4/source.yaml",                        // not in the version dir
				"tests/testdata/buf.build/owner1/plugin1/v1.2.3/Dockerfile", // testdata is not included
			},
		},
		{
			name:            "changed_files_with_testdata",
			includeTestdata: true,
			expected: []string{
				"plugins/owner1/plugin1/v1.2.3/Dockerfile",
				"plugins/owner2/plugin2/v4.5.6/some_other.file",
				"plugins/owner3/plugin3/v7.8.9/some/deeper.file",
				"tests/testdata/buf.build/owner1/plugin1/v1.2.3/Dockerfile",
				"tests/testdata/buf.build/owner2/plugin2/v1.2.3/some/deeper.file",
			},
			changedFiles: []string{
				"some/unrelated/file.txt",
				"plugins/owner1/plugin1/v1.2.3/Dockerfile",
				"plugins/owner2/plugin2/v4.5.6/some_other.file",
				"plugins/owner3/plugin3/v7.8.9/some/deeper.file",
				"plugins/owner4/plugin4/source.yaml", // not in the version dir
				"tests/testdata/buf.build/owner1/plugin1/v1.2.3/Dockerfile",
				"tests/testdata/buf.build/owner2/plugin2/v1.2.3/some/deeper.file",
				"tests/testdata/buf.build/owner3/plugin3/source.yaml", // not in the version dir
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(
				t,
				tc.expected,
				filterPluginPaths(tc.changedFiles, tc.includeTestdata),
			)
		})
	}
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
	filtered, err := FilterByBaseRefDiff(t.Context(), plugins, lookuper)
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
