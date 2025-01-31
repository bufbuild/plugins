package docker

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindBaseImageDir(t *testing.T) {
	t.Parallel()
	verifyDir := func(basedir, expected string) {
		baseImageDir, err := FindBaseImageDir(basedir)
		require.NoError(t, err)
		expectedAbs, err := filepath.Abs(expected)
		require.NoError(t, err)
		assert.Equal(t, expectedAbs, baseImageDir)
	}
	expected := "../../.github/docker"
	verifyDir(".", expected)
	verifyDir("..", expected)
	verifyDir("../..", expected)
	verifyDir("../../.github", expected)
	verifyDir("../../.github/docker", expected)
}

func TestBaseImages(t *testing.T) {
	t.Parallel()
	baseImageDir, err := FindBaseImageDir(".")
	require.NoError(t, err)
	baseImages, err := LoadLatestBaseImages(baseImageDir)
	require.NoError(t, err)
	assert.NotEmpty(t, baseImages)
	assert.NotEmpty(t, baseImages.ImageNameAndVersion("debian"))
	assert.Empty(t, baseImages.ImageNameAndVersion("untracked"))
	// Test distroless image upgrades
	javaImage := baseImages.ImageNameAndVersion("gcr.io/distroless/java11-debian11")
	assert.NotEmpty(t, javaImage)
	assert.NotContains(t, javaImage, "java11")   // Should be replaced with a later java image
	assert.NotContains(t, javaImage, "debian11") // Should be replaced with a later debian image
}

func TestBaseImagesNoDuplicateVersions(t *testing.T) {
	t.Parallel()
	_, err := LoadLatestBaseImages("testdata/duplicateversions")
	require.ErrorContains(t, err, "found duplicate dockerfiles")
}

func TestBaseImagesNoDuplicateDistroless(t *testing.T) {
	t.Parallel()
	_, err := LoadLatestBaseImages("testdata/duplicatedistroless")
	require.ErrorContains(t, err, "found duplicate distroless dockerfiles")
}

func TestDistrolessImageNameWithoutVersions(t *testing.T) {
	t.Parallel()
	verifyResult := func(imageName, expectedImage string) {
		imageWithoutVersions := distrolessImageNameWithoutVersions(imageName)
		assert.Equal(t, expectedImage, imageWithoutVersions)
	}
	verifyResult("gcr.io/distroless/base-debian11", "gcr.io/distroless/base-debian")
	verifyResult("gcr.io/distroless/java17-debian11", "gcr.io/distroless/java-debian")
	verifyResult("gcr.io/distroless/static", "")
	verifyResult("debian", "")
}
