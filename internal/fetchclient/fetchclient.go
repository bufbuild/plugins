package fetchclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/v48/github"
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
	// docs: https://central.sonatype.org/search/rest-api-guide/
	mavenSearchURL = "https://search.maven.org/solrsearch/select"
	mavenPageSize  = 50
	mavenMaxFetch  = 1000
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
		client = retryablehttp.NewClient().StandardClient()
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
		results, err := c.fetchMaven(ctx, config.Source.Maven.Group, config.Source.Maven.Name)
		if err != nil {
			return "", err
		}
		return results.latestVersion, nil
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
		if versionWithPrefix, ok := ensureSemverPrefix(version.Num); ok {
			versions = append(versions, versionWithPrefix)
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

type mavenResults struct {
	latestVersion       string
	latestPatchVersions map[string]string
}

func (c *Client) fetchMaven(ctx context.Context, group string, name string) (*mavenResults, error) {
	targetURL, err := url.Parse(mavenSearchURL)
	if err != nil {
		return nil, err
	}
	results := &mavenResults{
		latestPatchVersions: make(map[string]string),
	}
	numFound := math.MaxInt
	for start := 0; start < numFound; start += mavenPageSize {
		response, err := c.fetchMavenPage(ctx, targetURL, group, name, start)
		if err != nil {
			return nil, err
		}
		if numFound == math.MaxInt {
			numFound = response.Response.NumFound
		}
		// Limit pagination in case API calls return a large number of versions.
		if numFound > mavenMaxFetch {
			numFound = mavenMaxFetch
		}
		if len(response.Response.Docs) == 0 {
			break
		}
		for _, doc := range response.Response.Docs {
			version := doc.Version
			if !strings.HasPrefix(version, "v") {
				version = "v" + version
			}
			if !semver.IsValid(version) || semver.Prerelease(version) != "" {
				continue
			}
			version = semver.Canonical(version)
			if results.latestVersion == "" || semver.Compare(results.latestVersion, version) < 0 {
				results.latestVersion = version
			}
			release := semver.MajorMinor(version)
			if results.latestPatchVersions[release] == "" || semver.Compare(results.latestPatchVersions[release], version) < 0 {
				results.latestPatchVersions[release] = version
			}
		}
	}
	if results.latestVersion == "" {
		return nil, errors.New("failed to determine latest version from response docs")
	}
	return results, nil
}

type mavenResponse struct {
	Response struct {
		NumFound int `json:"numFound"` //nolint:tagliatelle
		Start    int `json:"start"`
		Docs     []struct {
			ID        string `json:"id"`
			Group     string `json:"g"`
			Artifact  string `json:"a"`
			Version   string `json:"v"`
			Packaging string `json:"p"`
		} `json:"docs"`
	} `json:"response"`
}

func (c *Client) fetchMavenPage(ctx context.Context, targetURL *url.URL, group string, name string, start int) (*mavenResponse, error) {
	q := url.Values{}
	q.Set("wt", "json")
	q.Set("rows", strconv.Itoa(mavenPageSize))
	q.Set("core", "gav")
	q.Set("start", strconv.Itoa(start))
	q.Set("q", fmt.Sprintf("g:%s+AND+a:%s", group, name))
	unescapedQuery, err := url.QueryUnescape(q.Encode())
	if err != nil {
		return nil, err
	}
	targetURL.RawQuery = unescapedQuery
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL.String(), nil)
	if err != nil {
		return nil, err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var data mavenResponse
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
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
			if version, ok := ensureSemverPrefix(*tag.Name); ok {
				versions = append(versions, version)
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

// ensureSemverPrefix ensures the given version is valid semver, optionally
// prefixing with "v". The output version is not guaranteed to be the same
// as input.
func ensureSemverPrefix(version string) (string, bool) {
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if !semver.IsValid(version) {
		return "", false
	}
	return version, true
}
