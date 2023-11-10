package version

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	skipCheckEnv = "SLIVER_NO_UPDATE_CHECK"
	dateLayout   = "2006-01-02T15:04:05Z"
)

var (
	// GithubReleasesURL - Check this Github releases API for updates
	GithubReleasesURL string
)

// Release - A single Github release object
// https://developer.github.com/v3/repos/releases/
type Release struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	URL         string  `json:"url"`
	HTMLURL     string  `json:"html_url"`
	TagName     string  `json:"tag_name"`
	Body        string  `json:"body"`
	Prerelease  bool    `json:"prerelease"`
	TarballURL  string  `json:"tarball_url"`
	ZipballURL  string  `json:"zipball_url"`
	CreatedAt   string  `json:"created_at"`
	PublishedAt string  `json:"published_at"`
	Assets      []Asset `json:"assets"`
}

// Asset - Asset from a release
// https://developer.github.com/v3/repos/releases/
type Asset struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Size int    `json:"size"`

	BrowserDownloadURL string `json:"browser_download_url"`
}

// Created - Get the time the release was created
func (r *Release) Created() (time.Time, error) {
	return time.Parse(dateLayout, r.CreatedAt)
}

// Published - Get the time the release was published
func (r *Release) Published() (time.Time, error) {
	return time.Parse(dateLayout, r.PublishedAt)
}

// CheckForUpdates - Checks Github releases for newer versions, if any error
// occurs we don't really try to recover. If client is nil we just use the Go
// default client with default settings.
func CheckForUpdates(client *http.Client, prereleases bool) (*Release, error) {
	skip := os.Getenv(skipCheckEnv)
	if skip != "" || GithubReleasesURL == "" {
		return nil, nil
	}

	if client == nil {
		client = &http.Client{}
	}

	resp, err := client.Get(GithubReleasesURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("API returned non-200 status code")
	}
	releases := &[]*Release{}
	err = json.Unmarshal(body, releases)
	if err != nil {
		return nil, err
	}

	mySemVer := SemanticVersion()
	for _, release := range *releases {
		if release.Prerelease && !prereleases {
			continue
		}
		releaseSemVer := parseGitTag(release.TagName)
		for index, myVer := range mySemVer {
			if index < len(releaseSemVer) {
				if releaseSemVer[index] == myVer {
					continue
				}
				if releaseSemVer[index] < myVer {
					break
				}
				if myVer < releaseSemVer[index] {
					return release, nil
				}
			}
		}
	}
	return nil, nil
}

// parseGitTag - Get the structured sematic version
func parseGitTag(tag string) []int {
	semVer := []int{}
	version := tag
	if strings.Contains(version, "-") {
		version = strings.Split(version, "-")[0]
	}
	if strings.HasPrefix(version, "v") {
		version = version[1:]
	}
	for _, part := range strings.Split(version, ".") {
		number, _ := strconv.Atoi(part)
		semVer = append(semVer, number)
	}
	return semVer
}
