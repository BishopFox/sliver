package shellcode

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/Binject/shellcode/api"
)

// Repo - Shellcode Repository, directory-backed
type Repo struct {
	Directory string
}

//NewRepo - create a new Repo object, if dirName is provided, directory structure will be created if it doesn't exist
func NewRepo(dirName string) *Repo {
	repo := new(Repo)
	if dirName != "" {
		repo.Directory = dirName
		if err := initShellcodeDir(dirName); err != nil {
			log.Fatal(err)
		}
	}
	return repo
}

func initShellcodeDir(dirName string) error {

	if !DirExists(dirName) {
		if err := os.Mkdir(dirName, os.FileMode(int(0755))); err != nil {
			return err
		}
	}
	for _, ose := range api.Oses {
		osDir := filepath.Join(dirName, ose)
		if !DirExists(osDir) {
			if err := os.Mkdir(osDir, os.FileMode(int(0755))); err != nil {
				return err
			}
		}
		for _, arch := range api.Arches {
			archDir := filepath.Join(osDir, arch)
			if !DirExists(archDir) {
				if err := os.Mkdir(archDir, os.FileMode(int(0755))); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Lookup - fetches a completed shellcode from the filesystem
func (r *Repo) Lookup(os api.Os, arch api.Arch, pattern string) ([]byte, error) {
	if r.Directory != "" {
		// check specific directory first
		dir := filepath.Join(r.Directory, string(os), string(arch))
		if DirExists(dir) {
			files, err := WalkMatch(dir, pattern)
			if err != nil {
				return nil, err
			}
			if len(files) > 0 {
				return ioutil.ReadFile(files[0])
			}
		}
		// todo: fallback from intel32 or 64 to 32y64
	}
	//todo: lookup in the compiled-in modules
	return nil, errors.New("No Matching Shellcode Found")
}
