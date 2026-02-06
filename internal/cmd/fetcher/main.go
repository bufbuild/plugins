package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"buf.build/go/interrupt"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/bufbuild/buf/private/pkg/encoding"
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

func main() {
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintf(os.Stderr, "usage: %s <directory>\n", os.Args)
		os.Exit(2)
	}
	root := os.Args[1]
	ctx := interrupt.Handle(context.Background())
	created, err := run(ctx, root)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to fetch versions: %v\n", err)
		os.Exit(1)
	}
	if err := postProcessCreatedPlugins(ctx, created); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to run post-processing on plugins: %v\n", err)
		os.Exit(1)
	}
}

type createdPlugin struct {
	org             string
	name            string
	pluginDir       string
	previousVersion string
	newVersion      string
}

func postProcessCreatedPlugins(ctx context.Context, plugins []createdPlugin) error {
	if len(plugins) == 0 {
		return nil
	}
	for _, plugin := range plugins {
		newPluginRef := fmt.Sprintf("%s/%s:%s", plugin.org, plugin.name, plugin.newVersion)
		if err := runGoModTidy(ctx, plugin); err != nil {
			return fmt.Errorf("failed to run go mod tidy for %s: %w", newPluginRef, err)
		}
		if err := recreateNPMPackageLock(ctx, plugin); err != nil {
			return fmt.Errorf("failed to recreate package-lock.json for %s: %w", newPluginRef, err)
		}
		if err := recreateSwiftPackageResolved(ctx, plugin); err != nil {
			return fmt.Errorf("failed to resolve Swift package for %s: %w", newPluginRef, err)
		}
	}
	if err := runPluginTests(ctx, plugins); err != nil {
		return fmt.Errorf("failed to run plugin tests: %w", err)
	}
	return nil
}

// runGoModTidy runs 'go mod tidy' for plugins (like twirp-go) which don't use modules.
// In order to get more reproducible builds, we check in a go.mod/go.sum file.
func runGoModTidy(ctx context.Context, plugin createdPlugin) error {
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
	log.Printf("running go mod tidy for %s/%s:%s", plugin.org, plugin.name, plugin.newVersion)
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = versionDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// recreateNPMPackageLock will remove an existing package-lock.json file and recreate it.
// This will ensure that we correctly resolve any updated versions in package.json.
func recreateNPMPackageLock(ctx context.Context, plugin createdPlugin) error {
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
	log.Printf("recreating package-lock.json for %s/%s:%s", plugin.org, plugin.name, plugin.newVersion)
	cmd := exec.CommandContext(ctx, "npm", "install")
	cmd.Dir = versionDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// recreateSwiftPackageResolved resolves Swift package dependencies for plugins that use Swift packages.
// It clones the git repository specified in the Dockerfile, runs 'swift package resolve',
// and moves the generated Package.resolved file to the version directory.
func recreateSwiftPackageResolved(ctx context.Context, plugin createdPlugin) (retErr error) {
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
	defer file.Close()

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

	log.Printf("resolving Swift package for %s/%s:%s", plugin.org, plugin.name, plugin.newVersion)

	// Create a tempdir for cloning the repo
	tmpDir, err := os.MkdirTemp("", "swift-repo-*")
	if err != nil {
		return fmt.Errorf("creating tmp dir: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, os.RemoveAll(tmpDir))
	}()

	// Execute the git clone command, cloning to the tmpDir
	cmd := exec.CommandContext(ctx, "sh", "-c", gitCloneCmd, "--", tmpDir)
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
func runPluginTests(ctx context.Context, plugins []createdPlugin) error {
	pluginsEnv := make([]string, 0, len(plugins))
	for _, plugin := range plugins {
		pluginsEnv = append(pluginsEnv, fmt.Sprintf("%s/%s:%s", plugin.org, plugin.name, plugin.newVersion))
	}
	env := os.Environ()
	env = append(env, "ALLOW_EMPTY_PLUGIN_SUM=true")
	start := time.Now()
	log.Printf("starting running tests for %d plugins", len(plugins))
	defer func() {
		log.Printf("finished running tests in: %.2fs", time.Since(start).Seconds())
	}()
	cmd := exec.CommandContext(ctx, "make", "test", fmt.Sprintf("PLUGINS=%s", strings.Join(pluginsEnv, ","))) //nolint:gosec
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// updatePluginDeps updates plugin dependencies in a buf.plugin.yaml file to their latest versions.
// It parses the YAML content, finds any entries in the "deps:" section with "plugin:" fields,
// and updates them to use the latest available version from latestVersions map.
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
func updatePluginDeps(content []byte, latestVersions map[string]string) ([]byte, error) {
	var config bufremotepluginconfig.ExternalConfig
	if err := encoding.UnmarshalJSONOrYAMLStrict(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse buf.plugin.yaml: %w", err)
	}

	// Check if there are any plugin dependencies
	if len(config.Deps) == 0 {
		// No deps, return original content
		return content, nil
	}

	modified := false
	for i := range config.Deps {
		dep := &config.Deps[i]
		if dep.Plugin == "" {
			continue
		}

		// Parse the plugin reference: buf.build/owner/name:version
		pluginName, currentVersion, ok := strings.Cut(dep.Plugin, ":")
		if !ok {
			continue
		}

		// Look up the latest version for this plugin
		if latestVersion, exists := latestVersions[pluginName]; exists && latestVersion != currentVersion {
			oldPluginRef := dep.Plugin
			newPluginRef := pluginName + ":" + latestVersion
			dep.Plugin = newPluginRef
			log.Printf("updating plugin dependency %s -> %s", oldPluginRef, newPluginRef)
			modified = true
		}
	}

	if !modified {
		// No changes made, return original content
		return content, nil
	}

	// Marshal back to YAML
	updatedContent, err := encoding.MarshalYAML(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated YAML: %w", err)
	}

	return updatedContent, nil
}

func run(ctx context.Context, root string) ([]createdPlugin, error) {
	now := time.Now()
	defer func() {
		log.Printf("finished running in: %.2fs\n", time.Since(now).Seconds())
	}()
	baseImageDir, err := docker.FindBaseImageDir(root)
	if err != nil {
		return nil, err
	}
	latestBaseImageVersions, err := docker.LoadLatestBaseImages(baseImageDir)
	if err != nil {
		return nil, err
	}

	// Load all existing plugins to determine latest versions for dependency bumping
	pluginsDir := filepath.Join(root, "plugins")
	allPlugins, err := plugin.FindAll(pluginsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load existing plugins: %w", err)
	}
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
	client := fetchclient.New(ctx)
	latestVersions := make(map[string]string, len(configs))
	created := make([]createdPlugin, 0, len(configs))
	for _, config := range configs {
		if config.Source.Disabled {
			log.Printf("skipping source: %s", config.Filename)
			continue
		}
		newVersion := latestVersions[config.CacheKey()]
		if newVersion == "" {
			newVersion, err = client.Fetch(ctx, config)
			if err != nil {
				if errors.Is(err, fetchclient.ErrSemverPrerelease) {
					log.Printf("skipping source: %s: %v", config.Filename, err)
					continue
				}
				return nil, err
			}
			latestVersions[config.CacheKey()] = newVersion
		}
		// Some plugins share the same source but specify different ignore versions.
		// Ensure we continue to only fetch the latest version once but still respect ignores.
		if slices.Contains(config.Source.IgnoreVersions, newVersion) {
			log.Printf("skipping source: %s: %v", config.Filename, newVersion)
			continue
		}
		// example: plugins/grpc
		pluginDir := filepath.Dir(config.Filename)
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
		if err := createPluginDir(pluginDir, previousVersion, newVersion, latestBaseImageVersions, latestPluginVersions); err != nil {
			return nil, err
		}
		log.Printf("created %v/%v\n", pluginDir, newVersion)
		created = append(created, createdPlugin{
			org:             filepath.Base(filepath.Dir(pluginDir)),
			name:            filepath.Base(pluginDir),
			pluginDir:       pluginDir,
			previousVersion: previousVersion,
			newVersion:      newVersion,
		})
	}
	return created, nil
}

// copyDirectory copies all files from the source directory to the target,
// creating the target directory if it does not exist.
// If the source directory contains subdirectories this function returns an error.
func copyDirectory(
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
		filepath.Join(dir, previousVersion),
		filepath.Join(dir, newVersion),
		previousVersion,
		newVersion,
		latestBaseImages,
		latestPluginVersions,
	)
}

func copyFile(
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
		content, err = updatePluginDeps(content, latestPluginVersions)
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
