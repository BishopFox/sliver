package assets

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"
)

func (r *runner) downloadFile(url, dest string) error {
	if err := ensureDir(filepath.Dir(dest)); err != nil {
		return err
	}
	return r.downloadFileTo(url, dest)
}

func (r *runner) downloadFileTo(url, dest string) error {
	tmpName, err := r.downloadToTemp(url, filepath.Dir(dest))
	if err != nil {
		return err
	}
	defer os.Remove(tmpName)

	if err := moveFile(tmpName, dest); err != nil {
		return fmt.Errorf("move download into place: %w", err)
	}
	return nil
}

func (r *runner) downloadToTemp(url, dir string) (string, error) {
	if err := ensureDir(dir); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()
	return r.downloadToTempWithContext(ctx, url, dir)
}

func (r *runner) downloadToTempWithContext(ctx context.Context, url, dir string) (string, error) {
	attempts := r.downloadAttempts
	if attempts <= 0 {
		attempts = defaultDownloadAttempts
	}
	delay := r.downloadRetryDelay
	if delay < 0 {
		delay = 0
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		name, retryable, err := r.downloadAttempt(ctx, url, dir)
		if err == nil {
			return name, nil
		}
		lastErr = err
		if !retryable || attempt == attempts {
			return "", err
		}

		r.logger.Warnf("Fetch %s failed (%d/%d): %v", downloadLabel(url), attempt, attempts, err)
		if err := waitForDownloadRetry(ctx, delay); err != nil {
			return "", fmt.Errorf("download %s: %w", url, err)
		}
		delay *= 2
	}

	return "", lastErr
}

func (r *runner) downloadAttempt(ctx context.Context, url, dir string) (string, bool, error) {
	file, err := os.CreateTemp(dir, "download-*")
	if err != nil {
		return "", false, fmt.Errorf("create temp file: %w", err)
	}
	tmpName := file.Name()
	closed := false
	success := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
		if !success {
			_ = os.Remove(tmpName)
		}
	}()

	if err := r.logger.RunSpinner(fmt.Sprintf("fetch %s", downloadLabel(url)), func() error {
		return r.fetchToWriter(ctx, url, file)
	}); err != nil {
		return "", ctx.Err() == nil && isRetryableDownloadError(err), err
	}
	if err := file.Close(); err != nil {
		closed = true
		return "", false, fmt.Errorf("flush download: %w", err)
	}
	closed = true
	success = true

	return tmpName, false, nil
}

func (r *runner) fetchToWriter(ctx context.Context, url string, w io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "sliver-assets")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
		return downloadHTTPStatusError{url: url, status: resp.Status, statusCode: resp.StatusCode}
	}

	written, err := io.Copy(w, resp.Body)
	if err != nil {
		return fmt.Errorf("stream %s: %w", url, err)
	}
	if resp.ContentLength >= 0 && written != resp.ContentLength {
		return fmt.Errorf(
			"stream %s: copied %d of %d bytes: %w",
			url,
			written,
			resp.ContentLength,
			io.ErrUnexpectedEOF,
		)
	}

	return nil
}

type downloadHTTPStatusError struct {
	url        string
	status     string
	statusCode int
}

func (e downloadHTTPStatusError) Error() string {
	return fmt.Sprintf("download %s: unexpected status %s", e.url, e.status)
}

func isRetryableDownloadError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	var statusErr downloadHTTPStatusError
	if errors.As(err, &statusErr) {
		return statusErr.statusCode == http.StatusRequestTimeout ||
			statusErr.statusCode == http.StatusTooEarly ||
			statusErr.statusCode == http.StatusTooManyRequests ||
			statusErr.statusCode >= http.StatusInternalServerError &&
				statusErr.statusCode <= 599
	}

	var netErr net.Error
	return errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary())
}

func waitForDownloadRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
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
