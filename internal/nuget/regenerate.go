package nuget

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
)

var selfClosingTagPattern = regexp.MustCompile(`(<\w+[^>]*?)></\w+>`)

// RegenerateNugetDeps processes a NuGet plugin version directory by
// collecting all transitive NuGet dependencies from the plugin's
// buf.plugin.yaml and regenerating the build.csproj file.
func RegenerateNugetDeps(pluginVersionDir, pluginsDir string) error {
	yamlPath := filepath.Join(pluginVersionDir, "buf.plugin.yaml")
	pluginConfig, err := bufremotepluginconfig.ParseConfig(yamlPath)
	if err != nil {
		return err
	}
	if pluginConfig.Registry == nil || pluginConfig.Registry.Nuget == nil {
		return nil
	}
	dependencies, err := collectAllNugetDeps(pluginConfig, pluginsDir)
	if err != nil {
		return fmt.Errorf("collecting nuget deps: %w", err)
	}
	csproj, err := renderCsproj(pluginConfig.Registry.Nuget.TargetFrameworks, dependencies)
	if err != nil {
		return fmt.Errorf("rendering csproj: %w", err)
	}
	csprojPath := filepath.Join(pluginVersionDir, "build.csproj")
	if err := os.WriteFile(csprojPath, []byte(csproj), 0644); err != nil { //nolint:gosec // file permissions are intentional
		return fmt.Errorf("writing build.csproj: %w", err)
	}
	return nil
}

// nugetDep represents a single NuGet package dependency.
type nugetDep struct {
	name    string
	version string
}

// collectAllNugetDeps walks the plugin's dependency tree and collects
// all NuGet dependencies, including transitive ones from plugin deps.
// Dependencies from deeper in the tree are collected first, matching
// the order used by the test's populateNugetDeps function.
func collectAllNugetDeps(
	pluginConfig *bufremotepluginconfig.Config,
	pluginsDir string,
) ([]nugetDep, error) {
	dependencies := make(map[string]string)
	visited := make(map[string]bool)
	if err := collectNugetDepsRecursive(pluginConfig, pluginsDir, visited, dependencies); err != nil {
		return nil, err
	}
	result := make([]nugetDep, 0, len(dependencies))
	for name, version := range dependencies {
		result = append(result, nugetDep{name: name, version: version})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].name < result[j].name
	})
	return result, nil
}

func collectNugetDepsRecursive(
	pluginConfig *bufremotepluginconfig.Config,
	pluginsDir string,
	visited map[string]bool,
	dependencies map[string]string,
) error {
	// First recurse into plugin dependencies.
	for _, dep := range pluginConfig.Dependencies {
		depKey := dep.IdentityString() + ":" + dep.Version()
		if visited[depKey] {
			continue
		}
		visited[depKey] = true
		depPath := filepath.Join(
			pluginsDir, dep.Owner(), dep.Plugin(),
			dep.Version(), "buf.plugin.yaml",
		)
		depConfig, err := bufremotepluginconfig.ParseConfig(depPath)
		if err != nil {
			return fmt.Errorf("loading dep config %s from %s: %w", depKey, depPath, err)
		}
		if err := collectNugetDepsRecursive(depConfig, pluginsDir, visited, dependencies); err != nil {
			return err
		}
	}
	// Then collect this plugin's own NuGet deps.
	if pluginConfig.Registry != nil && pluginConfig.Registry.Nuget != nil {
		for _, dep := range pluginConfig.Registry.Nuget.Deps {
			dependencies[dep.Name] = dep.Version
		}
	}
	return nil
}

// packageReference represents a PackageReference element in a csproj file.
type packageReference struct {
	XMLName xml.Name `xml:"PackageReference"`
	Include string   `xml:"Include,attr"`
	Version string   `xml:"Version,attr"`
}

// propertyGroup represents a PropertyGroup element in a csproj file.
type propertyGroup struct {
	XMLName          xml.Name `xml:"PropertyGroup"`
	TargetFramework  string   `xml:"TargetFramework,omitempty"`
	TargetFrameworks string   `xml:"TargetFrameworks,omitempty"`
}

// itemGroup represents an ItemGroup element in a csproj file.
type itemGroup struct {
	XMLName           xml.Name           `xml:"ItemGroup"`
	PackageReferences []packageReference `xml:"PackageReference"`
}

// csharpProject represents a .csproj XML file.
type csharpProject struct {
	XMLName       xml.Name      `xml:"Project"`
	SDK           string        `xml:"Sdk,attr"`
	PropertyGroup propertyGroup `xml:"PropertyGroup"`
	ItemGroup     itemGroup     `xml:"ItemGroup"`
}

// renderCsproj generates a build.csproj file from target frameworks and dependencies.
func renderCsproj(targetFrameworks []string, dependencies []nugetDep) (string, error) {
	project := csharpProject{
		SDK: "Microsoft.NET.Sdk",
	}
	if len(targetFrameworks) == 1 {
		project.PropertyGroup.TargetFramework = targetFrameworks[0]
	} else {
		project.PropertyGroup.TargetFrameworks = strings.Join(targetFrameworks, ";")
	}
	for _, dep := range dependencies {
		project.ItemGroup.PackageReferences = append(project.ItemGroup.PackageReferences, packageReference{
			Include: dep.name,
			Version: dep.version,
		})
	}
	output, err := xml.MarshalIndent(project, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling csproj: %w", err)
	}
	// Convert empty XML elements to self-closing tags to match .csproj conventions.
	result := selfClosingTagPattern.ReplaceAllString(string(output), "$1 />")
	return result + "\n", nil
}
