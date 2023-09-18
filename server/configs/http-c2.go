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
	"fmt"
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
	ErrDuplicateC2ProfileName     = errors.New("C2 Profile name is already in use")

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
		MaxFiles:                  8,
		MinFiles:                  2,
		MaxPaths:                  8,
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

	Cookies = []string{
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
	}
	CloseFiles = []string{
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
