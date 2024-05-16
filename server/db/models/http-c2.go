package models

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
	"fmt"
	"time"

	insecureRand "math/rand"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/gofrs/uuid"

	"gorm.io/gorm"
)

const (
	DefaultChromeBaseVer = 106
	DefaultMacOSVer      = "10_15_7"
)

// HttpC2Config -
type HttpC2Config struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	Name string `gorm:"unique;"`

	ServerConfig  HttpC2ServerConfig
	ImplantConfig HttpC2ImplantConfig
}

func (h *HttpC2Config) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	h.CreatedAt = time.Now()
	return nil
}

func (h *HttpC2Config) ToProtobuf() *clientpb.HTTPC2Config {
	return &clientpb.HTTPC2Config{
		ID:      h.ID.String(),
		Created: h.CreatedAt.Unix(),
		Name:    h.Name,

		ServerConfig:  h.ServerConfig.ToProtobuf(),
		ImplantConfig: h.ImplantConfig.ToProtobuf(),
	}
}

func (h *HttpC2Config) GenerateImplantHTTPC2Config() *clientpb.HTTPC2ImplantConfig {
	params := make([]*clientpb.HTTPC2URLParameter, len(h.ImplantConfig.ExtraURLParameters))
	for i, param := range h.ImplantConfig.ExtraURLParameters {
		params[i] = param.ToProtobuf()
	}
	return &clientpb.HTTPC2ImplantConfig{
		UserAgent:          h.ImplantConfig.UserAgent,
		ChromeBaseVersion:  h.ImplantConfig.ChromeBaseVersion,
		MacOSVersion:       h.ImplantConfig.MacOSVersion,
		NonceQueryArgChars: h.ImplantConfig.NonceQueryArgChars,
		ExtraURLParameters: params,
	}
}

// HttpC2ServerConfig - HTTP C2 Server Configuration
type HttpC2ServerConfig struct {
	ID             uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	HttpC2ConfigID uuid.UUID `gorm:"type:uuid;"`

	RandomVersionHeaders bool
	Headers              []HttpC2Header
	Cookies              []HttpC2Cookie
}

func (h *HttpC2ServerConfig) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID, err = uuid.NewV4()
	return err
}

func (h *HttpC2ServerConfig) ToProtobuf() *clientpb.HTTPC2ServerConfig {
	headers := make([]*clientpb.HTTPC2Header, len(h.Headers))
	for i, header := range h.Headers {
		headers[i] = header.ToProtobuf()
	}
	cookies := make([]*clientpb.HTTPC2Cookie, len(h.Cookies))
	for i, cookie := range h.Cookies {
		cookies[i] = cookie.ToProtobuf()
	}
	return &clientpb.HTTPC2ServerConfig{
		ID:                   h.ID.String(),
		RandomVersionHeaders: h.RandomVersionHeaders,
		Headers:              headers,
		Cookies:              cookies,
	}
}

// HttpC2ImplantConfig - HTTP C2 Implant Configuration
type HttpC2ImplantConfig struct {
	ID             uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	HttpC2ConfigID uuid.UUID `gorm:"type:uuid;"`

	UserAgent          string
	ChromeBaseVersion  int32
	MacOSVersion       string
	NonceQueryArgChars string
	ExtraURLParameters []HttpC2URLParameter
	Headers            []HttpC2Header

	MaxFiles int32
	MinFiles int32
	MaxPaths int32
	MinPaths int32

	StagerFileExtension       string
	PollFileExtension         string
	StartSessionFileExtension string
	SessionFileExtension      string
	CloseFileExtension        string

	PathSegments []HttpC2PathSegment
}

func (h *HttpC2ImplantConfig) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID, err = uuid.NewV4()
	return err
}

func (h *HttpC2ImplantConfig) ToProtobuf() *clientpb.HTTPC2ImplantConfig {
	params := make([]*clientpb.HTTPC2URLParameter, len(h.ExtraURLParameters))
	for i, param := range h.ExtraURLParameters {
		params[i] = param.ToProtobuf()
	}
	headers := make([]*clientpb.HTTPC2Header, len(h.Headers))
	for i, header := range h.Headers {
		headers[i] = header.ToProtobuf()
	}
	pathSegments := make([]*clientpb.HTTPC2PathSegment, len(h.PathSegments))
	for i, segment := range h.PathSegments {
		pathSegments[i] = segment.ToProtobuf()
	}
	return &clientpb.HTTPC2ImplantConfig{
		ID:                        h.ID.String(),
		UserAgent:                 h.UserAgent,
		ChromeBaseVersion:         h.ChromeBaseVersion,
		MacOSVersion:              h.MacOSVersion,
		NonceQueryArgChars:        h.NonceQueryArgChars,
		ExtraURLParameters:        params,
		Headers:                   headers,
		MaxFiles:                  h.MaxFiles,
		MinFiles:                  h.MinFiles,
		MaxPaths:                  h.MaxPaths,
		MinPaths:                  h.MinPaths,
		StagerFileExtension:       h.StagerFileExtension,
		PollFileExtension:         h.PollFileExtension,
		StartSessionFileExtension: h.StartSessionFileExtension,
		SessionFileExtension:      h.SessionFileExtension,
		CloseFileExtension:        h.CloseFileExtension,
		PathSegments:              pathSegments,
	}
}

//
// >>> Sub-Models <<<
//

// HttpC2Cookie - HTTP C2 Cookie (server only)
type HttpC2Cookie struct {
	ID                   uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	HttpC2ServerConfigID uuid.UUID `gorm:"type:uuid;"`

	Name string
}

func (h *HttpC2Cookie) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID, err = uuid.NewV4()
	return err
}

func (h *HttpC2Cookie) ToProtobuf() *clientpb.HTTPC2Cookie {
	return &clientpb.HTTPC2Cookie{
		ID:   h.ID.String(),
		Name: h.Name,
	}
}

// HttpC2Header - HTTP C2 Header (server and implant)
type HttpC2Header struct {
	ID                    uuid.UUID  `gorm:"primaryKey;->;<-:create;type:uuid;"`
	HttpC2ServerConfigID  *uuid.UUID `gorm:"type:uuid;"`
	HttpC2ImplantConfigID *uuid.UUID `gorm:"type:uuid;"`

	Method      string
	Name        string
	Value       string
	Probability int32
}

func (h *HttpC2Header) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID, err = uuid.NewV4()
	return err
}

func (h *HttpC2Header) ToProtobuf() *clientpb.HTTPC2Header {
	return &clientpb.HTTPC2Header{
		ID:          h.ID.String(),
		Method:      h.Method,
		Name:        h.Name,
		Value:       h.Value,
		Probability: h.Probability,
	}
}

// HttpC2URLParameter - Extra URL parameters (implant only)
type HttpC2URLParameter struct {
	ID                    uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	HttpC2ImplantConfigID uuid.UUID `gorm:"type:uuid;"`

	Method      string // HTTP Method
	Name        string // Name of URL parameter, must be 3+ characters
	Value       string // Value of the URL parameter
	Probability int32  // 0 - 100
}

func (h *HttpC2URLParameter) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID, err = uuid.NewV4()
	return err
}

func (h *HttpC2URLParameter) ToProtobuf() *clientpb.HTTPC2URLParameter {
	return &clientpb.HTTPC2URLParameter{
		ID:          h.ID.String(),
		Method:      h.Method,
		Name:        h.Name,
		Value:       h.Value,
		Probability: h.Probability,
	}
}

// HttpC2PathSegment - Represents a list of file/path URL segments (implant only)
type HttpC2PathSegment struct {
	ID                    uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	HttpC2ImplantConfigID uuid.UUID `gorm:"type:uuid;"`

	IsFile      bool
	SegmentType int32 // Poll, Session, Close
	Value       string
}

func (h *HttpC2PathSegment) BeforeCreate(tx *gorm.DB) (err error) {
	h.ID, err = uuid.NewV4()
	return err
}

func (h *HttpC2PathSegment) ToProtobuf() *clientpb.HTTPC2PathSegment {
	return &clientpb.HTTPC2PathSegment{
		ID:          h.ID.String(),
		IsFile:      h.IsFile,
		SegmentType: clientpb.HTTPC2SegmentType(h.SegmentType),
		Value:       h.Value,
	}
}

// HTTPC2ConfigFromProtobuf - Create a native config struct from Protobuf
func HTTPC2ConfigFromProtobuf(pbHttpC2Config *clientpb.HTTPC2Config) *HttpC2Config {
	cfg := &HttpC2Config{}

	// Server Config
	serverHeaders := []HttpC2Header{}
	for _, header := range pbHttpC2Config.ServerConfig.Headers {
		serverHeaders = append(serverHeaders, HttpC2Header{
			Method:      header.Method,
			Name:        header.Name,
			Value:       header.Value,
			Probability: header.Probability,
		})
	}

	cookies := []HttpC2Cookie{}
	for _, cookie := range pbHttpC2Config.ServerConfig.Cookies {
		cookies = append(cookies, HttpC2Cookie{
			Name: cookie.Name,
		})
	}

	cfg.ServerConfig = HttpC2ServerConfig{
		RandomVersionHeaders: pbHttpC2Config.ServerConfig.RandomVersionHeaders,
		Headers:              serverHeaders,
		Cookies:              cookies,
	}

	// Implant Config
	params := []HttpC2URLParameter{}
	for _, param := range pbHttpC2Config.ImplantConfig.ExtraURLParameters {
		params = append(params, HttpC2URLParameter{
			Method:      param.Method,
			Name:        param.Name,
			Value:       param.Value,
			Probability: param.Probability,
		})
	}

	implantHeaders := []HttpC2Header{}
	for _, header := range pbHttpC2Config.ImplantConfig.Headers {
		implantHeaders = append(implantHeaders, HttpC2Header{
			Method:      header.Method,
			Name:        header.Name,
			Value:       header.Value,
			Probability: header.Probability,
		})
	}

	pathSegments := []HttpC2PathSegment{}
	for _, pathSegment := range pbHttpC2Config.ImplantConfig.PathSegments {
		pathSegments = append(pathSegments, HttpC2PathSegment{
			IsFile:      pathSegment.IsFile,
			SegmentType: int32(pathSegment.SegmentType),
			Value:       pathSegment.Value,
		})
	}

	cfg.ImplantConfig = HttpC2ImplantConfig{
		UserAgent:                 pbHttpC2Config.ImplantConfig.UserAgent,
		ChromeBaseVersion:         pbHttpC2Config.ImplantConfig.ChromeBaseVersion,
		MacOSVersion:              pbHttpC2Config.ImplantConfig.MacOSVersion,
		NonceQueryArgChars:        pbHttpC2Config.ImplantConfig.NonceQueryArgChars,
		ExtraURLParameters:        params,
		Headers:                   implantHeaders,
		MaxFiles:                  pbHttpC2Config.ImplantConfig.MaxFiles,
		MinFiles:                  pbHttpC2Config.ImplantConfig.MinFiles,
		MaxPaths:                  pbHttpC2Config.ImplantConfig.MaxPaths,
		MinPaths:                  pbHttpC2Config.ImplantConfig.MinPaths,
		StagerFileExtension:       pbHttpC2Config.ImplantConfig.StagerFileExtension,
		PollFileExtension:         pbHttpC2Config.ImplantConfig.PollFileExtension,
		StartSessionFileExtension: pbHttpC2Config.ImplantConfig.StartSessionFileExtension,
		SessionFileExtension:      pbHttpC2Config.ImplantConfig.SessionFileExtension,
		CloseFileExtension:        pbHttpC2Config.ImplantConfig.CloseFileExtension,
		PathSegments:              pathSegments,
	}

	// C2 Config
	cfg.Name = pbHttpC2Config.Name

	return cfg
}

// RandomImplantConfig - Randomly generate a new implant config from the parent config,
// this is the primary configuration used by the implant generation.
func RandomizeImplantConfig(h *clientpb.HTTPC2ImplantConfig, goos string, goarch string) *clientpb.HTTPC2ImplantConfig {
	return &clientpb.HTTPC2ImplantConfig{

		NonceQueryArgChars: h.NonceQueryArgChars,
		ExtraURLParameters: h.ExtraURLParameters,
		Headers:            h.Headers,

		PollFileExtension:         h.PollFileExtension,
		StartSessionFileExtension: h.StartSessionFileExtension,
		SessionFileExtension:      h.SessionFileExtension,
		CloseFileExtension:        h.CloseFileExtension,
		PathSegments:              RandomPathSegments(h),
		UserAgent:                 GenerateUserAgent(goos, goarch, h.UserAgent, h.ChromeBaseVersion, h.MacOSVersion),
		ChromeBaseVersion:         h.ChromeBaseVersion,
		MacOSVersion:              h.MacOSVersion,
		MinFiles:                  h.MinFiles,
		MaxFiles:                  h.MaxFiles,
		MinPaths:                  h.MinPaths,
		MaxPaths:                  h.MaxPaths,
	}
}

// GenerateUserAgent - Generate a user-agent depending on OS/Arch
func GenerateUserAgent(goos string, goarch string, userAgent string, baseVer int32, macOsVer string) string {
	return generateChromeUserAgent(goos, goarch, userAgent, baseVer, macOsVer)
}

func generateChromeUserAgent(goos string, goarch string, userAgent string, baseVer int32, macOsVer string) string {
	if userAgent == "" {
		switch goos {
		case "windows":
			switch goarch {
			case "amd64":
				return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", ChromeVer(baseVer))
			}

		case "linux":
			switch goarch {
			case "amd64":
				return fmt.Sprintf("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", ChromeVer(baseVer))
			}

		case "darwin":
			switch goarch {
			case "arm64":
				fallthrough // https://source.chromium.org/chromium/chromium/src/+/master:third_party/blink/renderer/core/frame/navigator_id.cc;l=76
			case "amd64":
				return fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X %s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", MacOSVer(macOsVer), ChromeVer(baseVer))
			}

		}
	} else {
		return userAgent
	}

	// Default is a generic Windows/Chrome
	return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", ChromeVer(baseVer))
}

// ChromeVer - Generate a random Chrome user-agent
func ChromeVer(baseVer int32) string {
	chromeVer := baseVer
	if chromeVer == 0 {
		chromeVer = DefaultChromeBaseVer
	}
	return fmt.Sprintf("%d.0.%d.%d", baseVer+int32(insecureRand.Intn(3)), 1000+int32(insecureRand.Intn(8999)), int32(insecureRand.Intn(999)))
}

func MacOSVer(MacOSVersion string) string {
	macosVer := MacOSVersion
	if macosVer == "" {
		macosVer = DefaultMacOSVer
	}
	return macosVer
}

func RandomPathSegments(h *clientpb.HTTPC2ImplantConfig) []*clientpb.HTTPC2PathSegment {

	var (
		sessionPaths []*clientpb.HTTPC2PathSegment
		closePaths   []*clientpb.HTTPC2PathSegment
		pollPaths    []*clientpb.HTTPC2PathSegment
		sessionFiles []*clientpb.HTTPC2PathSegment
		closeFiles   []*clientpb.HTTPC2PathSegment
		pollFiles    []*clientpb.HTTPC2PathSegment
	)
	for _, pathSegment := range h.PathSegments {
		switch pathSegment.SegmentType {
		case 0:
			if pathSegment.IsFile {
				pollFiles = append(pollFiles, pathSegment)
			} else {
				pollPaths = append(pollPaths, pathSegment)
			}
		case 1:
			if pathSegment.IsFile {
				sessionFiles = append(sessionFiles, pathSegment)
			} else {
				sessionPaths = append(sessionPaths, pathSegment)
			}
		case 2:
			if pathSegment.IsFile {
				closeFiles = append(closeFiles, pathSegment)
			} else {
				closePaths = append(closePaths, pathSegment)
			}
		default:
			continue
		}
	}

	sessionPaths = RandomPaths(sessionPaths, h.MinPaths, h.MaxPaths)
	pollPaths = RandomPaths(pollPaths, h.MinPaths, h.MaxPaths)
	closePaths = RandomPaths(closePaths, h.MinPaths, h.MaxPaths)

	sessionFiles = RandomFiles(sessionFiles, h.MinFiles, h.MaxFiles)
	closeFiles = RandomFiles(closeFiles, h.MinFiles, h.MaxFiles)
	pollFiles = RandomFiles(pollFiles, h.MinFiles, h.MaxFiles)

	var res []*clientpb.HTTPC2PathSegment
	res = append(res, sessionPaths...)
	res = append(res, closePaths...)
	res = append(res, pollPaths...)
	res = append(res, sessionFiles...)
	res = append(res, closeFiles...)
	res = append(res, pollFiles...)
	return res
}

func RandomFiles(httpC2PathSegments []*clientpb.HTTPC2PathSegment, minFiles int32, maxFiles int32) []*clientpb.HTTPC2PathSegment {
	if minFiles < 1 {
		minFiles = 1
	}
	return randomSample(httpC2PathSegments, minFiles, maxFiles)
}

func RandomPaths(httpC2PathSegments []*clientpb.HTTPC2PathSegment, minPaths int32, maxPaths int32) []*clientpb.HTTPC2PathSegment {
	return randomSample(httpC2PathSegments, minPaths, maxPaths)
}

func randomSample(values []*clientpb.HTTPC2PathSegment, min int32, max int32) []*clientpb.HTTPC2PathSegment {
	count := int32(insecureRand.Intn(len(values)))
	if count < min {
		count = min
	}
	if max < count {
		count = max
	}
	var sample []*clientpb.HTTPC2PathSegment
	for i := 0; int32(len(sample)) < count; i++ {
		index := (count + int32(i)) % int32(len(values))
		sample = append(sample, values[index])
	}
	return sample
}
