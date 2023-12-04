package release

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

// Utilities for creating or consuming GitHub plugin releases.

const (
	PluginReleasesFile          = "plugin-releases.json"
	PluginReleasesSignatureFile = PluginReleasesFile + ".minisig"
)

type Status int

const (
	StatusExisting Status = iota
	StatusNew
	StatusUpdated
)

type PluginReleases struct {
	Releases []PluginRelease `json:"releases"`
}

type PluginRelease struct {
	PluginName       string    `json:"name"`           // org/name for the plugin (without remote)
	PluginVersion    string    `json:"version"`        // version of the plugin (including 'v' prefix)
	PluginZipDigest  string    `json:"zip_digest"`     // <digest-type>:<digest> for plugin .zip download
	PluginYAMLDigest string    `json:"yaml_digest"`    // <digest-type>:<digest> for buf.plugin.yaml
	ImageID          string    `json:"image_id"`       // <digest-type>:<digest> - https://github.com/opencontainers/image-spec/blob/main/config.md#imageid
	RegistryImage    string    `json:"registry_image"` // ghcr.io/bufbuild/plugins-<org>-<name>@<digest-type>:<digest>
	ReleaseTag       string    `json:"release_tag"`    // GitHub release tag - i.e. 20221121.1
	URL              string    `json:"url"`            // URL to GitHub release zip file for the plugin - i.e. https://github.com/bufbuild/plugins/releases/download/20221121.1/bufbuild-connect-go-v1.1.0.zip
	LastUpdated      time.Time `json:"last_updated"`
	Status           Status    `json:"-"`
	Dependencies     []string  `json:"dependencies,omitempty"` // direct dependencies on other plugins
}

// CalculateDigest will calculate the sha256 digest of the given file.
// It returns a string in the format: '<digest-type>:<digest>'.
func CalculateDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("failed to close: %v", err)
		}
	}()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	hashBytes := hash.Sum(nil)
	return "sha256:" + hex.EncodeToString(hashBytes), nil
}

// SortReleasesInDependencyOrder sorts the list of plugin releases so that a plugin's dependencies come before each plugin.
// The original slice is unmodified - it returns a copy in sorted order, or an error if there is a cycle or unmet dependency.
func SortReleasesInDependencyOrder(original []PluginRelease) ([]PluginRelease, error) {
	// Make a defensive copy of the original list
	plugins := make([]PluginRelease, len(original))
	copy(plugins, original)
	resolved := make([]PluginRelease, 0, len(plugins))
	resolvedMap := make(map[string]struct{}, len(plugins))
	for len(plugins) > 0 {
		var unresolved []PluginRelease
		for _, plugin := range plugins {
			foundDeps := true
			for _, dep := range plugin.Dependencies {
				// TODO: This is kinda ugly - we don't include the remote on names in plugin-releases.json but do on deps.
				if _, ok := resolvedMap[strings.TrimPrefix(dep, "buf.build/")]; !ok {
					foundDeps = false
					break
				}
			}
			if foundDeps {
				resolved = append(resolved, plugin)
				resolvedMap[plugin.PluginName+":"+plugin.PluginVersion] = struct{}{}
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
