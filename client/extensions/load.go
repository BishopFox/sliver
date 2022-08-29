package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"github.com/desertbit/grumble"
)

// ParseExtensionManifest - Parse extension manifest from buffer
func ParseExtensionManifest(data []byte) (*ExtensionManifest, error) {
	extManifest := &ExtensionManifest{}
	err := json.Unmarshal(data, &extManifest)
	if err != nil {
		return nil, err
	}
	if extManifest.Name == "" {
		return nil, errors.New("missing `name` field in extension manifest")
	}
	if extManifest.CommandName == "" {
		return nil, errors.New("missing `command_name` field in extension manifest")
	}
	if len(extManifest.Files) == 0 {
		return nil, errors.New("missing `files` field in extension manifest")
	}
	for _, extFiles := range extManifest.Files {
		if extFiles.OS == "" {
			return nil, errors.New("missing `files.os` field in extension manifest")
		}
		if extFiles.Arch == "" {
			return nil, errors.New("missing `files.arch` field in extension manifest")
		}
		extFiles.Path = util.ResolvePath(extFiles.Path)
		if extFiles.Path == "" || extFiles.Path == "/" {
			return nil, errors.New("missing `files.path` field in extension manifest")
		}
		extFiles.OS = strings.ToLower(extFiles.OS)
		extFiles.Arch = strings.ToLower(extFiles.Arch)
	}
	if extManifest.Help == "" {
		return nil, errors.New("missing `help` field in extension manifest")
	}
	return extManifest, nil
}

// LoadExtensionManifest - Parse extension files
func LoadExtensionManifest(manifestPath string) (*ExtensionManifest, error) {
	data, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	extManifest, err := ParseExtensionManifest(data)
	if err != nil {
		return nil, err
	}
	extManifest.RootPath = filepath.Dir(manifestPath)
	loadedExtensions[extManifest.CommandName] = extManifest
	return extManifest, nil
}

func LoadExtension(goos string, goarch string, checkCache bool, ext *ExtensionManifest, ctx *grumble.Context, con *console.SliverConsoleClient) error {
	var extensionList []string
	binPath, err := ext.GetFileForTarget(ctx.Command.Name, goos, goarch)
	if err != nil {
		return err
	}

	// Try to find the extension in the loaded extensions
	if checkCache {
		extList, err := con.Rpc.ListExtensions(context.Background(), &sliverpb.ListExtensionsReq{
			Request: con.ActiveTarget.Request(ctx),
		})
		if err != nil {
			con.PrintErrorf("List extensions error: %s\n", err.Error())
			return err
		}
		if extList.Response != nil && extList.Response.Err != "" {
			return errors.New(extList.Response.Err)
		}
		extensionList = extList.Names
	}
	depLoaded := false
	for _, extName := range extensionList {
		if !depLoaded && extName == ext.DependsOn {
			depLoaded = true
		}
		if ext.CommandName == extName {
			return nil
		}
	}
	// Extension not found, let's load it
	if filepath.Ext(binPath) == ".o" {
		// BOFs are not loaded by the DLL loader, but we make sure the loader itself is loaded
		// Auto load the coff loader if we have it
		if !depLoaded {
			if errLoad := loadDep(goos, goarch, ext.DependsOn, ctx, con); errLoad != nil {
				return errLoad
			}
		}
		return nil
	}
	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		return err
	}
	if errRegister := registerExtension(goos, ext, binData, ctx, con); errRegister != nil {
		return errRegister
	}
	return nil
}

func registerExtension(goos string, ext *ExtensionManifest, binData []byte, ctx *grumble.Context, con *console.SliverConsoleClient) error {
	registerResp, err := con.Rpc.RegisterExtension(context.Background(), &sliverpb.RegisterExtensionReq{
		Name:    ext.CommandName,
		Data:    binData,
		OS:      goos,
		Init:    ext.Init,
		Request: con.ActiveTarget.Request(ctx),
	})
	if err != nil {
		return err
	}
	if registerResp.Response != nil && registerResp.Response.Err != "" {
		return errors.New(registerResp.Response.Err)
	}
	return nil
}

func loadDep(goos string, goarch string, depName string, ctx *grumble.Context, con *console.SliverConsoleClient) error {
	depExt, ok := loadedExtensions[depName]
	if ok {
		depBinPath, err := depExt.GetFileForTarget(depExt.CommandName, goos, goarch)
		if err != nil {
			return err
		}
		depBinData, err := ioutil.ReadFile(depBinPath)
		if err != nil {
			return err
		}
		return registerExtension(goos, depExt, depBinData, ctx, con)
	}
	return fmt.Errorf("missing dependency %s", depName)
}
