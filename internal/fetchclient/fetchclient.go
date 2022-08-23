package fetchclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/bufbuild/plugins/internal/source"
	"github.com/google/go-github/v45/github"
	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/mod/semver"
)

const (
	// docs: https://pub.dev/help/api
	dartFlutterAPIURL = "https://pub.dev/api/packages"
	goProxyURL        = "https://proxy.golang.org"
	npmRegistryURL    = "https://registry.npmjs.org"
	// docs: https://central.sonatype.org/search/rest-api-guide/
	mavenSearchURL = "https://search.maven.org/solrsearch/select"
)

// Client is a client used to fetch latest package version.
type Client struct {
	httpClient *http.Client
	ghClient   *github.Client
}

// New returns a new client.
func New() *Client {
	return newClient(retryablehttp.NewClient().StandardClient())
}

func newClient(client *http.Client) *Client {
	return &Client{
		httpClient: client,
		ghClient:   github.NewClient(client),
	}
}

// Fetch fetches new versions based on the given config and returns a valid semver version
// that can be used with the Go semver package. The version is gauranteed to contain a "v" prefix.
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
	}
	return "", fmt.Errorf("failed to match a source")
}

func (c *Client) fetchDartFlutter(ctx context.Context, name string) (string, error) {
	request, err := http.NewRequest(
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

func (c *Client) fetchGoProxy(ctx context.Context, name string) (string, error) {
	request, err := http.NewRequest(
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
		Version string `json:"Version"`
	}
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return "", err
	}
	return data.Version, nil
}

func (c *Client) fetchNPMRegistry(ctx context.Context, name string) (string, error) {
	request, err := http.NewRequest(
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
		} `json:"dist-tags"`
	}
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return "", err
	}
	return data.DistTags.Latest, nil
}

func (c *Client) fetchMaven(ctx context.Context, group string, name string) (string, error) {
	targetURL, err := url.Parse(mavenSearchURL)
	if err != nil {
		return "", err
	}
	q := url.Values{}
	q.Set("wt", "json")
	q.Set("rows", "20")
	q.Set("core", "gav")
	q.Set("q", fmt.Sprintf("g:%s+AND+a:%s", group, name))
	unescapedQuery, err := url.QueryUnescape(q.Encode())
	if err != nil {
		return "", err
	}
	targetURL.RawQuery = unescapedQuery
	request, err := http.NewRequest(http.MethodGet, targetURL.String(), nil)
	if err != nil {
		return "", err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	var data struct {
		Response struct {
			Docs []struct {
				ID       string `json:"id"`
				Group    string `json:"g"`
				Artifact string `json:"a"`
				Version  string `json:"v"`
			} `json:"docs"`
		} `json:"response"`
	}
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return "", err
	}
	if len(data.Response.Docs) == 0 {
		return "", errors.New("failed to get versions from response docs")
	}
	return data.Response.Docs[0].Version, nil
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
			if tag.Name != nil {
				version, ok := ensureSemverPrefix(*tag.Name)
				if ok {
					versions = append(versions, version)
					continue
				}
				log.Printf("fetchclient: skipping invalid semver %s for package %s/%s\n", *tag.Name, owner, repository)
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
