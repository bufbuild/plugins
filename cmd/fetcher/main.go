package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bufbuild/plugins/internal/fetchclient"
	"github.com/bufbuild/plugins/internal/source"
	"go.uber.org/multierr"
	"golang.org/x/mod/semver"
)

var (
	// TODO(mf): use the plugin from bufbuild/buf
	re = regexp.MustCompile(`(?m)^plugin_version:\s(v.*)$`)
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
		latestVersion, err := client.Fetch(ctx, config)
		if err != nil {
			return err
		}
		baseDir := filepath.Dir(config.Filename)
		ok, err := checkDirVersionExists(baseDir, latestVersion)
		if err != nil {
			return err
		}
		if ok {
			log.Printf("skipping: %v/%v already exists\n", baseDir, latestVersion)
			continue
		}
		if err := createDirVersion(baseDir, latestVersion); err != nil {
			return err
		}
	}
	return nil
}

func createDirVersion(dir string, version string) (retErr error) {
	previousVersion, err := getLatestVersionFromDir(dir)
	if err != nil && !errors.Is(err, errNoVersions) {
		return err
	}
	// ^^^^ make sure to resolve the latest version before writing the new version
	// 		to the same directory.
	if err := os.Mkdir(filepath.Join(dir, version), 0755); err != nil {
		return err
	}
	defer func() {
		if retErr != nil {
			retErr = multierr.Append(retErr, os.RemoveAll(filepath.Join(dir, version)))
		}
	}()
	if previousVersion == "" {
		log.Printf("successfully created empty directory: %s\n", filepath.Join(dir, version))
		return nil
	}
	for _, name := range []string{"Dockerfile", ".dockerignore"} {
		if err := copy(
			filepath.Join(dir, previousVersion, name),
			filepath.Join(dir, version, name),
		); err != nil {
			return err
		}
	}
	previousBufPluginFile := filepath.Join(dir, previousVersion, "buf.plugin.yaml")
	previousBufPluginData, err := os.ReadFile(previousBufPluginFile)
	if err != nil {
		return err
	}
	match := re.FindSubmatch(previousBufPluginData)
	if len(match) != 2 {
		return fmt.Errorf("invalid match for plugin_version in buf.plugin.yaml file: got %d matches", len(match))
	}
	// Sanity check the existing plugin_version is lower than incoming version.
	match[1] = bytes.TrimSpace(match[1])
	if semver.Compare(string(match[1]), previousVersion) < 0 {
		return fmt.Errorf(
			"existing plugin_version version: %q in %s has higher semver precedence than found version: %q",
			string(match[1]),
			previousBufPluginFile,
			version,
		)
	}
	// We are certain the incoming version is latest, replace and write to new file.
	newBufPluginData := re.ReplaceAll(previousBufPluginData, []byte("plugin_version: "+version))
	newBufPluginFile := filepath.Join(dir, version, "buf.plugin.yaml")
	return os.WriteFile(newBufPluginFile, newBufPluginData, 0644)
}

func copy(src, dest string) error {
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
	for _, d := range dirs {
		if d.IsDir() {
			if semver.IsValid(d.Name()) {
				versions = append(versions, d.Name())
				continue
			}
		}
	}
	if len(versions) == 0 {
		return "", errNoVersions
	}
	semver.Sort(versions)
	return versions[len(versions)-1], nil
}

func checkDirVersionExists(dir string, version string) (bool, error) {
	info, err := os.Stat(filepath.Join(dir, version))
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
