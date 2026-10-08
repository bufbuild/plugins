package fetchclient

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/google/go-github/v72/github"
	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"

	"github.com/bufbuild/plugins/internal/source"
)

const (
	cratesURL = "https://crates.io/api/v1"
	// docs: https://pub.dev/help/api
	dartFlutterAPIURL = "https://pub.dev/api/packages"
	goProxyURL        = "https://proxy.golang.org"
	npmRegistryURL    = "https://registry.npmjs.org"
	mavenURL          = "https://repo.maven.apache.org/maven2"
	// docs: https://packaging.python.org/en/latest/specifications/simple-repository-api/
	pypiURL = "https://pypi.org/simple"
)

// Client is a client used to fetch package versions.
type Client struct {
	httpClient     *http.Client
	ghClient       *github.Client
	goProxyBaseURL string
	pypiBaseURL    string
}

// New returns a new client.
func New() *Client {
	httpClient := NewHTTPClient()
	ghClient := github.NewClient(httpClient)
	if ghToken := os.Getenv("GITHUB_TOKEN"); ghToken != "" {
		ghClient = ghClient.WithAuthToken(ghToken)
	}
	return &Client{
		httpClient:     httpClient,
		ghClient:       ghClient,
		goProxyBaseURL: goProxyURL,
		pypiBaseURL:    pypiURL,
	}
}

// NewHTTPClient returns an HTTP client which retries transient failures.
func NewHTTPClient() *http.Client {
	retryableClient := retryablehttp.NewClient()
	retryableClient.Logger = nil
	return retryableClient.StandardClient()
}

// FetchVersions returns every stable version published by the source's
// upstream. Each version is valid semver with a "v" prefix. Prereleases and
// versions that are not valid semver are omitted.
func (c *Client) FetchVersions(ctx context.Context, src *source.Source) ([]string, error) {
	versions, err := c.fetchVersions(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", src.Name(), err)
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("%s: no versions found", src.Name())
	}
	return versions, nil
}

func (c *Client) fetchVersions(ctx context.Context, src *source.Source) ([]string, error) {
	switch {
	case src.GitHub != nil:
		return c.fetchGithub(ctx, src.GitHub.Owner, src.GitHub.Repository)
	case src.DartFlutter != nil:
		return c.fetchDartFlutter(ctx, src.DartFlutter.Name)
	case src.GoProxy != nil:
		return c.fetchGoProxy(ctx, src.GoProxy.Name)
	case src.NPMRegistry != nil:
		return c.fetchNPMRegistry(ctx, src.NPMRegistry.Name)
	case src.Maven != nil:
		return c.fetchMaven(ctx, src.Maven.Group, src.Maven.Name)
	case src.Crates != nil:
		return c.fetchCrate(ctx, src.Crates.CrateName)
	case src.PyPI != nil:
		return c.fetchPyPI(ctx, src.PyPI.Name)
	}
	return nil, errors.New("failed to match a source")
}

func (c *Client) fetchDartFlutter(ctx context.Context, name string) (_ []string, retErr error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/%s", dartFlutterAPIURL, strings.TrimPrefix(name, "/")),
		nil,
	)
	if err != nil {
		return nil, err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, response.Body.Close())
	}()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received status code %d retrieving %q", response.StatusCode, request.URL.String())
	}

	var data struct {
		Versions []struct {
			Version string `json:"version"`
		} `json:"versions"`
	}
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return nil, err
	}
	versions := make([]string, 0, len(data.Versions))
	for _, version := range data.Versions {
		if v, ok := ensureSemverPrefix(version.Version); ok {
			versions = append(versions, v)
		}
	}
	return versions, nil
}

func (c *Client) fetchCrate(ctx context.Context, name string) (_ []string, retErr error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/crates/%s", cratesURL, strings.TrimPrefix(name, "/")),
		nil,
	)
	if err != nil {
		return nil, err
	}
	// See https://github.com/bufbuild/plugins/issues/252 for more information.
	// We must be careful with this API and respect the crawling policy.
	request.Header.Set("User-Agent", "bufbuild (github.com/bufbuild/plugins)")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, response.Body.Close())
	}()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received status code %d retrieving %q", response.StatusCode, request.URL.String())
	}

	var data struct {
		Versions []struct {
			Yanked bool   `json:"yanked"`
			Num    string `json:"num"`
		} `json:"versions"`
	}
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return nil, err
	}
	versions := make([]string, 0, len(data.Versions))
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
	return versions, nil
}

func (c *Client) fetchGoProxy(ctx context.Context, name string) ([]string, error) {
	modulePath := strings.TrimPrefix(name, "/")
	escapedPath, err := module.EscapePath(modulePath)
	if err != nil {
		return nil, err
	}
	list, err := c.getGoProxy(ctx, escapedPath+"/@v/list")
	if err != nil {
		return nil, err
	}
	var versions []string
	for line := range strings.Lines(string(list)) {
		if v, ok := ensureSemverPrefix(strings.TrimSpace(line)); ok {
			versions = append(versions, v)
		}
	}
	if len(versions) == 0 {
		return nil, nil
	}
	// Like the go command, honor retractions declared by the highest version.
	highestVersion := slices.MaxFunc(versions, semver.Compare)
	escapedVersion, err := module.EscapeVersion(highestVersion)
	if err != nil {
		return nil, err
	}
	modPath := escapedPath + "/@v/" + escapedVersion + ".mod"
	modData, err := c.getGoProxy(ctx, modPath)
	if err != nil {
		return nil, err
	}
	modFile, err := modfile.ParseLax(modPath, modData, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", modPath, err)
	}
	return slices.DeleteFunc(versions, func(version string) bool {
		return slices.ContainsFunc(modFile.Retract, func(retract *modfile.Retract) bool {
			return semver.Compare(version, retract.Low) >= 0 && semver.Compare(version, retract.High) <= 0
		})
	}), nil
}

func (c *Client) getGoProxy(ctx context.Context, path string) (_ []byte, retErr error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.goProxyBaseURL+"/"+path, nil)
	if err != nil {
		return nil, err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, response.Body.Close())
	}()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received status code %d retrieving %q", response.StatusCode, request.URL.String())
	}
	return io.ReadAll(response.Body)
}

func (c *Client) fetchNPMRegistry(ctx context.Context, name string) (_ []string, retErr error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/%s", npmRegistryURL, strings.TrimPrefix(name, "/")),
		nil,
	)
	if err != nil {
		return nil, err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, response.Body.Close())
	}()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received status code %d retrieving %q", response.StatusCode, request.URL.String())
	}

	var data struct {
		Versions map[string]any `json:"versions"`
	}
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return nil, err
	}
	versions := make([]string, 0, len(data.Versions))
	for version := range data.Versions {
		if v, ok := ensureSemverPrefix(version); ok {
			versions = append(versions, v)
		}
	}
	return versions, nil
}

func (c *Client) fetchMaven(ctx context.Context, group string, name string) (_ []string, retErr error) {
	groupComponents := strings.Split(group, ".")
	targetURL, err := url.JoinPath(mavenURL, append(groupComponents, name, "maven-metadata.xml")...)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, response.Body.Close())
	}()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received status code %d retrieving %q", response.StatusCode, request.URL.String())
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
		return nil, err
	}
	versions := make([]string, 0, len(metadata.Versioning.Versions))
	for _, version := range metadata.Versioning.Versions {
		if v, ok := ensureSemverPrefix(version); ok {
			versions = append(versions, semver.Canonical(v))
		}
	}
	return versions, nil
}

func (c *Client) fetchGithub(ctx context.Context, owner string, repository string) ([]string, error) {
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
			return nil, err
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
	return versions, nil
}

func (c *Client) fetchPyPI(ctx context.Context, name string) (_ []string, retErr error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/%s/", c.pypiBaseURL, strings.TrimPrefix(name, "/")),
		nil,
	)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.pypi.simple.v1+json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, response.Body.Close())
	}()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received status code %d retrieving %q", response.StatusCode, request.URL.String())
	}

	var data struct {
		Versions []string `json:"versions"`
	}
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return nil, err
	}
	versions := make([]string, 0, len(data.Versions))
	for _, version := range data.Versions {
		if v, ok := ensureSemverPrefix(version); ok {
			versions = append(versions, v)
		}
	}
	return versions, nil
}

// ensureSemverPrefix checks if the given version is valid semver, optionally
// prefixing with "v". The output version is not guaranteed to be the same
// as input. This function returns false if the version is not valid semver, is
// a prerelease, or has build metadata.
func ensureSemverPrefix(version string) (string, bool) {
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if !semver.IsValid(version) {
		return "", false
	}
	if semver.Prerelease(version) != "" || semver.Build(version) != "" {
		return "", false
	}
	return version, true
}
