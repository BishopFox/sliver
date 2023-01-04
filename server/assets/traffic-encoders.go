//go:build server

package assets

import (
	"embed"
	"os"
	"path"
	"path/filepath"
)

var (
	//go:embed traffic-encoders/*.wasm
	defaultTrafficEncoders embed.FS
)

func setupTrafficEncoders(appDir string) error {
	localTrafficEncodersDir := filepath.Join(appDir, "traffic-encoders")
	if _, err := os.Stat(localTrafficEncodersDir); os.IsNotExist(err) {
		err = os.MkdirAll(localTrafficEncodersDir, 0700)
		if err != nil {
			return err
		}
	}

	encoders, err := defaultTrafficEncoders.ReadDir("traffic-encoders")
	if err != nil {
		return err
	}
	for _, encoder := range encoders {
		if encoder.IsDir() {
			continue
		}
		encoderName := encoder.Name()
		encoderBytes, err := defaultTrafficEncoders.ReadFile(path.Join("traffic-encoders", encoderName))
		if err != nil {
			return err
		}
		err = os.WriteFile(filepath.Join(localTrafficEncodersDir, filepath.Base(encoderName)), encoderBytes, 0600)
		if err != nil {
			return err
		}
	}
	return nil
}
