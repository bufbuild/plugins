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

	"aead.dev/minisign"
	"github.com/google/go-github/v48/github"
	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/oauth2"
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
func NewClient(ctx context.Context) *Client {
	var httpClient *http.Client
	if ghToken := os.Getenv("GITHUB_TOKEN"); ghToken != "" {
		log.Printf("creating authenticated client with GITHUB_TOKEN")
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: ghToken},
		)
		httpClient = oauth2.NewClient(ctx, ts)
	} else {
		client := retryablehttp.NewClient()
		client.Logger = nil
		httpClient = client.StandardClient()
	}
	return &Client{
		GitHub: github.NewClient(httpClient),
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
// into '{owner}' and '{repo}'. If not in the expected format, it will return an non-nil error.
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

// LoadPluginReleases loads the plugin-releases.json file from the specified GitHub release.
// It will additionally verify the minisign signature of the release if passed a valid minisign.PublicKey.
func (c *Client) LoadPluginReleases(ctx context.Context, release *github.RepositoryRelease, publicKey minisign.PublicKey) (*PluginReleases, error) {
	releasesJSONBytes, err := c.downloadAsset(ctx, release, PluginReleasesFile)
	if err != nil {
		return nil, err
	}
	releasesJSONMinisigBytes, err := c.downloadAsset(ctx, release, PluginReleasesSignatureFile)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if !publicKey.Equal(minisign.PublicKey{}) && !minisign.Verify(publicKey, releasesJSONBytes, releasesJSONMinisigBytes) {
		return nil, fmt.Errorf("release %s file %q doesn't match signature %q", release.GetName(), PluginReleasesFile, PluginReleasesSignatureFile)
	}
	var releases PluginReleases
	if err := json.Unmarshal(releasesJSONBytes, &releases); err != nil {
		return nil, err
	}
	return &releases, nil
}

// downloadAsset uses the GitHub API to download the asset with the given name from the release.
// If the asset isn't found, returns ErrNotFound.
func (c *Client) downloadAsset(ctx context.Context, release *github.RepositoryRelease, assetName string) ([]byte, error) {
	var assetID int64
	for _, asset := range release.Assets {
		if asset.GetName() == assetName {
			assetID = asset.GetID()
			break
		}
	}
	if assetID == 0 {
		return nil, ErrNotFound
	}
	owner, repo, err := getOwnerRepoFromReleaseURL(release.GetURL())
	if err != nil {
		return nil, err
	}
	rc, _, err := c.GitHub.Repositories.DownloadReleaseAsset(ctx, owner, repo, assetID, http.DefaultClient)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rc.Close(); err != nil {
			log.Printf("failed to close: %v", err)
		}
	}()
	return io.ReadAll(rc)
}
