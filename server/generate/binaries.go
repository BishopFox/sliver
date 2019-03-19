package generate

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	clientpb "sliver/protobuf/client"
	"sliver/server/assets"
	"sliver/server/certs"
	gobfuscate "sliver/server/gobfuscate"
	gogo "sliver/server/gogo"
	"sliver/server/log"
	"text/template"

	"github.com/gobuffalo/packr"
)

var (
	buildLog = log.NamedLogger("generate", "build")
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

	defaultMingwPath = "/usr/bin/x86_64-w64-mingw32-gcc"
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
	ReconnectInterval   int    `json:"reconnect_interval"`
	MaxConnectionErrors int    `json:"max_connection_errors`

	C2            []string `json:"c2s"`
	MTLSc2Enabled bool     `json:"c2_mtls_enabled"`
	HTTPc2Enabled bool     `json:"c2_http_enabled"`
	DNSc2Enabled  bool     `json:"c2_dns_enabled"`

	// Limits
	LimitDomainJoined bool   `json:"limit_domainjoined"`
	LimitHostname     string `json:"limit_hostname"`
	LimitUsername     string `json:"limit_username"`
	LimitDatetime     string `json:"limit_datetime"`

	// DLL test
	IsSharedLib bool `json:"is_shared_lib"`
}

// ToProtobuf - Convert SliverConfig to protobuf equiv
func (c *SliverConfig) ToProtobuf() *clientpb.SliverConfig {
	return &clientpb.SliverConfig{
		GOOS:                c.GOOS,
		GOARCH:              c.GOARCH,
		Name:                c.Name,
		CACert:              c.CACert,
		Cert:                c.Cert,
		Key:                 c.Key,
		Debug:               c.Debug,
		ReconnectInterval:   uint32(c.ReconnectInterval),
		MaxConnectionErrors: uint32(c.MaxConnectionErrors),

		LimitDatetime:     c.LimitDatetime,
		LimitDomainJoined: c.LimitDomainJoined,
		LimitHostname:     c.LimitHostname,
		LimitUsername:     c.LimitUsername,

		IsSharedLib: c.IsSharedLib,
	}
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

	cfg.ReconnectInterval = int(pbConfig.ReconnectInterval)
	cfg.MaxConnectionErrors = int(pbConfig.MaxConnectionErrors)

	cfg.LimitDomainJoined = pbConfig.LimitDomainJoined
	cfg.LimitDatetime = pbConfig.LimitDatetime
	cfg.LimitUsername = pbConfig.LimitUsername
	cfg.LimitHostname = pbConfig.LimitHostname

	cfg.IsSharedLib = pbConfig.IsSharedLib

	cfg.C2 = copyC2List(pbConfig.C2)
	cfg.MTLSc2Enabled = isC2Enabled([]string{"mtls"}, cfg.C2)
	cfg.HTTPc2Enabled = isC2Enabled([]string{"http", "https"}, cfg.C2)
	cfg.DNSc2Enabled = isC2Enabled([]string{"dns"}, cfg.C2)

	return cfg
}

func copyC2List(src []*clientpb.SliverC2) []string {
	c2s := []string{}
	for _, srcC2 := range src {
		c2URL, err := url.Parse(srcC2.URL)
		if err != nil {
			buildLog.Warnf("Failed to parse c2 url %v", err)
			continue
		}
		c2s = append(c2s, c2URL.String())
	}
	return c2s
}

func isC2Enabled(schemes []string, c2s []string) bool {
	for _, c2 := range c2s {
		c2URL, err := url.Parse(c2)
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

// SliverSharedLibrary - Generates a sliver shared library (DLL/dylib/so) binary
func SliverSharedLibrary(config *SliverConfig) (string, error) {
	// Compile go code
	appDir := assets.GetRootAppDir()
	crossCompiler := getCCompiler()
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
	return dest, err
}

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
	rootDir := assets.GetRootAppDir()
	caCert, _, _ := certs.GetCertificateAuthorityPEM(rootDir, certs.SliversCertDir)
	sliverCert, sliverKey := certs.GenerateSliverCertificate(rootDir, config.Name, true)
	config.CACert = string(caCert)
	config.Cert = string(sliverCert)
	config.Key = string(sliverKey)

	// binDir - ~/.sliver/slivers/<os>/<arch>/<name>/bin
	binDir := path.Join(projectGoPathDir, "bin")
	os.MkdirAll(binDir, os.ModePerm)

	// srcDir - ~/.sliver/slivers/<os>/<arch>/<name>/src
	srcDir := path.Join(projectGoPathDir, "src")
	assets.SetupGoPath(srcDir) // Extract GOPATH dependancy files

	sliverPkgDir := path.Join(srcDir, "sliver") // "main"
	os.MkdirAll(sliverPkgDir, os.ModePerm)

	// Load code template
	sliverBox := packr.NewBox("../../sliver")
	for _, boxName := range srcFiles {
		sliverGoCode, _ := sliverBox.FindString(boxName)

		// We need to correct for the "sliver/sliver/foo" imports, since Go
		// doesn't allow relative imports and "sliver" is a subdirectory of
		// the main "sliver" repo we need to fake this when coping the code
		// to our per-compile "GOPATH"
		var sliverCodePath string
		dirName := filepath.Dir(boxName)
		fileName := filepath.Base(boxName)
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
		buildLog.Infof("[render] %s", sliverCodePath)
		sliverCodeTmpl, _ := template.New("sliver").Parse(sliverGoCode)
		err := sliverCodeTmpl.Execute(fSliver, config)
		if err != nil {
			buildLog.Infof("Failed to render go code: %v", err)
			return "", err
		}
	}

	if !config.Debug && !config.IsSharedLib {
		buildLog.Infof("Obfuscating source code ...")
		obfuscatedGoPath := path.Join(projectGoPathDir, "obfuscated")
		obfuscatedPkg, err := gobfuscate.Gobfuscate(*goConfig, randomObfuscationKey(), "sliver", obfuscatedGoPath)
		if err != nil {
			buildLog.Infof("Error while obfuscating sliver %v", err)
			return "", err
		}
		goConfig.GOPATH = obfuscatedGoPath
		buildLog.Infof("Obfuscated GOPATH = %s", obfuscatedGoPath)
		buildLog.Infof("Obfuscated sliver package: %s", obfuscatedPkg)
		sliverPkgDir = path.Join(obfuscatedGoPath, "src", obfuscatedPkg) // new "main"
	}
	return sliverPkgDir, nil
}

func getCCompiler() string {
	compiler := os.Getenv("SLIVER_CC")
	if compiler == "" {
		compiler = defaultMingwPath
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
