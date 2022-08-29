package extensions

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/util"
)

// RemoveExtensionByCommandName - Remove an extension by command name
func RemoveExtensionByCommandName(commandName string, con *console.SliverConsoleClient) error {
	if commandName == "" {
		return errors.New("command name is required")
	}
	if _, ok := loadedExtensions[commandName]; !ok {
		return errors.New("extension not loaded")
	}
	delete(loadedExtensions, commandName)
	con.App.Commands().Remove(commandName)
	extPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(commandName))
	if _, err := os.Stat(extPath); os.IsNotExist(err) {
		return nil
	}
	forceRemoveAll(extPath)
	return nil
}

func forceRemoveAll(rootPath string) {
	util.ChmodR(rootPath, 0o600, 0o700)
	os.RemoveAll(rootPath)
}
