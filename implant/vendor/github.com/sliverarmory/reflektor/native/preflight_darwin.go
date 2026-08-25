//go:build darwin && !ios

package native

import (
	"bytes"
	"debug/buildinfo"
	"debug/macho"
)

func imageHasGoBuildInfo(data []byte) bool {
	if _, err := buildinfo.Read(bytes.NewReader(data)); err == nil {
		return true
	}

	fat, err := macho.NewFatFile(bytes.NewReader(data))
	if err == nil {
		defer fat.Close()
		for index := range fat.Arches {
			arch := &fat.Arches[index]
			if machOFileHasGoBuildInfo(arch.File) {
				return true
			}

			start := uint64(arch.Offset)
			end := start + uint64(arch.Size)
			if end < start || end > uint64(len(data)) {
				continue
			}
			if _, err := buildinfo.Read(bytes.NewReader(data[start:end])); err == nil {
				return true
			}
		}
		return false
	}

	thin, err := macho.NewFile(bytes.NewReader(data))
	if err != nil {
		return false
	}
	defer thin.Close()
	return machOFileHasGoBuildInfo(thin)
}

func machOFileHasGoBuildInfo(file *macho.File) bool {
	return file != nil && file.Section("__go_buildinfo") != nil
}
