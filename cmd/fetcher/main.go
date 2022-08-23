package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginconfig"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/plugins/internal/fetchclient"
	"github.com/bufbuild/plugins/internal/source"
	"go.uber.org/multierr"
	"golang.org/x/mod/semver"
)

var errNoVersions = errors.New("no versions found")

func main() {
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintf(os.Stderr, "usage: %s <directory> or <directory/subdirectory>\n", os.Args)
		os.Exit(2)
	}
	root := os.Args[1]
	depth := strings.Count(root, string(os.PathSeparator))
	if depth > 1 {
		_, _ = fmt.Fprintf(os.Stderr, "usage: %s <directory> or <directory/subdirectory>\n", os.Args)
		os.Exit(2)
	}
	depth = 1 - depth
	if err := run(context.Background(), root, depth); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to fetch versions: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, root string, depth int) error {
	now := time.Now()
	defer func() {
		log.Printf("finished running in: %.2fs\n", time.Since(now).Seconds())
	}()
	configs, err := source.GatherConfigs(root, depth)
	if err != nil {
		return err
	}
	client := fetchclient.New()
	for _, config := range configs {
		if config.Source.Disabled {
			log.Printf("skipping source: %s\n", config.Filename)
			continue
		}
		newVersion, err := client.Fetch(ctx, config)
		if err != nil {
			return err
		}
		// example: library/grpc
		pluginDir := filepath.Dir(config.Filename)
		ok, err := checkDirExists(filepath.Join(pluginDir, newVersion))
		if err != nil {
			return err
		}
		if ok {
			log.Printf("skipping: %v/%v already exists\n", pluginDir, newVersion)
			continue
		}
		previousVersion, err := getLatestVersionFromDir(pluginDir)
		if err != nil {
			return fmt.Errorf("failed to get latest known version from dir %s with error: %w", pluginDir, err)
		}
		switch pluginDir {
		case "library/grpc", "library/protoc":
			// These directories don't follow the normal convention:
			//	library/{plugin_name}/{version}
			// 	Example: library/connect-go/v0.1.1
			//
			// Instead, they have an additional per language subdirectory:
			// 	library/{plugin_base}/{version}/{plugin_name}
			// 	Example: library/grpc/v1.48.0/ruby
			//
			// This means we need to make a copy for each of those subdirectories .
			if err := createPluginDirs(pluginDir, previousVersion, newVersion); err != nil {
				return err
			}
			if err := updatePluginDirs(pluginDir, newVersion); err != nil {
				return err
			}
		default:
			if err := createPluginDir(pluginDir, previousVersion, newVersion); err != nil {
				return err
			}
			// example: library/connect-go/v0.4.0/buf.plugin.yaml
			bufPluginFile := filepath.Join(pluginDir, newVersion, bufpluginconfig.ExternalConfigFilePath)
			if err := updateBufPluginFile(bufPluginFile, newVersion); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyDirectory copies all files from the source directory to the target,
// creating the target directory if does not exist.
// If the source directory contains subdirectories this function returns an error.
func copyDirectory(source, target string) (retErr error) {
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
		); err != nil {
			return err
		}
	}
	return nil
}

func createPluginDirs(pluginDir string, previousVersion string, newVersion string) error {
	// pluginDir: library/grpc
	// incomingVersion: v1.49.0-pre1
	// previousVersion: v1.48.0

	// example: library/grpc/v1.48.0
	oldPluginDir := filepath.Join(pluginDir, previousVersion)
	entries, err := os.ReadDir(oldPluginDir)
	if err != nil {
		return err
	}
	// example: library/grpc/v1.49.0-pre1
	newPluginDir := filepath.Join(pluginDir, newVersion)
	for _, entry := range entries {
		if !entry.IsDir() {
			return fmt.Errorf("expecting directories only: %s", entry.Name())
		}
		err = copyDirectory(
			filepath.Join(oldPluginDir, entry.Name()),
			filepath.Join(newPluginDir, entry.Name()),
		)
		if err != nil {
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
	)
}

func updatePluginDirs(pluginDir string, newVersion string) error {
	// example: library/grpc/v1.49.0-pre1
	newPluginDir := filepath.Join(pluginDir, newVersion)
	entries, err := os.ReadDir(newPluginDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// example: library/grpc/v1.49.0-pre1/python/buf.plugin.yaml
		bufPluginFile := filepath.Join(newPluginDir, entry.Name(), bufpluginconfig.ExternalConfigFilePath)
		ok, err := checkFileExists(bufPluginFile)
		if err != nil {
			return err
		}
		if ok {
			if err := updateBufPluginFile(bufPluginFile, newVersion); err != nil {
				return err
			}
		}
	}
	return nil
}

// updateBufPluginFile takes a named file, expected to be a buf.plugin.yaml, and
// updates the plugin_version with new version.
func updateBufPluginFile(name string, newVersion string) error {
	data, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	var config bufpluginconfig.ExternalConfig
	if err := encoding.UnmarshalYAMLStrict(data, &config); err != nil {
		return err
	}
	config.PluginVersion = newVersion
	// TODO(mf): can we also bump the registry.{npm|go}.deps assuming its the same version?
	newData, err := encoding.MarshalYAML(config)
	if err != nil {
		return fmt.Errorf("failed to write file %s with error: %s", name, err)
	}
	return os.WriteFile(name, newData, 0644)
}

func copyFile(src, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0644)
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

func checkFileExists(name string) (bool, error) {
	info, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("expecting normal file: %q", name)
	}
	return true, nil
}
