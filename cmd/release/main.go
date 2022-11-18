package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"aead.dev/minisign"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"
	"github.com/bufbuild/plugins/internal/plugin"
	"github.com/google/go-github/v48/github"
	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/mod/semver"
	"golang.org/x/oauth2"
)

const (
	githubOwner = "bufbuild"
	// This is separate from githubOwner for testing releases (can point at personal fork)
	githubReleaseOwner   = githubOwner // "pkwarren"
	githubRepo           = "plugins"
	packageTypeContainer = "container"
	pluginReleasesFile   = "plugin-releases.json"
)

type ReleaseStatus int

const (
	EXISTING ReleaseStatus = iota
	NEW
	UPDATED
)

type PluginReleases struct {
	Releases []PluginRelease `json:"releases"`
}

type PluginRelease struct {
	PluginName       string        `json:"name"`              // org/name for the plugin (without remote)
	PluginVersion    string        `json:"version"`           // version of the plugin (including 'v' prefix)
	PluginZipDigest  string        `json:"zip_digest"`        // <digest-type>:<digest> for plugin .zip download
	PluginYAMLDigest string        `json:"yaml_digest"`       // <digest-type>:<digest> for buf.plugin.yaml
	GHCRImageDigest  string        `json:"ghcr_image_digest"` // <digest-type>:<digest> for ghcr.io/bufbuild/plugins-<org>-<name>:v<version> image
	ReleaseTag       string        `json:"release_tag"`       // GitHub release tag - i.e. 20221121.1
	URL              string        `json:"url"`               // URL to GitHub release zip file for the plugin - i.e. https://github.com/bufbuild/plugins/releases/download/20221121.1/bufbuild-connect-go-v1.1.0.zip
	LastUpdated      time.Time     `json:"last_updated"`
	Status           ReleaseStatus `json:"-"`
}

type pluginNameVersion struct {
	name, version string
}

func main() {
	var (
		minisignPrivateKey string
		dryRun             bool
	)
	flag.BoolVar(&dryRun, "dry-run", false, "perform a dry-run (no GitHub modifications)")
	flag.StringVar(&minisignPrivateKey, "minisign-private-key", "", "path to minisign private key file")
	flag.Parse()

	if len(flag.Args()) != 1 {
		_, _ = fmt.Fprintf(os.Stderr, "usage: %s [-dry-run] [-minisign-private-key <file>] <directory>\n", os.Args)
		os.Exit(2)
	}
	root := flag.Args()[0]
	if err := run(root, minisignPrivateKey, dryRun); err != nil {
		log.Fatalln(err.Error())
	}
}

func run(root string, minisignPrivateKey string, dryRun bool) error {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "plugins-release")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	log.Printf("created tmp dir: %s", tmpDir)
	if !dryRun {
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				log.Printf("failed to remove %q: %v", tmpDir, err)
			}
		}()
	}

	ctx := context.Background()
	client := newGitHubClient()
	latestRelease, err := getLatestRelease(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to retrieve latest release: %w", err)
	}

	releases, err := loadPluginReleases(ctx, client, latestRelease)
	if err != nil {
		return fmt.Errorf("failed to determine latest plugin releases: %w", err)
	}
	if releases == nil {
		log.Printf("no current release found")
		releases = &PluginReleases{}
	}

	pluginNameVersionToRelease := make(map[pluginNameVersion]PluginRelease, len(releases.Releases))
	for _, release := range releases.Releases {
		key := pluginNameVersion{name: release.PluginName, version: release.PluginVersion}
		if _, ok := pluginNameVersionToRelease[key]; ok {
			return fmt.Errorf("duplicate plugin discovered in releases file: %+v", key)
		}
		pluginNameVersionToRelease[key] = release
	}

	var newPlugins []PluginRelease
	var existingPlugins []PluginRelease
	now := time.Now().UTC().Truncate(time.Second)
	releaseName, err := calculateNextRelease(now, latestRelease)
	if err != nil {
		return fmt.Errorf("failed to determine next release name: %w", err)
	}

	n := 0
	if err := plugin.Walk(root, func(plugin *plugin.Plugin) error {
		n++
		if n > 2 {
			return nil
		}
		identity, err := bufpluginref.PluginIdentityForString(plugin.Name)
		if err != nil {
			return nil
		}
		pluginYamlDigest, err := calculateDigest(plugin.Path)
		if err != nil {
			return err
		}
		imageName, imageDigest, err := fetchGHCRImageNameAndDigest(context.Background(), client, plugin)
		if err != nil {
			return err
		}
		key := pluginNameVersion{name: identity.Owner() + "/" + identity.Plugin(), version: plugin.PluginVersion}
		release := pluginNameVersionToRelease[key]
		// Found existing release - only rebuild if changed image digest or buf.plugin.yaml digest
		if release.GHCRImageDigest != imageDigest || release.PluginYAMLDigest != pluginYamlDigest {
			downloadURL, err := pluginDownloadURL(plugin, releaseName)
			if err != nil {
				return err
			}
			zipDigest, err := createPluginZip(tmpDir, plugin, imageName, imageDigest)
			if err != nil {
				return err
			}
			status := UPDATED
			if release.GHCRImageDigest == "" {
				status = NEW
			}
			newPlugins = append(newPlugins, PluginRelease{
				PluginName:       fmt.Sprintf("%s/%s", identity.Owner(), identity.Plugin()),
				PluginVersion:    plugin.PluginVersion,
				PluginZipDigest:  zipDigest,
				PluginYAMLDigest: pluginYamlDigest,
				GHCRImageDigest:  imageDigest,
				ReleaseTag:       releaseName,
				URL:              downloadURL,
				LastUpdated:      now,
				Status:           status,
			})
		} else {
			log.Printf("plugin %s:%s unchanged", release.PluginName, release.PluginVersion)
			release.Status = EXISTING
			existingPlugins = append(existingPlugins, release)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to discover plugins in path %q: %w", root, err)
	}

	if len(newPlugins) == 0 {
		log.Printf("no changes to plugins since %v", latestRelease.GetTagName())
		return nil
	}

	plugins := make([]PluginRelease, 0, len(newPlugins)+len(existingPlugins))
	plugins = append(plugins, newPlugins...)
	plugins = append(plugins, existingPlugins...)
	sort.Slice(plugins, func(i, j int) bool {
		p1, p2 := plugins[i], plugins[j]
		if p1.PluginName != p2.PluginName {
			return p1.PluginName < p2.PluginName
		}
		return semver.Compare(p1.PluginVersion, p2.PluginVersion) < 0
	})
	if err := createPluginReleases(tmpDir, plugins); err != nil {
		return fmt.Errorf("failed to create %s: %w", pluginReleasesFile, err)
	}

	minisignPrivateKeyPassword := os.Getenv("MINISIGN_PRIVATE_KEY_PASSWORD")
	if minisignPrivateKey != "" && minisignPrivateKeyPassword != "" {
		if err := signPluginReleases(tmpDir, minisignPrivateKey, minisignPrivateKeyPassword); err != nil {
			return fmt.Errorf("failed to sign %q: %w", filepath.Join(tmpDir, pluginReleasesFile), err)
		}
	} else {
		log.Printf("skipping signing of %s", pluginReleasesFile)
	}

	if dryRun {
		log.Printf("skipping GitHub release creation in dry-run mode")
		log.Printf("release assets created in %q", tmpDir)
		return nil
	}
	if err := createRelease(ctx, client, releaseName, plugins, tmpDir); err != nil {
		return fmt.Errorf("failed to create GitHub release: %w", err)
	}
	return nil
}

func createRelease(ctx context.Context, client *github.Client, releaseName string, plugins []PluginRelease, tmpDir string) error {
	// Create GitHub release
	release, _, err := client.Repositories.CreateRelease(ctx, githubReleaseOwner, githubRepo, &github.RepositoryRelease{
		Name:    github.String(releaseName),
		TagName: github.String("releases/" + releaseName),
		Body:    github.String(createReleaseBody(releaseName, plugins)),
	})
	if err != nil {
		return err
	}
	return filepath.WalkDir(tmpDir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		log.Printf("uploading: %s", d.Name())
		_, _, err = client.Repositories.UploadReleaseAsset(ctx, githubReleaseOwner, githubRepo, release.GetID(), &github.UploadOptions{
			Name: d.Name(),
		}, f)
		return err
	})
}

func createReleaseBody(name string, plugins []PluginRelease) string {
	var sb strings.Builder
	pluginsByStatus := make(map[ReleaseStatus][]PluginRelease)
	for _, p := range plugins {
		pluginsByStatus[p.Status] = append(pluginsByStatus[p.Status], p)
	}

	sb.WriteString(fmt.Sprintf("# Buf Remote Plugins Release %s\n\n", name))

	if newPlugins := pluginsByStatus[NEW]; len(newPlugins) > 0 {
		sb.WriteString(`## New Plugins

| Plugin | Version | Link |
|--------|---------|------|
`)
		for _, p := range newPlugins {
			sb.WriteString(fmt.Sprintf("| %s | %s | [Download](%s) |\n", p.PluginName, p.PluginVersion, p.URL))
		}
		sb.WriteString("\n")
	}

	if updatedPlugins := pluginsByStatus[UPDATED]; len(updatedPlugins) > 0 {
		sb.WriteString(`## Updated Plugins

| Plugin | Version | Link |
|--------|---------|------|
`)
		for _, p := range updatedPlugins {
			sb.WriteString(fmt.Sprintf("| %s | %s | [Download](%s) |\n", p.PluginName, p.PluginVersion, p.URL))
		}
		sb.WriteString("\n")
	}

	if existingPlugins := pluginsByStatus[EXISTING]; len(existingPlugins) > 0 {
		sb.WriteString(`## Previously Released Plugins

| Plugin | Version | Release | Link |
|--------|---------|---------|------|
`)
		for _, p := range existingPlugins {
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | [Download](%s) |\n", p.PluginName, p.PluginVersion, p.ReleaseTag, p.URL))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func signPluginReleases(dir string, keyPath string, password string) error {
	releasesFile := filepath.Join(dir, pluginReleasesFile)
	log.Printf("signing: %s", releasesFile)
	privateKey, err := minisign.PrivateKeyFromFile(password, keyPath)
	if err != nil {
		return err
	}
	releasesFileBytes, err := os.ReadFile(releasesFile)
	if err != nil {
		return err
	}
	signature := minisign.Sign(privateKey, releasesFileBytes)
	return os.WriteFile(releasesFile+".minisig", signature, 0644)
}

func createPluginZip(basedir string, plugin *plugin.Plugin, imageName string, imageDigest string) (string, error) {
	if err := pullImage(imageName, imageDigest); err != nil {
		return "", err
	}
	zipName, err := pluginZipName(plugin)
	if err != nil {
		return "", err
	}
	pluginTempDir, err := os.MkdirTemp(basedir, strings.TrimSuffix(zipName, filepath.Ext(zipName)))
	if err != nil {
		return "", err
	}
	defer func() {
		if err := os.RemoveAll(pluginTempDir); err != nil {
			log.Printf("failed to remove %q: %v", pluginTempDir, err)
		}
	}()
	if err := saveImageToDir(fmt.Sprintf("%s@%s", imageName, imageDigest), pluginTempDir); err != nil {
		return "", err
	}
	zipFile := filepath.Join(basedir, zipName)
	zf, err := os.OpenFile(zipFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := zf.Close(); !errors.Is(err, os.ErrClosed) {
			log.Printf("failed to close: %v", err)
		}
	}()
	zw := zip.NewWriter(zf)
	if err := addFileToZip(zw, plugin.Path); err != nil {
		return "", err
	}
	if err := addFileToZip(zw, filepath.Join(pluginTempDir, "image.tar")); err != nil {
		return "", err
	}
	if err := zw.Close(); err != nil {
		return "", err
	}
	if err := zf.Close(); err != nil {
		return "", err
	}
	digest, err := calculateDigest(zipFile)
	if err != nil {
		return "", err
	}
	return digest, nil
}

func addFileToZip(zipWriter *zip.Writer, path string) error {
	w, err := zipWriter.Create(filepath.Base(path))
	if err != nil {
		return err
	}
	r, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("failed to close: %v", err)
		}
	}()
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return nil
}

func saveImageToDir(imageRef string, dir string) error {
	dockerCmd, err := exec.LookPath("docker")
	if err != nil {
		return err
	}
	dockerSave := exec.Cmd{
		Path: dockerCmd,
		Args: []string{
			dockerCmd,
			"save",
			imageRef,
			"-o",
			"image.tar",
		},
		Dir:    dir,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	return dockerSave.Run()
}

func createPluginReleases(dir string, plugins []PluginRelease) error {
	f, err := os.OpenFile(filepath.Join(dir, pluginReleasesFile), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("failed to close: %v", err)
		}
	}()
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(&PluginReleases{Releases: plugins})
}

func pullImage(name string, digest string) error {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return err
	}
	image := fmt.Sprintf("%s@%s", name, digest)
	log.Printf("pulling image: %s", image)
	pullCmd := exec.Cmd{
		Path: dockerPath,
		Args: []string{
			dockerPath,
			"pull",
			image,
		},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	return pullCmd.Run()
}

func calculateNextRelease(now time.Time, latestRelease *github.RepositoryRelease) (string, error) {
	var releaseName string
	var latestReleaseName string
	if latestRelease != nil {
		latestReleaseName = strings.TrimPrefix(latestRelease.GetTagName(), "releases/")
	}
	currentDate := now.UTC().Format("20060102")
	if latestRelease == nil || !strings.HasPrefix(latestReleaseName, currentDate+".") {
		releaseName = currentDate + ".1"
	} else {
		_, revision, ok := strings.Cut(latestReleaseName, ".")
		if !ok {
			return "", fmt.Errorf("malformed latest release tag name: %v", latestRelease.GetTagName())
		}
		currentRevision, err := strconv.Atoi(revision)
		if err != nil {
			return "", err
		}
		releaseName = currentDate + "." + strconv.Itoa(currentRevision+1)
	}
	return releaseName, nil
}

func pluginDownloadURL(plugin *plugin.Plugin, releaseName string) (string, error) {
	zipName, err := pluginZipName(plugin)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", githubReleaseOwner, githubRepo, releaseName, zipName), nil
}

func pluginZipName(plugin *plugin.Plugin) (string, error) {
	identity, err := bufpluginref.PluginIdentityForString(plugin.Name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%s.zip", identity.Owner(), identity.Plugin(), plugin.PluginVersion), nil
}

func calculateDigest(path string) (string, error) {
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

func fetchGHCRImageNameAndDigest(ctx context.Context, client *github.Client, plugin *plugin.Plugin) (string, string, error) {
	identity, err := bufpluginref.PluginIdentityForString(plugin.Name)
	if err != nil {
		return "", "", err
	}
	packageName := fmt.Sprintf("plugins-%s-%s", identity.Owner(), identity.Plugin())
	versions, _, err := client.Organizations.PackageGetAllVersions(ctx, githubOwner, packageTypeContainer, packageName, &github.PackageListOptions{})
	if err != nil {
		return "", "", err
	}
	for _, version := range versions {
		if version.GetMetadata() == nil || version.GetMetadata().GetContainer() == nil {
			continue
		}
		for _, tag := range version.GetMetadata().GetContainer().Tags {
			if tag == plugin.PluginVersion {
				imageName := fmt.Sprintf("ghcr.io/%s/%s", githubOwner, packageName)
				return imageName, version.GetName(), nil
			}
		}
	}
	return "", "", fmt.Errorf("no digest found for: %v:%v", identity.IdentityString(), plugin.PluginVersion)
}

func getLatestRelease(ctx context.Context, client *github.Client) (*github.RepositoryRelease, error) {
	release, resp, err := client.Repositories.GetLatestRelease(ctx, githubReleaseOwner, githubRepo)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// no latest release
			return nil, nil
		}
		return nil, err
	}
	return release, nil
}

func loadPluginReleases(ctx context.Context, client *github.Client, latestRelease *github.RepositoryRelease) (*PluginReleases, error) {
	if latestRelease == nil {
		return nil, nil
	}

	var pluginVersionsAssetID int64
	for _, asset := range latestRelease.Assets {
		if asset.GetName() == pluginReleasesFile {
			// Found asset
			pluginVersionsAssetID = asset.GetID()
			break
		}
	}
	if pluginVersionsAssetID == 0 {
		// no plugin-versions.json in latest release
		return nil, nil
	}
	rc, _, err := client.Repositories.DownloadReleaseAsset(ctx, githubReleaseOwner, githubRepo, pluginVersionsAssetID, http.DefaultClient)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rc.Close(); err != nil {
			log.Printf("failed to close: %v", err)
		}
	}()
	var releases PluginReleases
	if err := json.NewDecoder(rc).Decode(&releases); err != nil {
		return nil, err
	}
	return &releases, nil
}

func newGitHubClient() *github.Client {
	var client *http.Client
	if ghToken := os.Getenv("GITHUB_TOKEN"); ghToken != "" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: ghToken},
		)
		client = oauth2.NewClient(ctx, ts)
	} else {
		client = retryablehttp.NewClient().StandardClient()
	}
	return github.NewClient(client)
}
