package assets

import (
	"log"
	"os"
	"os/user"
	"path"
)

const (
	// SliverClientDirName - Directory storing all of the client configs/logs
	SliverClientDirName = ".sliver-client"
)

// GetRootAppDir - Get the Sliver app dir ~/.sliver-client/
func GetRootAppDir() string {
	user, _ := user.Current()
	dir := path.Join(user.HomeDir, SliverClientDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}
