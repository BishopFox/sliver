package configs

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
	"encoding/json"
	"fmt"
	"io/ioutil"
	insecureRand "math/rand"
	"os"
	"path"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/log"
)

const (
	httpC2ConfigFileName = "http-c2.json"
	chromeBaseVer        = 89
)

// HTTPC2Config - Parent config file struct for implant/server
type HTTPC2Config struct {
	ImplantConfig *HTTPC2ImplantConfig `json:"implant_config"`
	ServerConfig  *HTTPC2ServerConfig  `json:"server_config"`
}

// GenerateUserAgent - Generate a user-agent depending on OS/Arch
func (h *HTTPC2Config) GenerateUserAgent(goos string, goarch string) string {
	return h.generateChromeUserAgent(goos, goarch)
}

func (h *HTTPC2Config) generateChromeUserAgent(goos string, goarch string) string {
	if h.ImplantConfig.UserAgent == "" {
		switch goos {
		case "windows":
			switch goarch {
			case "amd64":
				return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", h.ChromeVer())
			}

		case "linux":
			switch goarch {
			case "amd64":
				return fmt.Sprintf("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", h.ChromeVer())
			}

		case "darwin":
			switch goarch {
			case "arm64":
				fallthrough // https://source.chromium.org/chromium/chromium/src/+/master:third_party/blink/renderer/core/frame/navigator_id.cc;l=76
			case "amd64":
				return fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", h.ChromeVer())
			}

		}
	} else {
		return h.ImplantConfig.UserAgent
	}

	// Default is a generic Windows/Chrome
	return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", h.ChromeVer())
}

// ChromeVer - Generate a random Chrome user-agent
func (h *HTTPC2Config) ChromeVer() string {
	return fmt.Sprintf("%d.0.%d.%d", chromeBaseVer+insecureRand.Intn(3), 1000+insecureRand.Intn(8999), insecureRand.Intn(999))
}

// RandomImplantConfig - Randomly generate a config
func (h *HTTPC2Config) RandomImplantConfig() *HTTPC2ImplantConfig {
	return &HTTPC2ImplantConfig{
		UserAgent: h.ImplantConfig.UserAgent,

		MaxFiles: h.ImplantConfig.MaxFiles,
		MaxPaths: h.ImplantConfig.MaxPaths,

		CssFiles: h.ImplantConfig.RandomCssFiles(),
		CssPaths: h.ImplantConfig.RandomCssPaths(),

		JsFiles: h.ImplantConfig.RandomJsFiles(),
		JsPaths: h.ImplantConfig.RandomJsPaths(),

		TxtFiles: h.ImplantConfig.RandomTxtFiles(),
		TxtPaths: h.ImplantConfig.RandomTxtPaths(),

		PngFiles: h.ImplantConfig.RandomPngFiles(),
		PngPaths: h.ImplantConfig.RandomPngPaths(),

		PhpFiles: h.ImplantConfig.RandomPhpFiles(),
		PhpPaths: h.ImplantConfig.RandomPhpPaths(),
	}
}

// HTTPC2ServerConfig - Server configuration options
type HTTPC2ServerConfig struct {
	Headers []string `json:"headers"`
	Cookies []string `json:"cookies"`
}

// HTTPC2ImplantConfig - Implant configuration options
// Procedural C2
// ===============
// .txt = rsakey
// .css = start
// .php = session
//  .js = poll
// .png = stop
// .woff = sliver shellcode
type HTTPC2ImplantConfig struct {
	UserAgent     string   `json:"user_agent"`
	URLParameters []string `json:"url_parameters"`
	Headers       []string `json:"headers"`

	MaxFiles int `json:"max_files"`
	MinFiles int `json:"min_files"`
	MaxPaths int `json:"max_paths"`
	MinPaths int `json:"min_paths"`

	// CSS files and paths
	CssFiles []string `json:"css_files"`
	CssPaths []string `json:"css_paths"`

	// JS files and paths
	JsFiles []string `json:"js_files"`
	JsPaths []string `json:"js_paths"`

	// Txt files and paths
	TxtFiles []string `json:"txt_files"`
	TxtPaths []string `json:"txt_paths"`

	// Png files and paths
	PngFiles []string `json:"png_files"`
	PngPaths []string `json:"png_paths"`

	// Php files and paths
	PhpFiles []string `json:"php_files"`
	PhpPaths []string `json:"php_paths"`
}

func (h *HTTPC2ImplantConfig) RandomCssFiles() []string {
	min := h.MinFiles
	if min < 1 {
		min = 1
	}
	return h.randomSample(h.CssFiles, min, h.MaxFiles)
}

func (h *HTTPC2ImplantConfig) RandomCssPaths() []string {
	return h.randomSample(h.CssPaths, h.MinPaths, h.MaxPaths)
}

func (h *HTTPC2ImplantConfig) RandomJsFiles() []string {
	min := h.MinFiles
	if min < 1 {
		min = 1
	}
	return h.randomSample(h.JsFiles, min, h.MaxFiles)
}

func (h *HTTPC2ImplantConfig) RandomJsPaths() []string {
	return h.randomSample(h.JsPaths, h.MinPaths, h.MaxPaths)
}

func (h *HTTPC2ImplantConfig) RandomTxtFiles() []string {
	min := h.MinFiles
	if min < 1 {
		min = 1
	}
	return h.randomSample(h.TxtFiles, min, h.MaxFiles)
}

func (h *HTTPC2ImplantConfig) RandomTxtPaths() []string {
	return h.randomSample(h.TxtPaths, h.MinPaths, h.MaxPaths)
}

func (h *HTTPC2ImplantConfig) RandomPngFiles() []string {
	min := h.MinFiles
	if min < 1 {
		min = 1
	}
	return h.randomSample(h.PngFiles, min, h.MaxFiles)
}

func (h *HTTPC2ImplantConfig) RandomPngPaths() []string {
	return h.randomSample(h.PngPaths, h.MinPaths, h.MaxPaths)
}

func (h *HTTPC2ImplantConfig) RandomPhpFiles() []string {
	min := h.MinFiles
	if min < 1 {
		min = 1
	}
	return h.randomSample(h.PhpFiles, min, h.MaxFiles)
}

func (h *HTTPC2ImplantConfig) RandomPhpPaths() []string {
	return h.randomSample(h.PhpPaths, h.MinPaths, h.MaxPaths)
}

func (h *HTTPC2ImplantConfig) randomSample(values []string, min int, max int) []string {
	count := insecureRand.Intn(len(values))
	if count < min {
		count = min
	}
	if max < count {
		count = max
	}
	sample := []string{}
	for i := 0; len(sample) < count; i++ {
		index := (count + i) % len(values)
		sample = append(sample, values[index])
	}
	return sample
}

var (
	httpC2ConfigLog = log.NamedLogger("config", "http-c2")

	defaultHTTPC2Config = &HTTPC2Config{
		ServerConfig: &HTTPC2ServerConfig{
			Cookies: []string{"PHPSESSID", "SID", "SSID", "APISID", "csrf-state", "AWSALBCORS"},
		},
		ImplantConfig: &HTTPC2ImplantConfig{
			UserAgent: "", // Blank string is rendered as randomized platform user-agent
			MaxFiles:  8,
			MinFiles:  2,
			MaxPaths:  8,
			MinPaths:  2,

			CssFiles: []string{"bootstrap.css", "bootstrap.min.css", "vendor.css"},
			CssPaths: []string{"css", "styles", "style", "stylesheets", "stylesheet"},

			JsFiles: []string{"bootstrap.js", "bootstrap.min.js", "jquery.min.js", "jquery.js"},
			JsPaths: []string{"js", "umd", "assets", "bundle", "bundles", "scripts", "script", "javascripts", "javascript"},

			TxtFiles: []string{"robots.txt", "sample.txt", "readme.txt", "example.txt"},
			TxtPaths: []string{"static", "www", "assets", "text", "docs", "sample"},

			PngFiles: []string{"favicon.png", "sample.png", "example.png"},
			PngPaths: []string{"static", "www", "assets", "images", "icons"},

			PhpFiles: []string{"login.php", "signin.php", "api.php", "samples.php", "rpc.php"},
			PhpPaths: []string{"php", "api", "upload", "actions", "rest", "v1", "async"},
		},
	}
)

// GetHTTPC2ConfigPath - File path to http-c2.json
func GetHTTPC2ConfigPath() string {
	appDir := assets.GetRootAppDir()
	httpC2ConfigPath := path.Join(appDir, "configs", httpC2ConfigFileName)
	return httpC2ConfigPath
}

// GetHTTPC2Config - Get the current HTTP C2 config
func GetHTTPC2Config() *HTTPC2Config {
	configPath := GetHTTPC2ConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		err = generateDefaultConfig(configPath)
		if err != nil {
			httpC2ConfigLog.Errorf("Failed to generate http c2 config %s", err)
			return defaultHTTPC2Config
		}
	}
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		httpC2ConfigLog.Errorf("Failed to read http c2 config %s", err)
		return defaultHTTPC2Config
	}
	config := &HTTPC2Config{}
	err = json.Unmarshal(data, config)
	if err != nil {
		httpC2ConfigLog.Errorf("Failed to parse http c2 config %s", err)
		return defaultHTTPC2Config
	}
	return config
}

func generateDefaultConfig(saveTo string) error {
	data, err := json.MarshalIndent(defaultHTTPC2Config, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(saveTo, data, 0600)
}
