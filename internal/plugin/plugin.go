package plugin

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginconfig"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/sethvargo/go-envconfig"
	"golang.org/x/mod/semver"
)

// Plugin represents metadata (and filesystem path) information about a plugin.
type Plugin struct {
	Path    string `yaml:"-"`
	Relpath string `yaml:"-"`
	// Parsed external yaml config
	bufpluginconfig.ExternalConfig `yaml:"-"`
}

func (p *Plugin) String() string {
	return fmt.Sprintf("%+v", *p)
}

// Dependency represents a dependency one plugin has on another.
type Dependency struct {
	Plugin string `yaml:"plugin"`
}

// FindAll returns every plugin found in the specified root directory.
func FindAll(dir string) ([]*Plugin, error) {
	var plugins []*Plugin
	if err := Walk(dir, func(plugin *Plugin) error {
		plugins = append(plugins, plugin)
		return nil
	}); err != nil {
		return nil, err
	}
	return plugins, nil
}

// Walk loads every buf.plugin.yaml found in the specified root directory and calls the callback function with each plugin.
// The callback is called in dependency order (all plugin dependencies are printed before the plugin).
func Walk(dir string, f func(plugin *Plugin) error) error {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	var unsorted []*Plugin
	pluginNames := make(map[string]struct{}, 0)
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != "." && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			switch d.Name() {
			case "testdata", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "buf.plugin.yaml" {
			plugin, err := Load(path, dir)
			if err != nil {
				return err
			}
			unsorted = append(unsorted, plugin)
			pluginNames[plugin.Name] = struct{}{}
		}
		return nil
	}); err != nil {
		return err
	}
	sort.SliceStable(unsorted, func(i, j int) bool {
		p1, p2 := unsorted[i], unsorted[j]
		if p1.Name != p2.Name {
			return p1.Name < p2.Name
		}
		return semver.Compare(p1.PluginVersion, p2.PluginVersion) < 0
	})
	sorted, err := sortByDependencyOrder(unsorted)
	if err != nil {
		return err
	}
	for _, p := range sorted {
		if err := f(p); err != nil {
			return err
		}
	}
	return nil
}

// sortByDependencyOrder sorts the passed in plugins such that each dependency comes before a plugin with dependencies.
func sortByDependencyOrder(original []*Plugin) ([]*Plugin, error) {
	// Make a defensive copy of the original list
	plugins := make([]*Plugin, len(original))
	copy(plugins, original)
	resolved := make([]*Plugin, 0, len(plugins))
	resolvedMap := make(map[string]struct{}, len(plugins))
	for len(plugins) > 0 {
		var unresolved []*Plugin
		for _, plugin := range plugins {
			foundDeps := true
			for _, dep := range plugin.Deps {
				_, _, ok := strings.Cut(dep.Plugin, ":")
				if !ok {
					return nil, fmt.Errorf("invalid plugin dependency: %s", dep.Plugin)
				}
				if _, ok := resolvedMap[dep.Plugin]; !ok {
					foundDeps = false
					break
				}
			}
			if foundDeps {
				resolved = append(resolved, plugin)
				resolvedMap[plugin.Name+":"+plugin.PluginVersion] = struct{}{}
			} else {
				unresolved = append(unresolved, plugin)
			}
		}
		// We either have a cycle or a bug in dependency calculation
		if len(unresolved) == len(plugins) {
			return nil, fmt.Errorf("failed to resolve dependencies: %v", unresolved)
		}
		plugins = unresolved
	}
	return resolved, nil
}

// Load loads the buf.plugin.yaml at the specified path and returns a structure containing metadata for the plugin.
func Load(path string, basedir string) (*Plugin, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var plugin Plugin
	plugin.Path = filepath.ToSlash(path)
	plugin.Relpath, err = filepath.Rel(basedir, path)
	if err != nil {
		return nil, err
	}
	plugin.Relpath = filepath.ToSlash(plugin.Relpath)
	if err := encoding.UnmarshalJSONOrYAMLStrict(contents, &plugin.ExternalConfig); err != nil {
		return nil, err
	}
	return &plugin, nil
}

// FilterByPluginsEnv returns matching plugins based on a space separated list of plugins (and optional versions) to include.
func FilterByPluginsEnv(plugins []*Plugin, pluginsEnv string) ([]*Plugin, error) {
	if pluginsEnv == "" {
		return nil, nil
	}
	if strings.EqualFold(pluginsEnv, "all") {
		return plugins, nil
	}
	includes, err := ParsePluginsEnvVar(pluginsEnv)
	if err != nil {
		return nil, err
	}
	latestVersionByName := getLatestPluginVersionsByName(plugins)
	var filtered []*Plugin
	for _, plugin := range plugins {
		var matched bool
		for _, include := range includes {
			if matched = include.Matches(plugin.Name, plugin.PluginVersion, latestVersionByName[plugin.Name]); matched {
				break
			}
		}
		if matched {
			log.Printf("including plugin: %s", plugin.Relpath)
			filtered = append(filtered, plugin)
		}
	}
	return filtered, nil
}

// FilterByChangedFiles works with https://github.com/tj-actions/changed-files#outputs to filter out unchanged plugins.
// This allows PR builds to only build the plugins which changed instead of all plugins.
func FilterByChangedFiles(plugins []*Plugin, lookuper envconfig.Lookuper) ([]*Plugin, error) {
	var changedFiles changedFiles
	if err := envconfig.ProcessWith(context.Background(), &changedFiles, lookuper); err != nil {
		return nil, err
	}
	// ANY_MODIFIED env var not set - filter everything
	if len(changedFiles.AnyModified) == 0 {
		return nil, nil
	}
	anyModified, err := strconv.ParseBool(changedFiles.AnyModified)
	if err != nil {
		return nil, err
	}
	// None of our included file patterns were changed - filter everything
	if !anyModified {
		return nil, nil
	}
	// plugins/community/chrusty-jsonschema/v1.3.9/*: build plugins/community/chrusty-jsonschema/v1.3.9/buf.plugin.yaml
	// plugins/bufbuild/connect-go/v0.1.1/*: build plugins/bufbuild/connect-go/v0.1.1/buf.plugin.yaml
	var filtered []*Plugin
	for _, plugin := range plugins {
		include := false
		for _, changedFile := range changedFiles.AllModifiedFiles {
			changedDir := filepath.ToSlash(filepath.Dir(changedFile))
			if strings.HasPrefix(plugin.Relpath, changedDir) {
				include = true
				break
			}
			if strings.HasPrefix(changedDir, "tests/testdata/"+plugin.Name+"/"+plugin.PluginVersion+"/") {
				include = true
				break
			}
		}
		if include {
			log.Printf("including plugin: %s", plugin.Relpath)
			filtered = append(filtered, plugin)
		}
	}
	return filtered, nil
}

func ParsePluginsEnvVar(pluginsEnv string) ([]IncludePlugin, error) {
	var includes []IncludePlugin
	fields := strings.FieldsFunc(pluginsEnv, func(r rune) bool {
		return unicode.IsSpace(r) || r == ','
	})
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		name, version, ok := strings.Cut(field, ":")
		if ok { // Specified a version
			if !semver.IsValid(version) && version != "latest" {
				return nil, fmt.Errorf("invalid version: %s", version)
			}
			includes = append(includes, IncludePlugin{name: name, version: version})
		} else {
			includes = append(includes, IncludePlugin{name: name})
		}
	}
	return includes, nil
}

// getLatestPluginVersionsByName returns a map with keys set to plugin.Name and values set to the latest semver version for the plugin.
// For example, if plugins contains buf.build/bufbuild/connect-web v0.1.1, v0.2.0, and v0.2.1,
// the returned map will contain: {"buf.build/bufbuild/connect-web": "v0.2.1"}.
func getLatestPluginVersionsByName(plugins []*Plugin) map[string]string {
	latestVersions := make(map[string]string)
	for _, plugin := range plugins {
		current := latestVersions[plugin.Name]
		if current == "" || semver.Compare(current, plugin.PluginVersion) < 0 {
			latestVersions[plugin.Name] = plugin.PluginVersion
		}
	}
	return latestVersions
}

type IncludePlugin struct {
	name    string
	version string
}

func (p IncludePlugin) Matches(pluginName, pluginVersion, latestVersion string) bool {
	if !strings.HasSuffix(pluginName, "/"+p.name) {
		return false
	}
	if p.version == "" {
		return true
	}
	if p.version == "latest" {
		return pluginVersion == latestVersion
	}
	return p.version == pluginVersion
}

// changedFiles contains data from the tj-actions/changed-files action.
// See https://github.com/tj-actions/changed-files#outputs for more details.
type changedFiles struct {
	AnyModified      string   `env:"ANY_MODIFIED"`
	AllModifiedFiles []string `env:"ALL_MODIFIED_FILES"`
}
