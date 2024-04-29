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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	insecureRand "math/rand"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/bishopfox/sliver/implant"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/encoders"
	"github.com/bishopfox/sliver/server/gogo"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/util"
	utilEncoders "github.com/bishopfox/sliver/util/encoders"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	buildLog = log.NamedLogger("generate", "build")

	// SupportedCompilerTargets - Supported compiler targets
	SupportedCompilerTargets = map[string]bool{
		"darwin/amd64":  true,
		"darwin/arm64":  true,
		"linux/386":     true,
		"linux/amd64":   true,
		"linux/arm64":   true,
		"windows/386":   true,
		"windows/amd64": true,
	}
)

const (
	SliverTemplateName = "sliver"

	// WINDOWS OS
	WINDOWS = "windows"

	// DARWIN / MacOS
	DARWIN = "darwin"

	// LINUX OS
	LINUX = "linux"

	clientsDirName = "clients"
	sliversDirName = "slivers"

	// DefaultReconnectInterval - In seconds
	DefaultReconnectInterval = 60
	// DefaultMTLSLPort - Default listen port
	DefaultMTLSLPort = 8888
	// DefaultHTTPLPort - Default HTTP listen port
	DefaultHTTPLPort = 443 // Assume SSL, it'll fallback
	// DefaultPollInterval - In seconds
	DefaultPollInterval = 1

	// DefaultSuffix - Indicates a platform independent src file
	DefaultSuffix = "_default.go"

	// *** Default ***

	// SliverCC64EnvVar - Environment variable that can specify the 64 bit mingw path
	SliverCC64EnvVar = "SLIVER_CC_64"
	// SliverCC32EnvVar - Environment variable that can specify the 32 bit mingw path
	SliverCC32EnvVar = "SLIVER_CC_32"

	// SliverCXX64EnvVar - Environment variable that can specify the 64 bit mingw path
	SliverCXX64EnvVar = "SLIVER_CXX_64"
	// SliverCXX32EnvVar - Environment variable that can specify the 32 bit mingw path
	SliverCXX32EnvVar = "SLIVER_CXX_32"

	// *** Platform Specific ***

	// SliverPlatformCC64EnvVar - Environment variable that can specify the 64 bit mingw path
	SliverPlatformCC64EnvVar = "SLIVER_%s_CC_64"
	// SliverPlatformCC32EnvVar - Environment variable that can specify the 32 bit mingw path
	SliverPlatformCC32EnvVar = "SLIVER_%s_CC_32"
	// SliverPlatformCXX64EnvVar - Environment variable that can specify the 64 bit mingw path
	SliverPlatformCXX64EnvVar = "SLIVER_%s_CXX_64"
	// SliverPlatformCXX32EnvVar - Environment variable that can specify the 32 bit mingw path
	SliverPlatformCXX32EnvVar = "SLIVER_%s_CXX_32"
)

// GetSliversDir - Get the binary directory
func GetSliversDir() string {
	appDir := assets.GetRootAppDir()
	sliversDir := filepath.Join(appDir, sliversDirName)
	if _, err := os.Stat(sliversDir); os.IsNotExist(err) {
		buildLog.Debugf("Creating bin directory: %s", sliversDir)
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

// SliverShellcode - Generates a sliver shellcode using Donut
func SliverShellcode(name string, build *clientpb.ImplantBuild, config *clientpb.ImplantConfig, pbC2Implant *clientpb.HTTPC2ImplantConfig) (string, error) {
	if config.GOOS != "windows" {
		return "", fmt.Errorf("shellcode format is currently only supported on Windows")
	}
	appDir := assets.GetRootAppDir()
	goConfig := &gogo.GoConfig{
		CGO: "0",

		GOOS:       config.GOOS,
		GOARCH:     config.GOARCH,
		GOCACHE:    gogo.GetGoCache(appDir),
		GOMODCACHE: gogo.GetGoModCache(appDir),
		GOROOT:     gogo.GetGoRootDir(appDir),
		GOPROXY:    getGoProxy(),
		HTTPPROXY:  getGoHttpProxy(),
		HTTPSPROXY: getGoHttpsProxy(),

		Obfuscation: config.ObfuscateSymbols,
		GOGARBLE:    goGarble(config),
	}
	pkgPath, err := renderSliverGoCode(name, build, config, goConfig, pbC2Implant)
	if err != nil {
		return "", err
	}

	dest := filepath.Join(goConfig.ProjectDir, "bin", filepath.Base(name))
	dest += ".bin"

	// if the destination already exists, delete it
	if _, err := os.Stat(dest); err == nil {
		os.Remove(dest)
	}

	tags := []string{}
	if config.NetGoEnabled {
		tags = append(tags, "netgo")
	}
	ldflags := []string{""} // Garble will automatically add "-s -w -buildid="
	if !config.Debug && goConfig.GOOS == WINDOWS {
		ldflags[0] += " -H=windowsgui"
	}
	// Keep those for potential later use
	gcFlags := ""
	asmFlags := ""
	_, err = gogo.GoBuild(*goConfig, pkgPath, dest, "pie", tags, ldflags, gcFlags, asmFlags)
	if err != nil {
		return "", err
	}
	shellcode, err := DonutShellcodeFromFile(dest, config.GOARCH, false, "", "", "")
	if err != nil {
		return "", err
	}
	err = os.WriteFile(dest, shellcode, 0600)
	if err != nil {
		return "", err
	}
	config.Format = clientpb.OutputFormat_SHELLCODE

	return dest, err

}

// SliverSharedLibrary - Generates a sliver shared library (DLL/dylib/so) binary
func SliverSharedLibrary(name string, build *clientpb.ImplantBuild, config *clientpb.ImplantConfig, pbC2Implant *clientpb.HTTPC2ImplantConfig) (string, error) {
	appDir := assets.GetRootAppDir()

	var cc string
	var cxx string
	if runtime.GOOS != config.GOOS || runtime.GOARCH != config.GOARCH {
		buildLog.Debugf("Cross-compiling from %s/%s to %s/%s", runtime.GOOS, runtime.GOARCH, config.GOOS, config.GOARCH)
		cc, cxx = findCrossCompilers(config.GOOS, config.GOARCH)
	}

	buildLog.Infof(" CC: %s", cc)
	buildLog.Infof("CXX: %s", cxx)

	goConfig := &gogo.GoConfig{
		CGO: "1",
		CC:  cc,
		CXX: cxx,

		GOOS:       config.GOOS,
		GOARCH:     config.GOARCH,
		GOCACHE:    gogo.GetGoCache(appDir),
		GOMODCACHE: gogo.GetGoModCache(appDir),
		GOROOT:     gogo.GetGoRootDir(appDir),
		GOPROXY:    getGoProxy(),
		HTTPPROXY:  getGoHttpProxy(),
		HTTPSPROXY: getGoHttpsProxy(),

		Obfuscation: config.ObfuscateSymbols,
		GOGARBLE:    goGarble(config),
	}
	pkgPath, err := renderSliverGoCode(name, build, config, goConfig, pbC2Implant)
	if err != nil {
		return "", err
	}

	dest := filepath.Join(goConfig.ProjectDir, "bin", filepath.Base(name))
	if goConfig.GOOS == WINDOWS {
		dest += ".dll"
	}
	if goConfig.GOOS == DARWIN {
		dest += ".dylib"
	}
	if goConfig.GOOS == LINUX {
		dest += ".so"
	}

	tags := []string{}
	if config.NetGoEnabled {
		tags = append(tags, "netgo")
	}
	ldflags := []string{""} // Garble will automatically add "-s -w -buildid="
	if !config.Debug && goConfig.GOOS == WINDOWS {
		ldflags[0] += " -H=windowsgui"
	}
	// Statically link Linux .so files to avoid glibc hell
	if goConfig.GOOS == LINUX && goConfig.CC != "" && goConfig.CGO == "1" {
		ldflags[0] += " -linkmode external -extldflags \"-static\""
	}
	// Keep those for potential later use
	gcFlags := ""
	asmFlags := ""
	_, err = gogo.GoBuild(*goConfig, pkgPath, dest, "c-shared", tags, ldflags, gcFlags, asmFlags)
	if err != nil {
		return "", err
	}

	return dest, err
}

// SliverExecutable - Generates a sliver executable binary
func SliverExecutable(name string, build *clientpb.ImplantBuild, config *clientpb.ImplantConfig, pbC2Implant *clientpb.HTTPC2ImplantConfig) (string, error) {
	appDir := assets.GetRootAppDir()

	// var cc string
	// var cxx string
	// cgo := "0"
	// if runtime.GOOS != config.GOOS {
	// 	buildLog.Debugf("Cross-compiling from %s/%s to %s/%s", runtime.GOOS, runtime.GOARCH, config.GOOS, config.GOARCH)
	// 	cc, cxx = findCrossCompilers(config.GOOS, config.GOARCH)
	// 	cgo = "1"
	// }

	// buildLog.Infof(" CC: %s", cc)
	// buildLog.Infof("CXX: %s", cxx)

	goConfig := &gogo.GoConfig{
		CGO:        "0",
		GOOS:       config.GOOS,
		GOARCH:     config.GOARCH,
		GOROOT:     gogo.GetGoRootDir(appDir),
		GOCACHE:    gogo.GetGoCache(appDir),
		GOMODCACHE: gogo.GetGoModCache(appDir),
		GOPROXY:    getGoProxy(),
		HTTPPROXY:  getGoHttpProxy(),
		HTTPSPROXY: getGoHttpsProxy(),

		Obfuscation: config.ObfuscateSymbols,
		GOGARBLE:    goGarble(config),
	}

	pkgPath, err := renderSliverGoCode(name, build, config, goConfig, pbC2Implant)
	if err != nil {
		return "", err
	}

	dest := filepath.Join(goConfig.ProjectDir, "bin", filepath.Base(name))
	if goConfig.GOOS == WINDOWS {
		dest += ".exe"
	}
	tags := []string{}
	if config.NetGoEnabled {
		tags = append(tags, "netgo")
	}
	ldflags := []string{""} // Garble will automatically add "-s -w -buildid="
	if !config.Debug && goConfig.GOOS == WINDOWS {
		ldflags[0] += " -H=windowsgui"
	}
	gcFlags := ""
	asmFlags := ""
	if config.Debug {
		gcFlags = "all=-N -l"
		ldflags = []string{}
	}
	_, err = gogo.GoBuild(*goConfig, pkgPath, dest, "", tags, ldflags, gcFlags, asmFlags)
	if err != nil {
		return "", err
	}

	return dest, err
}

// This function is a little too long, we should probably refactor it as some point
func renderSliverGoCode(name string, build *clientpb.ImplantBuild, config *clientpb.ImplantConfig, goConfig *gogo.GoConfig, pbC2Implant *clientpb.HTTPC2ImplantConfig) (string, error) {
	target := fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH)
	if _, ok := gogo.ValidCompilerTargets(*goConfig)[target]; !ok {
		return "", fmt.Errorf("invalid compiler target: %s", target)
	}
	if name == "" {
		return "", fmt.Errorf("name cannot be empty")
	}

	buildLog.Debugf("Generating new sliver binary '%s'", name)
	pbC2Implant = models.RandomizeImplantConfig(pbC2Implant, config.GOOS, config.GOARCH)
	sliversDir := GetSliversDir() // ~/.sliver/slivers
	projectGoPathDir := filepath.Join(sliversDir, config.GOOS, config.GOARCH, filepath.Base(name))
	if _, err := os.Stat(projectGoPathDir); os.IsNotExist(err) {
		os.MkdirAll(projectGoPathDir, 0700)
	}
	goConfig.ProjectDir = projectGoPathDir

	// binDir - ~/.sliver/slivers/<os>/<arch>/<name>/bin
	binDir := filepath.Join(projectGoPathDir, "bin")
	os.MkdirAll(binDir, 0700)

	// srcDir - ~/.sliver/slivers/<os>/<arch>/<name>/src
	srcDir := filepath.Join(projectGoPathDir, "src")
	assets.SetupGoPath(srcDir, config.IncludeDNS) // Extract GOPATH dependency files
	err := util.ChmodR(srcDir, 0600, 0700)        // Ensures src code files are writable
	if err != nil {
		buildLog.Errorf("fs perms: %v", err)
		return "", err
	}

	sliverPkgDir := filepath.Join(srcDir, "github.com", "bishopfox", "sliver") // "main"
	err = os.MkdirAll(sliverPkgDir, 0700)
	if err != nil {
		return "", nil
	}

	err = fs.WalkDir(implant.FS, ".", func(fsPath string, f fs.DirEntry, err error) error {
		if f.IsDir() {
			return nil
		}
		buildLog.Debugf("Walking: %s %s %v", fsPath, f.Name(), err)

		sliverGoCodeRaw, err := implant.FS.ReadFile(fsPath)
		if err != nil {
			buildLog.Errorf("Failed to read %s: %s", fsPath, err)
			return nil
		}
		sliverGoCode := string(sliverGoCodeRaw)

		// Skip dllmain files for anything non windows
		if f.Name() == "sliver.c" || f.Name() == "sliver.h" {
			if !config.IsSharedLib && !config.IsShellcode {
				return nil
			}
		}

		var sliverCodePath string
		if f.Name() == "sliver.go" || f.Name() == "sliver.c" || f.Name() == "sliver.h" {
			sliverCodePath = filepath.Join(sliverPkgDir, f.Name())
		} else {
			sliverCodePath = filepath.Join(sliverPkgDir, "implant", fsPath)
		}
		dirPath := filepath.Dir(sliverCodePath)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			buildLog.Debugf("[mkdir] %#v", dirPath)
			err = os.MkdirAll(dirPath, 0700)
			if err != nil {
				return err
			}
		}
		fSliver, err := os.Create(sliverCodePath)
		if err != nil {
			return err
		}
		if !util.Contains([]string{".go", ".c", ".h"}, path.Ext(f.Name())) {
			buildLog.Debugf("Skipping render for %s, does not appear to be source code file", f.Name())
			_, err = fSliver.Write(sliverGoCodeRaw)
			return err
		}

		encoderStruct := utilEncoders.EncodersList{
			Base32EncoderID:  encoders.Base32EncoderID,
			Base58EncoderID:  encoders.Base58EncoderID,
			Base64EncoderID:  encoders.Base64EncoderID,
			EnglishEncoderID: encoders.EnglishEncoderID,
			GzipEncoderID:    encoders.GzipEncoderID,
			HexEncoderID:     encoders.HexEncoderID,
			PNGEncoderID:     encoders.PNGEncoderID,
		}

		// --------------
		// Render Code
		// --------------
		buf := bytes.NewBuffer([]byte{})
		buildLog.Debugf("[render] %s -> %s", f.Name(), sliverCodePath)

		sliverCode := template.New("sliver")
		sliverCode, err = sliverCode.Funcs(template.FuncMap{
			"GenerateUserAgent": func() string {
				return pbC2Implant.UserAgent
			},
		}).Parse(sliverGoCode)
		if err != nil {
			buildLog.Errorf("Template parsing error %s", err)
			return err
		}
		err = sliverCode.Execute(buf, struct {
			Name                string
			Config              *clientpb.ImplantConfig
			Build               *clientpb.ImplantBuild
			HTTPC2ImplantConfig *clientpb.HTTPC2ImplantConfig
			Encoders            utilEncoders.EncodersList
		}{
			name,
			config,
			build,
			pbC2Implant,
			encoderStruct,
		})
		if err != nil {
			buildLog.Errorf("Template execution error %s", err)
			return err
		}

		// Render canaries
		if len(config.CanaryDomains) > 0 {
			buildLog.Debugf("Canary domain(s): %v", config.CanaryDomains)
		}
		canaryTemplate := template.New("canary").Delims("[[", "]]")
		canaryGenerator := &CanaryGenerator{
			ImplantName:   name,
			ParentDomains: config.CanaryDomains,
		}
		canaryTemplate, err = canaryTemplate.Funcs(template.FuncMap{
			"GenerateCanary": canaryGenerator.GenerateCanary,
		}).Parse(buf.String())
		if err != nil {
			return err
		}
		err = canaryTemplate.Execute(fSliver, canaryGenerator)
		if err != nil {
			buildLog.Debugf("Failed to render go code: %s", err)
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	// Render encoder assets
	renderNativeEncoderAssets(build, config, sliverPkgDir)
	if config.TrafficEncodersEnabled {
		renderTrafficEncoderAssets(build, config, sliverPkgDir)
	}

	// Render GoMod
	buildLog.Info("Rendering go.mod file ...")
	goModPath := filepath.Join(sliverPkgDir, "go.mod")
	err = os.WriteFile(goModPath, []byte(implant.GoMod), 0600)
	if err != nil {
		return "", err
	}
	goSumPath := filepath.Join(sliverPkgDir, "go.sum")
	err = os.WriteFile(goSumPath, []byte(implant.GoSum), 0600)
	if err != nil {
		return "", err
	}
	// Render vendor dir
	err = fs.WalkDir(implant.Vendor, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return os.MkdirAll(filepath.Join(sliverPkgDir, path), 0700)
		}

		contents, err := implant.Vendor.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(filepath.Join(sliverPkgDir, path), contents, 0600)
	})
	if err != nil {
		buildLog.Errorf("Failed to copy vendor directory %v", err)
		return "", err
	}
	buildLog.Debugf("Created %s", goModPath)

	return sliverPkgDir, nil
}

// renderTrafficEncoderAssets - Copies and compresses any enabled WASM traffic encoders
func renderTrafficEncoderAssets(_ *clientpb.ImplantBuild, config *clientpb.ImplantConfig, sliverPkgDir string) {
	buildLog.Infof("Rendering traffic encoder assets ...")
	encoderAssetsPath := filepath.Join(sliverPkgDir, "implant", "sliver", "encoders", "assets")
	for _, asset := range config.Assets {
		if !strings.HasSuffix(asset.Name, ".wasm") {
			continue
		}
		wasm, err := encoders.TrafficEncoderFS.ReadFile(asset.Name)
		if err != nil {
			buildLog.Errorf("Failed to read %s: %v", asset.Name, err)
			continue
		}
		saveAssetPath := filepath.Join(encoderAssetsPath, filepath.Base(asset.Name))
		compressedWasm, _ := encoders.Gzip.Encode(wasm)
		buildLog.Infof("Embed traffic encoder %s (%s, %s compressed)",
			asset.Name, util.ByteCountBinary(int64(len(wasm))), util.ByteCountBinary(int64(len(compressedWasm))))
		err = os.WriteFile(saveAssetPath, compressedWasm, 0600)
		if err != nil {
			buildLog.Errorf("Failed to write %s: %v", saveAssetPath, err)
			continue
		}
	}
}

// renderNativeEncoderAssets - Render native encoder assets such as the english dictionary file
func renderNativeEncoderAssets(_ *clientpb.ImplantBuild, _ *clientpb.ImplantConfig, sliverPkgDir string) {
	buildLog.Infof("Rendering native encoder assets ...")
	encoderAssetsPath := filepath.Join(sliverPkgDir, "implant", "sliver", "encoders", "assets")

	// English assets
	dictionary := renderImplantEnglish()
	dictionaryPath := filepath.Join(encoderAssetsPath, "english.gz")
	data := []byte(strings.Join(dictionary, "\n"))
	compressedData, _ := encoders.Gzip.Encode(data)
	buildLog.Infof("Embed english dictionary (%s, %s compressed)",
		util.ByteCountBinary(int64(len(data))), util.ByteCountBinary(int64(len(compressedData))))
	err := os.WriteFile(dictionaryPath, compressedData, 0600)
	if err != nil {
		buildLog.Errorf("Failed to write %s: %v", dictionaryPath, err)
	}
}

// renderImplantEnglish - Render the english dictionary file, ensures that the returned dictionary
// contains at least one word that will encode to a given byte value (0-255). There is also a default
// dictionary that we'll try to overwrite (the default one is in the git repo).
func renderImplantEnglish() []string {
	allWords := assets.English() // 178,543 words -> server/assets/fs/english.txt
	meRiCaN := cases.Title(language.AmericanEnglish)
	for i := 0; i < len(allWords); i++ {
		switch insecureRand.Intn(3) {
		case 0:
			allWords[i] = strings.ToUpper(allWords[i])
		case 1:
			allWords[i] = strings.ToLower(allWords[i])
		case 2:
			allWords[i] = meRiCaN.String(allWords[i])
		}
	}

	// Calculate the sum for each word
	allWordsDictionary := map[int][]string{}
	for _, word := range allWords {
		word = strings.TrimSpace(word)
		sum := utilEncoders.SumWord(word)
		allWordsDictionary[sum] = append(allWordsDictionary[sum], word)
	}

	// Shuffle the words for each byte value
	for byteValue := 0; byteValue < 256; byteValue++ {
		insecureRand.Shuffle(len(allWordsDictionary[byteValue]), func(i, j int) {
			allWordsDictionary[byteValue][i], allWordsDictionary[byteValue][j] = allWordsDictionary[byteValue][j], allWordsDictionary[byteValue][i]
		})
	}

	// Build the implant's dictionary, two words per-byte value
	implantDictionary := []string{}
	for byteValue := 0; byteValue < 256; byteValue++ {
		wordsForByteValue := allWordsDictionary[byteValue]
		implantDictionary = append(implantDictionary, wordsForByteValue[0])
		if 1 < len(wordsForByteValue) {
			implantDictionary = append(implantDictionary, wordsForByteValue[1])
		}
	}
	return implantDictionary
}

// GenerateConfig - Generate the keys/etc for the implant
func GenerateConfig(name string, implantConfig *clientpb.ImplantConfig) (*clientpb.ImplantBuild, error) {
	var err error

	// Cert PEM encoded certificates
	serverCACert, _, _ := certs.GetCertificateAuthorityPEM(certs.MtlsServerCA)
	sliverCert, sliverKey, err := certs.MtlsC2ImplantGenerateECCCertificate(name)
	if err != nil {
		return nil, err
	}

	build := clientpb.ImplantBuild{
		Name: name,
	}

	// ECC keys - only generate if config is not set
	implantKeyPair, err := cryptography.RandomAgeKeyPair()
	if err != nil {
		return nil, err
	}
	serverKeyPair := cryptography.AgeServerKeyPair()
	digest := sha256.Sum256([]byte(implantKeyPair.Public))
	build.PeerPublicKey = implantKeyPair.Public
	build.PeerPublicKeyDigest = hex.EncodeToString(digest[:])
	build.PeerPrivateKey = implantKeyPair.Private
	build.PeerPublicKeySignature = cryptography.MinisignServerSign([]byte(implantKeyPair.Public))
	build.AgeServerPublicKey = serverKeyPair.Public
	build.MinisignServerPublicKey = cryptography.MinisignServerPublicKey()

	// MTLS keys
	if models.IsC2Enabled([]string{"mtls"}, implantConfig.C2) {
		build.MtlsCACert = string(serverCACert)
		build.MtlsCert = string(sliverCert)
		build.MtlsKey = string(sliverKey)
	}

	// Generate wg Keys as needed
	if models.IsC2Enabled([]string{"wg"}, implantConfig.C2) {
		implantPrivKey, _, err := certs.ImplantGenerateWGKeys(implantConfig.WGPeerTunIP)
		if err != nil {
			return nil, err
		}
		_, serverPubKey, err := certs.GetWGServerKeys()
		if err != nil {
			return nil, fmt.Errorf("failed to embed implant wg keys: %s", err)
		}
		build.WGImplantPrivKey = implantPrivKey
		build.WGServerPubKey = serverPubKey
	}

	build.Stage = false

	return &build, nil
}

// Platform specific ENV VARS take precedence over generic
func getCrossCompilersFromEnv(targetGoos string, targetGoarch string) (string, string) {
	var cc string
	var cxx string

	TARGET_GOOS := strings.ToUpper(targetGoos)

	// Get Defaults
	if targetGoarch == "amd64" {
		if os.Getenv(fmt.Sprintf(SliverPlatformCC64EnvVar, TARGET_GOOS)) != "" {
			cc = os.Getenv(fmt.Sprintf(SliverPlatformCC64EnvVar, TARGET_GOOS))
		}
		if cc == "" {
			cc = os.Getenv(SliverCC64EnvVar)
		}
		if os.Getenv(fmt.Sprintf(SliverPlatformCXX64EnvVar, TARGET_GOOS)) != "" {
			cc = os.Getenv(fmt.Sprintf(SliverPlatformCXX64EnvVar, TARGET_GOOS))
		}
		if cxx == "" {
			cxx = os.Getenv(SliverCXX64EnvVar)
		}
	}
	if targetGoarch == "386" {
		cc = os.Getenv(SliverCC32EnvVar)
		if os.Getenv(fmt.Sprintf(SliverPlatformCC32EnvVar, TARGET_GOOS)) != "" {
			cc = os.Getenv(fmt.Sprintf(SliverPlatformCC32EnvVar, TARGET_GOOS))
		}
		cxx = os.Getenv(SliverCXX64EnvVar)
		if os.Getenv(fmt.Sprintf(SliverPlatformCXX32EnvVar, TARGET_GOOS)) != "" {
			cc = os.Getenv(fmt.Sprintf(SliverPlatformCXX32EnvVar, TARGET_GOOS))
		}
	}
	return cc, cxx
}

func findCrossCompilers(targetGOOS string, targetGOARCH string) (string, string) {

	// Get CC and CXX from ENV -- First Priority
	cc, cxx := getCrossCompilersFromEnv(targetGOOS, targetGOARCH)
	if cc != "" && cxx != "" {
		buildLog.Debugf("CC and CXX found in ENV: cc=%s, cxx=%s", cc, cxx)
		return cc, cxx
	}

	// Server config file -- Second Priority
	serverConfig := configs.GetServerConfig()
	if serverConfig != nil {
		if cc == "" {
			if value, ok := serverConfig.CC[fmt.Sprintf("%s/%s", targetGOOS, targetGOARCH)]; ok {
				cc = value
			} else {
				buildLog.Debugf("CC for %s/%s not found in ENV or server config", targetGOOS, targetGOARCH)
			}
		}
		if cxx == "" {
			if value, ok := serverConfig.CXX[fmt.Sprintf("%s/%s", targetGOOS, targetGOARCH)]; ok {
				cxx = value
			} else {
				buildLog.Debugf("CXX for %s/%s not found in ENV or server config", targetGOOS, targetGOARCH)
			}
		}
		if cc != "" && cxx != "" {
			buildLog.Debugf("CC and CXX found in ENV/server config: cc=%s, cxx=%s", cc, cxx)
			return cc, cxx
		}
	}

	// Defaults -- Tertiary Priority
	// Darwin/zig doesn't work :( maybe it will in the future though
	if targetGOOS != DARWIN && (cc == "" || cxx == "") {
		zigTarget := map[string]string{
			"windows/amd64": "x86_64-windows-gnu",
			"windows/arm64": "aarch64-windows-gnu",
			"windows/386":   "x86-windows-gnu",
			"linux/amd64":   "x86_64-linux-musl",
			"linux/386":     "x86-linux-musl",
			"linux/arm64":   "aarch64-linux-musl",
			"linux/ppc64":   "powerpc64le-linux-musl",
		}[fmt.Sprintf("%s/%s", targetGOOS, targetGOARCH)]

		zigDir := assets.GetZigDir()
		if cc == "" {
			buildLog.Debugf("Using default zig cc for %s/%s", targetGOOS, targetGOARCH)
			cc = fmt.Sprintf("%s cc -target %s", filepath.Join(zigDir, "zig"), zigTarget)

		}
		if cxx == "" {
			buildLog.Debugf("Using default zig cxx for %s/%s", targetGOOS, targetGOARCH)
			cxx = fmt.Sprintf("%s c++ -target %s", filepath.Join(zigDir, "zig"), zigTarget)
		}
	}
	// Try to use OSXCross for cross-compiling to Darwin
	if targetGOOS == DARWIN && runtime.GOOS != DARWIN {
		if targetGOARCH == "amd64" && cc == "" {
			buildLog.Debugf("Using default osxcross cc/cxx for %s/%s", targetGOOS, targetGOARCH)
			cc = "/opt/osxcross/target/bin/o64-clang"
			if cxx == "" {
				cxx = cc
			}
		}
		if targetGOARCH == "arm64" && cc == "" {
			buildLog.Debugf("Using default osxcross cc/cxx for %s/%s", targetGOOS, targetGOARCH)
			cc = "/opt/osxcross/target/bin/aarch64-apple-darwin20.2-clang"
			if cxx == "" {
				cxx = cc
			}
		}
	}

	return cc, cxx
}

// GetCompilerTargets - This function attempts to determine what we can reasonably target
func GetCompilerTargets() []*clientpb.CompilerTarget {
	targets := []*clientpb.CompilerTarget{}

	// EXE - Any server should be able to target EXEs of each platform
	for longPlatform := range SupportedCompilerTargets {
		platform := strings.SplitN(longPlatform, "/", 2)
		targets = append(targets, &clientpb.CompilerTarget{
			GOOS:   platform[0],
			GOARCH: platform[1],
			Format: clientpb.OutputFormat_EXECUTABLE,
		})
	}

	// SHARED_LIB - Determine if we can probably build a dll/dylib/so
	for longPlatform := range SupportedCompilerTargets {
		platform := strings.SplitN(longPlatform, "/", 2)

		// We can always build our own platform
		if runtime.GOOS == platform[0] {
			targets = append(targets, &clientpb.CompilerTarget{
				GOOS:   platform[0],
				GOARCH: platform[1],
				Format: clientpb.OutputFormat_SHARED_LIB,
			})
			continue
		}

		// Cross-compile with the right configuration
		if runtime.GOOS == LINUX || runtime.GOOS == DARWIN {
			cc, _ := findCrossCompilers(platform[0], platform[1])
			if cc != "" {
				if runtime.GOOS == DARWIN && platform[0] == LINUX && platform[1] == "386" {
					continue // Darwin can't target 32-bit Linux, even with a cc/cxx
				}
				targets = append(targets, &clientpb.CompilerTarget{
					GOOS:   platform[0],
					GOARCH: platform[1],
					Format: clientpb.OutputFormat_SHARED_LIB,
				})
			}
		}

	}

	// SERVICE - Can generate service executables for Windows targets only
	for longPlatform := range SupportedCompilerTargets {
		platform := strings.SplitN(longPlatform, "/", 2)
		if platform[0] != WINDOWS {
			continue
		}

		targets = append(targets, &clientpb.CompilerTarget{
			GOOS:   platform[0],
			GOARCH: platform[1],
			Format: clientpb.OutputFormat_SERVICE,
		})
	}

	// SHELLCODE - Can generate shellcode for Windows targets only
	for longPlatform := range SupportedCompilerTargets {
		platform := strings.SplitN(longPlatform, "/", 2)
		if platform[0] != WINDOWS {
			continue
		}

		targets = append(targets, &clientpb.CompilerTarget{
			GOOS:   platform[0],
			GOARCH: platform[1],
			Format: clientpb.OutputFormat_SHELLCODE,
		})
	}

	return targets
}

// GetCrossCompilers - Get information about the server's cross-compiler configuration
func GetCrossCompilers() []*clientpb.CrossCompiler {
	compilers := []*clientpb.CrossCompiler{}
	for longPlatform := range SupportedCompilerTargets {
		platform := strings.SplitN(longPlatform, "/", 2)
		if runtime.GOOS == platform[0] {
			continue
		}
		cc, cxx := findCrossCompilers(platform[0], platform[1])
		if cc != "" {
			compilers = append(compilers, &clientpb.CrossCompiler{
				TargetGOOS:   platform[0],
				TargetGOARCH: platform[1],
				CCPath:       cc,
				CXXPath:      cxx,
			})
		}
	}
	return compilers
}

// GetUnsupportedTargets - Get compiler targets that are not "supported" on this platform
func GetUnsupportedTargets() []*clientpb.CompilerTarget {
	appDir := assets.GetRootAppDir()
	distList := gogo.GoToolDistList(gogo.GoConfig{
		GOCACHE:    gogo.GetGoCache(appDir),
		GOMODCACHE: gogo.GetGoModCache(appDir),
		GOROOT:     gogo.GetGoRootDir(appDir),
	})
	targets := []*clientpb.CompilerTarget{}
	for _, dist := range distList {
		if _, ok := SupportedCompilerTargets[dist]; ok {
			continue
		}
		parts := strings.SplitN(dist, "/", 2)
		if len(parts) != 2 {
			continue
		}
		targets = append(targets, &clientpb.CompilerTarget{
			GOOS:   parts[0],
			GOARCH: parts[1],
			Format: clientpb.OutputFormat_EXECUTABLE,
		})
	}
	return targets
}

func getGoProxy() string {
	serverConfig := configs.GetServerConfig()
	if serverConfig.GoProxy != "" {
		buildLog.Debugf("Using GOPROXY from server config = %s", serverConfig.GoProxy)
		return serverConfig.GoProxy
	}
	value, present := os.LookupEnv("GOPROXY")
	if present {
		buildLog.Debugf("Using GOPROXY from env: %s", value)
		return value
	}
	const defaultGoProxy = "off"
	buildLog.Debugf("No GOPROXY setting found, default to %s", defaultGoProxy)
	return defaultGoProxy
}

func getGoHttpProxy() string {
	value, present := os.LookupEnv("HTTP_PROXY")
	if present {
		buildLog.Debugf("Using HTTP_PROXY from env: %s", value)
		return value
	}
	buildLog.Debugf("No HTTP_PROXY found")
	return ""
}

func getGoHttpsProxy() string {
	value, present := os.LookupEnv("HTTPS_PROXY")
	if present {
		buildLog.Debugf("Using HTTPS_PROXY from env: %s", value)
		return value
	}
	buildLog.Debugf("No HTTPS_PROXY found")
	return ""
}

const (
	allGoPrivate = "*"
)

// goGarble - Can be used to conditionally modify the GOGARBLE env variable
// this is currently set to '*' (all packages) however in the past we've had
// to carve out specific packages, so we left this here just in case we need
// it in the future.
func goGarble(_ *clientpb.ImplantConfig) string {
	// for _, c2 := range config.C2 {
	// 	uri, err := url.Parse(c2.URL)
	// 	if err != nil {
	// 		return wgGoPrivate
	// 	}
	// 	if uri.Scheme == "wg" {
	// 		return wgGoPrivate
	// 	}
	// }
	return allGoPrivate
}
