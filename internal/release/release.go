package release

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"os"
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
