package assets

import (
	"fmt"
	"path/filepath"
)

type garblePlatform struct {
	os       string
	arch     string
	filename string
	url      string
}

func (r *runner) buildGarbleAssets() error {
	r.logger.Section("Garble")

	platforms := []garblePlatform{
		{
			os:       "linux",
			arch:     "amd64",
			filename: "garble",
			url:      fmt.Sprintf("https://github.com/moloch--/garble/releases/download/v%s/garble_linux-amd64", garbleVersion),
		},
		{
			os:       "linux",
			arch:     "arm64",
			filename: "garble",
			url:      fmt.Sprintf("https://github.com/moloch--/garble/releases/download/v%s/garble_linux-arm64", garbleVersion),
		},
		{
			os:       "windows",
			arch:     "amd64",
			filename: "garble.exe",
			url:      fmt.Sprintf("https://github.com/moloch--/garble/releases/download/v%s/garble_windows-amd64.exe", garbleVersion),
		},
		{
			os:       "windows",
			arch:     "arm64",
			filename: "garble.exe",
			url:      fmt.Sprintf("https://github.com/moloch--/garble/releases/download/v%s/garble_windows-arm64.exe", garbleVersion),
		},
		{
			os:       "darwin",
			arch:     "amd64",
			filename: "garble",
			url:      fmt.Sprintf("https://github.com/moloch--/garble/releases/download/v%s/garble_darwin-amd64", garbleVersion),
		},
		{
			os:       "darwin",
			arch:     "arm64",
			filename: "garble",
			url:      fmt.Sprintf("https://github.com/moloch--/garble/releases/download/v%s/garble_darwin-arm64", garbleVersion),
		},
	}

	for _, platform := range platforms {
		r.garbleIndex++
		r.logger.Logf("Fetch garble %s/%s (%d/%d)", platform.os, platform.arch, r.garbleIndex, garbleTotal)
		outputDir := filepath.Join(r.outputDir, platform.os, platform.arch)
		if err := ensureDir(outputDir); err != nil {
			return err
		}
		destPath := filepath.Join(outputDir, platform.filename)
		if err := r.downloadFile(platform.url, destPath); err != nil {
			return err
		}
	}

	return nil
}
