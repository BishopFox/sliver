package update

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox
	Copyright (C) 2019 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/util"
	"github.com/bishopfox/sliver/util/minisign"
	"github.com/cheggaaa/pb/v3"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	SliverPublicKey string
)

const (
	downloadAttempts = 3
	downloadBackoff  = 750 * time.Millisecond
)

// UpdateCmd - Check for updates.
// UpdateCmd - Check 代表 updates.
func UpdateCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	VerboseVersionsCmd(cmd, con, args)

	timeoutF, _ := cmd.Flags().GetInt("timeout")
	timeout := time.Duration(timeoutF) * time.Second

	insecure, _ := cmd.Flags().GetBool("insecure")
	if insecure {
		con.Println()
		con.Println(console.Warn + "You're trying to update over an insecure connection, this is a really bad idea!")
		confirm := false
		forms.Confirm("Recklessly update?", &confirm)
		if !confirm {
			return
		}
		confirm = false
		forms.Confirm("Seriously?", &confirm)
		if !confirm {
			return
		}
	}

	proxy, _ := cmd.Flags().GetString("proxy")
	var proxyURL *url.URL = nil
	var err error
	if proxy != "" {
		proxyURL, err = url.Parse(proxy)
		if err != nil {
			con.PrintErrorf("%s", err)
			return
		}
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: timeout,
			}).Dial,
			TLSHandshakeTimeout: timeout,
			Proxy:               http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecure,
			},
		},
	}

	con.Printf("\nChecking for updates ... ")
	prereleases, _ := cmd.Flags().GetBool("prereleases")
	force, _ := cmd.Flags().GetBool("force")
	var release *version.Release
	if force {
		release, err = version.LatestRelease(client, prereleases)
	} else {
		release, err = version.CheckForUpdates(client, prereleases)
	}
	con.Printf("done!\n\n")
	if err != nil {
		con.PrintErrorf("Update check failed %s\n", err)
		return
	}

	if release != nil {
		saveTo, err := updateSavePath(cmd)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		updateAvailable(con, client, release, saveTo, force)
	} else {
		if force {
			con.PrintInfof("No releases found.\n")
		} else {
			con.PrintInfof("No new releases.\n")
		}
	}
	now := time.Now()
	lastCheck := fmt.Appendf(nil, "%d", now.Unix())
	appDir := assets.GetRootAppDir()
	lastUpdateCheckPath := filepath.Join(appDir, consts.LastUpdateCheckFileName)
	err = os.WriteFile(lastUpdateCheckPath, lastCheck, 0o600)
	if err != nil {
		con.Printf("Failed to save update check time %s", err)
	}
}

// VerboseVersionsCmd - Get verbose version information about the client and server.
// VerboseVersionsCmd - Get 有关客户端和 server. 的详细版本信息
func VerboseVersionsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	clientVer := version.FullVersion()
	serverVer, err := con.Rpc.GetVersion(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("Failed to check server version %s\n", err)
		return
	}

	con.PrintInfof("Client %s - %s/%s\n", clientVer, runtime.GOOS, runtime.GOARCH)
	clientCompiledAt, _ := version.Compiled()
	con.Printf("    Compiled at %s\n", clientCompiledAt)
	con.Printf("    Compiled with %s\n\n", version.GoVersion)

	con.Println()
	con.PrintInfof("Server v%d.%d.%d - %s - %s/%s\n",
		serverVer.Major, serverVer.Minor, serverVer.Patch, serverVer.Commit,
		serverVer.OS, serverVer.Arch)
	serverCompiledAt := time.Unix(serverVer.CompiledAt, 0)
	con.Printf("    Compiled at %s\n", serverCompiledAt)
}

func updateSavePath(cmd *cobra.Command) (string, error) {
	saveTo, _ := cmd.Flags().GetString("save")
	if saveTo != "" {
		fi, err := os.Stat(saveTo)
		if err != nil {
			return "", err
		}
		if !fi.Mode().IsDir() {
			return "", fmt.Errorf("'%s' is not a directory", saveTo)
		}
		return saveTo, nil
	}
	user, err := user.Current()
	if err != nil {
		return os.TempDir(), nil
	}
	if fi, err := os.Stat(filepath.Join(user.HomeDir, "Downloads")); !os.IsNotExist(err) {
		if fi.Mode().IsDir() {
			return filepath.Join(user.HomeDir, "Downloads"), nil
		}
	}
	return user.HomeDir, nil
}

func hasAnySuffix(assetFileName string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(assetFileName, suffix) {
			return true
		}
	}
	return false
}

func findAssetFor(prefix string, suffixes []string, assets []version.Asset) *version.Asset {
	for _, asset := range assets {
		downloadURL, err := url.Parse(asset.BrowserDownloadURL)
		if err != nil {
			continue
		}
		assetFileName := filepath.Base(downloadURL.Path)
		if strings.HasPrefix(assetFileName, prefix) && hasAnySuffix(assetFileName, suffixes) {
			return &asset
		}
	}
	return nil
}

func serverAssetForGOOS(assets []version.Asset) *version.Asset {
	suffixes := assetSuffixes(runtime.GOOS, runtime.GOARCH)
	prefix := "sliver-server"
	return findAssetFor(prefix, suffixes, assets)
}

func clientAssetForGOOS(assets []version.Asset) *version.Asset {
	suffixes := assetSuffixes(runtime.GOOS, runtime.GOARCH)
	prefix := "sliver-client"
	return findAssetFor(prefix, suffixes, assets)
}

func assetSuffixes(goos, goarch string) []string {
	suffixes := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)
	add := func(suffix string) {
		if suffix == "" {
			return
		}
		if _, ok := seen[suffix]; ok {
			return
		}
		seen[suffix] = struct{}{}
		suffixes = append(suffixes, suffix)
	}

	add(fmt.Sprintf("_%s-%s.zip", goos, goarch))
	add(fmt.Sprintf("_%s-%s", goos, goarch))
	add(fmt.Sprintf("_%s.zip", goos))
	add(fmt.Sprintf("_%s", goos))

	if goos == "darwin" {
		add(fmt.Sprintf("_macos-%s.zip", goarch))
		add(fmt.Sprintf("_macos-%s", goarch))
		add("_macos.zip")
		add("_macos")
	}

	return suffixes
}

func updateAvailable(con *console.SliverClient, client *http.Client, release *version.Release, saveTo string, force bool) {
	serverAsset := serverAssetForGOOS(release.Assets)
	clientAsset := clientAssetForGOOS(release.Assets)

	msg := "New version available"
	if force {
		msg = "Latest release available"
	}
	con.Printf("%s %s\n", msg, release.TagName)
	if serverAsset != nil {
		con.Printf(" - Server: %s\n", util.ByteCountBinary(int64(serverAsset.Size)))
	}
	if clientAsset != nil {
		con.Printf(" - Client: %s\n", util.ByteCountBinary(int64(clientAsset.Size)))
	}
	if serverAsset == nil {
		con.PrintWarnf("No server asset available for %s/%s\n", runtime.GOOS, runtime.GOARCH)
	}
	if clientAsset == nil {
		con.PrintWarnf("No client asset available for %s/%s\n", runtime.GOOS, runtime.GOARCH)
	}
	con.Println()

	confirm := false
	forms.Confirm("Download update?", &confirm)
	if confirm {
		if serverAsset == nil && clientAsset == nil {
			con.PrintWarnf("No matching assets found for %s/%s\n", runtime.GOOS, runtime.GOARCH)
			return
		}
		publicKey, err := loadSliverPublicKey()
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		downloads := []struct {
			label string
			asset *version.Asset
		}{
			{label: "server", asset: serverAsset},
			{label: "client", asset: clientAsset},
		}
		for _, item := range downloads {
			if item.asset == nil {
				continue
			}
			con.Printf("Downloading %s update from %s ...\n", item.label, item.asset.BrowserDownloadURL)
			if err := downloadAssetWithSignature(con, client, item.asset, release.Assets, saveTo, publicKey); err != nil {
				con.PrintErrorf("Failed to download %s update: %s\n", item.label, err)
				return
			}
			con.Println(console.Clearln)
		}
		con.PrintInfof("Saved updates to: %s\n", saveTo)
	}
}

func downloadAssetWithSignature(con *console.SliverClient, client *http.Client, asset *version.Asset, assets []version.Asset, saveTo string, publicKey minisign.PublicKey) error {
	if asset == nil {
		return errors.New("no asset available for this platform")
	}
	downloadURL, err := url.Parse(asset.BrowserDownloadURL)
	if err != nil {
		return err
	}
	assetFileName := filepath.Base(downloadURL.Path)
	finalPath := filepath.Join(saveTo, assetFileName)

	sigURL := asset.BrowserDownloadURL + ".minisig"
	var sigLimit int64
	if sigAsset := signatureAssetFor(assetFileName, assets); sigAsset != nil && sigAsset.BrowserDownloadURL != "" {
		sigURL = sigAsset.BrowserDownloadURL
		sigLimit = int64(sigAsset.Size)
	}

	assetTempPath, err := downloadWithRetries(client, asset.BrowserDownloadURL, saveTo, int64(asset.Size), "asset")
	if err != nil {
		return fmt.Errorf("download asset: %w", err)
	}

	sigData, err := downloadBytesWithRetries(client, sigURL, sigLimit, "signature")
	if err != nil {
		cleanupAssetFiles(assetTempPath, finalPath)
		return fmt.Errorf("download signature: %w", err)
	}

	if err := verifyMinisignSignature(assetTempPath, sigData, assetFileName, publicKey); err != nil {
		cleanupAssetFiles(assetTempPath, finalPath)
		return err
	}
	con.PrintSuccessf("Signature verified for %s\n", finalPath)

	if err := replaceFile(assetTempPath, finalPath); err != nil {
		os.Remove(assetTempPath)
		return err
	}
	return nil
}

func signatureAssetFor(assetFileName string, assets []version.Asset) *version.Asset {
	want := assetFileName + ".minisig"
	for _, asset := range assets {
		if asset.Name == want {
			return &asset
		}
	}
	return nil
}

func loadSliverPublicKey() (minisign.PublicKey, error) {
	if strings.TrimSpace(SliverPublicKey) == "" {
		return minisign.PublicKey{}, errors.New("minisign public key not set at build time")
	}
	var publicKey minisign.PublicKey
	if err := publicKey.UnmarshalText([]byte(SliverPublicKey)); err != nil {
		return minisign.PublicKey{}, fmt.Errorf("invalid minisign public key: %w", err)
	}
	return publicKey, nil
}

type httpStatusError struct {
	url    string
	status int
}

func (e httpStatusError) Error() string {
	return fmt.Sprintf("unexpected HTTP status %d for %s", e.status, e.url)
}

func downloadWithRetries(client *http.Client, downloadURL, saveTo string, limit int64, label string) (string, error) {
	var lastErr error
	backoff := downloadBackoff
	for attempt := 1; attempt <= downloadAttempts; attempt++ {
		path, err := downloadOnce(client, downloadURL, saveTo, limit, label)
		if err == nil {
			return path, nil
		}
		lastErr = err
		if !isRetryableDownloadError(err) || attempt == downloadAttempts {
			break
		}
		time.Sleep(backoff)
		backoff *= 2
	}
	return "", lastErr
}

func downloadBytesWithRetries(client *http.Client, downloadURL string, limit int64, label string) ([]byte, error) {
	var lastErr error
	backoff := downloadBackoff
	for attempt := 1; attempt <= downloadAttempts; attempt++ {
		data, err := downloadBytesOnce(client, downloadURL, limit, label)
		if err == nil {
			return data, nil
		}
		lastErr = err
		if !isRetryableDownloadError(err) || attempt == downloadAttempts {
			break
		}
		time.Sleep(backoff)
		backoff *= 2
	}
	return nil, lastErr
}

func downloadOnce(client *http.Client, downloadURL, saveTo string, limit int64, label string) (string, error) {
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return "", err
	}

	baseName := filepath.Base(parsedURL.Path)
	tmpFile, err := os.CreateTemp(saveTo, baseName+".tmp-*")
	if err != nil {
		return "", err
	}
	success := false
	defer func() {
		if !success {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
		}
	}()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", httpStatusError{url: downloadURL, status: resp.StatusCode}
	}

	if limit <= 0 && resp.ContentLength > 0 {
		limit = resp.ContentLength
	}

	var reader io.Reader = resp.Body
	var bar *pb.ProgressBar
	if limit > 0 {
		bar = pb.Full.Start64(limit)
		bar.Set("prefix", fmt.Sprintf("%s: ", label))
		bar.Set(pb.Bytes, true)
		bar.Set(pb.CleanOnFinish, true)
		bar.Set(pb.ReturnSymbol, "\r")
		if term.IsTerminal(int(os.Stdin.Fd())) {
			bar.Set(pb.Terminal, true)
		}
		reader = bar.NewProxyReader(reader)
	}

	n, copyErr := io.Copy(tmpFile, reader)
	if bar != nil {
		bar.Finish()
	}
	if copyErr != nil {
		return "", copyErr
	}

	if limit > 0 && n != limit {
		return "", fmt.Errorf("downloaded %d bytes, expected %d", n, limit)
	}

	if err := tmpFile.Sync(); err != nil {
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	success = true
	return tmpFile.Name(), nil
}

func downloadBytesOnce(client *http.Client, downloadURL string, limit int64, label string) ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, httpStatusError{url: downloadURL, status: resp.StatusCode}
	}

	if limit <= 0 && resp.ContentLength > 0 {
		limit = resp.ContentLength
	}

	var reader io.Reader = resp.Body
	var bar *pb.ProgressBar
	if limit > 0 {
		bar = pb.Full.Start64(limit)
		bar.Set("prefix", fmt.Sprintf("%s: ", label))
		bar.Set(pb.Bytes, true)
		bar.Set(pb.CleanOnFinish, true)
		bar.Set(pb.ReturnSymbol, "\r")
		if term.IsTerminal(int(os.Stdin.Fd())) {
			bar.Set(pb.Terminal, true)
		}
		reader = bar.NewProxyReader(reader)
	}

	data, readErr := readAllWithLimit(reader, limit)
	if bar != nil {
		bar.Finish()
	}
	if readErr != nil {
		return nil, readErr
	}

	return data, nil
}

func isRetryableDownloadError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}
	var statusErr httpStatusError
	if errors.As(err, &statusErr) {
		switch statusErr.status {
		case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	}
	return false
}

func verifyMinisignSignature(artifactPath string, sigData []byte, expectedFile string, publicKey minisign.PublicKey) error {
	artifact, err := os.Open(artifactPath)
	if err != nil {
		return fmt.Errorf("open artifact: %w", err)
	}
	defer artifact.Close()

	reader := minisign.NewReader(artifact)
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return fmt.Errorf("stream artifact: %w", err)
	}

	if !reader.Verify(publicKey, sigData) {
		return errors.New("signature verification failed")
	}

	if err := verifyTrustedComment(sigData, expectedFile); err != nil {
		return err
	}

	return nil
}

func readAllWithLimit(reader io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return io.ReadAll(reader)
	}

	lr := &io.LimitedReader{R: reader, N: limit + 1}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("downloaded %d bytes, expected %d", len(data), limit)
	}
	if int64(len(data)) != limit {
		return nil, fmt.Errorf("downloaded %d bytes, expected %d", len(data), limit)
	}
	return data, nil
}

func verifyTrustedComment(signature []byte, expectedFile string) error {
	var sig minisign.Signature
	if err := sig.UnmarshalText(signature); err != nil {
		return fmt.Errorf("parse signature: %w", err)
	}

	fileField := trustedCommentFile(sig.TrustedComment)
	if fileField == "" {
		return fmt.Errorf("trusted comment missing file field: %q", sig.TrustedComment)
	}
	if fileField != expectedFile {
		return fmt.Errorf("trusted comment file mismatch: expected %q, got %q", expectedFile, fileField)
	}

	return nil
}

func trustedCommentFile(trustedComment string) string {
	for _, field := range strings.Fields(trustedComment) {
		if after, ok := strings.CutPrefix(field, "file:"); ok {
			return after
		}
	}
	return ""
}

func replaceFile(src, dst string) error {
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(src, dst)
}

func cleanupAssetFiles(tempPath, finalPath string) {
	if tempPath != "" {
		_ = os.Remove(tempPath)
	}
	if finalPath != "" {
		_ = os.Remove(finalPath)
	}
}
