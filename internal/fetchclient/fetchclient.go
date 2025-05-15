package fetchclient

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"buf.build/go/standard/xslices"
	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v72/github"
	"golang.org/x/mod/semver"

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

var (
	// ErrSemverPrerelease is returned when a version is a pre-release.
	ErrSemverPrerelease = errors.New("pre-release versions are not supported")
)

// Client is a client used to fetch latest package version.
type Client struct {
	httpClient *http.Client
	ghClient   *github.Client
}

// New returns a new client.
func New(ctx context.Context) *Client {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken != "" {
		log.Printf("creating authenticated GitHub client with GITHUB_TOKEN")
	} else {
		log.Printf("creating unauthenticated GitHub client")
	}
	rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(nil)
	if err != nil {
		log.Printf("failed to create rate limiter: %v", err)
		rateLimiter = http.DefaultClient
	}
	githubClient := github.NewClient(rateLimiter).WithAuthToken(githubToken)
	return &Client{
		httpClient: rateLimiter,
		ghClient:   githubClient,
	}
}

// Fetch fetches new versions based on the given config and returns a valid semver version
// that can be used with the Go semver package. The version is guaranteed to contain a "v" prefix.
func (c *Client) Fetch(ctx context.Context, config *source.Config) (string, error) {
	version, err := c.fetch(ctx, config)
	if err != nil {
		return "", fmt.Errorf("%s: %w", config.Source.Name(), err)
	}
	// We must ensure that the version is prefixed with "v" for the semver package.
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if !semver.IsValid(version) {
		return "", fmt.Errorf("%s: invalid semver: %s", config.Source.Name(), version)
	}
	if semver.Prerelease(version) != "" {
		return "", fmt.Errorf("%s: %w: %s", config.Source.Name(), ErrSemverPrerelease, version)
	}
	return version, nil
}

func (c *Client) fetch(ctx context.Context, config *source.Config) (string, error) {
	ignoreVersions := xslices.ToStructMap(config.Source.IgnoreVersions)
	switch {
	case config.Source.GitHub != nil:
		return c.fetchGithub(ctx, config.Source.GitHub.Owner, config.Source.GitHub.Repository, ignoreVersions)
	case config.Source.DartFlutter != nil:
		return c.fetchDartFlutter(ctx, config.Source.DartFlutter.Name, ignoreVersions)
	case config.Source.GoProxy != nil:
		return c.fetchGoProxy(ctx, config.Source.GoProxy.Name, ignoreVersions)
	case config.Source.NPMRegistry != nil:
		return c.fetchNPMRegistry(ctx, config.Source.NPMRegistry.Name, ignoreVersions)
	case config.Source.Maven != nil:
		return c.fetchMaven(ctx, config.Source.Maven.Group, config.Source.Maven.Name, ignoreVersions)
	case config.Source.Crates != nil:
		return c.fetchCrate(ctx, config.Source.Crates.CrateName, ignoreVersions)
	}
	return "", errors.New("failed to match a source")
}

func (c *Client) fetchDartFlutter(
	ctx context.Context,
	name string,
	ignoreVersions map[string]struct{},
) (string, error) {
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
		Versions []struct {
			Version string `json:"version"`
		} `json:"versions"`
	}
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return "", err
	}
	if len(ignoreVersions) == 0 {
		return data.Latest.Version, nil
	}
	var latestVersion string
	for _, version := range data.Versions {
		version, ok := ensureSemverPrefix(version.Version)
		if !ok {
			continue
		}
		if _, ok := ignoreVersions[version]; ok {
			continue
		}
		if latestVersion == "" || semver.Compare(latestVersion, version) < 0 {
			latestVersion = version
		}
	}
	// Shouldn't be possible unless we've ignored all versions
	if latestVersion == "" {
		return "", fmt.Errorf("failed to calculate latest version for dart source %s", name)
	}
	return latestVersion, nil
}

func (c *Client) fetchCrate(ctx context.Context, name string, ignoreVersions map[string]struct{}) (string, error) {
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
		v, ok := ensureSemverPrefix(version.Num)
		if !ok {
			continue
		}
		if _, ok := ignoreVersions[v]; !ok {
			versions = append(versions, v)
		}
	}
	if len(versions) == 0 {
		return "", errors.New("no versions found")
	}
	semver.Sort(versions)
	return versions[len(versions)-1], nil
}

func (c *Client) fetchGoProxy(ctx context.Context, name string, ignoreVersions map[string]struct{}) (string, error) {
	if len(ignoreVersions) > 0 {
		return "", errors.New("ignore_versions not supported yet for go sources")
	}
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

func (c *Client) fetchNPMRegistry(ctx context.Context, name string, ignoreVersions map[string]struct{}) (string, error) {
	if len(ignoreVersions) > 0 {
		return "", errors.New("ignore_versions not supported yet for NPM sources")
	}
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

func (c *Client) fetchMaven(
	ctx context.Context,
	group string,
	name string,
	ignoreVersions map[string]struct{},
) (string, error) {
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
		if _, ok := ignoreVersions[v]; ok {
			continue
		}
		if latestVersion == "" || semver.Compare(latestVersion, v) < 0 {
			latestVersion = v
		}
	}
	if latestVersion == "" {
		return "", errors.New("failed to determine latest version from metadata")
	}
	return latestVersion, nil
}

func (c *Client) fetchGithub(
	ctx context.Context,
	owner string,
	repository string,
	ignoreVersions map[string]struct{},
) (string, error) {
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
				if _, ok := ignoreVersions[v]; !ok {
					versions = append(versions, v)
				}
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
