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
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/util/minisign"
)

// ArmoryIndexParser - Generic interface to fetch armory indexes
type ArmoryIndexParser func(*assets.ArmoryConfig, ArmoryHTTPConfig) (*ArmoryIndex, error)

// ArmoryPackageParser - Generic interface to fetch armory package manifests
type ArmoryPackageParser func(*assets.ArmoryConfig, *ArmoryPackage, bool, ArmoryHTTPConfig) (*minisign.Signature, []byte, error)

var (
	indexParsers = map[string]ArmoryIndexParser{
		"api.github.com": GithubAPIArmoryIndexParser,
	}
	pkgParsers = map[string]ArmoryPackageParser{
		"api.github.com": GithubAPIArmoryPackageParser,
		"github.com":     GithubArmoryPackageParser,
	}
)

const (
	armoryIndexFileName    = "armory.json"
	armoryIndexSigFileName = "armory.minisig"
)

type armoryIndexResponse struct {
	Minisig     string `json:"minisig"`      // Minisig (Base64)
	ArmoryIndex string `json:"armory_index"` // Index JSON (Base64)
}

type armoryPkgResponse struct {
	Minisig  string `json:"minisig"`    // Minisig (Base64)
	TarGzURL string `json:"tar_gz_url"` // Raw tar.gz url
}

func calculatePackageHash(pkg *ArmoryPackage) string {
	if pkg == nil {
		return ""
	}

	hasher := sha256.New()
	// Hash some of the things that make the package unique
	packageIdentifier := []byte(pkg.RepoURL + pkg.PublicKey + pkg.ArmoryName + pkg.CommandName)
	hasher.Write(packageIdentifier)
	return fmt.Sprintf("%x", hasher.Sum(nil))
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

	resp, body, err := httpRequest(clientConfig, armoryConfig.RepoURL, armoryConfig, http.Header{})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("api returned non-200 status code")
	}

	indexResp := &armoryIndexResponse{}
	err = json.Unmarshal(body, indexResp)
	if err != nil {
		return nil, err
	}
	armoryIndexData, err := base64.StdEncoding.DecodeString(indexResp.ArmoryIndex)
	if err != nil {
		return nil, err
	}
	armoryIndexSigData, err := base64.StdEncoding.DecodeString(indexResp.Minisig)
	if err != nil {
		return nil, err
	}

	// Verify index is signed by trusted key
	valid := minisign.Verify(publicKey, armoryIndexData, armoryIndexSigData)
	if !valid {
		return nil, errors.New("index has invalid signature")
	}

	armoryIndex := &ArmoryIndex{
		ArmoryConfig: armoryConfig,
	}
	err = json.Unmarshal(armoryIndexData, armoryIndex)
	if err != nil {
		return nil, err
	}
	// Populate armory name and ID information for assets
	for _, bundle := range armoryIndex.Bundles {
		bundle.ArmoryName = armoryIndex.ArmoryConfig.Name
	}
	for _, alias := range armoryIndex.Aliases {
		alias.ArmoryName = armoryIndex.ArmoryConfig.Name
		alias.ArmoryPK = armoryIndex.ArmoryConfig.PublicKey
		if cached := packageCacheLookupByCmdAndArmory(alias.CommandName, armoryIndex.ArmoryConfig.PublicKey); cached != nil {
			// A package with this name is in the cache, so we will remove it here so that the latest version is added
			// when the index is parsed
			pkgCache.Delete(cached.ID)
		}
		alias.ID = calculatePackageHash(alias)
	}
	for _, extension := range armoryIndex.Extensions {
		extension.ArmoryName = armoryIndex.ArmoryConfig.Name
		extension.ArmoryPK = armoryIndex.ArmoryConfig.PublicKey
		if cached := packageCacheLookupByCmdAndArmory(extension.CommandName, armoryIndex.ArmoryConfig.PublicKey); cached != nil {
			// A package with this name is in the cache, so we will remove it here so that the latest version is added
			// when the index is parsed
			pkgCache.Delete(cached.ID)
		}
		extension.ID = calculatePackageHash(extension)
	}
	return armoryIndex, nil
}

func decodePackageSignature(sigString string) ([]byte, error) {
	// Base64 decode the signature
	decodedData := make([]byte, base64.StdEncoding.DecodedLen(len(sigString)))
	_, err := base64.StdEncoding.Decode(decodedData, []byte(sigString))
	if err != nil {
		return nil, err
	}
	// If decodedData does not end up being base64.StdEncoding.DecodedLen(len(data)) bytes long, trim the null bytes
	decodedData = bytes.Trim(decodedData, "\x00")
	return decodedData, nil
}

// DefaultArmoryPkgParser - Parse the armory package manifest directly from the url
func DefaultArmoryPkgParser(armoryConfig *assets.ArmoryConfig, armoryPkg *ArmoryPackage, sigOnly bool, clientConfig ArmoryHTTPConfig) (*minisign.Signature, []byte, error) {
	var publicKey minisign.PublicKey
	err := publicKey.UnmarshalText([]byte(armoryPkg.PublicKey))
	if err != nil {
		return nil, nil, err
	}

	resp, body, err := httpRequest(clientConfig, armoryPkg.RepoURL, armoryConfig, http.Header{})
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("api returned non-200 status code (%d)", resp.StatusCode)
	}
	pkgResp := &armoryPkgResponse{}
	err = json.Unmarshal(body, pkgResp)
	if err != nil {
		return nil, nil, err
	}
	decodedSig, err := decodePackageSignature(pkgResp.Minisig)
	if err != nil {
		return nil, nil, err
	}
	sig, err := parsePkgMinsig(decodedSig)
	if err != nil {
		return nil, nil, err
	}
	var tarGz []byte
	if !sigOnly {
		tarGzURL, err := url.Parse(pkgResp.TarGzURL)
		if err != nil {
			return nil, nil, err
		}
		if tarGzURL.Scheme != "https" && tarGzURL.Scheme != "http" {
			return nil, nil, errors.New("invalid url scheme")
		}
		tarGz, err = downloadRequest(clientConfig, tarGzURL.String(), armoryConfig)
		if err != nil {
			return nil, nil, err
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

	resp, body, err := httpRequest(clientConfig, armoryConfig.RepoURL, armoryConfig, http.Header{})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			return nil, errors.New("you hit the github api rate limit (60 req/hr), try later")
		}
		return nil, fmt.Errorf("api returned non-200 status code: %d", resp.StatusCode)
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
			armoryIndexData, err = downloadRequest(clientConfig, asset.URL, armoryConfig)
			if err != nil {
				return nil, err
			}
		}
		if asset.Name == armoryIndexSigFileName {
			sigData, err = downloadRequest(clientConfig, asset.URL, armoryConfig)
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

	armoryIndex := &ArmoryIndex{
		ArmoryConfig: armoryConfig,
	}
	err = json.Unmarshal(armoryIndexData, armoryIndex)
	if err != nil {
		return nil, err
	}
	// Populate armory name and ID information for assets
	for _, bundle := range armoryIndex.Bundles {
		bundle.ArmoryName = armoryIndex.ArmoryConfig.Name
	}
	for _, alias := range armoryIndex.Aliases {
		alias.ArmoryName = armoryIndex.ArmoryConfig.Name
		alias.ArmoryPK = armoryIndex.ArmoryConfig.PublicKey
		alias.ID = calculatePackageHash(alias)
	}
	for _, extension := range armoryIndex.Extensions {
		extension.ArmoryName = armoryIndex.ArmoryConfig.Name
		extension.ArmoryPK = armoryIndex.ArmoryConfig.PublicKey
		extension.ID = calculatePackageHash(extension)
	}
	return armoryIndex, nil
}

// GithubAPIArmoryPackageParser - Retrieve the minisig and tar.gz for an armory package from a GitHub release
func GithubAPIArmoryPackageParser(armoryConfig *assets.ArmoryConfig, armoryPkg *ArmoryPackage, sigOnly bool, clientConfig ArmoryHTTPConfig) (*minisign.Signature, []byte, error) {
	var publicKey minisign.PublicKey
	err := publicKey.UnmarshalText([]byte(armoryPkg.PublicKey))
	if err != nil {
		return nil, nil, err
	}

	resp, body, err := httpRequest(clientConfig, armoryPkg.RepoURL, armoryConfig, http.Header{})
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			return nil, nil, errors.New("you hit the github api rate limit (60 req/hr), try later")
		}
		return nil, nil, fmt.Errorf("api returned non-200 status code: %d", resp.StatusCode)
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
			body, err := downloadRequest(clientConfig, asset.URL, armoryConfig)
			if err != nil {
				break
			}
			sig, err = parsePkgMinsig(body)
			if err != nil {
				break
			}
		}
		if asset.Name == fmt.Sprintf("%s.tar.gz", armoryPkg.CommandName) && !sigOnly {
			tarGz, err = downloadRequest(clientConfig, asset.URL, armoryConfig)
			if err != nil {
				break
			}
		}
	}
	return sig, tarGz, err
}

//
// GitHub Parsers
//

// GithubArmoryPackageParser - Uses github.com instead of api.github.com to download packages
func GithubArmoryPackageParser(_ *assets.ArmoryConfig, armoryPkg *ArmoryPackage, sigOnly bool, clientConfig ArmoryHTTPConfig) (*minisign.Signature, []byte, error) {
	latestTag, err := githubLatestTagParser(armoryPkg, clientConfig)
	if err != nil {
		return nil, nil, err
	}

	sigURL, err := url.Parse(armoryPkg.RepoURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse armory pkg url '%s': %s", armoryPkg.RepoURL, err)
	}
	sigURL.Path = path.Join(sigURL.Path, "releases", "download", latestTag, fmt.Sprintf("%s.minisig", armoryPkg.CommandName))

	// Setup dummy auth here as the non-api endpoints don't support the Authorization header
	noAuth := &assets.ArmoryConfig{
		Authorization: "",
	}
	body, err := downloadRequest(clientConfig, sigURL.String(), noAuth)
	if err != nil {
		return nil, nil, err
	}

	sig, err := parsePkgMinsig(body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse pkg sig '%s': %s", armoryPkg.RepoURL, err)
	}

	var tarGz []byte
	if !sigOnly {
		tarGzURL, err := url.Parse(armoryPkg.RepoURL)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse armory pkg url '%s': %s", armoryPkg.RepoURL, err)
		}
		tarGzURL.Path = path.Join(tarGzURL.Path, "releases", "download", latestTag, fmt.Sprintf("%s.tar.gz", armoryPkg.CommandName))
		tarGz, err = downloadRequest(clientConfig, tarGzURL.String(), noAuth)
		if err != nil {
			return nil, nil, err
		}
	}

	return sig, tarGz, nil
}

// We need to intercept the 302 redirect to determine the latest version tag
func githubLatestTagParser(armoryPkg *ArmoryPackage, clientConfig ArmoryHTTPConfig) (string, error) {
	client := httpClient(clientConfig)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	latestURL, err := url.Parse(armoryPkg.RepoURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse armory pkg url '%s': %s", armoryPkg.RepoURL, err)
	}
	latestURL.Path = path.Join(latestURL.Path, "releases", "latest")
	latestRedirect, err := client.Get(latestURL.String())
	if err != nil {
		return "", fmt.Errorf("http get failed armory pkg url '%s': %s", armoryPkg.RepoURL, err)
	}
	defer latestRedirect.Body.Close()
	if latestRedirect.StatusCode != http.StatusFound {
		return "", fmt.Errorf("unexpected response status (wanted 302) '%s': %s", armoryPkg.RepoURL, latestRedirect.Status)
	}
	if latestRedirect.Header.Get("Location") == "" {
		return "", fmt.Errorf("no location header in response '%s'", armoryPkg.RepoURL)
	}
	latestLocationURL, err := url.Parse(latestRedirect.Header.Get("Location"))
	if err != nil {
		return "", fmt.Errorf("failed to parse location header '%s'->'%s': %s",
			armoryPkg.RepoURL, latestRedirect.Header.Get("Location"), err)
	}
	pathSegments := strings.Split(latestLocationURL.Path, "/")
	for index, segment := range pathSegments {
		if segment == "tag" && index+1 < len(pathSegments) {
			return pathSegments[index+1], nil
		}
	}
	return "", errors.New("tag not found in location header")
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
			IdleConnTimeout:     time.Millisecond,
			Proxy:               http.ProxyURL(config.ProxyURL),
			TLSHandshakeTimeout: config.Timeout,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.DisableTLSValidation,
			},
		},
	}
}

func httpRequest(clientConfig ArmoryHTTPConfig, reqURL string, armoryConfig *assets.ArmoryConfig, extraHeaders http.Header) (*http.Response, []byte, error) {
	client := httpClient(clientConfig)
	req, err := http.NewRequest(http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, nil, err
	}

	if len(extraHeaders) > 0 {
		for key := range extraHeaders {
			req.Header.Add(key, strings.Join(extraHeaders[key], ","))
		}
	}
	if armoryConfig.Authorization != "" {
		req.Header.Set("Authorization", armoryConfig.Authorization)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	return resp, body, err
}

func downloadRequest(clientConfig ArmoryHTTPConfig, reqURL string, armoryConfig *assets.ArmoryConfig) ([]byte, error) {
	downloadHdr := http.Header{
		"Accept": {"application/octet-stream"},
	}
	resp, body, err := httpRequest(clientConfig, reqURL, armoryConfig, downloadHdr)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("invalid response when downloading %s, try again later", reqURL)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("error downloading asset: http %d", resp.StatusCode)
	}

	return body, err
}
