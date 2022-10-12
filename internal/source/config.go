package source

import (
	"io"
	"strconv"

	"gopkg.in/yaml.v3"
)

// NewConfig returns a new config.
func NewConfig(reader io.Reader) (*Config, error) {
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)
	var config *Config
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	return config, nil
}

// Cacheable indicates that a given config can be cached by the returned key.
// This allows duplicate configurations to only fetch latest versions once.
type Cacheable interface {
	CacheKey() string
}

// Config is the source config.
type Config struct {
	Filename string       `yaml:"-"`
	Source   SourceConfig `yaml:"source"`
	// IncludePrerelease includes semver prereleases when fetching versions
	// from upstream.
	IncludePrerelease bool `yaml:"include_prerelease"`
}

func (c Config) CacheKey() string {
	return c.Source.CacheKey() + "-" + strconv.FormatBool(c.IncludePrerelease)
}

var _ Cacheable = (*Config)(nil)

// SourceConfig is the configuration for the fetch source.
type SourceConfig struct {
	Disabled bool `yaml:"disabled"`
	// Only one field will be set.
	GitHub      *GitHubConfig      `yaml:"github"`
	DartFlutter *DartFlutterConfig `yaml:"dart_flutter"`
	GoProxy     *GoProxyConfig     `yaml:"goproxy"`
	NPMRegistry *NPMRegistryConfig `yaml:"npm_registry"`
	Maven       *MavenConfig       `yaml:"maven"`
}

var _ Cacheable = (*SourceConfig)(nil)

func (s *SourceConfig) Name() string {
	switch {
	case s.GitHub != nil:
		return "github"
	case s.DartFlutter != nil:
		return "dart_flutter"
	case s.GoProxy != nil:
		return "go_proxy"
	case s.NPMRegistry != nil:
		return "npm_registry"
	case s.Maven != nil:
		return "maven"
	}
	return "unknown"
}

func (s *SourceConfig) CacheKey() string {
	name := s.Name()
	switch {
	case s.GitHub != nil:
		return name + "-" + s.GitHub.CacheKey()
	case s.DartFlutter != nil:
		return name + "-" + s.DartFlutter.CacheKey()
	case s.GoProxy != nil:
		return name + "-" + s.GoProxy.CacheKey()
	case s.NPMRegistry != nil:
		return name + "-" + s.NPMRegistry.CacheKey()
	case s.Maven != nil:
		return name + "-" + s.Maven.CacheKey()
	}
	return name
}

// GitHubConfig is the GitHub configuration.
type GitHubConfig struct {
	Owner      string `yaml:"owner"`
	Repository string `yaml:"repository"`
}

var _ Cacheable = (*GitHubConfig)(nil)

func (g GitHubConfig) CacheKey() string {
	return g.Owner + "-" + g.Repository
}

// DartFlutterConfig is the dart and flutter configuration.
type DartFlutterConfig struct {
	Name string `yaml:"name"`
}

var _ Cacheable = (*DartFlutterConfig)(nil)

func (d DartFlutterConfig) CacheKey() string {
	return d.Name
}

// GoProxyConfig is the go proxy configuration.
type GoProxyConfig struct {
	Name string `yaml:"name"`
}

var _ Cacheable = (*GoProxyConfig)(nil)

func (g GoProxyConfig) CacheKey() string {
	return g.Name
}

// NPMRegistryConfig is the npm registry configuration.
type NPMRegistryConfig struct {
	Name string `yaml:"name"`
}

var _ Cacheable = (*NPMRegistryConfig)(nil)

func (n NPMRegistryConfig) CacheKey() string {
	return n.Name
}

// MavenConfig is the maven search configuration.
type MavenConfig struct {
	Group string `yaml:"group"`
	Name  string `yaml:"name"`
}

var _ Cacheable = (*MavenConfig)(nil)

func (m MavenConfig) CacheKey() string {
	return m.Group + "-" + m.Name
}
