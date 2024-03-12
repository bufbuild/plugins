package main

import (
	"archive/zip"
	"cmp"
	"compress/flate"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"aead.dev/minisign"
	"github.com/bufbuild/buf/private/pkg/interrupt"
	githubkeychain "github.com/google/go-containerregistry/pkg/authn/github"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-github/v58/github"
	"golang.org/x/mod/semver"

	"github.com/bufbuild/plugins/internal/plugin"
	"github.com/bufbuild/plugins/internal/release"
)

type pluginNameVersion struct {
	name, version string
}

func main() {
	dryRun := flag.Bool("dry-run", false, "perform a dry-run (no GitHub modifications)")
	githubCommit := flag.String("commit", "", "GitHub commit for the release")
	githubReleaseOwner := flag.String(
		"github-release-owner",
		string(release.GithubOwnerBufbuild),
		"GitHub release owner (set to personal account to test against a fork)",
	)
	minisignPrivateKey := flag.String("minisign-private-key", "", "path to minisign private key file")
	minisignPublicKey := flag.String(
		"minisign-public-key",
		"",
		"path to public key used to verify the latest release's plugin-releases.json file (if different than private key)",
	)
	flag.Parse()

	if len(flag.Args()) != 1 {
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "usage: release <directory>")
		flag.PrintDefaults()
		os.Exit(2)
	}
	root := flag.Args()[0]
	cmd := &command{
		minisignPrivateKey: *minisignPrivateKey,
		minisignPublicKey:  *minisignPublicKey,
		githubCommit:       *githubCommit,
		githubReleaseOwner: release.GithubOwner(*githubReleaseOwner),
		dryRun:             *dryRun,
		rootDir:            root,
	}
	if err := cmd.run(); err != nil {
		log.Fatalln(err.Error())
	}
}

type command struct {
	minisignPrivateKey string
	minisignPublicKey  string
	githubCommit       string
	githubReleaseOwner release.GithubOwner
	dryRun             bool
	rootDir            string
}

func (c *command) run() error {
	ctx, cancel := interrupt.WithCancel(context.Background())
	defer cancel()
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "plugins-release")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	log.Printf("created tmp dir: %s", tmpDir)
	defer func() {
		if c.dryRun {
			return
		}
		if err := os.RemoveAll(tmpDir); err != nil {
			log.Printf("failed to remove %q: %v", tmpDir, err)
		}
	}()
	client := release.NewClient(ctx)
	latestRelease, err := client.GetLatestRelease(ctx, c.githubReleaseOwner, release.GithubRepoPlugins)
	if err != nil && !errors.Is(err, release.ErrNotFound) {
		return fmt.Errorf("failed to retrieve latest release: %w", err)
	}

	var privateKey minisign.PrivateKey
	if minisignPrivateKeyPassword := os.Getenv("MINISIGN_PRIVATE_KEY_PASSWORD"); minisignPrivateKeyPassword != "" {
		privateKey, err = minisign.PrivateKeyFromFile(minisignPrivateKeyPassword, c.minisignPrivateKey)
		if err != nil {
			return err
		}
	}
	publicKey, err := c.loadMinisignPublicKeyFromFileOrPrivateKey(privateKey)
	if err != nil {
		return err
	}
	releases, err := client.DownloadPluginReleasesToDir(ctx, latestRelease, publicKey, tmpDir)
	if err != nil && !errors.Is(err, release.ErrNotFound) {
		return fmt.Errorf("failed to determine latest plugin releases: %w", err)
	}
	if releases == nil {
		log.Printf("no current release found")
		releases = &release.PluginReleases{}
	}

	now := time.Now().UTC().Truncate(time.Second)
	releaseName, err := calculateNextRelease(now, latestRelease)
	if err != nil {
		return fmt.Errorf("failed to determine next release name: %w", err)
	}

	plugins, err := c.calculateNewReleasePlugins(ctx, releases, releaseName, now, tmpDir)
	if err != nil {
		return fmt.Errorf("failed to calculate new release contents: %w", err)
	}
	if len(plugins) == 0 {
		if tagName := latestRelease.GetTagName(); tagName != "" {
			log.Printf("no changes to plugins since %v", tagName)
		} else {
			log.Printf("no changes to plugins - not creating initial release")
		}
		return nil
	}
	if err := createPluginReleases(tmpDir, plugins); err != nil {
		return fmt.Errorf("failed to create %s: %w", release.PluginReleasesFile, err)
	}

	if err := signPluginReleases(tmpDir, privateKey); err != nil {
		return fmt.Errorf("failed to sign %q: %w", filepath.Join(tmpDir, release.PluginReleasesFile), err)
	}

	if c.dryRun {
		releaseBody, err := c.createReleaseBody(releaseName, plugins, privateKey)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "RELEASE.md"), []byte(releaseBody), 0644); err != nil { //nolint:gosec
			return err
		}
		log.Printf("skipping GitHub release creation in dry-run mode")
		log.Printf("release assets created in %q", tmpDir)
		return nil
	}
	if err := c.createRelease(ctx, client, releaseName, plugins, tmpDir, privateKey); err != nil {
		return fmt.Errorf("failed to create GitHub release: %w", err)
	}
	return nil
}

func (c *command) calculateNewReleasePlugins(ctx context.Context, currentRelease *release.PluginReleases, releaseName string, now time.Time, tmpDir string) (
	[]release.PluginRelease, error,
) {
	pluginNameVersionToRelease := make(map[pluginNameVersion]release.PluginRelease, len(currentRelease.Releases))
	for _, pluginRelease := range currentRelease.Releases {
		key := pluginNameVersion{name: pluginRelease.PluginName, version: pluginRelease.PluginVersion}
		if _, ok := pluginNameVersionToRelease[key]; ok {
			return nil, fmt.Errorf("duplicate plugin discovered in releases file: %+v", key)
		}
		pluginNameVersionToRelease[key] = pluginRelease
	}

	var newPlugins []release.PluginRelease
	var existingPlugins []release.PluginRelease

	if err := plugin.Walk(c.rootDir, func(plugin *plugin.Plugin) error {
		pluginYamlDigest, err := release.CalculateDigest(plugin.Path)
		if err != nil {
			return err
		}
		registryImage, imageID, err := fetchRegistryImageAndImageID(plugin)
		if err != nil {
			return err
		}
		identity := plugin.Identity
		if registryImage == "" || imageID == "" {
			log.Printf("unable to detect registry image and image ID for plugin %s/%s:%s", identity.Owner(), identity.Plugin(), plugin.PluginVersion)
			return nil
		}
		key := pluginNameVersion{name: identity.Owner() + "/" + identity.Plugin(), version: plugin.PluginVersion}
		pluginRelease := pluginNameVersionToRelease[key]
		// Found existing release - only rebuild if changed image digest or buf.plugin.yaml digest
		if pluginRelease.ImageID != imageID || pluginRelease.PluginYAMLDigest != pluginYamlDigest {
			downloadURL := c.pluginDownloadURL(plugin, releaseName)
			zipDigest, err := createPluginZip(ctx, tmpDir, plugin, registryImage, imageID)
			if err != nil {
				return err
			}
			status := release.StatusUpdated
			if pluginRelease.ImageID == "" {
				status = release.StatusNew
			}
			newPlugins = append(newPlugins, release.PluginRelease{
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
				Dependencies:     pluginDependencies(plugin),
			})
		} else {
			log.Printf("plugin %s:%s unchanged", pluginRelease.PluginName, pluginRelease.PluginVersion)
			pluginRelease.Status = release.StatusExisting
			pluginRelease.Dependencies = pluginDependencies(plugin)
			existingPlugins = append(existingPlugins, pluginRelease)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to discover plugins in path %q: %w", c.rootDir, err)
	}

	if len(newPlugins) == 0 {
		return nil, nil
	}

	plugins := make([]release.PluginRelease, 0, len(newPlugins)+len(existingPlugins))
	plugins = append(plugins, newPlugins...)
	plugins = append(plugins, existingPlugins...)
	sortPluginsByNameVersion(plugins)
	return plugins, nil
}

func pluginDependencies(plugin *plugin.Plugin) []string {
	if len(plugin.Deps) == 0 {
		return nil
	}
	deps := make([]string, len(plugin.Deps))
	for i, dep := range plugin.Deps {
		if dep.Revision != 0 {
			log.Fatalf("unsupported plugin dependency revision: %v", dep.Revision)
		}
		deps[i] = dep.Plugin
	}
	slices.Sort(deps)
	return deps
}

func (c *command) loadMinisignPublicKeyFromFileOrPrivateKey(privateKey minisign.PrivateKey) (minisign.PublicKey, error) {
	var publicKey minisign.PublicKey
	if c.minisignPublicKey != "" {
		var err error
		publicKey, err = minisign.PublicKeyFromFile(c.minisignPublicKey)
		if err != nil {
			return minisign.PublicKey{}, err
		}
	} else if !privateKey.Equal(minisign.PrivateKey{}) {
		var ok bool
		publicKey, ok = privateKey.Public().(minisign.PublicKey)
		if !ok {
			return minisign.PublicKey{}, errors.New("unable to retrieve minisign public key from private key")
		}
	}
	return publicKey, nil
}

func sortPluginsByNameVersion(plugins []release.PluginRelease) {
	slices.SortFunc(plugins, func(a, b release.PluginRelease) int {
		if c := cmp.Compare(a.PluginName, b.PluginName); c != 0 {
			return c
		}
		return semver.Compare(a.PluginVersion, b.PluginVersion)
	})
}

func (c *command) createRelease(ctx context.Context, client *release.Client, releaseName string, plugins []release.PluginRelease, tmpDir string, privateKey minisign.PrivateKey) error {
	releaseBody, err := c.createReleaseBody(releaseName, plugins, privateKey)
	if err != nil {
		return err
	}
	// Create GitHub release
	repositoryReleaseParams := &github.RepositoryRelease{
		TagName: github.String(releaseName),
		Name:    github.String(releaseName),
		Body:    github.String(releaseBody),
		// Start release as a draft until all assets are uploaded
		Draft: github.Bool(true),
	}
	if c.githubCommit != "" {
		repositoryReleaseParams.TargetCommitish = github.String(c.githubCommit)
	}
	repositoryRelease, err := client.CreateRelease(ctx, c.githubReleaseOwner, release.GithubRepoPlugins, repositoryReleaseParams)
	if err != nil {
		return err
	}
	if err := filepath.WalkDir(tmpDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		log.Printf("uploading: %s", d.Name())
		return client.UploadReleaseAsset(ctx, c.githubReleaseOwner, release.GithubRepoPlugins, repositoryRelease.GetID(), path)
	}); err != nil {
		return err
	}
	// Publish release
	if _, err := client.EditRelease(ctx, c.githubReleaseOwner, release.GithubRepoPlugins, repositoryRelease.GetID(), &github.RepositoryRelease{
		Draft: github.Bool(false),
	}); err != nil {
		return err
	}
	return nil
}

func (c *command) createReleaseBody(name string, plugins []release.PluginRelease, privateKey minisign.PrivateKey) (string, error) {
	var sb strings.Builder
	pluginsByStatus := make(map[release.Status][]release.PluginRelease)
	for _, p := range plugins {
		pluginsByStatus[p.Status] = append(pluginsByStatus[p.Status], p)
	}

	sb.WriteString(fmt.Sprintf("# Buf Remote Plugins Release %s\n\n", name))

	if newPlugins := pluginsByStatus[release.StatusNew]; len(newPlugins) > 0 {
		sb.WriteString(`## New Plugins

| Plugin | Version | Link |
|--------|---------|------|
`)
		for _, p := range newPlugins {
			sb.WriteString(fmt.Sprintf("| %s | %s | [Download](%s) |\n", p.PluginName, p.PluginVersion, p.URL))
		}
		sb.WriteString("\n")
	}

	if updatedPlugins := pluginsByStatus[release.StatusUpdated]; len(updatedPlugins) > 0 {
		sb.WriteString(`## Updated Plugins

| Plugin | Version | Link |
|--------|---------|------|
`)
		for _, p := range updatedPlugins {
			sb.WriteString(fmt.Sprintf("| %s | %s | [Download](%s) |\n", p.PluginName, p.PluginVersion, p.URL))
		}
		sb.WriteString("\n")
	}

	if existingPlugins := pluginsByStatus[release.StatusExisting]; len(existingPlugins) > 0 {
		sb.WriteString("## Previously Released Plugins\n\n")
		sb.WriteString(fmt.Sprintf("The previously released plugins can be found in the [plugin-releases.json](%s) file.\n", c.pluginReleasesURL(name)))
	}

	if !privateKey.Equal(minisign.PrivateKey{}) {
		publicKey, ok := privateKey.Public().(minisign.PublicKey)
		if !ok {
			return "", errors.New("failed to retrieve minisign public key from private key")
		}
		sb.WriteString("## Verifying a release\n\n")
		sb.WriteString("Releases are signed using our [minisign](https://github.com/jedisct1/minisign) public key:\n\n")
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", publicKey.String()))
		sb.WriteString("The release assets can be verified using this command (assuming that minisign is installed):\n\n")
		releasesFile := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", c.githubReleaseOwner, release.GithubRepoPlugins, name, release.PluginReleasesFile)
		sb.WriteString(fmt.Sprintf("```\ncurl -OL %s && \\\n", releasesFile))
		sb.WriteString(fmt.Sprintf("  curl -OL %s && \\\n", releasesFile+".minisig"))
		sb.WriteString(fmt.Sprintf("  minisign -Vm %s -P %s\n```\n", release.PluginReleasesFile, publicKey.String()))
	}
	return sb.String(), nil
}

func signPluginReleases(dir string, privateKey minisign.PrivateKey) error {
	releasesFile := filepath.Join(dir, release.PluginReleasesFile)
	if privateKey.Equal(minisign.PrivateKey{}) { // Private key not initialized
		log.Printf("skipping signing of %s", releasesFile)
		return nil
	}
	log.Printf("signing: %s", releasesFile)
	releasesFileBytes, err := os.ReadFile(releasesFile)
	if err != nil {
		return err
	}
	signature := minisign.Sign(privateKey, releasesFileBytes)
	if err := os.WriteFile(filepath.Join(dir, release.PluginReleasesSignatureFile), signature, 0644); err != nil { //nolint:gosec
		return err
	}
	return nil
}

func createPluginZip(ctx context.Context, basedir string, plugin *plugin.Plugin, registryImage string, imageID string) (string, error) {
	if err := pullImage(ctx, registryImage); err != nil {
		return "", err
	}
	zipName := pluginZipName(plugin)
	pluginTempDir, err := os.MkdirTemp(basedir, strings.TrimSuffix(zipName, filepath.Ext(zipName)))
	if err != nil {
		return "", err
	}
	defer func() {
		if err := os.RemoveAll(pluginTempDir); err != nil {
			log.Printf("failed to remove %q: %v", pluginTempDir, err)
		}
	}()
	if err := saveImageToDir(ctx, imageID, pluginTempDir); err != nil {
		return "", err
	}
	log.Printf("creating %s", zipName)
	zipFile := filepath.Join(basedir, zipName)
	zf, err := os.OpenFile(zipFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := zf.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
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
	digest, err := release.CalculateDigest(zipFile)
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

func saveImageToDir(ctx context.Context, imageRef string, dir string) error {
	cmd := dockerCmd(ctx, "save", imageRef, "-o", "image.tar")
	cmd.Dir = dir
	return cmd.Run()
}

func createPluginReleases(dir string, plugins []release.PluginRelease) error {
	f, err := os.OpenFile(filepath.Join(dir, release.PluginReleasesFile), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
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
	return encoder.Encode(&release.PluginReleases{Releases: plugins})
}

func pullImage(ctx context.Context, name string) error {
	log.Printf("pulling image: %s", name)
	return dockerCmd(ctx, "pull", name).Run()
}

func dockerCmd(ctx context.Context, command string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "docker", append([]string{command}, args...)...) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
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

func (c *command) pluginDownloadURL(plugin *plugin.Plugin, releaseName string) string {
	zipName := pluginZipName(plugin)
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", c.githubReleaseOwner, release.GithubRepoPlugins, releaseName, zipName)
}

func (c *command) pluginReleasesURL(releaseName string) string {
	return fmt.Sprintf("https://github.com/bufbuild/plugins/releases/download/%s/%s", releaseName, release.PluginReleasesFile)
}

func pluginZipName(plugin *plugin.Plugin) string {
	identity := plugin.Identity
	return fmt.Sprintf("%s-%s-%s.zip", identity.Owner(), identity.Plugin(), plugin.PluginVersion)
}

func fetchRegistryImageAndImageID(plugin *plugin.Plugin) (string, string, error) {
	identity := plugin.Identity
	imageName := fmt.Sprintf("ghcr.io/%s/plugins-%s-%s:%s", release.GithubOwnerBufbuild, identity.Owner(), identity.Plugin(), plugin.PluginVersion)
	parsedName, err := name.ParseReference(imageName)
	if err != nil {
		return "", "", err
	}
	remoteImage, err := remote.Image(parsedName, remote.WithAuthFromKeychain(githubkeychain.Keychain))
	if err != nil {
		return "", "", err
	}
	manifest, err := remoteImage.Manifest()
	if err != nil {
		return "", "", err
	}
	remoteDigest, err := remoteImage.Digest()
	if err != nil {
		return "", "", err
	}
	return fmt.Sprintf("%s@%s", imageName, remoteDigest.String()), manifest.Config.Digest.String(), nil
}
