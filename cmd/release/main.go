package main

import (
	"archive/zip"
	"bytes"
	"compress/flate"
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
	"github.com/google/go-github/v48/github"
	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/mod/semver"
	"golang.org/x/oauth2"

	"github.com/bufbuild/plugins/internal/plugin"
)

const (
	githubOwner = "bufbuild"
	// This is separate from githubOwner for testing releases (can point at personal fork).
	githubReleaseOwner = githubOwner // "pkwarren"
	githubRepo         = "plugins"
	pluginReleasesFile = "plugin-releases.json"
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
	PluginName       string        `json:"name"`           // org/name for the plugin (without remote)
	PluginVersion    string        `json:"version"`        // version of the plugin (including 'v' prefix)
	PluginZipDigest  string        `json:"zip_digest"`     // <digest-type>:<digest> for plugin .zip download
	PluginYAMLDigest string        `json:"yaml_digest"`    // <digest-type>:<digest> for buf.plugin.yaml
	ImageID          string        `json:"image_id"`       // <digest-type>:<digest> - https://github.com/opencontainers/image-spec/blob/main/config.md#imageid
	RegistryImage    string        `json:"registry_image"` // ghcr.io/bufbuild/plugins-<org>-<name>@<digest-type>:<digest>
	ReleaseTag       string        `json:"release_tag"`    // GitHub release tag - i.e. 20221121.1
	URL              string        `json:"url"`            // URL to GitHub release zip file for the plugin - i.e. https://github.com/bufbuild/plugins/releases/download/20221121.1/bufbuild-connect-go-v1.1.0.zip
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
	client := newGitHubClient(ctx)
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

	if err := plugin.Walk(root, func(plugin *plugin.Plugin) error {
		identity, err := bufpluginref.PluginIdentityForString(plugin.Name)
		if err != nil {
			return err
		}
		pluginYamlDigest, err := calculateDigest(plugin.Path)
		if err != nil {
			return err
		}
		registryImage, imageID, err := fetchRegistryImageAndImageID(plugin)
		if err != nil {
			return err
		}
		if registryImage == "" || imageID == "" {
			log.Printf("unable to detect registry image and image ID for plugin %s/%s:%s", identity.Owner(), identity.Plugin(), plugin.PluginVersion)
			return nil
		}
		key := pluginNameVersion{name: identity.Owner() + "/" + identity.Plugin(), version: plugin.PluginVersion}
		release := pluginNameVersionToRelease[key]
		// Found existing release - only rebuild if changed image digest or buf.plugin.yaml digest
		if release.ImageID != imageID || release.PluginYAMLDigest != pluginYamlDigest {
			downloadURL, err := pluginDownloadURL(plugin, releaseName)
			if err != nil {
				return err
			}
			zipDigest, err := createPluginZip(tmpDir, plugin, registryImage, imageID)
			if err != nil {
				return err
			}
			status := UPDATED
			if release.ImageID == "" {
				status = NEW
			}
			newPlugins = append(newPlugins, PluginRelease{
				PluginName:       fmt.Sprintf("%s/%s", identity.Owner(), identity.Plugin()),
				PluginVersion:    plugin.PluginVersion,
				PluginZipDigest:  zipDigest,
				PluginYAMLDigest: pluginYamlDigest,
				RegistryImage:    registryImage,
				ImageID:          imageID,
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
		if tagName := latestRelease.GetTagName(); tagName != "" {
			log.Printf("no changes to plugins since %v", tagName)
		} else {
			log.Printf("no changes to plugins - not creating initial release")
		}
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
	publicKey, err := signPluginReleases(tmpDir, minisignPrivateKey, minisignPrivateKeyPassword)
	if err != nil {
		return fmt.Errorf("failed to sign %q: %w", filepath.Join(tmpDir, pluginReleasesFile), err)
	}

	if dryRun {
		releaseBody := createReleaseBody(releaseName, plugins, publicKey)
		if err := os.WriteFile(filepath.Join(tmpDir, "RELEASE.md"), []byte(releaseBody), 0644); err != nil { //nolint:gosec
			return err
		}
		log.Printf("skipping GitHub release creation in dry-run mode")
		log.Printf("release assets created in %q", tmpDir)
		return nil
	}
	if err := createRelease(ctx, client, releaseName, plugins, tmpDir, publicKey); err != nil {
		return fmt.Errorf("failed to create GitHub release: %w", err)
	}
	return nil
}

func createRelease(ctx context.Context, client *github.Client, releaseName string, plugins []PluginRelease, tmpDir string, publicKey *minisign.PublicKey) error {
	// Create GitHub release
	release, _, err := client.Repositories.CreateRelease(ctx, githubReleaseOwner, githubRepo, &github.RepositoryRelease{
		Name: github.String(releaseName),
		Body: github.String(createReleaseBody(releaseName, plugins, publicKey)),
	})
	if err != nil {
		return err
	}
	return filepath.WalkDir(tmpDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
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

func createReleaseBody(name string, plugins []PluginRelease, publicKey *minisign.PublicKey) string {
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

	if publicKey != nil {
		sb.WriteString("## Verifying a release\n\n")
		sb.WriteString("Releases are signed using our [minisign](https://github.com/jedisct1/minisign) public key:\n\n")
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", publicKey.String()))
		sb.WriteString("The release assets can be verified using this command (assuming that minisign is installed):\n\n")
		releasesFile := fmt.Sprintf("https://github.com/%s/plugins/releases/download/%s/%s", githubReleaseOwner, name, pluginReleasesFile)
		sb.WriteString(fmt.Sprintf("```\ncurl -OL %s && \\\n", releasesFile))
		sb.WriteString(fmt.Sprintf("  curl -OL %s && \\\n", releasesFile+".minisig"))
		sb.WriteString(fmt.Sprintf("  minisign -Vm %s -P %s\n```\n", pluginReleasesFile, publicKey.String()))
	}
	return sb.String()
}

func signPluginReleases(dir string, keyPath string, password string) (*minisign.PublicKey, error) {
	if keyPath == "" || password == "" {
		log.Printf("skipping signing of %s", pluginReleasesFile)
		return nil, nil
	}
	releasesFile := filepath.Join(dir, pluginReleasesFile)
	log.Printf("signing: %s", releasesFile)
	privateKey, err := minisign.PrivateKeyFromFile(password, keyPath)
	if err != nil {
		return nil, err
	}
	releasesFileBytes, err := os.ReadFile(releasesFile)
	if err != nil {
		return nil, err
	}
	signature := minisign.Sign(privateKey, releasesFileBytes)
	if err := os.WriteFile(releasesFile+".minisig", signature, 0644); err != nil { //nolint:gosec
		return nil, err
	}
	publicKey, ok := privateKey.Public().(minisign.PublicKey)
	if !ok {
		return nil, errors.New("unable to type assert minisign public key")
	}
	return &publicKey, nil
}

func createPluginZip(basedir string, plugin *plugin.Plugin, registryImage string, imageID string) (string, error) {
	if err := pullImage(registryImage); err != nil {
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
	if err := saveImageToDir(imageID, pluginTempDir); err != nil {
		return "", err
	}
	log.Printf("creating %s", zipName)
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
	zw.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(w, flate.BestCompression)
	})
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
	cmd, err := dockerCmd("save", imageRef, "-o", "image.tar")
	if err != nil {
		return err
	}
	cmd.Dir = dir
	return cmd.Run()
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

func pullImage(name string) error {
	cmd, err := dockerCmd("pull", name)
	if err != nil {
		return err
	}
	log.Printf("pulling image: %s", name)
	return cmd.Run()
}

func dockerCmd(command string, args ...string) (*exec.Cmd, error) {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return nil, err
	}
	cmd := &exec.Cmd{
		Path: dockerPath,
		Args: append([]string{
			dockerPath,
			command,
		}, args...),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	return cmd, nil
}

func calculateNextRelease(now time.Time, latestRelease *github.RepositoryRelease) (string, error) {
	var releaseName string
	var latestReleaseName string
	if latestRelease != nil {
		latestReleaseName = latestRelease.GetTagName()
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

func fetchRegistryImageAndImageID(plugin *plugin.Plugin) (string, string, error) {
	identity, err := bufpluginref.PluginIdentityForString(plugin.Name)
	if err != nil {
		return "", "", err
	}
	imageName := fmt.Sprintf("ghcr.io/%s/plugins-%s-%s", githubOwner, identity.Owner(), identity.Plugin())
	cmd, err := dockerCmd("manifest", "inspect", "--verbose", imageName+":"+plugin.PluginVersion)
	if err != nil {
		return "", "", err
	}
	var bb bytes.Buffer
	cmd.Stdout = &bb
	if err := cmd.Run(); err != nil {
		// this may occur if this runs prior to publishing a newly added plugin
		return "", "", nil //nolint:nilerr
	}
	type manifestJSON struct {
		Descriptor struct {
			Digest string `json:"digest"`
		} `json:"Descriptor"` //nolint:tagliatelle
		SchemaV2Manifest struct {
			Config struct {
				Digest string `json:"digest"`
			} `json:"config"`
		} `json:"SchemaV2Manifest"` //nolint:tagliatelle
	}
	var result manifestJSON
	if err := json.Unmarshal(bb.Bytes(), &result); err != nil {
		return "", "", fmt.Errorf("unable to parse docker manifest inspect output: %w", err)
	}
	descriptorDigest := result.Descriptor.Digest
	if descriptorDigest == "" {
		return "", "", errors.New("unable to parse descriptor digest from docker manifest inspect output")
	}
	imageDigest := result.SchemaV2Manifest.Config.Digest
	if imageDigest == "" {
		return "", "", errors.New("unable to parse image config digest from docker manifest inspect output")
	}
	return fmt.Sprintf("%s@%s", imageName, descriptorDigest), imageDigest, nil
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

func newGitHubClient(ctx context.Context) *github.Client {
	var client *http.Client
	if ghToken := os.Getenv("GITHUB_TOKEN"); ghToken != "" {
		log.Printf("creating authenticated client with GITHUB_TOKEN")
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: ghToken},
		)
		client = oauth2.NewClient(ctx, ts)
	} else {
		client = retryablehttp.NewClient().StandardClient()
	}
	return github.NewClient(client)
}
