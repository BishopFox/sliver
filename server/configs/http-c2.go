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
	"errors"
	"regexp"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/log"
)

const (
	DefaultChromeBaseVer = 106
	DefaultMacOSVer      = "10_15_7"
)

// CheckHTTPC2ConfigErrors - Get the current HTTP C2 config
func CheckHTTPC2ConfigErrors(config *clientpb.HTTPC2Config) error {
	err := checkHTTPC2Config(config)
	if err != nil {
		httpC2ConfigLog.Errorf("Invalid http c2 config: %s", err)
		return err
	}
	return nil
}

var (
	ErrMissingCookies             = errors.New("server config must specify at least one cookie")
	ErrMissingExtensions          = errors.New("implant config must specify at least one file extension")
	ErrMissingFiles               = errors.New("implant config must specify at least one files value")
	ErrMissingPaths               = errors.New("implant config must specify at least one paths value")
	ErrDuplicateC2ProfileName     = errors.New("C2 Profile name is already in use")
	ErrMissingC2ProfileName       = errors.New("C2 Profile name is required")
	ErrC2ProfileNotFound          = errors.New("C2 Profile does not exist")
	ErrUserAgentIllegalCharacters = errors.New("user agent cannot contain the ` character")

	fileNameExp = regexp.MustCompile(`[^a-zA-Z0-9\\._-]+`)
)

// checkHTTPC2Config - Validate the HTTP C2 config, coerces common mistakes
func checkHTTPC2Config(config *clientpb.HTTPC2Config) error {
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

func checkServerConfig(config *clientpb.HTTPC2ServerConfig) error {
	if len(config.Cookies) < 1 {
		return ErrMissingCookies
	}
	return nil
}

func checkImplantConfig(config *clientpb.HTTPC2ImplantConfig) error {

	// min and max used during http c2 config generation
	if config.MinFileGen < 1 {
		config.MinFileGen = 1
	}
	if config.MaxFileGen < config.MinFileGen {
		config.MaxFileGen = config.MinFileGen
	}
	if config.MinPathGen < 0 {
		config.MinPathGen = 0
	}
	if config.MaxPathGen < config.MinPathGen {
		config.MaxPathGen = config.MinPathGen
	}

	// Path length used during implant operation
	if config.MinPathLength < 1 {
		config.MinPathLength = 1
	}
	if config.MaxPathLength < config.MinPathLength {
		config.MaxPathLength = config.MinPathLength
	}

	// File Extensions
	if len(config.Extensions) < 1 {
		return ErrMissingExtensions
	}

	/*
		User agent

		Do not allow backticks in user agents because that breaks compilation of the
		implant.
	*/
	if strings.Contains(config.UserAgent, "`") {
		// Blank out the user agent so that a default one will be filled in later
		config.UserAgent = ""
		return ErrUserAgentIllegalCharacters
	}

	return nil
}

func GenerateDefaultHTTPC2Config() *clientpb.HTTPC2Config {

	// Implant Config
	httpC2UrlParameters := []*clientpb.HTTPC2URLParameter{}
	httpC2Headers := []*clientpb.HTTPC2Header{}
	pathSegments := GenerateHTTPC2DefaultPathSegment()

	implantConfig := clientpb.HTTPC2ImplantConfig{
		UserAgent:          "",
		ChromeBaseVersion:  DefaultChromeBaseVer,
		MacOSVersion:       DefaultMacOSVer,
		NonceQueryArgChars: "abcdefghijklmnopqrstuvwxyz",
		NonceQueryLength:   1,
		NonceMode:          "UrlParam", // Url or UrlParam
		ExtraURLParameters: httpC2UrlParameters,
		Headers:            httpC2Headers,
		MaxFileGen:         4,
		MinFileGen:         2,
		MaxPathGen:         4,
		MinPathGen:         2,
		MaxPathLength:      4,
		MinPathLength:      2,
		Extensions: []string{
			"js", "", "php",
		},
		PathSegments: pathSegments,
	}

	// Server Config
	serverHeaders := []*clientpb.HTTPC2Header{
		{
			Method:      "GET",
			Name:        "Cache-Control",
			Value:       "no-store, no-cache, must-revalidate",
			Probability: 100,
		},
	}
	cookies := GenerateDefaultHTTPC2Cookies()
	serverConfig := clientpb.HTTPC2ServerConfig{
		RandomVersionHeaders: false,
		Headers:              serverHeaders,
		Cookies:              cookies,
	}

	// HTTPC2Config

	defaultConfig := clientpb.HTTPC2Config{
		ServerConfig:  &serverConfig,
		ImplantConfig: &implantConfig,
		Name:          "default",
	}

	return &defaultConfig
}

func GenerateDefaultHTTPC2Cookies() []*clientpb.HTTPC2Cookie {
	cookies := []*clientpb.HTTPC2Cookie{}
	for _, cookie := range Cookies {
		cookies = append(cookies, &clientpb.HTTPC2Cookie{
			Name: cookie,
		})
	}
	return cookies
}

func GenerateHTTPC2DefaultPathSegment() []*clientpb.HTTPC2PathSegment {
	pathSegments := []*clientpb.HTTPC2PathSegment{}

	/*
		IsFile      bool
		Value       string
	*/

	// files
	for _, file := range Files {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile: true,
			Value:  file,
		})
	}

	for _, path := range Paths {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile: false,
			Value:  path,
		})
	}

	return pathSegments
}

var (
	httpC2ConfigLog = log.NamedLogger("config", "http-c2")

	// Pro-tip: You can use ChatGPT to generate this stuff for you!
	Cookies = []string{
		"JSESSIONID",
		"rememberMe",
		"authToken",
		"userId",
		"userName",
		"language",
		"theme",
		"locale",
		"currency",
		"lastVisited",
		"loggedIn",
		"userRole",
		"cartId",
		"accessToken",
		"refreshToken",
		"consent",
		"notificationPreference",
		"userSettings",
		"sessionTimeout",
		"error",
		"errorMessage",
		"successMessage",
		"infoMessage",
		"warningMessage",
		"errorMessageKey",
		"successMessageKey",
		"infoMessageKey",
		"warningMessageKey",
		"sessionID",
		"userID",
		"username",
		"authToken",
		"rememberMe",
		"language",
		"theme",
		"locale",
		"currency",
		"lastVisit",
		"loggedIn",
		"userRole",
		"cartID",
		"accessToken",
		"refreshToken",
		"consent",
		"notificationPref",
		"userSettings",
		"sessionTimeout",
		"visitedPages",
		"favoriteItems",
		"searchHistory",
		"basketID",
		"promoCode",
		"campaignID",
		"referrer",
		"source",
		"utmCampaign",
		"utmSource",
		"utmMedium",
		"utmContent",
		"utmTerm",
		"deviceType",
		"OSVersion",
		"browser",
		"screenResolution",
		"timezone",
		"firstVisit",
		"feedbackGiven",
		"surveyID",
		"errorAlerts",
		"successAlerts",
		"infoAlerts",
		"warningAlerts",
		"darkMode",
		"emailSubscription",
		"privacyConsent",
		"PHPSESSID",
		"SID",
		"SSID",
		"APISID",
		"csrf-state",
		"AWSALBCORS",
	}
	Files = []string{
		"bootstrap",
		"bootstrap.min",
		"jquery.min",
		"jquery",
		"route",
		"app",
		"app.min",
		"array",
		"backbone",
		"script",
		"email",
		"main",
		"index",
		"utils",
		"jquery.min",
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
	}
	Paths = []string{
		"js",
		"umd",
		"assets",
		"bundle",
		"bundles",
		"scripts",
		"script",
		"javascripts",
		"javascript",
		"jscript",
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
		"jsfiles",
		"javascript",
		"jslib",
		"jslibraries",
		"jsdir",
		"jsfolder",
		"jsfilesdir",
		"jsfilesfolder",
		"scriptsdir",
		"assetsjs",
		"srcjs",
		"srcscripts",
		"libjs",
		"libscripts",
		"publicjs",
		"publicscripts",
		"staticjs",
		"staticscripts",
		"appjs",
		"appscripts",
		"distjs",
		"distscripts",
		"frontendjs",
		"frontendscripts",
		"clientjs",
		"clientscripts",
		"serverjs",
		"resourcesjs",
		"jsfilesjs",
		"jsfilesscripts",
		"javascriptjs",
		"jslibscripts",
		"assetsjs",
		"assetsjsscripts",
		"assetsjs",
		"assetsscripts",
		"srcjs",
		"srcjsscripts",
		"srcscripts",
		"srcscripts",
		"libjs",
		"libjsscripts",
	}
)
