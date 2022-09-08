package plugin

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sethvargo/go-envconfig"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

// Plugin represents metadata (and filesystem path) information about a plugin.
type Plugin struct {
	Path        string       `yaml:"-"`
	Relpath     string       `yaml:"-"`
	Name        string       `yaml:"name"`
	Version     string       `yaml:"plugin_version"`
	Deps        []Dependency `yaml:"deps,omitempty"`
	DefaultOpts []string     `yaml:"default_opts,omitempty"`
}

func (p *Plugin) String() string {
	return fmt.Sprintf("%+v", *p)
}

// Dependency represents a dependency one plugin has on another.
type Dependency struct {
	Plugin string `yaml:"plugin"`
}

// Walk loads every buf.plugin.yaml found in the specified root directory and calls the callback function with each plugin.
// The callback is called in dependency order (all plugin dependencies are printed before the plugin).
func Walk(dir string, f func(plugin *Plugin)) error {
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
	sorted, err := sortByDependencyOrder(unsorted, pluginNames)
	if err != nil {
		return err
	}
	for _, p := range sorted {
		f(p)
	}
	return nil
}

// sortByDependencyOrder sorts the passed in plugins such that each dependency comes before a plugin with dependencies.
func sortByDependencyOrder(original []*Plugin, pluginNames map[string]struct{}) ([]*Plugin, error) {
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
				name, _, ok := strings.Cut(dep.Plugin, ":")
				if !ok {
					return nil, fmt.Errorf("invalid plugin dependency: %s", dep)
				}
				if _, ok := pluginNames[name]; !ok {
					// Plugin dependency defined elsewhere - this is ok.
					// We only calculate dependency order for plugins found within the specified path.
					log.Printf("dependency on external plugin - ignoring: %s", name)
					continue
				}
				if _, ok := resolvedMap[dep.Plugin]; !ok {
					foundDeps = false
					break
				}
			}
			if foundDeps {
				resolved = append(resolved, plugin)
				resolvedMap[plugin.Name+":"+plugin.Version] = struct{}{}
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
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var plugin Plugin
	if err := yaml.NewDecoder(f).Decode(&plugin); err != nil {
		return nil, err
	}
	plugin.Path = filepath.ToSlash(path)
	plugin.Relpath, err = filepath.Rel(basedir, path)
	if err != nil {
		return nil, err
	}
	plugin.Relpath = filepath.ToSlash(plugin.Relpath)
	return &plugin, nil
}

// FilterByPluginsEnv returns matching plugins based on a space separated list of plugins (and optional versions) to include.
func FilterByPluginsEnv(plugins []*Plugin, pluginsEnv string) ([]*Plugin, error) {
	if pluginsEnv == "" {
		return plugins, nil
	}
	includes, err := parsePluginsEnvVar(pluginsEnv)
	if err != nil {
		return nil, err
	}
	latestVersionByName := getLatestPluginVersionsByName(plugins)
	var filtered []*Plugin
	for _, plugin := range plugins {
		var matched bool
		for _, include := range includes {
			if matched = include.Matches(plugin, latestVersionByName[plugin.Name]); matched {
				break
			}
		}
		if matched {
			filtered = append(filtered, plugin)
		} else {
			log.Printf("excluding plugin: %s", plugin.Relpath)
		}
	}
	return filtered, nil
}

// FilterByChangedFiles works with https://github.com/tj-actions/changed-files#outputs to filter out unchanged plugins.
// This allows PR builds to only build the plugins which changed (and their base images) instead of all plugins.
func FilterByChangedFiles(plugins []*Plugin, lookuper envconfig.Lookuper) ([]*Plugin, error) {
	var changedFiles changedFiles
	if err := envconfig.ProcessWith(context.Background(), &changedFiles, lookuper); err != nil {
		return nil, err
	}
	// ANY_MODIFIED env var not set - don't filter anything
	if len(changedFiles.AnyModified) == 0 {
		return plugins, nil
	}
	anyModified, err := strconv.ParseBool(changedFiles.AnyModified)
	if err != nil {
		return nil, err
	}
	// None of our included file patterns were changed - filter everything
	if !anyModified {
		return nil, nil
	}
	// Makefile: build all
	// contrib/chrusty/jsonschema/v1.3.9/*: build contrib/chrusty/jsonschema/v1.3.9/buf.plugin.yaml
	// library/connect-go/v0.1.1/*: build library/connect-go/v0.1.1/buf.plugin.yaml
	// ...
	// library/grpc/base-build/*: build library/grpc/**/buf.plugin.yaml
	// library/grpc/v1.48.0/base/*: build library/grpc/v1.48.0/*/buf.plugin.yaml
	// tests/*.go: build all
	// tests/testdata/buf.build/contrib/chrusty-jsonschema/v1.3.9/**: build contrib/chrusty/jsonschema/v1.3.9/buf.plugin.yaml
	// tests/testdata/images/*: build all
	includeAll := false
	for _, modifiedFile := range changedFiles.AllModifiedFiles {
		if modifiedFile == "Makefile" {
			includeAll = true
			break
		}
		if strings.HasPrefix(modifiedFile, "tests/") {
			if strings.HasSuffix(modifiedFile, ".go") || strings.HasSuffix(modifiedFile, ".bin.gz") {
				includeAll = true
				break
			}
		}
	}
	if includeAll {
		return plugins, nil
	}
	var filtered []*Plugin
	for _, plugin := range plugins {
		include := false
		for _, changedFile := range changedFiles.AllModifiedFiles {
			changedDir := filepath.ToSlash(filepath.Dir(changedFile))
			switch filepath.Base(changedDir) {
			case "base", "base-build":
				changedDir = filepath.Dir(changedDir) + "/"
			}
			if strings.HasPrefix(plugin.Relpath, changedDir) {
				include = true
				break
			}
			if strings.HasPrefix(changedDir, "tests/testdata/"+plugin.Name+"/"+plugin.Version+"/") {
				include = true
				break
			}
		}
		if include {
			filtered = append(filtered, plugin)
		} else {
			log.Printf("excluding plugin: %s", plugin.Relpath)
		}
	}
	return filtered, nil
}

// GetDockerfiles returns the relative path of all Dockerfile files found in the specified list of plugins.
func GetDockerfiles(basedir string, plugins []*Plugin) ([]string, error) {
	baseDockerfiles, err := GetBaseDockerfiles(basedir)
	if err != nil {
		return nil, err
	}
	var dockerFiles []string
	for _, plugin := range plugins {
		remaining := make([]string, 0, len(baseDockerfiles))
		for _, baseDockerfile := range baseDockerfiles {
			pluginBasedir := filepath.ToSlash(filepath.Dir(filepath.Dir(baseDockerfile)))
			if strings.HasPrefix(plugin.Relpath, pluginBasedir+"/") {
				dockerFiles = append(dockerFiles, baseDockerfile)
			} else {
				remaining = append(remaining, baseDockerfile)
			}
		}
		baseDockerfiles = remaining
		dockerFiles = append(dockerFiles, filepath.ToSlash(filepath.Join(filepath.Dir(plugin.Relpath), "Dockerfile")))
	}
	return dockerFiles, nil
}

// GetBaseDockerfiles returns the relative paths to base Dockerfiles in the specified base directory.
func GetBaseDockerfiles(basedir string) ([]string, error) {
	var baseDockerFiles []string
	if err := filepath.WalkDir(basedir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "tests", "testdata", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if IsBaseDockerfile(path) {
			relativePath, err := filepath.Rel(basedir, path)
			if err != nil {
				return err
			}
			baseDockerFiles = append(baseDockerFiles, filepath.ToSlash(relativePath))
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return baseDockerFiles, nil
}

// IsBaseDockerfile returns true if the path's base name is 'Dockerfile' and the name of the parent directory begins with "base".
// For example, 'library/protoc/base-build/Dockerfile' and 'library/protoc/v21.3/base/Dockerfile' will return true.
func IsBaseDockerfile(path string) bool {
	if filepath.Base(path) != "Dockerfile" {
		return false
	}
	parentDir := filepath.Base(filepath.Dir(path))
	return strings.HasPrefix(parentDir, "base")
}

func parsePluginsEnvVar(pluginsEnv string) ([]includePlugin, error) {
	var includes []includePlugin
	fields := strings.Fields(pluginsEnv)
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
			includes = append(includes, includePlugin{name: name, version: version})
		} else {
			includes = append(includes, includePlugin{name: name})
		}
	}
	return includes, nil
}

// getLatestPluginVersionsByName returns a map with keys set to plugin.Name and values set to the latest semver version for the plugin.
// For example, if plugins contains buf.build/library/connect-web v0.1.1, v0.2.0, and v0.2.1,
// the returned map will contain: {"buf.build/library/connect-web": "v0.2.1"}.
func getLatestPluginVersionsByName(plugins []*Plugin) map[string]string {
	latestVersions := make(map[string]string)
	for _, plugin := range plugins {
		current := latestVersions[plugin.Name]
		if current == "" || semver.Compare(current, plugin.Version) < 0 {
			latestVersions[plugin.Name] = plugin.Version
		}
	}
	return latestVersions
}

type includePlugin struct {
	name    string
	version string
}

func (p includePlugin) Matches(plugin *Plugin, latestVersion string) bool {
	if !strings.HasSuffix(plugin.Name, "/"+p.name) {
		return false
	}
	if p.version == "" {
		return true
	}
	if p.version == "latest" {
		return plugin.Version == latestVersion
	}
	return p.version == plugin.Version
}

// changedFiles contains data from the tj-actions/changed-files action.
// See https://github.com/tj-actions/changed-files#outputs for more details.
type changedFiles struct {
	AnyModified      string   `env:"ANY_MODIFIED"`
	AllModifiedFiles []string `env:"ALL_MODIFIED_FILES"`
}
