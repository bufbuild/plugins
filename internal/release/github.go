package release

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aead.dev/minisign"
	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v72/github"
)

type GithubOwner string
type GithubRepo string

const (
	GithubOwnerBufbuild GithubOwner = "bufbuild"
	GithubRepoPlugins   GithubRepo  = "plugins"
)

var ErrNotFound = errors.New("release not found")

type Client struct {
	GitHub *github.Client
}

// NewClient returns a new HTTP client which can be used to perform actions on GitHub releases.
// The returned client is authenticated if the GITHUB_TOKEN environment variable is set.
// The returned context.Context is altered to support rate limiting.
func NewClient(ctx context.Context) (context.Context, *Client) {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken != "" {
		log.Printf("creating authenticated GitHub client with GITHUB_TOKEN")
	} else {
		log.Printf("creating unauthenticated GitHub client")
	}
	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(nil)
	if err != nil {
		log.Printf("failed to create rate limiter: %v", err)
		// Fallback to default client if rate limiter creation fails.
		ctx = context.WithValue(ctx, github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)
		rateLimiter = http.DefaultClient
	} else {
		// Disable the github-go rate limiter for github_ratelimit.
		ctx = context.WithValue(ctx, github.BypassRateLimitCheck, true)
	}
	githubClient := github.NewClient(rateLimiter).WithAuthToken(githubToken)
	return ctx, &Client{
		GitHub: githubClient,
	}
}

// CreateRelease creates a new GitHub release under the owner and repo.
func (c *Client) CreateRelease(ctx context.Context, owner GithubOwner, repo GithubRepo, release *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	repositoryRelease, _, err := c.GitHub.Repositories.CreateRelease(ctx, string(owner), string(repo), release)
	if err != nil {
		return nil, err
	}
	return repositoryRelease, nil
}

// EditRelease performs an update of editable properties of a release (i.e. marking it not as a draft).
func (c *Client) EditRelease(ctx context.Context, owner GithubOwner, repo GithubRepo, releaseID int64, releaseChanges *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	release, _, err := c.GitHub.Repositories.EditRelease(ctx, string(owner), string(repo), releaseID, releaseChanges)
	if err != nil {
		return nil, err
	}
	return release, nil
}

// UploadReleaseAsset uploads the specified file to the release.
func (c *Client) UploadReleaseAsset(ctx context.Context, owner GithubOwner, repo GithubRepo, releaseID int64, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	_, _, err = c.GitHub.Repositories.UploadReleaseAsset(ctx, string(owner), string(repo), releaseID, &github.UploadOptions{
		Name: filepath.Base(filename),
	}, f)
	return err
}

// GetLatestRelease returns information about the latest GitHub release for the given org and repo (i.e. 'bufbuild', 'plugins').
// If no release is found, returns ErrReleaseNotFound.
func (c *Client) GetLatestRelease(ctx context.Context, owner GithubOwner, repo GithubRepo) (*github.RepositoryRelease, error) {
	repositoryRelease, resp, err := c.GitHub.Repositories.GetLatestRelease(ctx, string(owner), string(repo))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// no latest release
			return nil, ErrNotFound
		}
		return nil, err
	}
	return repositoryRelease, nil
}

// GetReleaseByTag returns information about a given release (by tag name).
func (c *Client) GetReleaseByTag(ctx context.Context, owner GithubOwner, repo GithubRepo, tag string) (*github.RepositoryRelease, error) {
	repositoryRelease, resp, err := c.GitHub.Repositories.GetReleaseByTag(ctx, string(owner), string(repo), tag)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// no latest release
			return nil, ErrNotFound
		}
		return nil, err
	}
	return repositoryRelease, nil
}

// getOwnerRepoFromReleaseURL parses a URL in the format:
//
//	https://api.github.com/repos/{owner}/{repo}/releases/{release_id}
//
// into '{owner}' and '{repo}'. If not in the expected format, it will return a non-nil error.
func getOwnerRepoFromReleaseURL(url string) (string, string, error) {
	_, ownerRepo, found := strings.Cut(url, "/repos/")
	if !found {
		return "", "", fmt.Errorf("unsupported release URL format - no /repos/ found: %q", url)
	}
	ownerRepo, _, found = strings.Cut(ownerRepo, "/releases/")
	if !found {
		return "", "", fmt.Errorf("unsupported release URL format - no /releases/ found: %q", url)
	}
	owner, repo, found := strings.Cut(ownerRepo, "/")
	if !found {
		return "", "", fmt.Errorf("unsupported release URL format no owner/repo found: %q", url)
	}
	return owner, repo, nil
}

// DownloadPluginReleasesToDir loads the plugin-releases.json file from the specified GitHub release.
// It will additionally verify the minisign signature of the release if passed a valid minisign.PublicKey.
// It will download both the plugin-releases.json and plugin-releases.json.minisig to the specified dir.
func (c *Client) DownloadPluginReleasesToDir(
	ctx context.Context,
	release *github.RepositoryRelease,
	publicKey minisign.PublicKey,
	dir string,
) (*PluginReleases, error) {
	releasesJSONBytes, releasesJSONLastModified, err := c.DownloadAsset(ctx, release, PluginReleasesFile)
	if err != nil {
		return nil, err
	}
	releasesJSONMinisigBytes, releasesJSONMinisigLastModified, err := c.DownloadAsset(ctx, release, PluginReleasesSignatureFile)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if !publicKey.Equal(minisign.PublicKey{}) && !minisign.Verify(publicKey, releasesJSONBytes, releasesJSONMinisigBytes) {
		return nil, fmt.Errorf("release %s file %q doesn't match signature %q", release.GetName(), PluginReleasesFile, PluginReleasesSignatureFile)
	}
	pluginReleasesDownload := filepath.Join(dir, PluginReleasesFile)
	if err := os.WriteFile(pluginReleasesDownload, releasesJSONBytes, 0644); err != nil { //nolint:gosec
		return nil, err
	}
	if err := os.Chtimes(pluginReleasesDownload, releasesJSONLastModified, releasesJSONLastModified); err != nil {
		return nil, err
	}
	if releasesJSONMinisigBytes != nil {
		pluginReleasesSignatureDownload := filepath.Join(dir, PluginReleasesSignatureFile)
		if err := os.WriteFile(pluginReleasesSignatureDownload, releasesJSONMinisigBytes, 0644); err != nil { //nolint:gosec
			return nil, err
		}
		if err := os.Chtimes(pluginReleasesSignatureDownload, releasesJSONMinisigLastModified, releasesJSONMinisigLastModified); err != nil {
			return nil, err
		}
	}
	var releases PluginReleases
	if err := json.Unmarshal(releasesJSONBytes, &releases); err != nil {
		return nil, err
	}
	return &releases, nil
}

// DownloadAsset uses the GitHub API to download the asset with the given name from the release.
// If the asset isn't found, returns ErrNotFound.
func (c *Client) DownloadAsset(ctx context.Context, release *github.RepositoryRelease, assetName string) ([]byte, time.Time, error) {
	var assetID int64
	var assetLastModified time.Time
	for _, asset := range release.Assets {
		if asset.GetName() == assetName {
			assetID = asset.GetID()
			assetLastModified = asset.GetUpdatedAt().Time
			break
		}
	}
	if assetID == 0 {
		return nil, time.Time{}, ErrNotFound
	}
	owner, repo, err := getOwnerRepoFromReleaseURL(release.GetURL())
	if err != nil {
		return nil, time.Time{}, err
	}
	rc, _, err := c.GitHub.Repositories.DownloadReleaseAsset(ctx, owner, repo, assetID, http.DefaultClient)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer func() {
		if err := rc.Close(); err != nil {
			log.Printf("failed to close: %v", err)
		}
	}()
	contents, err := io.ReadAll(rc)
	if err != nil {
		return nil, time.Time{}, err
	}
	return contents, assetLastModified, nil
}
