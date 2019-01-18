package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
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
func GenerateImplantBinary(goos string, goarch string) (string, error) {

	goos = path.Base(goos)
	goarch = path.Base(goarch)
	target := fmt.Sprintf("%s/%s", goos, goarch)
	if _, ok := validCompilerTargets[target]; !ok {
		return "", fmt.Errorf("Invalid compiler target: %s", target)
	}

	config := SliverConfig{
		DefaultServer:      "localhost",
		DefaultServerLPort: 8444,
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
	workingDir := path.Join(binDir, SliversDir, goos, goarch, config.Name)
	os.MkdirAll(workingDir, os.ModePerm)

	// Load code template
	sliverBox := packr.NewBox("../sliver")
	savePlatformCode(sliverBox, windowsPlatform, workingDir)
	savePlatformCode(sliverBox, linuxPlatform, workingDir)
	savePlatformCode(sliverBox, darwinPlatform, workingDir)

	sliverGoCode, _ := sliverBox.MustString("sliver.go")
	sliverCodePath := path.Join(workingDir, "sliver.go")
	fSliver, _ := os.Create(sliverCodePath)
	log.Printf("Rendering sliver code to: %s", sliverCodePath)
	sliverCodeTmpl, _ := template.New("sliver").Parse(sliverGoCode)
	err := sliverCodeTmpl.Execute(fSliver, config)
	if err != nil {
		log.Printf("Failed to render go code: %v", err)
		return "", err
	}

	// Compile go code
	goConfig := GoConfig{
		GOOS:   goos,
		GOARCH: goarch,
		GOROOT: GetGoRootDir(),
		GOPATH: GetGoPathDir(),
	}

	dst := path.Join(workingDir, config.Name)
	tags := []string{"netgo"}
	ldflags := []string{"-s -w"}
	_, err = GoBuild(goConfig, workingDir, dst, tags, ldflags)
	return dst, err
}

func savePlatformCode(sliverBox packr.Box, platform string, workingDir string) {
	fileName := fmt.Sprintf("sliver_%s.go", platform)
	sliverPlatformCode, _ := sliverBox.MustString(fileName)
	sliverPlatformCodePath := path.Join(workingDir, fileName)
	err := ioutil.WriteFile(sliverPlatformCodePath, []byte(sliverPlatformCode), os.ModePerm)
	if err != nil {
		log.Printf("Error writing file %s: %s", sliverPlatformCodePath, err)
	}
}
