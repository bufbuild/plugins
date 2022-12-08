package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"aead.dev/minisign"
	"github.com/google/go-github/v48/github"

	"github.com/bufbuild/plugins/internal/release"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("failed to download release: %v", err)
	}
}

func run() error {
	var (
		minisignPublicKey string
		releaseTag        string
	)
	flag.StringVar(&minisignPublicKey, "minisign-public-key", "", "path to minisign public key file")
	flag.StringVar(&releaseTag, "release-tag", "", "release to download (default: latest release)")
	flag.Parse()

	if len(flag.Args()) != 1 {
		_, _ = os.Stderr.WriteString("usage: download-plugins <dir>\n")
		flag.PrintDefaults()
		os.Exit(2)
	}
	downloadDir := flag.Arg(0)

	ctx := context.Background()
	client := release.NewClient(ctx)

	publicKey, err := loadPublicKey(minisignPublicKey)
	if err != nil {
		return err
	}

	var githubRelease *github.RepositoryRelease
	if releaseTag == "" {
		githubRelease, err = client.GetLatestRelease(ctx, release.GithubOwnerBufbuild, release.GithubRepoPlugins)
		if err != nil {
			return err
		}
	} else {
		githubRelease, err = client.GetReleaseByTag(ctx, release.GithubOwnerBufbuild, release.GithubRepoPlugins, releaseTag)
		if err != nil {
			return err
		}
	}

	pluginReleases, err := client.LoadPluginReleases(ctx, githubRelease, publicKey)
	if err != nil {
		return err
	}
	for _, pluginRelease := range pluginReleases.Releases {
		exists, err := pluginExistsMatchingDigest(pluginRelease, downloadDir)
		if err != nil {
			return err
		}
		if exists {
			log.Printf("already downloaded: %s", filepath.Join(downloadDir, filepath.Base(pluginRelease.URL)))
		} else if err := downloadReleaseToDir(ctx, client.GitHub.Client(), pluginRelease, downloadDir); err != nil {
			return err
		}
	}
	return nil
}

func pluginExistsMatchingDigest(plugin release.PluginRelease, downloadDir string) (bool, error) {
	filename := filepath.Join(downloadDir, filepath.Base(plugin.URL))
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	digest, err := release.CalculateDigest(filename)
	if err != nil {
		return false, fmt.Errorf("failed to calculate digest for plugin %s: %w", plugin.URL, err)
	}
	return digest == plugin.PluginZipDigest, nil
}

func downloadReleaseToDir(ctx context.Context, client *http.Client, plugin release.PluginRelease, downloadDir string) error {
	expectedDigest, err := parseDigest(plugin.PluginZipDigest)
	if err != nil {
		return fmt.Errorf("failed to parse digest for plugin: %w", err)
	}
	f, err := os.CreateTemp(downloadDir, "."+strings.ReplaceAll(plugin.PluginName, "/", "-"))
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			log.Printf("failed to close temporary file: %v", err)
		}
		if err := os.Remove(f.Name()); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("failed to remove temporary file %q: %v", f.Name(), err)
		}
	}()
	log.Printf("downloading: %v", plugin.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, plugin.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to make HTTP request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform HTTP request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response: %v", err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: %s", plugin.URL, resp.Status)
	}
	digest := sha256.New()
	w := io.MultiWriter(f, digest)
	if _, err := io.Copy(w, resp.Body); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	sha256Digest := hex.EncodeToString(digest.Sum(nil))
	if sha256Digest != expectedDigest {
		return fmt.Errorf("checksum mismatch for %s: %q (expected) != %q (actual)", plugin.URL, expectedDigest, sha256Digest)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}
	if err := os.Rename(f.Name(), filepath.Join(downloadDir, filepath.Base(plugin.URL))); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}
	return nil
}

func parseDigest(digestStr string) (string, error) {
	digestType, digest, found := strings.Cut(digestStr, ":")
	if !found {
		return "", fmt.Errorf("malformed digest: %q", digestStr)
	}
	if digestType != "sha256" {
		return "", fmt.Errorf("unsupported digest: %q", digestType)
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return "", fmt.Errorf("malformed digest %q: %w", digest, err)
	}
	return digest, nil
}

func loadPublicKey(path string) (minisign.PublicKey, error) {
	if path == "" {
		return release.DefaultPublicKey()
	}
	return minisign.PublicKeyFromFile(path)
}
