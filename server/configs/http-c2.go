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
	DefaultChromeBaseVer = 106
	DefaultMacOSVer      = "10_15_7"
)

// HTTPC2Config - Parent config file struct for implant/server
type HTTPC2Config struct {
	ImplantConfig *HTTPC2ImplantConfig `json:"implant_config"`
	ServerConfig  *HTTPC2ServerConfig  `json:"server_config"`
}

// RandomImplantConfig - Randomly generate a new implant config from the parent config,
// this is the primary configuration used by the implant generation.
func (h *HTTPC2Config) RandomImplantConfig() *HTTPC2ImplantConfig {
	return &HTTPC2ImplantConfig{

		NonceQueryArgs: h.ImplantConfig.NonceQueryArgs,
		URLParameters:  h.ImplantConfig.URLParameters,
		Headers:        h.ImplantConfig.Headers,

		PollFileExt: h.ImplantConfig.PollFileExt,
		PollFiles:   h.ImplantConfig.RandomPollFiles(),
		PollPaths:   h.ImplantConfig.RandomPollPaths(),

		StartSessionFileExt: h.ImplantConfig.StartSessionFileExt,
		SessionFileExt:      h.ImplantConfig.SessionFileExt,
		SessionFiles:        h.ImplantConfig.RandomSessionFiles(),
		SessionPaths:        h.ImplantConfig.RandomSessionPaths(),

		CloseFileExt: h.ImplantConfig.CloseFileExt,
		CloseFiles:   h.ImplantConfig.RandomCloseFiles(),
		ClosePaths:   h.ImplantConfig.RandomClosePaths(),
	}
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
				return fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X %s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", h.MacOSVer(), h.ChromeVer())
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
	chromeVer := h.ImplantConfig.ChromeBaseVersion
	if chromeVer == 0 {
		chromeVer = DefaultChromeBaseVer
	}
	return fmt.Sprintf("%d.0.%d.%d", chromeVer+insecureRand.Intn(3), 1000+insecureRand.Intn(8999), insecureRand.Intn(999))
}

func (h *HTTPC2Config) MacOSVer() string {
	macosVer := h.ImplantConfig.MacOSVersion
	if macosVer == "" {
		macosVer = DefaultMacOSVer
	}
	return macosVer
}

// HTTPC2ServerConfig - Server configuration options
type HTTPC2ServerConfig struct {
	RandomVersionHeaders bool                   `json:"random_version_headers"`
	Headers              []NameValueProbability `json:"headers"`
	Cookies              []string               `json:"cookies"`
}

type NameValueProbability struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Probability int    `json:"probability"`
	Methods     []string
}

// HTTPC2ImplantConfig - Implant configuration options
// Procedural C2
// ===============
// .txt = rsakey
// .css = start
// .php = session
//
//	.js = poll
//
// .png = stop
// .woff = sliver shellcode
type HTTPC2ImplantConfig struct {
	UserAgent         string `json:"user_agent"`
	ChromeBaseVersion int    `json:"chrome_base_version"`
	MacOSVersion      string `json:"macos_version"`

	NonceQueryArgs string                 `json:"nonce_query_args"`
	URLParameters  []NameValueProbability `json:"url_parameters"`
	Headers        []NameValueProbability `json:"headers"`

	MaxFiles int `json:"max_files"`
	MinFiles int `json:"min_files"`
	MaxPaths int `json:"max_paths"`
	MinPaths int `json:"min_paths"`

	// Stager File Extension
	StagerFileExt string `json:"stager_file_ext"`

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
				"PHPSESSID",
				"SID",
				"SSID",
				"APISID",
				"csrf-state",
				"AWSALBCORS",

				"cookie",
				"session",
				"lang",
				"user",
				"auth",
				"remember_me",
				"preferences",
				"cart",
				"basket",
				"order",
				"recent_items",
				"search_history",
				"tracking",
				"analytics",
				"marketing",
				"consent",
				"notification",
				"popup",
				"ad",
				"banner",
				"survey",
				"feedback",
				"login",
				"logged_in",
				"logout",
				"visit",
				"visitor",
				"viewed",
				"visited",
				"liked",
				"favorite",
				"last_visit",
				"first_visit",
				"referral",
				"source",
				"utm_campaign",
				"utm_source",
				"utm_medium",
				"utm_content",
				"utm_term",
				"affiliate",
				"coupon",
				"discount",
				"promo",
				"newsletter",
				"subscription",
				"consent_tracking",
				"consent_analytics",
				"consent_marketing",
				"consent_personalization",
				"consent_advertising",
				"consent_preferences",
				"consent_statistics",
				"consent_security",
				"consent_performance",
				"consent_functionality",
				"consent_other",
				"consent_required",
				"consent_given",
				"consent_revoked",
				"error",
				"alert",
				"message",
				"notification",
				"language",
				"currency",
				"timezone",
				"geolocation",
				"device",
				"screen_resolution",
				"browser",
				"os",
				"platform",
				"session_timeout",
				"remember_me",
				"cart_items",
				"order_total",
				"shipping_address",
				"billing_address",
				"payment_method",
				"discount_code",
				"login_status",
				"username",
				"email",
				"role",
				"permission",
				"authentication_token",
				"csrf_token",
				"form_data",
				"popup_closed",
				"consent_given",
				"consent_declined",
				"consent_age_verification",
				"consent_cookie_policy",
				"consent_gdpr",
				"consent_ccpa",
				"consent_eprivacy",
				"consent_cookie_notice",
				"consent_terms_conditions",
				"consent_privacy_policy",
			},
			Headers: []NameValueProbability{
				// {Name: "Cache-Control", Value: "no-store, no-cache, must-revalidate", Probability: 100},
			},
		},
		ImplantConfig: &HTTPC2ImplantConfig{
			UserAgent:         "", // Blank string is rendered as randomized platform user-agent
			ChromeBaseVersion: DefaultChromeBaseVer,
			MacOSVersion:      DefaultMacOSVer,
			MaxFiles:          8,
			MinFiles:          2,
			MaxPaths:          8,
			MinPaths:          2,

			Headers: []NameValueProbability{
				{Name: "Accept-Language", Value: "en-US,en;q=0.9", Methods: []string{"GET"}, Probability: 100},
				{Name: "Accept", Value: "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9", Methods: []string{"GET"}, Probability: 100},
			},

			NonceQueryArgs: "abcdefghijklmnopqrstuvwxyz",

			StagerFileExt: ".woff",

			PollFileExt: ".js",
			PollFiles: []string{
				"main",
				"app",
				"script",
				"index",
				"utils",
				"jquery",
				"bootstrap",
				"angular",
				"react",
				"vue",
				"lodash",
				"moment",
				"axios",
				"underscore",
				"d3",
				"chart",
				"map",
				"validation",
				"animation",
				"slider",
				"modal",
				"form",
				"ajax",
				"cookie",
				"dom",
				"events",
				"navigation",
				"menu",
				"dropdown",
				"carousel",
				"scroll",
				"pagination",
				"tabs",
				"accordion",
				"tooltip",
				"popover",
				"alert",
				"notification",
				"progress",
				"loader",
				"countdown",
				"lazyload",
				"parallax",
				"video",
				"audio",
				"slideshow",
				"gallery",
				"lightbox",
				"share",
				"social",
				"analytics",
				"tracking",
				"search",
				"autocomplete",
				"filter",
				"sort",
				"table",
				"chart",
				"graph",
				"calendar",
				"datepicker",
				"timepicker",
				"dropdown",
				"multi-select",
				"form-validation",
				"tooltip",
				"popover",
				"modal",
				"sidebar",
				"drawer",
				"sticky",
				"scrollspy",
				"smoothscroll",
				"anchor",
				"slideshow",
				"testimonial",
				"newsletter",
				"login",
				"registration",
				"cart",
				"checkout",
				"payment",
				"validation",
				"maps",
				"geolocation",
				"geocoding",
				"canvas",
				"webgl",
				"particles",
				"barcode",
				"qr-code",
				"encryption",
				"decryption",
				"localization",
				"translation",
				"i18n",
				"routing",
				"router",
				"storage",
				"offline",
			},
			PollPaths: []string{
				"js",
				"scripts",
				"assets",
				"src",
				"lib",
				"public",
				"static",
				"app",
				"www",
				"dist",
				"frontend",
				"client",
				"server",
				"resources",
				"js_files",
				"javascript",
				"js-lib",
				"js-libraries",
				"js_dir",
				"js_folder",
				"js_files_dir",
				"js_files_folder",
				"scripts_dir",
				"scripts_folder",
				"scripts_files",
				"scripts_files_dir",
				"scripts_files_folder",
				"assets_js",
				"assets_scripts",
				"src_js",
				"src_scripts",
				"lib_js",
				"lib_scripts",
				"public_js",
				"public_scripts",
				"static_js",
				"static_scripts",
				"app_js",
				"app_scripts",
				"www_js",
				"www_scripts",
				"dist_js",
				"dist_scripts",
				"frontend_js",
				"frontend_scripts",
				"client_js",
				"client_scripts",
				"server_js",
				"server_scripts",
				"resources_js",
				"resources_scripts",
				"js_files_js",
				"js_files_scripts",
				"javascript_js",
				"javascript_scripts",
				"js-lib_js",
				"js-lib_scripts",
				"js-libraries_js",
				"js-libraries_scripts",
				"js_dir_js",
				"js_dir_scripts",
				"js_folder_js",
				"js_folder_scripts",
				"js_files_dir_js",
				"js_files_dir_scripts",
				"js_files_folder_js",
				"js_files_folder_scripts",
				"scripts_dir_js",
				"scripts_dir_scripts",
				"scripts_folder_js",
				"scripts_folder_scripts",
				"scripts_files_js",
				"scripts_files_scripts",
				"scripts_files_dir_js",
				"scripts_files_dir_scripts",
				"scripts_files_folder_js",
				"scripts_files_folder_scripts",
				"assets_js_js",
				"assets_js_scripts",
				"assets_scripts_js",
				"assets_scripts_scripts",
				"src_js_js",
				"src_js_scripts",
				"src_scripts_js",
				"src_scripts_scripts",
				"lib_js_js",
				"lib_js_scripts",
				"lib_scripts_js",
				"lib_scripts_scripts",
			},

			StartSessionFileExt: ".html",
			SessionFileExt:      ".php",
			SessionFiles: []string{
				"index",
				"home",
				"login",
				"register",
				"dashboard",
				"profile",
				"settings",
				"config",
				"functions",
				"header",
				"footer",
				"navigation",
				"database",
				"connection",
				"form",
				"action",
				"validation",
				"upload",
				"download",
				"search",
				"results",
				"gallery",
				"blog",
				"article",
				"category",
				"archive",
				"single",
				"contact",
				"about",
				"faq",
				"error",
				"maintenance",
				"admin",
				"admin_login",
				"admin_dashboard",
				"admin_users",
				"admin_settings",
				"admin_products",
				"admin_categories",
				"admin_orders",
				"admin_reports",
				"admin_logs",
				"admin_logout",
				"api",
				"webhook",
				"cron",
				"email",
				"newsletter",
				"invoice",
				"payment",
				"cart",
				"checkout",
				"confirmation",
				"success",
				"error",
				"thank_you",
				"subscribe",
				"unsubscribe",
				"contact_us",
				"privacy",
				"terms",
				"cookie",
				"sitemap",
				"rss",
				"feed",
				"mobile",
				"desktop",
				"responsive",
				"ajax",
				"json",
				"xml",
				"captcha",
				"authentication",
				"authorization",
				"session",
				"cookies",
				"cache",
				"logging",
				"utilities",
				"helpers",
				"constants",
				"routes",
				"error_handler",
				"page_not_found",
				"maintenance_mode",
				"backup",
				"restore",
				"upgrade",
				"install",
				"uninstall",
				"cron_job",
				"script",
				"widget",
				"template",
				"theme",
				"plugin",
				"language",
				"style",
				"script",
				"utility",
			},
			SessionPaths: []string{
				"home",
				"about",
				"contact",
				"products",
				"services",
				"blog",
				"news",
				"login",
				"register",
				"shop",
				"search",
				"faq",
				"support",
				"terms",
				"privacy",
				"careers",
				"gallery",
				"events",
				"download",
				"portfolio",
				"help",
				"resources",
				"checkout",
				"cart",
				"account",
				"pricing",
				"features",
				"documentation",
				"api",
				"tutorials",
				"testimonials",
				"partners",
				"team",
				"media",
				"forum",
				"feedback",
				"settings",
				"dashboard",
				"profile",
				"messages",
				"notifications",
				"deals",
				"offers",
				"projects",
				"surveys",
				"newsroom",
				"videos",
				"marketplace",
				"donations",
				"community",
				"newsletter",
				"reviews",
				"sign-up",
				"terms-of-service",
				"privacy-policy",
				"returns",
				"subscribe",
				"jobs",
				"training",
				"courses",
				"tickets",
				"orders",
				"shipping",
				"tracking",
				"affiliates",
				"sign-in",
				"sign-out",
				"unsubscribe",
				"learn",
				"solutions",
				"library",
				"stats",
				"contests",
				"promotions",
				"book-now",
				"specials",
			},

			CloseFileExt: ".png",
			CloseFiles: []string{
				"image",
				"logo",
				"icon",
				"background",
				"banner",
				"button",
				"avatar",
				"photo",
				"picture",
				"header",
				"footer",
				"thumbnail",
				"screenshot",
				"cover",
				"badge",
				"illustration",
				"graphic",
				"map",
				"diagram",
				"chart",
				"emoji",
				"flag",
				"arrow",
				"social",
				"media",
				"document",
				"product",
				"menu",
				"navigation",
				"search",
				"result",
				"loading",
				"progress",
				"error",
				"success",
				"warning",
				"info",
				"question",
				"exclamation",
				"play",
				"pause",
				"stop",
				"next",
				"previous",
				"rewind",
				"forward",
				"volume",
				"mute",
				"speaker",
				"microphone",
				"camera",
				"video",
				"audio",
				"file",
				"folder",
				"download",
				"upload",
				"share",
				"like",
				"heart",
				"star",
				"comment",
				"chat",
				"speech bubble",
				"message",
				"envelope",
				"mail",
				"clock",
				"calendar",
				"location",
				"pin",
				"home",
				"settings",
				"gear",
				"tools",
				"user",
				"profile",
				"login",
				"logout",
				"register",
				"lock",
				"unlock",
				"shield",
				"security",
				"privacy",
				"checkmark",
				"cross",
				"delete",
				"trash",
				"restore",
				"recycle",
				"favorite",
				"bookmark",
				"star",
				"eye",
				"magnifier",
				"question-mark",
				"information",
				"exclamation-mark",
				"help",
			},
			ClosePaths: []string{
				"images",
				"photos",
				"pictures",
				"icons",
				"graphics",
				"assets",
				"media",
				"gallery",
				"uploads",
				"resources",
				"media_files",
				"media_assets",
				"media_library",
				"img",
				"logos",
				"banners",
				"thumbnails",
				"avatars",
				"screenshots",
				"headers",
				"footers",
				"backgrounds",
				"buttons",
				"illustrations",
				"icons_folder",
				"images_folder",
				"photos_folder",
				"pictures_folder",
				"graphics_folder",
				"assets_folder",
				"media_folder",
				"gallery_folder",
				"uploads_folder",
				"resources_folder",
				"media_files_folder",
				"media_assets_folder",
				"media_library_folder",
				"img_folder",
				"logos_folder",
				"banners_folder",
				"thumbnails_folder",
				"avatars_folder",
				"screenshots_folder",
				"headers_folder",
				"footers_folder",
				"backgrounds_folder",
				"buttons_folder",
				"illustrations_folder",
				"image_files",
				"photo_files",
				"picture_files",
				"icon_files",
				"graphic_files",
				"asset_files",
				"media_files_files",
				"gallery_files",
				"upload_files",
				"resource_files",
				"media_files_files_folder",
				"media_assets_files",
				"media_library_files",
				"img_files",
				"logo_files",
				"banner_files",
				"thumbnail_files",
				"avatar_files",
				"screenshot_files",
				"header_files",
				"footer_files",
				"background_files",
				"button_files",
				"illustration_files",
				"icons_dir",
				"images_dir",
				"photos_dir",
				"pictures_dir",
				"graphics_dir",
				"assets_dir",
				"media_dir",
				"gallery_dir",
				"uploads_dir",
				"resources_dir",
				"media_files_dir",
				"media_assets_dir",
				"media_library_dir",
				"img_dir",
				"logos_dir",
				"banners_dir",
				"thumbnails_dir",
				"avatars_dir",
				"screenshots_dir",
				"headers_dir",
				"footers_dir",
				"backgrounds_dir",
				"buttons_dir",
				"illustrations_dir",
				"png",
				"png_folder",
				"png_files",
				"pngs",
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
	data, err := os.ReadFile(configPath)
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
	err = checkHTTPC2Config(config)
	if err != nil {
		httpC2ConfigLog.Errorf("Invalid http c2 config: %s", err)
		return &defaultHTTPC2Config
	}
	return config
}

// CheckHTTPC2ConfigErrors - Get the current HTTP C2 config
func CheckHTTPC2ConfigErrors() error {
	configPath := GetHTTPC2ConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		err = generateDefaultConfig(configPath)
		if err != nil {
			httpC2ConfigLog.Errorf("Failed to generate http c2 config %s", err)
			return err
		}
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		httpC2ConfigLog.Errorf("Failed to read http c2 config %s", err)
		return err
	}
	config := &HTTPC2Config{}
	err = json.Unmarshal(data, config)
	if err != nil {
		httpC2ConfigLog.Errorf("Failed to parse http c2 config %s", err)
		return err
	}
	err = checkHTTPC2Config(config)
	if err != nil {
		httpC2ConfigLog.Errorf("Invalid http c2 config: %s", err)
		return err
	}
	return nil
}

func generateDefaultConfig(saveTo string) error {
	data, err := json.MarshalIndent(defaultHTTPC2Config, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(saveTo, data, 0600)
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
	ErrQueryParamNameLen          = errors.New("implant config url query parameter names must be 3 or more characters")

	fileNameExp = regexp.MustCompile(`[^a-zA-Z0-9\\._-]+`)
)

// checkHTTPC2Config - Validate the HTTP C2 config, coerces common mistakes
func checkHTTPC2Config(config *HTTPC2Config) error {
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

	// Query Parameter Names
	for _, queryParam := range config.URLParameters {
		if len(queryParam.Name) < 3 {
			return ErrQueryParamNameLen
		}
	}

	return nil
}
