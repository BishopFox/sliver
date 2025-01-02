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
	ErrMissingStagerFileExt       = errors.New("implant config must specify a stager_file_ext")
	ErrTooFewStagerFiles          = errors.New("implant config must specify at least one stager_files value")
	ErrMissingPollFileExt         = errors.New("implant config must specify a poll_file_ext")
	ErrTooFewPollFiles            = errors.New("implant config must specify at least one poll_files value")
	ErrMissingKeyExchangeFileExt  = errors.New("implant config must specify a key_exchange_file_ext")
	ErrTooFewKeyExchangeFiles     = errors.New("implant config must specify at least one key_exchange_files value")
	ErrMissingCloseFileExt        = errors.New("implant config must specify a close_file_ext")
	ErrTooFewCloseFiles           = errors.New("implant config must specify at least one close_files value")
	ErrMissingStartSessionFileExt = errors.New("implant config must specify a start_session_file_ext")
	ErrMissingSessionFileExt      = errors.New("implant config must specify a session_file_ext")
	ErrTooFewSessionFiles         = errors.New("implant config must specify at least one session_files value")
	ErrNonUniqueFileExt           = errors.New("implant config must specify unique file extensions")
	ErrQueryParamNameLen          = errors.New("implant config url query parameter names must be 3 or more characters")
	ErrDuplicateStageExt          = errors.New("stager extension is already used in another C2 profile")
	ErrDuplicateStartSessionExt   = errors.New("start session extension is already used in another C2 profile")
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
	config.StagerFileExtension = coerceFileExt(config.StagerFileExtension)
	if config.StagerFileExtension == "" {
		return ErrMissingStagerFileExt
	}

	// File Extensions
	config.PollFileExtension = coerceFileExt(config.PollFileExtension)
	if config.PollFileExtension == "" {
		return ErrMissingPollFileExt
	}
	config.StartSessionFileExtension = coerceFileExt(config.StartSessionFileExtension)
	if config.StartSessionFileExtension == "" {
		return ErrMissingStartSessionFileExt
	}
	config.SessionFileExtension = coerceFileExt(config.SessionFileExtension)
	if config.SessionFileExtension == "" {
		return ErrMissingSessionFileExt
	}
	config.CloseFileExtension = coerceFileExt(config.CloseFileExtension)
	if config.CloseFileExtension == "" {
		return ErrMissingCloseFileExt
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
		UserAgent:                 "",
		ChromeBaseVersion:         DefaultChromeBaseVer,
		MacOSVersion:              DefaultMacOSVer,
		NonceQueryArgChars:        "abcdefghijklmnopqrstuvwxyz",
		ExtraURLParameters:        httpC2UrlParameters,
		Headers:                   httpC2Headers,
		MaxFiles:                  4,
		MinFiles:                  2,
		MaxPaths:                  4,
		MinPaths:                  2,
		StagerFileExtension:       "woff",
		PollFileExtension:         "js",
		StartSessionFileExtension: "html",
		SessionFileExtension:      "php",
		CloseFileExtension:        "png",
		PathSegments:              pathSegments,
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
		SegmentType int32 // Poll, Session, Close, stager
		Value       string
	*/

	// files
	for _, poll := range PollFiles {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile:      true,
			SegmentType: 0,
			Value:       poll,
		})
	}

	for _, session := range SessionFiles {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile:      true,
			SegmentType: 1,
			Value:       session,
		})
	}

	for _, close := range CloseFiles {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile:      true,
			SegmentType: 2,
			Value:       close,
		})
	}

	for _, stager := range StagerFiles {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile:      true,
			SegmentType: 3,
			Value:       stager,
		})
	}

	// paths
	for _, poll := range PollPaths {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile:      false,
			SegmentType: 0,
			Value:       poll,
		})
	}

	for _, session := range SessionPaths {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile:      false,
			SegmentType: 1,
			Value:       session,
		})
	}

	for _, close := range ClosePaths {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile:      false,
			SegmentType: 2,
			Value:       close,
		})
	}

	for _, stager := range StagerPaths {
		pathSegments = append(pathSegments, &clientpb.HTTPC2PathSegment{
			IsFile:      false,
			SegmentType: 3,
			Value:       stager,
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
	PollFiles = []string{
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
	PollPaths = []string{
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
	SessionFiles = []string{
		"login",
		"signin",
		"api",
		"samples",
		"rpc",
		"index",
		"admin",
		"register",
		"sign-up",
		"index",
		"config",
		"functions",
		"database",
		"header",
		"footer",
		"login",
		"register",
		"logout",
		"profile",
		"dashboard",
		"home",
		"about",
		"contact",
		"services",
		"utils",
		"common",
		"init",
		"auth",
		"session",
		"error",
		"handler",
		"api",
		"ajax",
		"form",
		"validation",
		"upload",
		"download",
		"report",
		"admin",
		"user",
		"account",
		"settings",
		"utility",
		"script",
		"mailer",
		"cron",
		"cache",
		"template",
		"page",
		"model",
		"view",
		"controller",
		"middleware",
		"router",
		"route",
		"helper",
		"library",
		"plugin",
		"widget",
		"widgetized",
		"search",
		"filter",
		"sort",
		"pagination",
		"backup",
		"restore",
		"upgrade",
		"install",
		"uninstall",
		"upgrade",
		"maintenance",
		"sitemap",
		"rss",
		"atom",
		"xml",
		"json",
		"rss",
		"log",
		"debug",
		"test",
		"mock",
		"stub",
		"mockup",
		"mockdata",
		"temp",
		"tmp",
		"backup",
		"old",
		"new",
		"demo",
		"example",
		"sample",
		"prototype",
		"backup",
		"import",
		"export",
		"sync",
		"async",
		"validate",
		"authenticate",
	}
	SessionPaths = []string{
		"php",
		"api",
		"upload",
		"actions",
		"rest",
		"v1",
		"auth",
		"authenticate",
		"oauth",
		"oauth2",
		"oauth2callback",
		"database",
		"db",
		"namespaces",
		"home",
		"about",
		"contact",
		"services",
		"products",
		"blog",
		"news",
		"events",
		"gallery",
		"portfolio",
		"shop",
		"store",
		"login",
		"register",
		"profile",
		"dashboard",
		"account",
		"settings",
		"faq",
		"help",
		"support",
		"download",
		"upload",
		"subscribe",
		"unsubscribe",
		"search",
		"sitemap",
		"privacy",
		"terms",
		"conditions",
		"policy",
		"cookie",
		"checkout",
		"cart",
		"order",
		"payment",
		"confirmation",
		"feedback",
		"survey",
		"testimonial",
		"newsletter",
		"membership",
		"forum",
		"signin",
		"signup",
		"forum",
		"contact-us",
		"pricing",
		"donate",
		"partners",
		"team",
		"career",
		"join-us",
		"downloads",
		"events",
		"explore",
		"insights",
		"newsroom",
		"press",
		"media",
		"blog",
		"articles",
		"research",
		"library",
		"resources",
		"inspiration",
		"academy",
		"labs",
		"developers",
		"api",
		"integration",
		"status",
		"changelog",
		"roadmap",
		"solutions",
		"clients",
		"testimonials",
		"success-stories",
		"clients",
		"partners",
		"404",
		"500",
		"error",
		"maintenance",
		"upgrade",
	}
	CloseFiles = []string{
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
		"question mark",
		"information",
		"exclamationmark",
		"help",
		"favicon",
		"sample",
		"example",
	}
	ClosePaths = []string{
		"static",
		"www",
		"assets",
		"images",
		"icons",
		"image",
		"icon",
		"png",
		"images",
		"photos",
		"pictures",
		"pics",
		"gallery",
		"media",
		"uploads",
		"assets",
		"img",
		"graphics",
		"visuals",
		"media-library",
		"artwork",
		"design",
		"diagrams",
		"icons",
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
		"diagrams",
		"photoshoots",
		"picoftheday",
		"imagebank",
		"album",
		"memories",
		"memes",
		"snapshots",
		"portraits",
		"posters",
		"renders",
		"cover-photos",
		"wallpapers",
		"photography",
		"visualizations",
		"moodboard",
		"infographics",
		"renders",
		"icons",
		"layers",
		"picgallery",
		"picdump",
		"imagelibrary",
	}
	StagerFiles = []string{
		"attribute_text_w01_regular", "ZillaSlab-Regular.subset.bbc33fb47cf6",
		"ZillaSlab-Bold.subset.e96c15f68c68", "Inter-Regular",
		"Inter-Medium",
	}
	StagerPaths = []string{
		"static", "assets", "fonts", "locales",
	}
)
