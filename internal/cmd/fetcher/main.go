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

	"go.uber.org/multierr"
	"golang.org/x/mod/semver"

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
	created, err := run(context.Background(), root)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to fetch versions: %v\n", err)
		os.Exit(1)
	}
	if err := postProcessCreatedPlugins(created); err != nil {
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

func postProcessCreatedPlugins(plugins []createdPlugin) error {
	if len(plugins) == 0 {
		return nil
	}
	for _, plugin := range plugins {
		newPluginRef := fmt.Sprintf("%s/%s:%s", plugin.org, plugin.name, plugin.newVersion)
		if err := runGoModTidy(plugin); err != nil {
			return fmt.Errorf("failed to run go mod tidy for %s: %w", newPluginRef, err)
		}
		if err := recreateNPMPackageLock(plugin); err != nil {
			return fmt.Errorf("failed to recreate package-lock.json for %s: %w", newPluginRef, err)
		}
	}
	if err := runPluginTests(plugins); err != nil {
		return fmt.Errorf("failed to run plugin tests: %w", err)
	}
	return nil
}

// runGoModTidy runs 'go mod tidy' for plugins (like twirp-go) which don't use modules.
// In order to get more reproducible builds, we check in a go.mod/go.sum file.
func runGoModTidy(plugin createdPlugin) error {
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
	goPath, err := exec.LookPath("go")
	if err != nil {
		return err
	}
	log.Printf("running go mod tidy for %s/%s:%s", plugin.org, plugin.name, plugin.newVersion)
	cmd := exec.Cmd{
		Path:   goPath,
		Args:   []string{goPath, "mod", "tidy"},
		Dir:    versionDir,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	return cmd.Run()
}

// recreateNPMPackageLock will remove an existing package-lock.json file and recreate it.
// This will ensure that we correctly resolve any updated versions in package.json.
func recreateNPMPackageLock(plugin createdPlugin) error {
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
	npmPath, err := exec.LookPath("npm")
	if err != nil {
		return err
	}
	log.Printf("recreating package-lock.json for %s/%s:%s", plugin.org, plugin.name, plugin.newVersion)
	cmd := exec.Cmd{
		Path:   npmPath,
		Args:   []string{npmPath, "install"},
		Dir:    versionDir,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	return cmd.Run()
}

// runPluginTests runs 'make test PLUGINS="org/name:v<new>"' in order to generate plugin.sum files.
func runPluginTests(plugins []createdPlugin) error {
	pluginsEnv := make([]string, 0, len(plugins))
	for _, plugin := range plugins {
		pluginsEnv = append(pluginsEnv, fmt.Sprintf("%s/%s:%s", plugin.org, plugin.name, plugin.newVersion))
	}
	makePath, err := exec.LookPath("make")
	if err != nil {
		return err
	}
	env := os.Environ()
	env = append(env, "ALLOW_EMPTY_PLUGIN_SUM=true")
	cmd := exec.Cmd{
		Path: makePath,
		Args: []string{
			makePath,
			"test",
			fmt.Sprintf("PLUGINS=%s", strings.Join(pluginsEnv, ",")),
		},
		Env: env,
	}
	start := time.Now()
	log.Printf("starting running tests for %d plugins", len(plugins))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, out)
	}
	log.Printf("finished running tests in: %.2fs", time.Since(start).Seconds())
	return nil
}

func run(ctx context.Context, root string) ([]createdPlugin, error) {
	now := time.Now()
	defer func() {
		log.Printf("finished running in: %.2fs\n", time.Since(now).Seconds())
	}()
	latestBaseImageVersions, err := getLatestBaseImageVersions(root)
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

func getLatestBaseImageVersions(basedir string) (_ map[string]string, retErr error) {
	// Walk up from plugins dir to find .github dir
	rootDir := basedir
	var githubDir string
	for {
		githubDir = filepath.Join(rootDir, ".github")
		if st, err := os.Stat(githubDir); err == nil && st.IsDir() {
			break
		}
		newRootDir := filepath.Dir(filepath.Dir(githubDir))
		if newRootDir == rootDir {
			return nil, fmt.Errorf("failed to find .github directory from %s", basedir)
		}
		rootDir = newRootDir
	}
	dockerDir := filepath.Join(githubDir, "docker")
	d, err := os.Open(dockerDir)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, d.Close())
	}()
	entries, err := d.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	latestVersions := make(map[string]string, len(entries))
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "Dockerfile.") {
			continue
		}
		imageName, version, err := parseDockerfileBaseImageNameVersion(filepath.Join(dockerDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		latestVersions[imageName] = version
	}
	return latestVersions, nil
}

func parseDockerfileBaseImageNameVersion(dockerfile string) (_ string, _ string, retErr error) {
	f, err := os.Open(dockerfile)
	if err != nil {
		return "", "", nil
	}
	defer func() {
		retErr = errors.Join(retErr, f.Close())
	}()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if !strings.EqualFold(fields[0], "from") {
			continue
		}
		var image string
		for i := 1; i < len(fields); i++ {
			if strings.HasPrefix(fields[i], "--") {
				// Ignore --platform and other args
				continue
			}
			image = fields[i]
			break
		}
		if image == "" {
			return "", "", fmt.Errorf("missing image in FROM: %q", line)
		}
		imageName, version, found := strings.Cut(image, ":")
		if !found {
			return "", "", fmt.Errorf("invalid FROM line: %q", line)
		}
		return imageName, version, nil
	}
	if err := s.Err(); err != nil {
		return "", "", err
	}
	return "", "", fmt.Errorf("failed to detect base image in %s", dockerfile)
}

// copyDirectory copies all files from the source directory to the target,
// creating the target directory if it does not exist.
// If the source directory contains subdirectories this function returns an error.
func copyDirectory(
	source string,
	target string,
	prevVersion string,
	newVersion string,
	latestBaseImageVersions map[string]string,
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
			latestBaseImageVersions,
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
	latestBaseImageVersions map[string]string,
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
		latestBaseImageVersions,
	)
}

func copyFile(
	src string,
	dest string,
	prevVersion string,
	newVersion string,
	latestBaseImageVersions map[string]string,
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
	s := bufio.NewScanner(srcFile)
	for s.Scan() {
		line := strings.ReplaceAll(s.Text(), prevVersion, newVersion)
		line = bazelDownloadRegexp.ReplaceAllString(
			line,
			fmt.Sprintf(`bazelbuild/bazel/releases/download/%[1]s/bazel-%[1]s-linux`,
				latestBaseImageVersions[bazelImageName],
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
			if name, _, found := strings.Cut(image, ":"); found {
				if newVersion := latestBaseImageVersions[name]; newVersion != "" {
					fields[imageIndex] = name + ":" + newVersion
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
