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
	"errors"
	"fmt"
	"io/ioutil"
	insecureRand "math/rand"
	"os"
	"path"
	"regexp"
	"strings"

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
	config := &HTTPC2ImplantConfig{}
	*config = *h.ImplantConfig

	config.KeyExchangeFiles = h.ImplantConfig.RandomKeyExchangeFiles()
	config.KeyExchangePaths = h.ImplantConfig.RandomKeyExchangePaths()

	config.PollFiles = h.ImplantConfig.RandomPollFiles()
	config.PollPaths = h.ImplantConfig.RandomPollPaths()

	config.SessionFiles = h.ImplantConfig.RandomSessionFiles()
	config.SessionPaths = h.ImplantConfig.RandomSessionPaths()

	config.CloseFiles = h.ImplantConfig.RandomCloseFiles()
	config.ClosePaths = h.ImplantConfig.RandomClosePaths()

	return config
}

type HTTPHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HTTPC2ServerConfig - Server configuration options
type HTTPC2ServerConfig struct {
	RandomVersionHeaders bool         `json:"random_version_headers"`
	ExtraHeaders         []HTTPHeader `json:"headers"`
	Cookies              []string     `json:"cookies"`
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

	// Stager File Extension
	StagerFileExt string `json:"stager_file_ext"`

	// Key Exchange (default .txt) files and paths
	KeyExchangeFileExt string   `json:"key_exchange_file_ext"`
	KeyExchangeFiles   []string `json:"key_exchange_files"`
	KeyExchangePaths   []string `json:"key_exchange_paths"`

	// Poll files and paths
	PollFileExt string   `json:"poll_file_ext"`
	PollFiles   []string `json:"poll_files"`
	PollPaths   []string `json:"poll_paths"`

	// Session files and paths
	StartSessionFileExt string   `json:"start_session_file_ext"`
	SessionFileExt      string   `json:"session_file_ext"`
	SessionFiles        []string `json:"session_files"`
	SessionPaths        []string `json:"session_paths"`

	// Close session files and paths
	CloseFileExt string   `json:"close_file_ext"`
	CloseFiles   []string `json:"close_files"`
	ClosePaths   []string `json:"close_paths"`
}

func (h *HTTPC2ImplantConfig) RandomKeyExchangeFiles() []string {
	min := h.MinFiles
	if min < 1 {
		min = 1
	}
	return h.randomSample(h.KeyExchangeFiles, h.KeyExchangeFileExt, min, h.MaxFiles)
}

func (h *HTTPC2ImplantConfig) RandomKeyExchangePaths() []string {
	return h.randomSample(h.KeyExchangePaths, "", h.MinPaths, h.MaxPaths)
}

func (h *HTTPC2ImplantConfig) RandomPollFiles() []string {
	min := h.MinFiles
	if min < 1 {
		min = 1
	}
	return h.randomSample(h.PollFiles, h.PollFileExt, min, h.MaxFiles)
}

func (h *HTTPC2ImplantConfig) RandomPollPaths() []string {
	return h.randomSample(h.PollPaths, "", h.MinPaths, h.MaxPaths)
}

func (h *HTTPC2ImplantConfig) RandomCloseFiles() []string {
	min := h.MinFiles
	if min < 1 {
		min = 1
	}
	return h.randomSample(h.CloseFiles, h.CloseFileExt, min, h.MaxFiles)
}

func (h *HTTPC2ImplantConfig) RandomClosePaths() []string {
	return h.randomSample(h.ClosePaths, "", h.MinPaths, h.MaxPaths)
}

func (h *HTTPC2ImplantConfig) RandomSessionFiles() []string {
	min := h.MinFiles
	if min < 1 {
		min = 1
	}
	return h.randomSample(h.SessionFiles, h.SessionFileExt, min, h.MaxFiles)
}

func (h *HTTPC2ImplantConfig) RandomSessionPaths() []string {
	return h.randomSample(h.SessionPaths, "", h.MinPaths, h.MaxPaths)
}

func (h *HTTPC2ImplantConfig) randomSample(values []string, ext string, min int, max int) []string {
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

	defaultHTTPC2Config = HTTPC2Config{
		ServerConfig: &HTTPC2ServerConfig{
			RandomVersionHeaders: false,
			Cookies: []string{
				"PHPSESSID", "SID", "SSID", "APISID", "csrf-state", "AWSALBCORS",
			},
			ExtraHeaders: []HTTPHeader{
				{"Cache-Control", "no-store, no-cache, must-revalidate"},
			},
		},
		ImplantConfig: &HTTPC2ImplantConfig{
			UserAgent: "", // Blank string is rendered as randomized platform user-agent
			MaxFiles:  8,
			MinFiles:  2,
			MaxPaths:  8,
			MinPaths:  2,

			StagerFileExt: ".woff",

			KeyExchangeFileExt: ".txt",
			KeyExchangeFiles: []string{
				"robots", "sample", "readme", "example",
			},
			KeyExchangePaths: []string{
				"static", "www", "assets", "text", "docs", "sample", "data", "readme",
				"examples",
			},

			PollFileExt: ".js",
			PollFiles: []string{
				"bootstrap", "bootstrap.min", "jquery.min", "jquery", "route",
				"app", "app.min", "array", "backbone", "script", "email",
			},
			PollPaths: []string{
				"js", "umd", "assets", "bundle", "bundles", "scripts", "script", "javascripts",
				"javascript", "jscript",
			},

			StartSessionFileExt: ".phtml",
			SessionFileExt:      ".php",
			SessionFiles: []string{
				"login", "signin", "api", "samples", "rpc", "index",
				"admin", "register", "sign-up",
			},
			SessionPaths: []string{
				"php", "api", "upload", "actions", "rest", "v1", "auth", "authenticate",
				"oauth", "oauth2", "oauth2callback", "database", "db", "namespaces",
			},

			CloseFileExt: ".png",
			CloseFiles: []string{
				"favicon", "sample", "example",
			},
			ClosePaths: []string{
				"static", "www", "assets", "images", "icons", "image", "icon", "png",
			},
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
			return &defaultHTTPC2Config
		}
	}
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		httpC2ConfigLog.Errorf("Failed to read http c2 config %s", err)
		return &defaultHTTPC2Config
	}
	config := &HTTPC2Config{}
	err = json.Unmarshal(data, config)
	if err != nil {
		httpC2ConfigLog.Errorf("Failed to parse http c2 config %s", err)
		return &defaultHTTPC2Config
	}
	err = CheckHTTPC2Config(config)
	if err != nil {
		httpC2ConfigLog.Errorf("Invalid http c2 config: %s", err)
		return &defaultHTTPC2Config
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

var (
	ErrMissingCookies             = errors.New("server config must specify at least one cookie")
	ErrMissingStagerFileExt       = errors.New("implant config must specify a stager_file_ext")
	ErrMissingPollFileExt         = errors.New("implant config must specify a poll_file_ext")
	ErrTooFewPollFiles            = errors.New("implant config must specify at least one poll_files value")
	ErrMissingKeyExchangeFileExt  = errors.New("implant config must specify a key_exchange_file_ext")
	ErrTooFewKeyExchangeFiles     = errors.New("implant config must specify at least one key_exchange_files value")
	ErrMissingCloseFileExt        = errors.New("implant config must specify a close_file_ext")
	ErrTooFewCloseFiles           = errors.New("implant config must specify at least one close_files value")
	ErrMissingStartSessionFileExt = errors.New("implant config must specify a start_session_file_ext")
	ErrMissingSessionFileExt      = errors.New("implant config must specify a session_file_ext")
	ErrTooFewSessionFiles         = errors.New("implant config must specify at least one session_files value")
	ErrNonuniqueFileExt           = errors.New("implant config must specify unique file extensions")

	fileNameExp = regexp.MustCompile("[^a-zA-Z0-9\\._-]+")
)

// CheckHTTPC2Config - Validate the HTTP C2 config, coerces common mistakes
func CheckHTTPC2Config(config *HTTPC2Config) error {
	err := checkServerConfig(config.ServerConfig)
	if err != nil {
		return err
	}
	return checkImplantConfig(config.ImplantConfig)
}

func coerceFileExt(value string) string {
	value = fileNameExp.ReplaceAllString(value, "")
	for strings.HasPrefix(value, ".") {
		value = strings.TrimPrefix(value, ".")
	}
	return value
}

func coerceFiles(values []string, ext string) []string {
	values = uniqueFileName(values)
	coerced := []string{}
	for _, value := range values {
		if strings.HasSuffix(value, fmt.Sprintf(".%s", ext)) {
			value = strings.TrimSuffix(value, fmt.Sprintf(".%s", ext))
		}
		coerced = append(coerced, value)
	}
	return coerced
}

func uniqueFileName(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		item = fileNameExp.ReplaceAllString(item, "")
		if len(item) < 1 {
			continue
		}
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func checkServerConfig(config *HTTPC2ServerConfig) error {
	if len(config.Cookies) < 1 {
		return ErrMissingCookies
	}
	return nil
}

func checkImplantConfig(config *HTTPC2ImplantConfig) error {

	// MinFiles and MaxFiles
	if config.MinFiles < 1 {
		config.MinFiles = 1
	}
	if config.MaxFiles < config.MinFiles {
		config.MaxFiles = config.MinFiles
	}

	// MinPaths and MaxPaths
	if config.MinPaths < 0 {
		config.MinPaths = 0
	}
	if config.MaxPaths < config.MinPaths {
		config.MaxPaths = config.MinPaths
	}

	// Stager
	config.StagerFileExt = coerceFileExt(config.StagerFileExt)
	if config.StagerFileExt == "" {
		return ErrMissingStagerFileExt
	}

	// Key Exchange
	config.KeyExchangeFileExt = coerceFileExt(config.KeyExchangeFileExt)
	if config.KeyExchangeFileExt == "" {
		return ErrMissingKeyExchangeFileExt
	}
	config.KeyExchangeFiles = coerceFiles(config.KeyExchangeFiles, config.KeyExchangeFileExt)
	if len(config.KeyExchangeFiles) < 1 {
		return ErrTooFewKeyExchangeFiles
	}

	// Poll Settings
	config.PollFileExt = coerceFileExt(config.PollFileExt)
	if config.PollFileExt == "" {
		return ErrMissingPollFileExt
	}
	config.PollFiles = coerceFiles(config.PollFiles, config.PollFileExt)
	if len(config.PollFiles) < 1 {
		return ErrTooFewPollFiles
	}

	// Session Settings
	config.StartSessionFileExt = coerceFileExt(config.StartSessionFileExt)
	if config.StartSessionFileExt == "" {
		return ErrMissingStartSessionFileExt
	}
	config.SessionFileExt = coerceFileExt(config.SessionFileExt)
	if config.SessionFileExt == "" {
		return ErrMissingSessionFileExt
	}
	config.SessionFiles = coerceFiles(config.SessionFiles, config.StartSessionFileExt)
	config.SessionFiles = coerceFiles(config.SessionFiles, config.SessionFileExt)
	if len(config.SessionFiles) < 1 {
		return ErrTooFewSessionFiles
	}

	// Close Settings
	config.CloseFileExt = coerceFileExt(config.CloseFileExt)
	if config.CloseFileExt == "" {
		return ErrMissingCloseFileExt
	}
	config.CloseFiles = coerceFiles(config.CloseFiles, config.CloseFileExt)
	if len(config.CloseFiles) < 1 {
		return ErrTooFewCloseFiles
	}

	// Unique file extensions
	allExtensions := map[string]bool{}
	extensions := []string{
		config.StagerFileExt,
		config.KeyExchangeFileExt,
		config.PollFileExt,
		config.StartSessionFileExt,
		config.SessionFileExt,
		config.CloseFileExt,
	}
	for _, ext := range extensions {
		if _, ok := allExtensions[ext]; ok {
			return ErrNonuniqueFileExt
		}
		allExtensions[ext] = true
	}

	return nil
}
