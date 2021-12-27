package armory

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/server/cryptography/minisign"
)

// ArmoryIndexParser - Generic interface to fetch armory indexes
type ArmoryIndexParser func(*assets.ArmoryConfig, ArmoryHTTPConfig) (*ArmoryIndex, error)

// ArmoryPackageParser - Generic interface to fetch armory package manifests
type ArmoryPackageParser func(string, string, bool, ArmoryHTTPConfig) (*ArmoryPackage, []byte, error)

var (
	indexParsers = map[string]ArmoryIndexParser{
		"github.com": GitHubArmoryIndexParser,
	}

	pkgParsers = map[string]ArmoryPackageParser{
		"github.com": GitHubArmoryPackageParser,
	}
)

// "armory.json"
const (
	armoryIndexResponseFileName = "armory.json"
)

type armoryIndexResponse struct {
	Minisig     string `json:"minisig"`      // Minisig
	ArmoryIndex string `json:"armory_index"` // Base64 String
}

type armoryPkgResponse struct {
	Minisig  string `json:"minisig"` // Minisig
	TarGzURL string `json:"tar_gz_url"`
}

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
	if resp.StatusCode != 200 {
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

// GitHubArmoryIndexParser - Parse the armory index from a GitHub release
func GitHubArmoryIndexParser(armoryConfig *assets.ArmoryConfig, clientConfig ArmoryHTTPConfig) (*ArmoryIndex, error) {
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
	if resp.StatusCode != 200 {
		return nil, errors.New("api returned non-200 status code")
	}

	release := &GithubRelease{}
	err = json.Unmarshal(body, release)
	if err != nil {
		return nil, err
	}

	for _, asset := range release.Assets {
		if asset.Name == armoryIndexResponseFileName {
			resp, err := client.Get(asset.BrowserDownloadURL)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
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
	}
	return nil, fmt.Errorf("no %s found", armoryIndexResponseFileName)
}

func GitHubArmoryPackageParser(rawRepoURL string, rawPublicKey string, sigOnly bool, clientConfig ArmoryHTTPConfig) (*ArmoryPackage, []byte, error) {
	var publicKey minisign.PublicKey
	err := publicKey.UnmarshalText([]byte(rawPublicKey))
	if err != nil {
		return nil, nil, err
	}

	client := httpClient(clientConfig)
	resp, err := client.Get(rawRepoURL)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != 200 {
		return nil, nil, errors.New("api returned non-200 status code")
	}

	release := &GithubRelease{}
	err = json.Unmarshal(body, release)
	if err != nil {
		return nil, nil, err
	}

	var pkg *ArmoryPackage
	var tarGz []byte
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ".minisig") {
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
			pkg, err = parsePkgMinsig(body)
			if err != nil {
				break
			}
		}
		if strings.HasSuffix(asset.Name, ".tar.gz") && !sigOnly {
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

	return pkg, tarGz, err
}

func parsePkgMinsig(data []byte) (*ArmoryPackage, error) {
	var sig minisign.Signature
	err := sig.UnmarshalText(data)
	if err != nil {
		return nil, err
	}
	if len(sig.TrustedComment) < 1 {
		return nil, errors.New("missing trusted comment")
	}
	manifestData, err := base64.StdEncoding.DecodeString(sig.TrustedComment)
	if err != nil {
		return nil, err
	}
	var manifest *ArmoryPackage
	err = json.Unmarshal(manifestData, manifest)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func httpClient(config ArmoryHTTPConfig) *http.Client {
	return &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: config.Timeout,
			}).Dial,
			TLSHandshakeTimeout: config.Timeout,
			Proxy:               http.ProxyURL(config.ProxyURL),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.DisableTLSValidation,
			},
		},
	}
}
