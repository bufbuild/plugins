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
	"regexp"
	"strings"
	"time"

	"github.com/bufbuild/buf/private/pkg/interrupt"
	"go.uber.org/multierr"
	"golang.org/x/mod/semver"

	"github.com/bufbuild/plugins/internal/docker"
	"github.com/bufbuild/plugins/internal/fetchclient"
	"github.com/bufbuild/plugins/internal/source"
)

var (
	bazelDownloadRegexp = regexp.MustCompile(`bazelbuild/bazel/releases/download/[^/]+/bazel-[^-]+-linux`)
	bazelImageName      = "gcr.io/bazel-public/bazel"
	errNoVersions       = errors.New("no versions found")
)

func main() {
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintf(os.Stderr, "usage: %s <directory>\n", os.Args)
		os.Exit(2)
	}
	root := os.Args[1]
	ctx, _ := interrupt.WithCancel(context.Background())
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
		if err := createPluginDir(pluginDir, previousVersion, newVersion, latestBaseImageVersions); err != nil {
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
			retErr = multierr.Append(retErr, os.RemoveAll(target))
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
) (retErr error) {
	if err := os.Mkdir(filepath.Join(dir, newVersion), 0755); err != nil {
		return err
	}
	defer func() {
		if retErr != nil {
			retErr = multierr.Append(retErr, os.RemoveAll(filepath.Join(dir, newVersion)))
		}
	}()
	return copyDirectory(
		filepath.Join(dir, previousVersion),
		filepath.Join(dir, newVersion),
		previousVersion,
		newVersion,
		latestBaseImages,
	)
}

func copyFile(
	src string,
	dest string,
	prevVersion string,
	newVersion string,
	latestBaseImages *docker.BaseImages,
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
	case "Dockerfile", "Dockerfile.wasm", "buf.plugin.yaml", "package.json":
		// We want to update these with the new version
	default:
		// Everything else just copy as-is
		if _, err := io.Copy(destFile, srcFile); err != nil {
			return err
		}
		return nil
	}
	isDockerfile := strings.HasPrefix(filename, "Dockerfile")
	prevVersion = strings.TrimPrefix(prevVersion, "v")
	newVersion = strings.TrimPrefix(newVersion, "v")
	latestBazelVersion := latestBaseImages.ImageVersion(bazelImageName)
	if latestBazelVersion == "" {
		return fmt.Errorf("failed to find latest version for bazel image %q", bazelImageName)
	}
	s := bufio.NewScanner(srcFile)
	for s.Scan() {
		line := strings.ReplaceAll(s.Text(), prevVersion, newVersion)
		line = bazelDownloadRegexp.ReplaceAllString(
			line,
			fmt.Sprintf(`bazelbuild/bazel/releases/download/%[1]s/bazel-%[1]s-linux`,
				latestBazelVersion,
			),
		)
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
