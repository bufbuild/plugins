package source

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/multierr"
)

var (
	ErrSourceFileNotFound = errors.New("source file not found")
)

func GatherConfigs(root string, depth int) (_ []*Config, retErr error) {
	filenames, err := gatherSourceFilenames(root, depth)
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

func gatherSourceFilenames(root string, depth int) ([]string, error) {
	var filenames []string
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			count := strings.Count(strings.TrimPrefix(path, root), string(os.PathSeparator))
			if count == 0 {
				return nil
			} else if count > depth {
				return fs.SkipDir
			}
			files, err := os.ReadDir(path)
			if err != nil {
				return err
			}
			var found bool
			for _, file := range files {
				if file.Name() == "source.yaml" {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("%w: %s", ErrSourceFileNotFound, path)
			}
			return nil
		}
		if d.Name() == "source.yaml" {
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
