package release

import (
	"archive/zip"
	"compress/flate"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bufbuild/plugins/internal/plugin"
)

func addFileToZip(zipWriter *zip.Writer, path string) error {
	w, err := zipWriter.Create(filepath.Base(path))
	if err != nil {
		return err
	}
	r, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("failed to close: %v", err)
		}
	}()
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return nil
}

func saveImageToDir(ctx context.Context, imageRef string, dir string) error {
	cmd := exec.CommandContext(ctx, "docker", "save", imageRef, "-o", "image.tar")
	cmd.Dir = dir
	return cmd.Run()
}

func PluginZipName(plugin *plugin.Plugin) string {
	identity := plugin.Identity
	return fmt.Sprintf("%s-%s-%s.zip", identity.Owner(), identity.Plugin(), plugin.PluginVersion)
}

// CreatePluginZip creates a plugin zip file containing the buf.plugin.yaml and Docker image.
// Returns the path to the created zip file and a digest.
func CreatePluginZip(ctx context.Context, basedir string, plugin *plugin.Plugin, imageID string) (string, error) {
	zipName := PluginZipName(plugin)
	pluginTempDir, err := os.MkdirTemp(basedir, strings.TrimSuffix(zipName, filepath.Ext(zipName)))
	if err != nil {
		return "", err
	}
	defer func() {
		if err := os.RemoveAll(pluginTempDir); err != nil {
			log.Printf("failed to remove %q: %v", pluginTempDir, err)
		}
	}()
	if err := saveImageToDir(ctx, imageID, pluginTempDir); err != nil {
		return "", err
	}
	log.Printf("creating %s", zipName)
	zipFile := filepath.Join(basedir, zipName)
	zf, err := os.OpenFile(zipFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := zf.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			log.Printf("failed to close: %v", err)
		}
	}()
	zw := zip.NewWriter(zf)
	zw.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(w, flate.BestCompression)
	})
	if err := addFileToZip(zw, plugin.Path); err != nil {
		return "", err
	}
	if err := addFileToZip(zw, filepath.Join(pluginTempDir, "image.tar")); err != nil {
		return "", err
	}
	if err := zw.Close(); err != nil {
		return "", err
	}
	if err := zf.Close(); err != nil {
		return "", err
	}

	digest, err := CalculateDigest(zipFile)
	if err != nil {
		return "", err
	}
	return digest, nil
}
