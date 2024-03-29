package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatherSourceFilenames(t *testing.T) {
	t.Parallel()
	// Walk entire directory with a depth of 1
	filenames, err := gatherSourceFilenames("testdata/success")
	require.NoError(t, err)
	assert.Len(t, filenames, 2)
	filenames, err = gatherSourceFilenames("testdata/success/connect-go")
	require.NoError(t, err)
	assert.Len(t, filenames, 1)
	filenames, err = gatherSourceFilenames("testdata/success")
	require.NoError(t, err)
	assert.Len(t, filenames, 2)

	filenames, err = gatherSourceFilenames("testdata/fail")
	require.NoError(t, err)
	assert.Len(t, filenames, 2)

	// Invalid directory
	_, err = gatherSourceFilenames("notexists")
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestLoadSourceFile(t *testing.T) {
	t.Parallel()
	filenames, err := gatherSourceFilenames("testdata/success/connect-go")
	require.NoError(t, err)
	assert.Len(t, filenames, 1)
	config, err := loadConfigFile(filenames[0])
	require.NoError(t, err)
	assert.Equal(t, filenames[0], config.Filename)
}

func TestGatherConfigs(t *testing.T) {
	t.Parallel()
	configs, err := GatherConfigs("testdata/success")
	require.NoError(t, err)
	assert.Len(t, configs, 2)

	for _, config := range configs {
		name := filepath.Base(filepath.Dir(config.Filename))
		switch name {
		case "connect-go":
			source := config.Source.GitHub
			require.NotNil(t, source)
			assert.Equal(t, "bufbuild", source.Owner)
			assert.Equal(t, "connect-go", source.Repository)
			assert.Nil(t, config.Source.DartFlutter)
		case "connect-web":
			source := config.Source.NPMRegistry
			require.NotNil(t, source)
			assert.Equal(t, "@bufbuild/protoc-gen-connect-web", source.Name)
			assert.True(t, config.Source.Disabled)
			assert.Nil(t, config.Source.DartFlutter)
		default:
			assert.FailNow(t, "unknown plugin name", name)
		}
	}

	// invalid source file
	_, err = GatherConfigs("testdata/fail/invalid")
	require.Error(t, err)
}

func TestConfigLoad(t *testing.T) {
	t.Parallel()
	// Strict marshal, detect unknown fields and fail fast.
	sourceData := `source:
	disabled: true
	unknown_field: true
	npm_registry:
	  name: "@bufbuild/protoc-gen-connect-web"
  `
	_, err := NewConfig(strings.NewReader(sourceData))
	require.Error(t, err)

	_, err = NewConfig(strings.NewReader(""))
	require.Error(t, err)
	_, err = loadConfigFile("unknown_dir")
	require.Error(t, err)
}
