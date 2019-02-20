package assets

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"

	"sliver/server/certs"

	"github.com/gobuffalo/packr"
)

const (
	GoDirName     = "go"
	goPathDirName = "gopath"
)

var (
	assetsBox   = packr.NewBox("../../assets")
	protobufBox = packr.NewBox("../../protobuf")
)

// GetRootAppDir - Get the Sliver app dir ~/.sliver/
func GetRootAppDir() string {
	user, _ := user.Current()
	dir := path.Join(user.HomeDir, ".sliver")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}

// GetDataDir - Returns the full path to the data directory
func GetDataDir() string {
	dir := GetRootAppDir() + "/data"
	return dir
}

// Setup - Extract or create local assets
// TODO: Add some type of version awareness
func Setup() {
	appDir := GetRootAppDir()
	SetupCerts(appDir)
	setupGo(appDir)
	setupCodenames(appDir)
	SetupDataPath(appDir)
}

// SetupCerts - Creates directories for certs
func SetupCerts(appDir string) {
	os.MkdirAll(path.Join(appDir, "certs"), os.ModePerm)
	rootDir := GetRootAppDir()
	certs.GenerateCertificateAuthority(rootDir, certs.SliversCertDir, true)
	certs.GenerateCertificateAuthority(rootDir, certs.ClientsCertDir, true)
}

// SetupGo - Unzip Go compiler assets
func setupGo(appDir string) error {

	log.Printf("Unpacking to '%s'", appDir)

	// Go compiler and stdlib
	goZip, err := assetsBox.Find(path.Join(runtime.GOOS, "go.zip"))
	if err != nil {
		log.Printf("static asset not found: go.zip")
		return err
	}

	goZipPath := path.Join(appDir, "go.zip")
	defer os.Remove(goZipPath)
	ioutil.WriteFile(goZipPath, goZip, 0644)
	_, err = unzip(goZipPath, appDir)
	if err != nil {
		log.Printf("Failed to unzip file %s -> %s", goZipPath, appDir)
		return err
	}

	goSrcZip, err := assetsBox.Find("src.zip")
	if err != nil {
		log.Printf("static asset not found: src.zip")
		return err
	}
	goSrcZipPath := path.Join(appDir, "src.zip")
	defer os.Remove(goSrcZipPath)
	ioutil.WriteFile(goSrcZipPath, goSrcZip, 0644)
	_, err = unzip(goSrcZipPath, path.Join(appDir, GoDirName))
	if err != nil {
		log.Printf("Failed to unzip file %s -> %s/go", goSrcZipPath, appDir)
		return err
	}

	return nil
}

// SetupGoPath - Extracts dependancies to goPathSrc
func SetupGoPath(goPathSrc string) error {

	// GOPATH setup
	if _, err := os.Stat(goPathSrc); os.IsNotExist(err) {
		log.Printf("Creating GOPATH directory: %s", goPathSrc)
		os.MkdirAll(goPathSrc, os.ModePerm)
	}

	// Protobuf dependencies
	pbGoSrc, err := protobufBox.Find("sliver/sliver.pb.go")
	if err != nil {
		log.Printf("static asset not found: sliver.pb.go")
		return err
	}
	pbConstSrc, err := protobufBox.Find("sliver/constants.go")
	if err != nil {
		log.Printf("static asset not found: constants.go")
		return err
	}

	protobufDir := path.Join(goPathSrc, "sliver", "protobuf", "sliver")
	os.MkdirAll(protobufDir, os.ModePerm)
	ioutil.WriteFile(path.Join(protobufDir, "constants.go"), pbGoSrc, 0644)

	ioutil.WriteFile(path.Join(protobufDir, "sliver.pb.go"), pbConstSrc, 0644)

	// GOPATH 3rd party dependencies
	protobufPath := path.Join(goPathSrc, "github.com", "golang")
	err = unzipGoDependency("protobuf.zip", protobufPath, assetsBox)
	if err != nil {
		log.Fatalf("Failed to unzip go dependency: %v", err)
	}

	return nil
}

// SetupDataPath - Sets the data directory up
func SetupDataPath(appDir string) error {
	dataDir := appDir + "/data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		log.Printf("Creating data directory: %s", dataDir)
		os.MkdirAll(dataDir, os.ModePerm)
	}
	hostingDll, err := assetsBox.Find("HostingCLRx64.dll")
	if err != nil {
		log.Printf("failed to find the dll")
		return err
	}
	err = ioutil.WriteFile(dataDir+"/HostingCLRx64.dll", hostingDll, 0644)
	return err
}

func unzipGoDependency(fileName string, targetPath string, assetsBox packr.Box) error {
	log.Printf("Unpacking go dependency %s -> %s", fileName, targetPath)
	appDir := GetRootAppDir()
	godep, err := assetsBox.Find(fileName)
	if err != nil {
		log.Printf("static asset not found: %s", fileName)
		return err
	}

	godepZipPath := path.Join(appDir, fileName)
	defer os.Remove(godepZipPath)
	ioutil.WriteFile(godepZipPath, godep, 0644)
	_, err = unzip(godepZipPath, targetPath)
	if err != nil {
		log.Printf("Failed to unzip file %s -> %s", godepZipPath, appDir)
		return err
	}

	return nil
}

func setupCodenames(appDir string) error {
	nouns, err := assetsBox.Find("nouns.txt")
	adjectives, err := assetsBox.Find("adjectives.txt")

	err = ioutil.WriteFile(path.Join(appDir, "nouns.txt"), nouns, 0600)
	if err != nil {
		log.Printf("Failed to write noun data to: %s", appDir)
		return err
	}

	err = ioutil.WriteFile(path.Join(appDir, "adjectives.txt"), adjectives, 0600)
	if err != nil {
		log.Printf("Failed to write adjective data to: %s", appDir)
		return err
	}
	return nil
}

func unzip(src string, dest string) ([]string, error) {

	var filenames []string

	reader, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer reader.Close()

	for _, file := range reader.File {

		rc, err := file.Open()
		if err != nil {
			return filenames, err
		}
		defer rc.Close()

		fpath := filepath.Join(dest, file.Name)
		filenames = append(filenames, fpath)

		if file.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
		} else {
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return filenames, err
			}
			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
			if err != nil {
				return filenames, err
			}
			_, err = io.Copy(outFile, rc)

			outFile.Close()

			if err != nil {
				return filenames, err
			}

		}
	}
	return filenames, nil
}
