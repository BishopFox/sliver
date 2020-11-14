package commands

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
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/connection"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"gopkg.in/AlecAivazis/survey.v1"
)

const (
	lastCheckFileName = "last_update_check"
)

// Updates - Check for newer Sliver console/server releases.
type Updates struct {
	Options struct {
		Insecure    bool   `long:"insecure" description:"Check for newer Sliver console/server releases"`
		Timeout     int    `long:"timeout" description:"Command timeout in seconds (default: 10)" default:"10"`
		PreReleases bool   `long:"prereleases" description:"include pre-released (unstable) versions"`
		Proxy       string `long:"proxy" description:"specify a proxy url (e.g. http://localhost:8080)"`
	} `group:"Update Check options"`
}

// Execute - Check for Sliver release updates.
func (u *Updates) Execute(args []string) (err error) {

	verboseVersions()

	insecure := u.Options.Insecure
	if insecure {
		fmt.Println()
		fmt.Println(util.Warn + "You're trying to update over an insecure connection, this is a really bad idea!")
		confirm := false
		prompt := &survey.Confirm{Message: "Recklessly update?"}
		survey.AskOne(prompt, &confirm, nil)
		if !confirm {
			return
		}
		confirm = false
		prompt = &survey.Confirm{Message: "Seriously?"}
		survey.AskOne(prompt, &confirm, nil)
		if !confirm {
			return
		}
	}

	proxy := u.Options.Proxy
	var proxyURL *url.URL = nil
	if proxy != "" {
		proxyURL, err = url.Parse(proxy)
		if err != nil {
			fmt.Printf(util.Error+"%s", err)
			return
		}
	}

	timeout, err := time.ParseDuration(fmt.Sprintf("%ds", u.Options.Timeout))
	if err != nil {
		err = fmt.Errorf(util.Error + "Could not parse timeout value (int) in time.Duration")
		fmt.Printf(util.Error+"Update check failed %s", err)
		return
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

	fmt.Printf("\nChecking for updates ... ")
	prereleases := u.Options.PreReleases
	release, err := version.CheckForUpdates(client, prereleases)
	fmt.Printf("done!\n\n")
	if err != nil {
		fmt.Printf(util.Error+"Update check failed %s", err)
		return
	}

	if release != nil {
		fmt.Printf("New version available: %s\n", release.TagName)
		fmt.Println(release.HTMLURL)
	} else {
		fmt.Printf(util.Info + "No new releases.\n")
	}
	now := time.Now()
	lastCheck := []byte(fmt.Sprintf("%d", now.Unix()))
	appDir := assets.GetRootAppDir()
	lastUpdateCheckPath := path.Join(appDir, lastCheckFileName)
	err = ioutil.WriteFile(lastUpdateCheckPath, lastCheck, 0600)
	if err != nil {
		log.Printf("Failed to save update check time %s", err)
	}

	return
}

// GetLastUpdateCheck - Get the timestap of the last update check, nil if none
func GetLastUpdateCheck() *time.Time {
	appDir := assets.GetRootAppDir()
	lastUpdateCheckPath := path.Join(appDir, lastCheckFileName)
	data, err := ioutil.ReadFile(lastUpdateCheckPath)
	if err != nil {
		log.Printf("Failed to read last update check %s", err)
		return nil
	}
	unixTime, err := strconv.Atoi(string(data))
	if err != nil {
		log.Printf("Failed to parse last update check %s", err)
		return nil
	}
	lastUpdate := time.Unix(int64(unixTime), 0)
	return &lastUpdate
}

// Version - Display version information
type Version struct{}

// Execute - Display version information
func (v *Version) Execute(args []string) (err error) {
	verboseVersions()
	return
}

func verboseVersions() {
	clientVer := version.FullVersion()
	serverVer, err := connection.RPC.GetVersion(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.Warn+"Failed to check server version %s", err)
		return
	}

	fmt.Printf(util.Info+"Client v%s - %s/%s\n", clientVer, runtime.GOOS, runtime.GOARCH)
	clientCompiledAt, _ := version.Compiled()
	fmt.Printf("    Compiled at %s\n\n", clientCompiledAt)

	fmt.Println()
	fmt.Printf(util.Info+"Server v%d.%d.%d - %s - %s/%s\n",
		serverVer.Major, serverVer.Minor, serverVer.Patch, serverVer.Commit,
		serverVer.OS, serverVer.Arch)
	serverCompiledAt := time.Unix(serverVer.CompiledAt, 0)
	fmt.Printf("    Compiled at %s\n", serverCompiledAt)
}
