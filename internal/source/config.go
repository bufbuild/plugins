package source

import (
	"io"

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

// Config is the source config.
type Config struct {
	Filename string       `yaml:"-"`
	Source   SourceConfig `yaml:"source"`
	// IncludePrerelease includes semver prereleases when fecthing versions
	// from upstream.
	IncludePrerelease bool `yaml:"include_prerelease"`
}

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

// GitHubConfig is the github configuration.
type GitHubConfig struct {
	Owner      string `yaml:"owner"`
	Repository string `yaml:"repository"`
}

// DartFlutterConfig is the dart and flutter configuration.
type DartFlutterConfig struct {
	Name string `yaml:"name"`
}

// GoProxyConfig is the go proxy configuration.
type GoProxyConfig struct {
	Name string `yaml:"name"`
}

// NPMRegistryConfig is the npm registry configuration.
type NPMRegistryConfig struct {
	Name string `yaml:"name"`
}

// MavenConfig is the maven search configuration.
type MavenConfig struct {
	Group string `yaml:"group"`
	Name  string `yaml:"name"`
}
