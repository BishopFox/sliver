package armory

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/server/cryptography/minisign"
)

// ArmoryIndexParser - Generic interface to fetch armory indexes
type ArmoryIndexParser func(*assets.ArmoryConfig, ArmoryHTTPConfig) (*ArmoryIndex, error)

// ArmoryPackageParser - Generic interface to fetch armory package manifests
type ArmoryPackageParser func(*ArmoryPackage, bool, ArmoryHTTPConfig) (*minisign.Signature, []byte, error)

var (
	indexParsers = map[string]ArmoryIndexParser{
		"api.github.com": GithubAPIArmoryIndexParser,
	}
	pkgParsers = map[string]ArmoryPackageParser{
		"api.github.com": GithubAPIArmoryPackageParser,
	}
)

const (
	armoryIndexFileName    = "armory.json"
	armoryIndexSigFileName = "armory.minisig"
)

type armoryIndexResponse struct {
	Minisig     string `json:"minisig"`      // Minisig
	ArmoryIndex string `json:"armory_index"` // Base64 String
}

type armoryPkgResponse struct {
	Minisig  string `json:"minisig"` // Minisig
	TarGzURL string `json:"tar_gz_url"`
}

//
// Default Parsers for Self-Hosted Armories
//

// DefaultArmoryParser - Parse the armory index directly from the url
func DefaultArmoryIndexParser(armoryConfig *assets.ArmoryConfig, clientConfig ArmoryHTTPConfig) (*ArmoryIndex, error) {
	var publicKey minisign.PublicKey
	err := publicKey.UnmarshalText([]byte(armoryConfig.PublicKey))
	if err != nil {
		return nil, err
	}

	client := httpClient(clientConfig)
	resp, err := client.Get(armoryConfig.RepoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("api returned non-200 status code")
	}
	index := &armoryIndexResponse{}
	err = json.Unmarshal(body, index)
	if err != nil {
		return nil, err
	}

	// Verify index is signed by trusted key
	valid := minisign.Verify(publicKey, []byte(index.ArmoryIndex), []byte(index.ArmoryIndex))
	if !valid {
		return nil, errors.New("invalid signature")
	}

	armoryIndex := &ArmoryIndex{}
	armoryIndexData, err := base64.StdEncoding.DecodeString(index.ArmoryIndex)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(armoryIndexData, armoryIndex)
	if err != nil {
		return nil, err
	}
	return armoryIndex, nil
}

// DefaultArmoryPkgParser - Parse the armory package manifest directly from the url
func DefaultArmoryPkgParser(armoryPkg *ArmoryPackage, sigOnly bool, clientConfig ArmoryHTTPConfig) (*minisign.Signature, []byte, error) {
	var publicKey minisign.PublicKey
	err := publicKey.UnmarshalText([]byte(armoryPkg.PublicKey))
	if err != nil {
		return nil, nil, err
	}

	client := httpClient(clientConfig)
	resp, err := client.Get(armoryPkg.RepoURL)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, errors.New("api returned non-200 status code")
	}
	pkgResp := &armoryPkgResponse{}
	err = json.Unmarshal(body, pkgResp)
	if err != nil {
		return nil, nil, err
	}
	sig, err := parsePkgMinsig([]byte(pkgResp.Minisig))
	if err != nil {
		return nil, nil, err
	}
	var tarGz []byte
	if !sigOnly {
		tarGzURL, err := url.Parse(armoryPkg.RepoURL)
		if err != nil {
			return nil, nil, err
		}
		if tarGzURL.Scheme != "https" && tarGzURL.Scheme != "http" {
			return nil, nil, errors.New("invalid url scheme")
		}
		resp, err := client.Get(tarGzURL.String())
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()
		tarGz, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, nil, errors.New("api returned non-200 status code")
		}
	}
	return sig, tarGz, nil
}

//
// GitHub API Parsers
//

type GithubAsset struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Size int    `json:"size"`

	BrowserDownloadURL string `json:"browser_download_url"`
}

type GithubRelease struct {
	ID          int           `json:"id"`
	Name        string        `json:"name"`
	URL         string        `json:"url"`
	HTMLURL     string        `json:"html_url"`
	TagName     string        `json:"tag_name"`
	Body        string        `json:"body"`
	Prerelease  bool          `json:"prerelease"`
	TarballURL  string        `json:"tarball_url"`
	ZipballURL  string        `json:"zipball_url"`
	CreatedAt   string        `json:"created_at"`
	PublishedAt string        `json:"published_at"`
	Assets      []GithubAsset `json:"assets"`
}

// GithubAPIArmoryIndexParser - Parse the armory index from a GitHub release
func GithubAPIArmoryIndexParser(armoryConfig *assets.ArmoryConfig, clientConfig ArmoryHTTPConfig) (*ArmoryIndex, error) {
	var publicKey minisign.PublicKey
	err := publicKey.UnmarshalText([]byte(armoryConfig.PublicKey))
	if err != nil {
		return nil, err
	}

	client := httpClient(clientConfig)
	resp, err := client.Get(armoryConfig.RepoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			return nil, errors.New("you hit the github api rate limit (60 req/hr), try later")
		}
		return nil, errors.New("api returned non-200 status code")
	}

	releases := []GithubRelease{}
	err = json.Unmarshal(body, &releases)
	if err != nil {
		return nil, err
	}
	if len(releases) < 1 {
		return nil, errors.New("no releases found")
	}
	release := releases[0] // Latest only right now

	var armoryIndexData []byte
	var sigData []byte
	for _, asset := range release.Assets {
		if asset.Name == armoryIndexFileName {
			resp, err := client.Get(asset.BrowserDownloadURL)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			armoryIndexData, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
		}
		if asset.Name == armoryIndexSigFileName {
			resp, err := client.Get(asset.BrowserDownloadURL)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			sigData, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
		}
	}

	// Verify index is signed by trusted key
	valid := minisign.Verify(publicKey, armoryIndexData, sigData)
	if !valid {
		return nil, errors.New("invalid signature")
	}

	armoryIndex := &ArmoryIndex{}
	err = json.Unmarshal(armoryIndexData, armoryIndex)
	if err != nil {
		return nil, err
	}
	return armoryIndex, nil
}

// GithubAPIArmoryPackageParser - Retrieve the minisig and tar.gz for an armory package from a GitHub release
func GithubAPIArmoryPackageParser(armoryPkg *ArmoryPackage, sigOnly bool, clientConfig ArmoryHTTPConfig) (*minisign.Signature, []byte, error) {
	var publicKey minisign.PublicKey
	err := publicKey.UnmarshalText([]byte(armoryPkg.PublicKey))
	if err != nil {
		return nil, nil, err
	}

	client := httpClient(clientConfig)
	resp, err := client.Get(armoryPkg.RepoURL)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			return nil, nil, errors.New("you hit the github api rate limit (60 req/hr), try later")
		}
		return nil, nil, errors.New("api returned non-200 status code")
	}

	releases := []GithubRelease{}
	err = json.Unmarshal(body, &releases)
	if err != nil {
		return nil, nil, err
	}
	release := releases[0] // Latest only right now

	var sig *minisign.Signature
	var tarGz []byte
	for _, asset := range release.Assets {
		if asset.Name == fmt.Sprintf("%s.minisig", armoryPkg.CommandName) {
			var resp *http.Response
			resp, err = client.Get(asset.BrowserDownloadURL)
			if err != nil {
				break
			}
			defer resp.Body.Close()
			var body []byte
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				break
			}
			sig, err = parsePkgMinsig(body)
			if err != nil {
				break
			}
		}
		if asset.Name == fmt.Sprintf("%s.tar.gz", armoryPkg.CommandName) && !sigOnly {
			var resp *http.Response
			resp, err = client.Get(asset.BrowserDownloadURL)
			if err != nil {
				break
			}
			defer resp.Body.Close()
			tarGz, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				break
			}
		}
	}
	return sig, tarGz, err
}

func parsePkgMinsig(data []byte) (*minisign.Signature, error) {
	var sig minisign.Signature
	err := sig.UnmarshalText(data)
	if err != nil {
		return nil, err
	}
	if len(sig.TrustedComment) < 1 {
		return nil, errors.New("missing trusted comment")
	}
	return &sig, nil
}

func httpClient(config ArmoryHTTPConfig) *http.Client {
	return &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: config.Timeout,
			}).Dial,

			Proxy: http.ProxyURL(config.ProxyURL),

			TLSHandshakeTimeout: config.Timeout,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.DisableTLSValidation,
			},
		},
	}
}
