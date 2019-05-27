package generate

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	clientpb "github.com/bishopfox/sliver/protobuf/client"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/gobfuscate"
	"github.com/bishopfox/sliver/server/gogo"
	"github.com/bishopfox/sliver/server/log"

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

	// SliverCCEnvVar - Environment variable that can specify the mingw path
	SliverCCEnvVar = "SLIVER_CC"
)

// SliverConfig - Parameters when generating a implant
type SliverConfig struct {
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

	C2            []SliverC2 `json:"c2s"`
	MTLSc2Enabled bool       `json:"c2_mtls_enabled"`
	HTTPc2Enabled bool       `json:"c2_http_enabled"`
	DNSc2Enabled  bool       `json:"c2_dns_enabled"`
	CanaryDomains []string   `json:"canary_domains"`

	// Limits
	LimitDomainJoined bool   `json:"limit_domainjoined"`
	LimitHostname     string `json:"limit_hostname"`
	LimitUsername     string `json:"limit_username"`
	LimitDatetime     string `json:"limit_datetime"`

	// Output Format
	Format clientpb.SliverConfig_OutputFormat `json:"format"`

	// For 	IsSharedLib bool `json:"is_shared_lib"`
	IsSharedLib bool `json:"is_shared_lib"`

	FileName string
}

// ToProtobuf - Convert SliverConfig to protobuf equiv
func (c *SliverConfig) ToProtobuf() *clientpb.SliverConfig {
	config := &clientpb.SliverConfig{
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
		Format:      c.Format,

		FileName: c.FileName,
	}
	config.C2 = []*clientpb.SliverC2{}
	for _, c2 := range c.C2 {
		config.C2 = append(config.C2, c2.ToProtobuf())
	}
	return config
}

// SliverConfigFromProtobuf - Create a native config struct from Protobuf
func SliverConfigFromProtobuf(pbConfig *clientpb.SliverConfig) *SliverConfig {
	cfg := &SliverConfig{}

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

	cfg.C2 = copyC2List(pbConfig.C2)
	cfg.MTLSc2Enabled = isC2Enabled([]string{"mtls"}, cfg.C2)
	cfg.HTTPc2Enabled = isC2Enabled([]string{"http", "https"}, cfg.C2)
	cfg.DNSc2Enabled = isC2Enabled([]string{"dns"}, cfg.C2)

	cfg.FileName = pbConfig.FileName
	return cfg
}

func copyC2List(src []*clientpb.SliverC2) []SliverC2 {
	c2s := []SliverC2{}
	for _, srcC2 := range src {
		c2URL, err := url.Parse(srcC2.URL)
		if err != nil {
			buildLog.Warnf("Failed to parse c2 url %v", err)
			continue
		}
		c2s = append(c2s, SliverC2{
			Priority: srcC2.Priority,
			URL:      c2URL.String(),
			Options:  srcC2.Options,
		})
	}
	return c2s
}

func isC2Enabled(schemes []string, c2s []SliverC2) bool {
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

// SliverC2 - C2 struct
type SliverC2 struct {
	Priority uint32 `json:"priority"`
	URL      string `json:"url"`
	Options  string `json:"options"`
}

// ToProtobuf - Convert to protobuf version
func (s SliverC2) ToProtobuf() *clientpb.SliverC2 {
	return &clientpb.SliverC2{
		Priority: s.Priority,
		URL:      s.URL,
		Options:  s.Options,
	}
}

func (s SliverC2) String() string {
	return s.URL
}

// GetSliversDir - Get the binary directory
func GetSliversDir() string {
	appDir := assets.GetRootAppDir()
	sliversDir := path.Join(appDir, sliversDirName)
	if _, err := os.Stat(sliversDir); os.IsNotExist(err) {
		buildLog.Infof("Creating bin directory: %s", sliversDir)
		err = os.MkdirAll(sliversDir, os.ModePerm)
		if err != nil {
			buildLog.Fatal(err)
		}
	}
	return sliversDir
}

// SliverEgg - Generates a sliver egg (stager) binary
func SliverEgg(config SliverConfig) (string, error) {

	return "", nil
}

// -----------------------
// Sliver Generation Code
// -----------------------

// SliverSharedLibrary - Generates a sliver shared library (DLL/dylib/so) binary
func SliverSharedLibrary(config *SliverConfig) (string, error) {
	// Compile go code
	appDir := assets.GetRootAppDir()
	crossCompiler := getCCompiler(config.GOARCH)
	if crossCompiler == "" {
		return "", errors.New("No cross-compiler (mingw) found")
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
	ldflags := []string{"-s -w"}
	if !config.Debug && goConfig.GOOS == WINDOWS {
		ldflags[0] += " -H=windowsgui"
	}
	_, err = gogo.GoBuild(*goConfig, pkgPath, dest, "c-shared", tags, ldflags)
	config.FileName = path.Base(dest)
	saveFileErr := SliverFileSave(config.Name, dest)
	saveCfgErr := SliverConfigSave(config)
	if saveFileErr != nil || saveCfgErr != nil {
		buildLog.Errorf("Failed to save file to db %s %s", saveFileErr, saveCfgErr)
	}
	return dest, err
}

// SliverExecutable - Generates a sliver executable binary
func SliverExecutable(config *SliverConfig) (string, error) {

	// Compile go code
	appDir := assets.GetRootAppDir()
	goConfig := &gogo.GoConfig{
		CGO:    "0",
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
	ldflags := []string{"-s -w"}
	if !config.Debug && goConfig.GOOS == WINDOWS {
		ldflags[0] += " -H=windowsgui"
	}
	_, err = gogo.GoBuild(*goConfig, pkgPath, dest, "", tags, ldflags)
	config.FileName = path.Base(dest)
	saveFileErr := SliverFileSave(config.Name, dest)
	saveCfgErr := SliverConfigSave(config)
	if saveFileErr != nil || saveCfgErr != nil {
		buildLog.Errorf("Failed to save file to db %s %s", saveFileErr, saveCfgErr)
	}
	return dest, err
}

// This function is a little too long, we should probably refactor it as some point
func renderSliverGoCode(config *SliverConfig, goConfig *gogo.GoConfig) (string, error) {
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

	sliversDir := GetSliversDir() // ~/.sliver/slivers
	projectGoPathDir := path.Join(sliversDir, config.GOOS, config.GOARCH, config.Name)
	os.MkdirAll(projectGoPathDir, os.ModePerm)
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
	os.MkdirAll(binDir, os.ModePerm)

	// srcDir - ~/.sliver/slivers/<os>/<arch>/<name>/src
	srcDir := path.Join(projectGoPathDir, "src")
	assets.SetupGoPath(srcDir) // Extract GOPATH dependancy files

	sliverPkgDir := path.Join(srcDir, "github.com", "bishopfox", "sliver") // "main"
	os.MkdirAll(sliverPkgDir, os.ModePerm)

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
		if config.Debug {
			fileName = filepath.Base(boxName)
		} else {
			fileName = fmt.Sprintf("s%d%s", index, suffix)
		}
		if dirName != "." {
			// Add an extra "sliver" dir
			dirPath := path.Join(sliverPkgDir, "sliver", dirName)
			if _, err := os.Stat(dirPath); os.IsNotExist(err) {
				buildLog.Infof("[mkdir] %#v", dirPath)
				os.MkdirAll(dirPath, os.ModePerm)
			}
			sliverCodePath = path.Join(dirPath, fileName)
		} else {
			sliverCodePath = path.Join(sliverPkgDir, fileName)
		}

		fSliver, _ := os.Create(sliverCodePath)
		buf := bytes.NewBuffer([]byte{})
		buildLog.Infof("[render] %s", sliverCodePath)

		// Render code
		sliverCodeTmpl, _ := template.New("sliver").Parse(sliverGoCode)
		sliverCodeTmpl.Execute(buf, config)

		// Render canaries
		buildLog.Infof("Canary domain(s): %v", config.CanaryDomains)
		canaryTempl := template.New("canary").Delims("[[", "]]")
		canaryGenerator := &CanaryGenerator{
			SliverName:    config.Name,
			ParentDomains: config.CanaryDomains,
		}
		canaryTempl, err := canaryTempl.Funcs(template.FuncMap{
			"GenerateCanary": canaryGenerator.GenerateCanary,
		}).Parse(buf.String())
		canaryTempl.Execute(fSliver, canaryGenerator)

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
		obfuscatedPkg, err := gobfuscate.Gobfuscate(*goConfig, randomObfuscationKey(), pkgName, obfgoPath, obfSymbols)
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
	compiler := os.Getenv(SliverCCEnvVar)
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
		compiler = ""
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
