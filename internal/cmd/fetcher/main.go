package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/spf13/pflag"
	"golang.org/x/mod/semver"

	"github.com/bufbuild/plugins/internal/docker"
	"github.com/bufbuild/plugins/internal/fetchclient"
	"github.com/bufbuild/plugins/internal/plugin"
	"github.com/bufbuild/plugins/internal/source"
)

var (
	dockerfileImageName    = "docker/dockerfile"
	dockerfileSyntaxPrefix = "# syntax=docker/dockerfile:"
	errNoVersions          = errors.New("no versions found")
)

type flags struct {
	include []string
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringArrayVar(
		&f.include,
		"include",
		nil,
		`Only fetch plugins matching these patterns (org or org/name). May be specified multiple times.`,
	)
}

type pluginFilter struct {
	orgs    map[string]struct{}
	plugins map[string]struct{}
}

func newPluginFilter(includes []string) *pluginFilter {
	if len(includes) == 0 {
		return nil
	}
	f := &pluginFilter{
		orgs:    make(map[string]struct{}),
		plugins: make(map[string]struct{}),
	}
	for _, pattern := range includes {
		if strings.Contains(pattern, "/") {
			f.plugins[pattern] = struct{}{}
		} else {
			f.orgs[pattern] = struct{}{}
		}
	}
	return f
}

func (f *pluginFilter) includes(org, name string) bool {
	if f == nil {
		return true
	}
	if _, ok := f.orgs[org]; ok {
		return true
	}
	_, ok := f.plugins[org+"/"+name]
	return ok
}

// Fetcher is an interface for fetching plugin versions from external sources.
type Fetcher interface {
	Fetch(ctx context.Context, config *source.Config) (string, error)
}

func main() {
	appcmd.Main(context.Background(), newRootCommand("fetcher"))
}

func newRootCommand(name string) *appcmd.Command {
	builder := appext.NewBuilder(name)
	f := &flags{}
	return &appcmd.Command{
		Use:   name + " [directory]",
		Short: "Fetches latest plugin versions from external sources.",
		Args:  appcmd.MaximumNArgs(1),
		Run: builder.NewRunFunc(func(ctx context.Context, container appext.Container) error {
			client := fetchclient.New(ctx)
			created, err := run(ctx, container, client, f)
			if err != nil {
				return fmt.Errorf("failed to fetch versions: %w", err)
			}
			if err := postProcessCreatedPlugins(ctx, container.Logger(), created); err != nil {
				return fmt.Errorf("failed to run post-processing on plugins: %w", err)
			}
			return nil
		}),
		BindFlags:           f.Bind,
		BindPersistentFlags: builder.BindRoot,
	}
}

type createdPlugin struct {
	org             string
	name            string
	pluginDir       string
	previousVersion string
	newVersion      string
}

func (p createdPlugin) String() string {
	return fmt.Sprintf("%s/%s:%s", p.org, p.name, p.newVersion)
}

func postProcessCreatedPlugins(ctx context.Context, logger *slog.Logger, plugins []createdPlugin) error {
	if len(plugins) == 0 {
		return nil
	}
	for _, plugin := range plugins {
		newPluginRef := plugin.String()
		if err := runGoModTidy(ctx, logger, plugin); err != nil {
			return fmt.Errorf("failed to run go mod tidy for %s: %w", newPluginRef, err)
		}
		if err := recreateNPMPackageLock(ctx, logger, plugin); err != nil {
			return fmt.Errorf("failed to recreate package-lock.json for %s: %w", newPluginRef, err)
		}
		if err := recreateSwiftPackageResolved(ctx, logger, plugin); err != nil {
			return fmt.Errorf("failed to resolve Swift package for %s: %w", newPluginRef, err)
		}
	}
	if err := runPluginTests(ctx, logger, plugins); err != nil {
		return fmt.Errorf("failed to run plugin tests: %w", err)
	}
	return nil
}

// runGoModTidy runs 'go mod tidy' for plugins (like twirp-go) which don't use modules.
// In order to get more reproducible builds, we check in a go.mod/go.sum file.
func runGoModTidy(ctx context.Context, logger *slog.Logger, plugin createdPlugin) error {
	versionDir := filepath.Join(plugin.pluginDir, plugin.newVersion)
	goMod := filepath.Join(versionDir, "go.mod")
	_, err := os.Stat(goMod)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		// no go.mod/go.sum to update
		return nil
	}
	logger.InfoContext(ctx, "running go mod tidy", slog.Any("plugin", plugin))
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = versionDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// recreateNPMPackageLock will remove an existing package-lock.json file and recreate it.
// This will ensure that we correctly resolve any updated versions in package.json.
func recreateNPMPackageLock(ctx context.Context, logger *slog.Logger, plugin createdPlugin) error {
	versionDir := filepath.Join(plugin.pluginDir, plugin.newVersion)
	npmPackageLock := filepath.Join(versionDir, "package-lock.json")
	_, err := os.Stat(npmPackageLock)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		// no package-lock to update
		return nil
	}
	if err := os.Remove(npmPackageLock); err != nil {
		return err
	}
	logger.InfoContext(ctx, "recreating package-lock.json", slog.Any("plugin", plugin))
	cmd := exec.CommandContext(ctx, "npm", "install")
	cmd.Dir = versionDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// recreateSwiftPackageResolved resolves Swift package dependencies for plugins that use Swift packages.
// It clones the git repository specified in the Dockerfile, runs 'swift package resolve',
// and moves the generated Package.resolved file to the version directory.
func recreateSwiftPackageResolved(ctx context.Context, logger *slog.Logger, plugin createdPlugin) (retErr error) {
	versionDir := filepath.Join(plugin.pluginDir, plugin.newVersion)
	packageResolved := filepath.Join(versionDir, "Package.resolved")
	_, err := os.Stat(packageResolved)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		// no Package.resolved to update
		return nil
	}

	// Read the Dockerfile to find the git clone command
	dockerfile := filepath.Join(versionDir, "Dockerfile")
	file, err := os.Open(dockerfile)
	if err != nil {
		return fmt.Errorf("failed to open Dockerfile: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, file.Close())
	}()

	var gitCloneCmd string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "RUN git clone") {
			// Strip the "RUN " prefix
			gitCloneCmd = strings.TrimSpace(strings.TrimPrefix(line, "RUN "))
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read Dockerfile: %w", err)
	}
	if gitCloneCmd == "" {
		return errors.New("no 'RUN git clone' command found in Dockerfile")
	}

	logger.InfoContext(ctx, "resolving Swift package", slog.Any("plugin", plugin))

	// Create a tempdir for cloning the repo
	tmpDir, err := os.MkdirTemp("", "swift-repo-*")
	if err != nil {
		return fmt.Errorf("creating tmp dir: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, os.RemoveAll(tmpDir))
	}()

	// Execute the git clone command, cloning to the tmpDir
	cmd := exec.CommandContext(ctx, "sh", "-c", gitCloneCmd+" -- "+tmpDir) //nolint:gosec // We control the arguments here.
	cmd.Dir = versionDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run git clone: %w", err)
	}

	// Run `swift package resolve` in the cloned directory
	cmd = exec.CommandContext(ctx, "swift", "package", "resolve")
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run swift package resolve: %w", err)
	}

	// Move the Package.resolved file from the cloned directory to the version directory
	src := filepath.Join(tmpDir, "Package.resolved")
	dest := packageResolved
	if err := os.Rename(src, dest); err != nil {
		return fmt.Errorf("failed to move Package.resolved: %w", err)
	}

	return nil
}

// runPluginTests runs 'make test PLUGINS="org/name:v<new>"' in order to generate plugin.sum files.
func runPluginTests(ctx context.Context, logger *slog.Logger, plugins []createdPlugin) error {
	pluginsEnv := make([]string, 0, len(plugins))
	for _, plugin := range plugins {
		pluginsEnv = append(pluginsEnv, plugin.String())
	}
	env := os.Environ()
	env = append(env, "ALLOW_EMPTY_PLUGIN_SUM=true")
	start := time.Now()
	logger.InfoContext(ctx, "starting running tests", slog.Int("num_plugins", len(plugins)))
	defer func() {
		logger.InfoContext(ctx, "finished running tests", slog.Duration("duration", time.Since(start)))
	}()
	cmd := exec.CommandContext(ctx, "make", "test", fmt.Sprintf("PLUGINS=%s", strings.Join(pluginsEnv, ","))) //nolint:gosec
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// updatePluginDeps updates plugin dependencies in a buf.plugin.yaml file to their latest versions.
// It parses the YAML content to find deps entries, then uses text replacement to update
// version references in-place, preserving the original formatting and comments.
// For example, if the YAML contains:
//
//	deps:
//	  - plugin: buf.build/protocolbuffers/go:v1.30.0
//
// and latestVersions maps "buf.build/protocolbuffers/go" to "v1.36.11",
// the function will update it to:
//
//	deps:
//	  - plugin: buf.build/protocolbuffers/go:v1.36.11
//
// It returns the modified content with updated dependency versions.
func updatePluginDeps(ctx context.Context, logger *slog.Logger, content []byte, latestVersions map[string]string) ([]byte, error) {
	var config bufremotepluginconfig.ExternalConfig
	if err := encoding.UnmarshalJSONOrYAMLStrict(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse buf.plugin.yaml: %w", err)
	}

	// Check if there are any plugin dependencies
	if len(config.Deps) == 0 {
		// No deps, return original content
		return content, nil
	}

	// Use text replacement rather than re-marshaling the struct to avoid introducing
	// empty fields from zero-value nested structs in ExternalConfig.
	result := string(content)
	for _, dep := range config.Deps {
		if dep.Plugin == "" {
			continue
		}

		// Parse the plugin reference: buf.build/owner/name:version
		pluginName, currentVersion, ok := strings.Cut(dep.Plugin, ":")
		if !ok {
			continue
		}

		// Look up the latest version for this plugin
		latestVersion, exists := latestVersions[pluginName]
		if !exists || latestVersion == currentVersion {
			continue
		}

		oldPluginRef := dep.Plugin
		newPluginRef := pluginName + ":" + latestVersion
		logger.InfoContext(ctx, "updating plugin dependency", slog.String("old", oldPluginRef), slog.String("new", newPluginRef))
		result = strings.ReplaceAll(result, oldPluginRef, newPluginRef)
	}

	return []byte(result), nil
}

// pluginToCreate represents a plugin that needs a new version created.
type pluginToCreate struct {
	pluginDir       string
	previousVersion string
	newVersion      string
}

func run(ctx context.Context, container appext.Container, fetcher Fetcher, f *flags) ([]createdPlugin, error) {
	var root string
	if container.NumArgs() > 0 {
		root = container.Arg(0)
	} else {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	logger := container.Logger()
	now := time.Now()
	defer func() {
		logger.InfoContext(ctx, "finished running", slog.Duration("duration", time.Since(now)))
	}()
	baseImageDir, err := docker.FindBaseImageDir(root)
	if err != nil {
		return nil, err
	}
	latestBaseImageVersions, err := docker.LoadLatestBaseImages(baseImageDir)
	if err != nil {
		return nil, err
	}

	// Load all existing plugins (already sorted in dependency order by plugin.FindAll)
	pluginsDir := filepath.Join(root, "plugins")
	allPlugins, err := plugin.FindAll(pluginsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load existing plugins: %w", err)
	}

	// Build initial map of latest plugin versions
	latestPluginVersions := make(map[string]string)
	for _, p := range allPlugins {
		current := latestPluginVersions[p.Name]
		if current == "" || semver.Compare(current, p.PluginVersion) < 0 {
			latestPluginVersions[p.Name] = p.PluginVersion
		}
	}

	configs, err := source.GatherConfigs(root)
	if err != nil {
		return nil, err
	}

	// First pass: fetch all new versions and determine which plugins need updates
	filter := newPluginFilter(f.include)
	latestVersions := make(map[string]string, len(configs))
	pendingCreations := make(map[string]*pluginToCreate) // keyed by plugin directory

	for _, config := range configs {
		if config.Source.Disabled {
			logger.InfoContext(ctx, "skipping source", slog.String("filename", config.Filename))
			continue
		}
		configDir := filepath.Dir(config.Filename)
		pluginName := filepath.Base(configDir)
		pluginOrg := filepath.Base(filepath.Dir(configDir))
		if !filter.includes(pluginOrg, pluginName) {
			logger.DebugContext(ctx, "skipping source (not in --include list)", slog.String("filename", config.Filename))
			continue
		}
		newVersion := latestVersions[config.CacheKey()]
		if newVersion == "" {
			newVersion, err = fetcher.Fetch(ctx, config)
			if err != nil {
				if errors.Is(err, fetchclient.ErrSemverPrerelease) {
					logger.InfoContext(ctx, "skipping source", slog.String("filename", config.Filename), slog.Any("error", err))
					continue
				}
				return nil, err
			}
			latestVersions[config.CacheKey()] = newVersion
		}
		// Some plugins share the same source but specify different ignore versions.
		// Ensure we continue to only fetch the latest version once but still respect ignores.
		if slices.Contains(config.Source.IgnoreVersions, newVersion) {
			logger.InfoContext(ctx, "skipping source", slog.String("filename", config.Filename), slog.String("version", newVersion))
			continue
		}
		// Convert to absolute path to match plugin.Walk behavior (which converts paths via filepath.Abs)
		pluginDir, err := filepath.Abs(filepath.Dir(config.Filename))
		if err != nil {
			return nil, err
		}
		ok, err := checkDirExists(filepath.Join(pluginDir, newVersion))
		if err != nil {
			return nil, err
		}
		if ok {
			continue
		}
		previousVersion, err := getLatestVersionFromDir(pluginDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest known version from dir %s with error: %w", pluginDir, err)
		}

		pendingCreations[pluginDir] = &pluginToCreate{
			pluginDir:       pluginDir,
			previousVersion: previousVersion,
			newVersion:      newVersion,
		}
	}

	// Second pass: create plugins in dependency order (using the order from allPlugins)
	// Update latestPluginVersions as we go so subsequent plugins reference new versions
	created := make([]createdPlugin, 0, len(pendingCreations))
	processedDirs := make(map[string]bool, len(pendingCreations))
	for _, p := range allPlugins {
		// Extract the plugin directory from the plugin's path
		// p.Path is the full path to buf.plugin.yaml, directory is two levels up (dir/version/buf.plugin.yaml)
		// Convert to absolute to match the keys in pendingCreations
		pluginDir, err := filepath.Abs(filepath.Dir(filepath.Dir(p.Path)))
		if err != nil {
			return nil, err
		}

		// Skip if we've already processed this plugin directory (multiple versions of same plugin)
		if processedDirs[pluginDir] {
			continue
		}

		pending, needsCreation := pendingCreations[pluginDir]
		if !needsCreation {
			continue
		}

		if err := createPluginDir(ctx, logger, pending.pluginDir, pending.previousVersion, pending.newVersion, latestBaseImageVersions, latestPluginVersions); err != nil {
			return nil, err
		}
		logger.InfoContext(ctx, "created", slog.String("path", fmt.Sprintf("%v/%v", pending.pluginDir, pending.newVersion)))

		// Mark this directory as processed
		processedDirs[pluginDir] = true

		// Update latestPluginVersions so subsequent plugins in this run can reference this new version
		latestPluginVersions[p.Name] = pending.newVersion

		created = append(created, createdPlugin{
			org:             filepath.Base(filepath.Dir(pending.pluginDir)),
			name:            filepath.Base(pending.pluginDir),
			pluginDir:       pending.pluginDir,
			previousVersion: pending.previousVersion,
			newVersion:      pending.newVersion,
		})
	}
	return created, nil
}

// copyDirectory copies all files from the source directory to the target,
// creating the target directory if it does not exist.
// If the source directory contains subdirectories this function returns an error.
func copyDirectory(
	ctx context.Context,
	logger *slog.Logger,
	source string,
	target string,
	prevVersion string,
	newVersion string,
	latestBaseImages *docker.BaseImages,
	latestPluginVersions map[string]string,
) (retErr error) {
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, os.RemoveAll(target))
		}
	}()
	for _, file := range entries {
		if file.IsDir() {
			return fmt.Errorf("failed to copy directory. Expecting files only: %s", source)
		}
		if err := copyFile(
			ctx,
			logger,
			filepath.Join(source, file.Name()),
			filepath.Join(target, file.Name()),
			prevVersion,
			newVersion,
			latestBaseImages,
			latestPluginVersions,
		); err != nil {
			return err
		}
	}
	return nil
}

func createPluginDir(
	ctx context.Context,
	logger *slog.Logger,
	dir string,
	previousVersion string,
	newVersion string,
	latestBaseImages *docker.BaseImages,
	latestPluginVersions map[string]string,
) (retErr error) {
	if err := os.Mkdir(filepath.Join(dir, newVersion), 0755); err != nil {
		return err
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, os.RemoveAll(filepath.Join(dir, newVersion)))
		}
	}()
	return copyDirectory(
		ctx,
		logger,
		filepath.Join(dir, previousVersion),
		filepath.Join(dir, newVersion),
		previousVersion,
		newVersion,
		latestBaseImages,
		latestPluginVersions,
	)
}

func copyFile(
	ctx context.Context,
	logger *slog.Logger,
	src string,
	dest string,
	prevVersion string,
	newVersion string,
	latestBaseImages *docker.BaseImages,
	latestPluginVersions map[string]string,
) (retErr error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		retErr = errors.Join(retErr, srcFile.Close())
	}()
	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		retErr = errors.Join(retErr, destFile.Close())
	}()
	filename := filepath.Base(dest)
	switch filename {
	case "Dockerfile", "Dockerfile.wasm", "buf.plugin.yaml", "build.csproj", "package.json", "requirements.txt":
		// We want to update these with the new version
	default:
		// Everything else just copy as-is
		if _, err := io.Copy(destFile, srcFile); err != nil {
			return err
		}
		return nil
	}

	// Special handling for buf.plugin.yaml to update plugin dependencies
	if filename == "buf.plugin.yaml" {
		content, err := io.ReadAll(srcFile)
		if err != nil {
			return fmt.Errorf("failed to read buf.plugin.yaml: %w", err)
		}
		// Update plugin dependencies to latest versions
		content, err = updatePluginDeps(ctx, logger, content, latestPluginVersions)
		if err != nil {
			return fmt.Errorf("failed to update plugin deps: %w", err)
		}
		// Now do the version string replacement
		prevVersionStripped := strings.TrimPrefix(prevVersion, "v")
		newVersionStripped := strings.TrimPrefix(newVersion, "v")
		content = []byte(strings.ReplaceAll(string(content), prevVersionStripped, newVersionStripped))
		if _, err := destFile.Write(content); err != nil {
			return err
		}
		return nil
	}

	isDockerfile := strings.HasPrefix(filename, "Dockerfile")
	prevVersion = strings.TrimPrefix(prevVersion, "v")
	newVersion = strings.TrimPrefix(newVersion, "v")
	latestDockerfileVersion := latestBaseImages.ImageVersion(dockerfileImageName)
	if latestDockerfileVersion == "" {
		return fmt.Errorf("failed to find latest version for dockerfile image %q", dockerfileImageName)
	}
	s := bufio.NewScanner(srcFile)
	for s.Scan() {
		line := strings.ReplaceAll(s.Text(), prevVersion, newVersion)
		if isDockerfile && len(line) > 5 && strings.EqualFold(line[0:5], "from ") {
			// Replace FROM line with the latest base image (if found)
			fields := strings.Fields(line)
			var imageIndex int
			var image string
			for i := 1; i <= len(fields); i++ {
				field := fields[i]
				if !strings.HasPrefix(field, "--") {
					image, imageIndex = field, i
					break
				}
			}
			name, _, _ := strings.Cut(image, ":")
			if name != "" {
				if newImageNameAndVersion := latestBaseImages.ImageNameAndVersion(name); newImageNameAndVersion != "" {
					fields[imageIndex] = newImageNameAndVersion
					line = strings.Join(fields, " ")
				}
			}
		}
		if isDockerfile && strings.HasPrefix(line, dockerfileSyntaxPrefix) {
			line = dockerfileSyntaxPrefix + latestDockerfileVersion
		}
		if _, err := fmt.Fprintln(destFile, line); err != nil {
			return err
		}
	}
	return s.Err()
}

func getLatestVersionFromDir(basedir string) (string, error) {
	entries, err := os.ReadDir(basedir)
	if err != nil {
		return "", err
	}
	var versions []string
	for _, entry := range entries {
		if entry.IsDir() && semver.IsValid(entry.Name()) {
			versions = append(versions, entry.Name())
		}
	}
	if len(versions) == 0 {
		return "", errNoVersions
	}
	semver.Sort(versions)
	return versions[len(versions)-1], nil
}

func checkDirExists(dir string) (bool, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("expecting directory: %q", dir)
	}
	return true, nil
}
