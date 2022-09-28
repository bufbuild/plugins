package source

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/multierr"
)

var (
	ErrSourceFileNotFound = errors.New("source file not found")
)

func GatherConfigs(root string) (_ []*Config, retErr error) {
	filenames, err := gatherSourceFilenames(root)
	if err != nil {
		return nil, err
	}
	configs := make([]*Config, 0, len(filenames))
	for _, filename := range filenames {
		config, err := loadConfigFile(filename)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}
	return configs, nil
}

func loadConfigFile(filename string) (_ *Config, retErr error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, file.Close())
	}()
	config, err := NewConfig(file)
	if err != nil {
		return nil, err
	}
	config.Filename = filename
	return config, nil
}

func gatherSourceFilenames(root string) ([]string, error) {
	var filenames []string
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".": // Allow relative path to current directory
			case "cmd", "internal", "tests":
				return filepath.SkipDir
			default:
				if strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
			}
		} else if d.Name() == "source.yaml" {
			filenames = append(filenames, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if len(filenames) == 0 {
		return nil, ErrSourceFileNotFound
	}
	return filenames, nil
}
