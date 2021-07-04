package update

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/util"
	"github.com/cheggaaa/pb/v3"
	"github.com/desertbit/grumble"
)

// UpdateCmd - Check for updates
func UpdateCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	VerboseVersionsCmd(ctx, con)

	timeout := time.Duration(ctx.Flags.Int("timeout")) * time.Second

	insecure := ctx.Flags.Bool("insecure")
	if insecure {
		con.Println()
		con.Println(console.Warn + "You're trying to update over an insecure connection, this is a really bad idea!")
		confirm := false
		prompt := &survey.Confirm{Message: "Recklessly update?"}
		survey.AskOne(prompt, &confirm)
		if !confirm {
			return
		}
		confirm = false
		prompt = &survey.Confirm{Message: "Seriously?"}
		survey.AskOne(prompt, &confirm)
		if !confirm {
			return
		}
	}

	proxy := ctx.Flags.String("proxy")
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
	prereleases := ctx.Flags.Bool("prereleases")
	release, err := version.CheckForUpdates(client, prereleases)
	con.Printf("done!\n\n")
	if err != nil {
		con.PrintErrorf("Update check failed %s\n", err)
		return
	}

	if release != nil {
		saveTo, err := updateSavePath(ctx)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		updateAvailable(con, client, release, saveTo)
	} else {
		con.PrintInfof("No new releases.\n")
	}
	now := time.Now()
	lastCheck := []byte(fmt.Sprintf("%d", now.Unix()))
	appDir := assets.GetRootAppDir()
	lastUpdateCheckPath := path.Join(appDir, consts.LastUpdateCheckFileName)
	err = ioutil.WriteFile(lastUpdateCheckPath, lastCheck, 0600)
	if err != nil {
		log.Printf("Failed to save update check time %s", err)
	}
}

// VerboseVersionsCmd - Get verbose version information about the client and server
func VerboseVersionsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
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

func updateSavePath(ctx *grumble.Context) (string, error) {
	saveTo := ctx.Flags.String("save")
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
	suffixes := []string{fmt.Sprintf("_%s.zip", runtime.GOOS), runtime.GOOS}
	if runtime.GOOS == "darwin" {
		suffixes = []string{"_macos.zip", "_macos"}
		if runtime.GOARCH == "arm64" {
			suffixes = []string{"_macos-arm64.zip", "_macos-arm64"}
		}
	}
	prefix := "sliver-server"
	return findAssetFor(prefix, suffixes, assets)
}

func clientAssetForGOOS(assets []version.Asset) *version.Asset {
	suffixes := []string{fmt.Sprintf("_%s.zip", runtime.GOOS), runtime.GOOS}
	if runtime.GOOS == "darwin" {
		suffixes = []string{"_macos.zip", "_macos"}
		if runtime.GOARCH == "arm64" {
			suffixes = []string{"_macos-arm64.zip", "_macos-arm64"}
		}
	}
	prefix := "sliver-client"
	return findAssetFor(prefix, suffixes, assets)
}

func updateAvailable(con *console.SliverConsoleClient, client *http.Client, release *version.Release, saveTo string) {

	serverAsset := serverAssetForGOOS(release.Assets)
	clientAsset := clientAssetForGOOS(release.Assets)

	con.Printf("New version available %s\n", release.TagName)
	if serverAsset != nil {
		con.Printf(" - Server: %s\n", util.ByteCountBinary(int64(serverAsset.Size)))
	}
	if clientAsset != nil {
		con.Printf(" - Client: %s\n", util.ByteCountBinary(int64(clientAsset.Size)))
	}
	con.Println()

	confirm := false
	prompt := &survey.Confirm{
		Message: "Download update?",
	}
	survey.AskOne(prompt, &confirm)
	if confirm {
		con.Printf("Please wait ...")
		err := downloadAsset(client, serverAsset, saveTo)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		err = downloadAsset(client, clientAsset, saveTo)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.Println(console.Clearln)
		con.PrintInfof("Saved updates to: %s\n", saveTo)
	}
}

func downloadAsset(client *http.Client, asset *version.Asset, saveTo string) error {
	downloadURL, err := url.Parse(asset.BrowserDownloadURL)
	if err != nil {
		return err
	}
	assetFileName := filepath.Base(downloadURL.Path)

	limit := int64(asset.Size)
	writer, err := os.Create(filepath.Join(saveTo, assetFileName))
	if err != nil {
		return err
	}

	resp, err := client.Get(asset.BrowserDownloadURL)
	if err != nil {
		return err
	}

	bar := pb.Full.Start64(limit)
	barReader := bar.NewProxyReader(resp.Body)
	io.Copy(writer, barReader)
	bar.Finish()
	return nil
}
