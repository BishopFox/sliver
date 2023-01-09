//go:build client

package assets

import (
	"embed"
	"io/fs"
	"os"

	"github.com/bishopfox/sliver/util/encoders"
)

var (
	//go:embed fs/english.txt fs/sliver.asc
	assetsFs embed.FS

	defaultTrafficEncoders embed.FS // Always empty for client
)

// Blank for client, no need for traffic encoders
func setupTrafficEncoders(appDir string) error {
	return nil
}

type EmptyEncoderFS struct{}

func (e EmptyEncoderFS) Open(name string) (fs.File, error) {
	return nil, os.ErrNotExist
}

func (e EmptyEncoderFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return []fs.DirEntry{}, nil
}

func (e EmptyEncoderFS) ReadFile(name string) ([]byte, error) {
	return nil, os.ErrNotExist
}

func loadTrafficEncoders(appDir string) encoders.EncoderFS {
	return EmptyEncoderFS{}
}

func setupGo(appDir string) error {
	return nil
}

func setupCodenames(appDir string) error {
	return nil
}
