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
	// Walk entire directory with a depth of 1
	filenames, err := gatherSourceFilenames("testdata/success", 1)
	require.NoError(t, err)
	assert.Equal(t, 2, len(filenames))
	// Walk explicit directory with a depth 0 (source is expected to be in the root directory)
	filenames, err = gatherSourceFilenames("testdata/success/connect-go", 0)
	require.NoError(t, err)
	assert.Equal(t, 1, len(filenames))
	// Wrong depth, no source files found in root
	_, err = gatherSourceFilenames("testdata/success", 0)
	require.ErrorIs(t, err, ErrSourceFileNotFound)
	// Wrong depth, no source files found in subdirectory. Too far.
	_, err = gatherSourceFilenames("testdata/success", 2)
	require.ErrorIs(t, err, ErrSourceFileNotFound)

	// One directory does not contain source file.
	_, err = gatherSourceFilenames("testdata/fail", 0)
	require.ErrorIs(t, err, ErrSourceFileNotFound)

	// Invalid directory
	_, err = gatherSourceFilenames("notexists", 0)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestLoadSourceFile(t *testing.T) {
	filenames, err := gatherSourceFilenames("testdata/success/connect-go", 0)
	require.NoError(t, err)
	assert.Equal(t, 1, len(filenames))
	config, err := loadConfigFile(filenames[0])
	require.NoError(t, err)
	assert.Equal(t, filenames[0], config.Filename)
}

func TestGatherConfigs(t *testing.T) {
	configs, err := GatherConfigs("testdata/success", 1)
	require.NoError(t, err)
	assert.Equal(t, 2, len(configs))

	for _, config := range configs {
		name := filepath.Base(filepath.Dir(config.Filename))
		switch name {
		case "connect-go":
			source := config.Source.GitHub
			require.NotNil(t, source)
			assert.Equal(t, source.Owner, "bufbuild")
			assert.Equal(t, source.Repository, "connect-go")
			assert.Nil(t, config.Source.DartFlutter)
		case "connect-web":
			source := config.Source.NPMRegistry
			require.NotNil(t, source)
			assert.Equal(t, source.Name, "@bufbuild/protoc-gen-connect-web")
			assert.Equal(t, true, config.Source.Disabled)
			assert.Nil(t, config.Source.DartFlutter)
		default:
			assert.FailNow(t, "unknown plugin name", name)
		}
	}

	// invalid source file
	_, err = GatherConfigs("testdata/fail/invalid", 1)
	require.Error(t, err)
}

func TestConfigLoad(t *testing.T) {
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
