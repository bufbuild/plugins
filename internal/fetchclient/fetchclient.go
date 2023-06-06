package fetchclient

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/google/go-github/v50/github"
	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/mod/semver"
	"golang.org/x/oauth2"

	"github.com/bufbuild/plugins/internal/source"
)

const (
	cratesURL = "https://crates.io/api/v1"
	// docs: https://pub.dev/help/api
	dartFlutterAPIURL = "https://pub.dev/api/packages"
	goProxyURL        = "https://proxy.golang.org"
	npmRegistryURL    = "https://registry.npmjs.org"
	mavenURL          = "https://repo1.maven.org/maven2"
)

// Client is a client used to fetch latest package version.
type Client struct {
	httpClient *http.Client
	ghClient   *github.Client
}

// New returns a new client.
func New(ctx context.Context) *Client {
	var client *http.Client
	if ghToken := os.Getenv("GITHUB_TOKEN"); ghToken != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: ghToken},
		)
		client = oauth2.NewClient(ctx, ts)
	} else {
		retryableClient := retryablehttp.NewClient()
		retryableClient.Logger = nil
		client = retryableClient.StandardClient()
	}
	return &Client{
		httpClient: client,
		ghClient:   github.NewClient(client),
	}
}

// Fetch fetches new versions based on the given config and returns a valid semver version
// that can be used with the Go semver package. The version is guaranteed to contain a "v" prefix.
func (c *Client) Fetch(ctx context.Context, config *source.Config) (string, error) {
	version, err := c.fetch(ctx, config)
	if err != nil {
		return "", fmt.Errorf("%s: %w", config.Source.Name(), err)
	}
	validSemver, ok := ensureSemverPrefix(version)
	if !ok {
		return "", fmt.Errorf("%s: invalid semver: %s", config.Source.Name(), version)
	}
	return validSemver, nil
}

func (c *Client) fetch(ctx context.Context, config *source.Config) (string, error) {
	switch {
	case config.Source.GitHub != nil:
		return c.fetchGithub(ctx, config.Source.GitHub.Owner, config.Source.GitHub.Repository)
	case config.Source.DartFlutter != nil:
		return c.fetchDartFlutter(ctx, config.Source.DartFlutter.Name)
	case config.Source.GoProxy != nil:
		return c.fetchGoProxy(ctx, config.Source.GoProxy.Name)
	case config.Source.NPMRegistry != nil:
		return c.fetchNPMRegistry(ctx, config.Source.NPMRegistry.Name)
	case config.Source.Maven != nil:
		return c.fetchMaven(ctx, config.Source.Maven.Group, config.Source.Maven.Name)
	case config.Source.Crates != nil:
		return c.fetchCrate(ctx, config.Source.Crates.CrateName)
	}
	return "", errors.New("failed to match a source")
}

func (c *Client) fetchDartFlutter(ctx context.Context, name string) (string, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/%s", dartFlutterAPIURL, strings.TrimPrefix(name, "/")),
		nil,
	)
	if err != nil {
		return "", err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received status code %d retrieving %q", response.StatusCode, request.URL.String())
	}

	var data struct {
		Latest struct {
			Version string `json:"version"`
		} `json:"latest"`
	}
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return "", err
	}
	return data.Latest.Version, nil
}

func (c *Client) fetchCrate(ctx context.Context, name string) (string, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/crates/%s", cratesURL, strings.TrimPrefix(name, "/")),
		nil,
	)
	if err != nil {
		return "", err
	}
	// See https://github.com/bufbuild/plugins/issues/252 for more information.
	// We must be careful with this API and respect the crawling policy.
	request.Header.Set("User-Agent", "bufbuild (github.com/bufbuild/plugins)")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received status code %d retrieving %q", response.StatusCode, request.URL.String())
	}

	var data struct {
		Versions []struct {
			Yanked bool   `json:"yanked"`
			Num    string `json:"num"`
		} `json:"versions"`
	}
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return "", err
	}
	var versions []string
	for _, version := range data.Versions {
		if version.Yanked {
			// A yanked version a is a published crate's version that has been removed
			// from the server's index.
			continue
		}
		if v, ok := ensureSemverPrefix(version.Num); ok {
			versions = append(versions, v)
		}
	}
	if len(versions) == 0 {
		return "", errors.New("no versions found")
	}
	semver.Sort(versions)
	return versions[len(versions)-1], nil
}

func (c *Client) fetchGoProxy(ctx context.Context, name string) (string, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/%s/@latest", goProxyURL, strings.TrimPrefix(name, "/")),
		nil,
	)
	if err != nil {
		return "", err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received status code %d retrieving %q", response.StatusCode, request.URL.String())
	}

	var data struct {
		Version string `json:"Version"` //nolint:tagliatelle
	}
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return "", err
	}
	return data.Version, nil
}

func (c *Client) fetchNPMRegistry(ctx context.Context, name string) (string, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/%s", npmRegistryURL, strings.TrimPrefix(name, "/")),
		nil,
	)
	if err != nil {
		return "", err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received status code %d retrieving %q", response.StatusCode, request.URL.String())
	}

	var data struct {
		DistTags struct {
			Latest string `json:"latest"`
		} `json:"dist-tags"` //nolint:tagliatelle
	}
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return "", err
	}
	return data.DistTags.Latest, nil
}

func (c *Client) fetchMaven(ctx context.Context, group string, name string) (string, error) {
	groupComponents := strings.Split(group, ".")
	targetURL, err := url.JoinPath(mavenURL, append(groupComponents, name, "maven-metadata.xml")...)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return "", err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received status code %d retrieving %q", response.StatusCode, request.URL.String())
	}
	var metadata struct {
		GroupID    string `xml:"groupId"`
		ArtifactID string `xml:"artifactId"`
		Versioning struct {
			Latest      string   `xml:"latest"`
			Release     string   `xml:"release"`
			Versions    []string `xml:"versions>version"`
			LastUpdated string   `xml:"lastUpdated"`
		} `xml:"versioning"`
	}
	if err := xml.NewDecoder(response.Body).Decode(&metadata); err != nil {
		return "", err
	}
	latestVersion := ""
	for _, version := range metadata.Versioning.Versions {
		v, ok := ensureSemverPrefix(version)
		if !ok {
			continue
		}
		v = semver.Canonical(v)
		if latestVersion == "" || semver.Compare(latestVersion, v) < 0 {
			latestVersion = v
		}
	}
	if latestVersion == "" {
		return "", errors.New("failed to determine latest version from metadata")
	}
	return latestVersion, nil
}

func (c *Client) fetchGithub(ctx context.Context, owner string, repository string) (string, error) {
	// With the GitHub API we have a few options:
	//
	// ✅ 1. list all git tags
	// 		https://docs.github.com/en/rest/repos/repos#list-repository-tags
	// ❌ 2. get latest by release only (does not include prereleases)
	// 		https://docs.github.com/en/rest/releases/releases#get-the-latest-release
	// ❌ 3. list all releases (does not include regular Git tags that have not been associated with a release)
	// 		https://docs.github.com/en/rest/releases/releases#list-releases
	var page int
	var versions []string
	for {
		tags, response, err := c.ghClient.Repositories.ListTags(ctx, owner, repository, &github.ListOptions{
			Page:    page,
			PerPage: 100,
		})
		if err != nil {
			return "", err
		}
		for _, tag := range tags {
			if tag.Name == nil {
				continue
			}
			if v, ok := ensureSemverPrefix(*tag.Name); ok {
				versions = append(versions, v)
			}
		}
		page = response.NextPage
		if page == 0 {
			break
		}
	}
	if len(versions) == 0 {
		return "", errors.New("no versions found")
	}
	semver.Sort(versions)
	return versions[len(versions)-1], nil
}

// ensureSemverPrefix checks if the given version is valid semver, optionally
// prefixing with "v". The output version is not guaranteed to be the same
// as input. This function returns false if the version is not valid semver or
// is a prerelease.
func ensureSemverPrefix(version string) (string, bool) {
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if !semver.IsValid(version) {
		return "", false
	}
	if semver.Prerelease(version) != "" {
		return "", false
	}
	return version, true
}
