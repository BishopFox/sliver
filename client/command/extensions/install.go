package extensions

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/desertbit/grumble"
)

// ExtensionsInstallCmd - Install an extension
func ExtensionsInstallCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	extLocalPath := ctx.Args.String("path")
	fi, err := os.Stat(extLocalPath)
	if os.IsNotExist(err) {
		con.PrintErrorf("Extension path '%s' does not exist", extLocalPath)
		return
	}
	if !fi.IsDir() {
		installFromFile(extLocalPath, con)
	} else {
		installFromDir(extLocalPath, con)
	}
}

// Install an extension from a directory
func installFromDir(extLocalPath string, con *console.SliverConsoleClient) {
	manifestData, err := ioutil.ReadFile(filepath.Join(extLocalPath, "manifest.json"))
	if err != nil {
		con.PrintErrorf("Error reading manifest.json: %s", err)
		return
	}
	manifest, err := parseExtensionManifest(string(manifestData))
	if err != nil {
		con.PrintErrorf("Error parsing manifest.json: %s", err)
		return
	}
	installPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(manifest.Name))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		con.PrintInfof("Extension '%s' already exists", manifest.Name)
		confirm := false
		prompt := &survey.Confirm{Message: "Overwrite current install?"}
		survey.AskOne(prompt, &confirm)
		if !confirm {
			return
		}
		os.RemoveAll(installPath)
	}

	con.Printf("Installing extension '%s' ... ", manifest.Name)
	err = os.MkdirAll(installPath, 0o700)
	if err != nil {
		con.PrintErrorf("\nError creating extension directory: %s\n", err)
		return
	}
	err = ioutil.WriteFile(filepath.Join(installPath, "manifest.json"), manifestData, 0o600)
	if err != nil {
		con.PrintErrorf("\nFailed to write manifest.json: %s\n", err)
		os.RemoveAll(installPath)
		return
	}

	for _, manifestFile := range manifest.Files {
		if manifestFile.Files.Ext32Path != "" {
			src := filepath.Join(extLocalPath, path.Clean("/"+manifestFile.Files.Ext32Path))
			dst := filepath.Join(installPath, path.Clean("/"+manifestFile.Files.Ext32Path))
			data, err := ioutil.ReadFile(src)
			if err != nil {
				con.PrintErrorf("\nError reading file '%s': %s\n", src, err)
				os.RemoveAll(installPath)
				return
			}
			err = ioutil.WriteFile(dst, data, 0o600)
			if err != nil {
				con.PrintErrorf("\nError writing file '%s': %s\n", dst, err)
				os.RemoveAll(installPath)
				return
			}
		}
		if manifestFile.Files.Ext64Path != "" {
			src := filepath.Join(extLocalPath, path.Clean("/"+manifestFile.Files.Ext64Path))
			dst := filepath.Join(installPath, path.Clean("/"+manifestFile.Files.Ext64Path))
			data, err := ioutil.ReadFile(src)
			if err != nil {
				con.PrintErrorf("\nError reading file '%s': %s\n", src, err)
				os.RemoveAll(installPath)
				return
			}
			err = ioutil.WriteFile(dst, data, 0o600)
			if err != nil {
				con.PrintErrorf("\nError writing file '%s': %s\n", dst, err)
				os.RemoveAll(installPath)
				return
			}
		}
	}

}

// Install an extension from a .tar.gz file
func installFromFile(extLocalPath string, con *console.SliverConsoleClient) {
	manifestData, err := readFileFromTarGz(extLocalPath, "./manifest.json")
	if err != nil {
		con.PrintErrorf("Failed to read manifest.json from '%s': %s\n", extLocalPath, err)
		return
	}
	manifest, err := parseExtensionManifest(string(manifestData))
	if err != nil {
		con.PrintErrorf("Failed to parse manifest.json: %s\n", err)
		return
	}
	installPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(manifest.Name))
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		con.PrintInfof("Extension '%s' already exists\n", manifest.Name)
		confirm := false
		prompt := &survey.Confirm{Message: "Overwrite current install?"}
		survey.AskOne(prompt, &confirm)
		if !confirm {
			return
		}
		os.RemoveAll(installPath)
	}

	con.Printf("Installing extension '%s' ... ", manifest.Name)
	err = os.MkdirAll(installPath, 0o700)
	if err != nil {
		con.PrintErrorf("\nFailed to create extension directory: %s\n", err)
		return
	}
	err = ioutil.WriteFile(filepath.Join(installPath, "manifest.json"), manifestData, 0o600)
	if err != nil {
		con.PrintErrorf("\nFailed to write manifest.json: %s\n", err)
		os.RemoveAll(installPath)
		return
	}
	for _, manifestFile := range manifest.Files {
		if manifestFile.Files.Ext32Path != "" {
			err = installArtifact(extLocalPath, installPath, manifestFile.Files.Ext32Path, con)
			if err != nil {
				con.PrintErrorf("\nFailed to install file: %s\n", err)
				os.RemoveAll(installPath)
				return
			}
		}
		if manifestFile.Files.Ext64Path != "" {
			err = installArtifact(extLocalPath, installPath, manifestFile.Files.Ext64Path, con)
			if err != nil {
				con.PrintErrorf("\nFailed to install file: %s\n", err)
				os.RemoveAll(installPath)
				return
			}
		}
	}
	con.Printf("done!\n")
}

func installArtifact(extLocalPath string, installPath string, artifactPath string, con *console.SliverConsoleClient) error {
	data, err := readFileFromTarGz(extLocalPath, artifactPath)
	if err != nil {
		return err
	}
	localArtifactPath := filepath.Join(installPath, path.Clean("/"+artifactPath))
	artifactDir := filepath.Dir(localArtifactPath)
	if _, err := os.Stat(artifactDir); os.IsNotExist(err) {
		os.MkdirAll(artifactDir, 0o700)
	}
	err = ioutil.WriteFile(localArtifactPath, data, 0o600)
	if err != nil {
		return err
	}
	return nil
}

func readFileFromTarGz(tarGzFile string, tarPath string) ([]byte, error) {
	f, err := os.Open(tarGzFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gzf, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gzf.Close()

	tarReader := tar.NewReader(gzf)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Name == tarPath {
			switch header.Typeflag {
			case tar.TypeDir: // = directory
				continue
			case tar.TypeReg: // = regular file
				return ioutil.ReadAll(tarReader)
			}
		}
	}
	return nil, nil
}
