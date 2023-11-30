package docker

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

var (
	distrolessNamePrefix = "gcr.io/distroless/"
)

// BaseImages contains state about the latest versions of base images in the .github/docker directory.
// These are automatically kept up to date by dependabot.
type BaseImages struct {
	latestVersions             map[string]string
	latestDistrolessImageNames map[string]string
}

// ImageNameAndVersion returns the latest image name and version (if tracked in the .github/docker directory).
// For example, passing "debian" will return "debian:bookworm-yyyyMMdd" (where "yyyyMMdd" is the latest image date).
// If the image is not tracked in the .github/docker directory, returns an empty string.
// This is used to automate updating Dockerfile base image versions when fetching new versions of plugins.
func (b *BaseImages) ImageNameAndVersion(imageName string) string {
	latestImageName := imageName
	if nameWithoutVersions := distrolessImageNameWithoutVersions(imageName); nameWithoutVersions != "" {
		latestImageName = b.latestDistrolessImageNames[nameWithoutVersions]
	}
	latestVersion, ok := b.latestVersions[latestImageName]
	if !ok {
		return ""
	}
	return latestImageName + ":" + latestVersion
}

// ImageVersion returns the latest version for the image name (if tracked in the .github/docker directory).
// For example, passing "debian" will return "bookworm-yyyyMMdd" (where "yyyyMMdd" is the latest image date).
// If the image is not tracked in the .github/docker directory, returns an empty string.
func (b *BaseImages) ImageVersion(imageName string) string {
	latestImageName := imageName
	if nameWithoutVersions := distrolessImageNameWithoutVersions(imageName); nameWithoutVersions != "" {
		latestImageName = b.latestDistrolessImageNames[nameWithoutVersions]
	}
	return b.latestVersions[latestImageName]
}

// FindBaseImageDir looks for the .github/docker folder starting from basedir.
// It continues to search through parent directories till found (or at the root).
func FindBaseImageDir(basedir string) (string, error) {
	// Walk up from plugins dir to find .github dir
	rootDir, err := filepath.Abs(basedir)
	if err != nil {
		return "", err
	}
	var dockerDir string
	for {
		dockerDir = filepath.Join(rootDir, ".github", "docker")
		if st, err := os.Stat(dockerDir); err == nil && st.IsDir() {
			break
		}
		newRootDir := filepath.Dir(rootDir)
		if newRootDir == rootDir {
			return "", fmt.Errorf("failed to find .github directory from %s", basedir)
		}
		rootDir = newRootDir
	}
	return dockerDir, nil
}

// LoadLatestBaseImages returns the latest base image information from images found in the .github/docker directory.
func LoadLatestBaseImages(baseImageDir string) (_ *BaseImages, retErr error) {
	d, err := os.Open(baseImageDir)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, d.Close())
	}()
	entries, err := d.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	latestVersions := make(map[string]string, len(entries))
	latestDistrolessImages := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "Dockerfile") {
			continue
		}
		imageName, version, err := parseDockerfileBaseImageNameVersion(filepath.Join(baseImageDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if _, ok := latestVersions[imageName]; ok {
			return nil, fmt.Errorf("found duplicate dockerfiles for image %q", imageName)
		}
		latestVersions[imageName] = version
		if imageNameWithoutVersions := distrolessImageNameWithoutVersions(imageName); imageNameWithoutVersions != "" {
			if _, ok := latestDistrolessImages[imageNameWithoutVersions]; ok {
				return nil, fmt.Errorf("found duplicate distroless dockerfiles for image %q", imageNameWithoutVersions)
			}
			latestDistrolessImages[imageNameWithoutVersions] = imageName
		}
	}
	return &BaseImages{
		latestVersions:             latestVersions,
		latestDistrolessImageNames: latestDistrolessImages,
	}, nil
}

func parseDockerfileBaseImageNameVersion(dockerfile string) (_ string, _ string, retErr error) {
	f, err := os.Open(dockerfile)
	if err != nil {
		return "", "", nil
	}
	defer func() {
		retErr = errors.Join(retErr, f.Close())
	}()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if !strings.EqualFold(fields[0], "from") {
			continue
		}
		var image string
		for i := 1; i < len(fields); i++ {
			if strings.HasPrefix(fields[i], "--") {
				// Ignore --platform and other args
				continue
			}
			image = fields[i]
			break
		}
		if image == "" {
			return "", "", fmt.Errorf("missing image in FROM: %q", line)
		}
		imageName, version, found := strings.Cut(image, ":")
		if !found {
			return "", "", fmt.Errorf("invalid FROM line: %q", line)
		}
		return imageName, version, nil
	}
	if err := s.Err(); err != nil {
		return "", "", err
	}
	return "", "", fmt.Errorf("failed to detect base image in %s", dockerfile)
}

// distrolessImageNameWithoutVersions returns a distroless image name without version numbers.
// If the passed in image name is a distroless image and contains versions, it returns the name without versions.
// Otherwise, it returns an empty string.
func distrolessImageNameWithoutVersions(nameWithVersions string) string {
	if !strings.HasPrefix(nameWithVersions, distrolessNamePrefix) || !strings.ContainsFunc(nameWithVersions, unicode.IsDigit) {
		return ""
	}
	var sb strings.Builder
	for _, r := range nameWithVersions {
		if !unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
