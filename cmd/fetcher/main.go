package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/multierr"
	"golang.org/x/mod/semver"

	"github.com/bufbuild/plugins/internal/fetchclient"
	"github.com/bufbuild/plugins/internal/source"
)

var errNoVersions = errors.New("no versions found")

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
		_, _ = fmt.Fprintf(os.Stderr, "failed to run post-processing on plugins: %v", err)
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
	for _, plugin := range plugins {
		if err := runGoModTidy(plugin); err != nil {
			return err
		}
		if err := recreateNPMPackageLock(plugin); err != nil {
			return err
		}
		if err := runPluginTests(plugin); err != nil {
			return err
		}
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

// runPluginTests runs 'make test PLUGINS="org/name:v<old> org/name:v<new>"' in order to generate plugin.sum files.
// Additionally, it prints out the diff of generated code between the previous and latest plugin.
func runPluginTests(plugin createdPlugin) error {
	basedir := filepath.Dir(filepath.Dir(filepath.Dir(plugin.pluginDir)))
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
			fmt.Sprintf("PLUGINS=%[1]s/%[2]s:%[3]s %[1]s/%[2]s:%[4]s", plugin.org, plugin.name, plugin.previousVersion, plugin.newVersion),
		},
		Dir:    basedir,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Env:    env,
	}
	log.Printf("running tests for %[1]s/%[2]s:%[3]s and %[1]s/%[2]s:%[4]s", plugin.org, plugin.name, plugin.previousVersion, plugin.newVersion)
	if err := cmd.Run(); err != nil {
		return err
	}
	diffPath, err := exec.LookPath("diff")
	if err != nil {
		return err
	}
	log.Printf("diff between generated code for %s/%s (%s -> %s)", plugin.org, plugin.name, plugin.previousVersion, plugin.newVersion)
	diffCmd := exec.Cmd{
		Path: diffPath,
		Dir:  filepath.Join(basedir, "tests", "testdata", "buf.build", plugin.org, plugin.name),
		Args: []string{
			diffPath,
			"--exclude",
			"plugin.sum",
			"--exclude",
			"protoc-gen-plugin",
			"--recursive",
			"--unified",
			plugin.previousVersion,
			plugin.newVersion,
		},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	if err := diffCmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// This is expected if there are differences
			if exitErr.ExitCode() == 1 {
				return nil
			}
		}
		return err
	}
	return nil
}

func run(ctx context.Context, root string) ([]createdPlugin, error) {
	now := time.Now()
	defer func() {
		log.Printf("finished running in: %.2fs\n", time.Since(now).Seconds())
	}()
	configs, err := source.GatherConfigs(root)
	if err != nil {
		return nil, err
	}
	client := fetchclient.New(ctx)
	latestVersions := make(map[string]string, len(configs))
	created := make([]createdPlugin, 0, len(configs))
	for _, config := range configs {
		if config.Source.Disabled {
			log.Printf("skipping source: %s\n", config.Filename)
			continue
		}
		newVersion := latestVersions[config.CacheKey()]
		if newVersion == "" {
			newVersion, err = client.Fetch(ctx, config)
			if err != nil {
				return nil, err
			}
			latestVersions[config.CacheKey()] = newVersion
		}
		// example: library/grpc
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
		if err := createPluginDir(pluginDir, previousVersion, newVersion); err != nil {
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
func copyDirectory(source, target, prevVersion, newVersion string) (retErr error) {
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
		); err != nil {
			return err
		}
	}
	return nil
}

func createPluginDir(dir string, previousVersion string, newVersion string) (retErr error) {
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
	)
}

func copyFile(src, dest string, prevVersion, newVersion string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	replaced := strings.ReplaceAll(string(data), strings.TrimPrefix(prevVersion, "v"), strings.TrimPrefix(newVersion, "v"))
	return os.WriteFile(dest, []byte(replaced), 0644) //nolint:gosec
}

func getLatestVersionFromDir(dir string) (string, error) {
	dirs, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var versions []string
	for _, dir := range dirs {
		if dir.IsDir() && semver.IsValid(dir.Name()) {
			versions = append(versions, dir.Name())
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
