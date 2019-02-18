package generate

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	pb "sliver/protobuf/client"
	"sliver/server/assets"
	"sliver/server/certs"
	gobfuscate "sliver/server/gobfuscate"
	gogo "sliver/server/gogo"
	"text/template"

	"github.com/gobuffalo/packr"
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
)

var (
	srcFiles = []string{

		"crypto.go",
		"handlers.go",
		"handlers_windows.go",
		"handlers_linux.go",
		"handlers_darwin.go",
		"tcp-mtls.go",
		"udp-dns.go",
		"sliver.go",

		"shell/shell.go",
		"shell/shell_windows.go",
		"shell/shell_darwin.go",
		"shell/pty/pty_darwin.go",
		"shell/shell_linux.go",
		"shell/pty/pty_linux.go",

		"shell/pty/run.go",
		"shell/pty/util.go",
		"shell/pty/doc.go",
		"shell/pty/types.go",
		"shell/pty/ztypes_386.go",
		"shell/pty/ztypes_amd64.go",
		"shell/pty/ioctl.go",
		"shell/pty/ioctl_bsd.go",
		"shell/pty/pty_unsupported.go",

		"limits/limits.go",
		"limits/limits_windows.go",
		"limits/limits_darwin.go",
		"limits/limits_linux.go",

		"ps/ps.go",
		"ps/ps_windows.go",
		"ps/ps_linux.go",
		"ps/ps_darwin.go",

		"taskrunner/task.go",
		"taskrunner/task_windows.go",
		"taskrunner/task_darwin.go",
		"taskrunner/task_linux.go",

		"procdump/dump.go",
		"procdump/dump_windows.go",
		"procdump/dump_linux.go",
		"procdump/dump_darwin.go",
	}
)

// SliverConfig - Parameters when generating a implant
type SliverConfig struct {
	// Go
	GOOS   string `json:"go_os"`
	GOARCH string `json:"go_arch"`

	// Standard
	Name              string `json:"name"`
	CACert            string `json:"ca_cert"`
	Cert              string `json:"cert"`
	Key               string `json:"key"`
	Debug             bool   `json:"debug"`
	ReconnectInterval int    `json:"reconnect_interval"`

	// mTLS
	MTLSServer string `json:"mtls_server"`
	MTLSLPort  uint16 `json:"mtls_lport"`

	// DNS
	DNSParent string `json:"dns_parent"`

	// Limits
	LimitDomainJoined bool   `json:"limit_domainjoined"`
	LimitHostname     string `json:"limit_hostname"`
	LimitUsername     string `json:"limit_username"`
	LimitDatetime     string `json:"limit_datetime"`
}

// ToProtobuf - Convert SliverConfig to protobuf equiv
func (c *SliverConfig) ToProtobuf() *pb.SliverConfig {
	return &pb.SliverConfig{
		GOOS:              c.GOOS,
		GOARCH:            c.GOARCH,
		Name:              c.Name,
		CACert:            c.CACert,
		Cert:              c.Cert,
		Key:               c.Key,
		Debug:             c.Debug,
		ReconnectInterval: int32(c.ReconnectInterval),
		MTLSServer:        c.MTLSServer,
		MTLSLPort:         int32(c.MTLSLPort),
		DNSParent:         c.DNSParent,

		LimitDatetime:     c.LimitDatetime,
		LimitDomainJoined: c.LimitDomainJoined,
		LimitHostname:     c.LimitHostname,
		LimitUsername:     c.LimitUsername,
	}
}

// SliverConfigFromProtobuf - Create config from Protobuf
func SliverConfigFromProtobuf(pbConfig *pb.SliverConfig) *SliverConfig {
	cfg := &SliverConfig{}
	cfg.GOOS = pbConfig.GOOS
	cfg.GOARCH = pbConfig.GOARCH
	cfg.Name = pbConfig.Name
	cfg.CACert = pbConfig.CACert
	cfg.Cert = pbConfig.Cert
	cfg.Key = pbConfig.Key
	cfg.Debug = pbConfig.Debug

	cfg.LimitDomainJoined = pbConfig.LimitDomainJoined
	cfg.LimitDatetime = pbConfig.LimitDatetime
	cfg.LimitUsername = pbConfig.LimitUsername
	cfg.LimitHostname = pbConfig.LimitHostname

	if pbConfig.ReconnectInterval != 0 {
		cfg.ReconnectInterval = int(pbConfig.ReconnectInterval)
	} else {
		cfg.ReconnectInterval = DefaultReconnectInterval
	}

	cfg.MTLSServer = pbConfig.MTLSServer
	if pbConfig.MTLSLPort != 0 {
		cfg.MTLSLPort = uint16(pbConfig.MTLSLPort)
	} else {
		cfg.MTLSLPort = DefaultMTLSLPort
	}

	cfg.DNSParent = pbConfig.DNSParent
	return cfg
}

// GetSliversDir - Get the binary directory
func GetSliversDir() string {
	appDir := assets.GetRootAppDir()
	sliversDir := path.Join(appDir, sliversDirName)
	if _, err := os.Stat(sliversDir); os.IsNotExist(err) {
		log.Printf("Creating bin directory: %s", sliversDir)
		err = os.MkdirAll(sliversDir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}
	return sliversDir
}

// SliverEgg - Generates a sliver egg (stager) binary
func SliverEgg(config SliverConfig) (string, error) {

	return "", nil
}

// SliverSharedLibrary - Generates a sliver shared library (DLL/dylib/so) binary
// NOTE: This doesn't work just yet, just a prototype
func SliverSharedLibrary(config *SliverConfig) (string, error) {
	// Compile go code
	appDir := assets.GetRootAppDir()
	goConfig := &gogo.GoConfig{
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
	log.Printf("Generating new sliver binary '%s'", config.Name)

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
				log.Printf("[mkdir] %#v", dirPath)
				os.MkdirAll(dirPath, os.ModePerm)
			}
			sliverCodePath = path.Join(dirPath, fileName)
		} else {
			sliverCodePath = path.Join(sliverPkgDir, fileName)
		}

		fSliver, _ := os.Create(sliverCodePath)
		log.Printf("[render] %s", sliverCodePath)
		sliverCodeTmpl, _ := template.New("sliver").Parse(sliverGoCode)
		err := sliverCodeTmpl.Execute(fSliver, config)
		if err != nil {
			log.Printf("Failed to render go code: %v", err)
			return "", err
		}
	}

	if !config.Debug {
		log.Printf("Obfuscating source code ...")
		obfuscatedGoPath := path.Join(projectGoPathDir, "obfuscated")
		obfuscatedPkg, err := gobfuscate.Gobfuscate(*goConfig, randomObfuscationKey(), "sliver", obfuscatedGoPath)
		if err != nil {
			log.Printf("Error while obfuscating sliver %v", err)
			return "", err
		}
		goConfig.GOPATH = obfuscatedGoPath
		log.Printf("Obfuscated GOPATH = %s", obfuscatedGoPath)
		log.Printf("Obfuscated sliver package: %s", obfuscatedPkg)
		sliverPkgDir = path.Join(obfuscatedGoPath, "src", obfuscatedPkg) // new "main"
	}
	return sliverPkgDir, nil
}

func randomObfuscationKey() string {
	randBuf := make([]byte, 64) // 64 bytes of randomness
	rand.Read(randBuf)
	digest := sha256.Sum256(randBuf)
	return fmt.Sprintf("%x", digest[:encryptKeySize])
}
