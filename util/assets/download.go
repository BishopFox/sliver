package assets

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
)

func (r *runner) downloadFile(url, dest string) error {
	if err := ensureDir(filepath.Dir(dest)); err != nil {
		return err
	}
	return r.downloadFileTo(url, dest)
}

func (r *runner) downloadFileTo(url, dest string) error {
	tmpFile, err := os.CreateTemp(filepath.Dir(dest), ".download-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpName)
	}()

	if err := r.logger.RunSpinner(fmt.Sprintf("fetch %s", downloadLabel(url)), func() error {
		return r.fetchToWriter(url, tmpFile)
	}); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("flush download: %w", err)
	}
	if err := moveFile(tmpName, dest); err != nil {
		return fmt.Errorf("move download into place: %w", err)
	}
	return nil
}

func (r *runner) downloadToTemp(url, dir string) (string, error) {
	if err := ensureDir(dir); err != nil {
		return "", err
	}

	file, err := os.CreateTemp(dir, "download-*")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	name := file.Name()

	if err := r.logger.RunSpinner(fmt.Sprintf("fetch %s", downloadLabel(url)), func() error {
		return r.fetchToWriter(url, file)
	}); err != nil {
		file.Close()
		os.Remove(name)
		return "", err
	}
	if err := file.Close(); err != nil {
		os.Remove(name)
		return "", fmt.Errorf("flush download: %w", err)
	}

	return name, nil
}

func (r *runner) fetchToWriter(url string, w io.Writer) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "sliver-assets")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("download %s: unexpected status %s", url, resp.Status)
	}

	if _, err := io.Copy(w, resp.Body); err != nil {
		return fmt.Errorf("stream %s: %w", url, err)
	}

	return nil
}

func moveFile(src, dest string) error {
	if err := os.Rename(src, dest); err == nil {
		return nil
	}
	if err := copyFile(src, dest); err != nil {
		return err
	}
	return os.Remove(src)
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	if err := ensureDir(filepath.Dir(dest)); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("flush dest: %w", err)
	}
	return nil
}

func downloadLabel(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	base := path.Base(parsed.Path)
	if base == "." || base == "/" || base == "" {
		return rawURL
	}
	return base
}
