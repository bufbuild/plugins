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

	"github.com/bufbuild/plugins/internal/fetchclient"
	"github.com/bufbuild/plugins/internal/source"
	"go.uber.org/multierr"
	"golang.org/x/mod/semver"
)

var errNoVersions = errors.New("no versions found")

func main() {
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintf(os.Stderr, "usage: %s <directory>\n", os.Args)
		os.Exit(2)
	}
	root := os.Args[1]
	depth := strings.Count(root, string(os.PathSeparator))
	if depth > 1 {
		_, _ = fmt.Fprintf(os.Stderr, "usage: %s <directory>\n", os.Args)
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
	configs, err := source.GatherConfigs(root)
	if err != nil {
		return err
	}
	client := fetchclient.New()
	latestVersions := make(map[string]string, len(configs))
	for _, config := range configs {
		if config.Source.Disabled {
			log.Printf("skipping source: %s\n", config.Filename)
			continue
		}
		newVersion := latestVersions[config.CacheKey()]
		if newVersion == "" {
			newVersion, err = client.Fetch(ctx, config)
			if err != nil {
				return err
			}
			latestVersions[config.CacheKey()] = newVersion
		}
		// For now we ignore prerelease versions. But this may change in the future.
		if semver.Prerelease(newVersion) != "" && !config.IncludePrerelease {
			continue
		}
		// example: library/grpc
		pluginDir := filepath.Dir(config.Filename)
		ok, err := checkDirExists(filepath.Join(pluginDir, newVersion))
		if err != nil {
			return err
		}
		if ok {
			continue
		}
		previousVersion, err := getLatestVersionFromDir(pluginDir)
		if err != nil {
			return fmt.Errorf("failed to get latest known version from dir %s with error: %w", pluginDir, err)
		}
		if err := createPluginDir(pluginDir, previousVersion, newVersion); err != nil {
			return err
		}
		log.Printf("created %v/%v\n", pluginDir, newVersion)
	}
	return nil
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
	return os.WriteFile(dest, []byte(replaced), 0644)
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
