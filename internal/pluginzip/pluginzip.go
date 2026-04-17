// Package pluginzip creates distributable zip archives for plugins. Each
// archive contains the plugin's buf.plugin.yaml and an image.tar produced by
// "docker save" of a locally available image.
package pluginzip

import (
	"archive/zip"
	"compress/flate"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/bufbuild/plugins/internal/plugin"
)

// Name returns the zip filename for a plugin: "<owner>-<name>-<version>.zip".
func Name(p *plugin.Plugin) string {
	identity := p.Identity
	return fmt.Sprintf("%s-%s-%s.zip", identity.Owner(), identity.Plugin(), p.PluginVersion)
}

// Create writes <outputDir>/<Name(p)> containing the plugin's buf.plugin.yaml
// and an image.tar produced by "docker save imageRef". The image must already
// be available to the local Docker daemon. Returns the path to the zip file.
func Create(
	ctx context.Context,
	logger *slog.Logger,
	p *plugin.Plugin,
	imageRef string,
	outputDir string,
) (string, error) {
	stagingDir, err := os.MkdirTemp(outputDir, ".pluginzip-")
	if err != nil {
		return "", fmt.Errorf("create staging dir: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(stagingDir); err != nil {
			logger.WarnContext(ctx, "failed to remove staging dir",
				slog.String("dir", stagingDir),
				slog.Any("error", err),
			)
		}
	}()
	imageTar := filepath.Join(stagingDir, "image.tar")
	if err := saveImage(ctx, imageRef, imageTar); err != nil {
		return "", fmt.Errorf("docker save %q: %w", imageRef, err)
	}
	zipPath := filepath.Join(outputDir, Name(p))
	logger.InfoContext(ctx, "creating zip", slog.String("path", zipPath))
	if err := writeZip(zipPath, p.Path, imageTar); err != nil {
		return "", fmt.Errorf("write zip: %w", err)
	}
	return zipPath, nil
}

func saveImage(ctx context.Context, imageRef, outputPath string) error {
	cmd := exec.CommandContext(ctx, "docker", "save", imageRef, "-o", outputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeZip(zipPath, pluginYAMLPath, imageTarPath string) (retErr error) {
	zf, err := os.OpenFile(zipPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		retErr = errors.Join(retErr, zf.Close())
		if retErr != nil {
			if removeErr := os.Remove(zipPath); removeErr != nil && !errors.Is(retErr, os.ErrNotExist) {
				retErr = errors.Join(retErr, fmt.Errorf("failed to remove zip: %w", removeErr))
			}
		}
	}()
	zw := zip.NewWriter(zf)
	defer func() {
		retErr = errors.Join(retErr, zw.Close())
	}()
	zw.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(w, flate.BestCompression)
	})
	if err := addFileToZip(zw, pluginYAMLPath); err != nil {
		return err
	}
	return addFileToZip(zw, imageTarPath)
}

func addFileToZip(zw *zip.Writer, path string) (retErr error) {
	w, err := zw.Create(filepath.Base(path))
	if err != nil {
		return err
	}
	r, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		retErr = errors.Join(retErr, r.Close())
	}()
	_, err = io.Copy(w, r)
	return err
}
