package assets

import (
	"fmt"
	"os"
	"path/filepath"
)

type goPlatform struct {
	os          string
	arch        string
	archiveExt  string
	includeSrc  bool
	toolRemoves []string
}

func (r *runner) buildGoAssets() error {
	r.logger.Logf("-----------------------------------------------------------------")
	r.logger.Logf(" Go")
	r.logger.Logf("-----------------------------------------------------------------")

	platforms := []goPlatform{
		{
			os:         "darwin",
			arch:       "amd64",
			archiveExt: "tar.gz",
			includeSrc: true,
			toolRemoves: []string{
				"pkg/tool/darwin_amd64/doc",
				"pkg/tool/darwin_amd64/tour",
				"pkg/tool/darwin_amd64/test2json",
			},
		},
		{
			os:         "darwin",
			arch:       "arm64",
			archiveExt: "tar.gz",
			includeSrc: true,
			toolRemoves: []string{
				"pkg/tool/darwin_arm64/doc",
				"pkg/tool/darwin_arm64/tour",
				"pkg/tool/darwin_arm64/test2json",
			},
		},
		{
			os:         "linux",
			arch:       "amd64",
			archiveExt: "tar.gz",
			toolRemoves: []string{
				"pkg/tool/linux_amd64/doc",
				"pkg/tool/linux_amd64/tour",
				"pkg/tool/linux_amd64/test2json",
			},
		},
		{
			os:         "linux",
			arch:       "arm64",
			archiveExt: "tar.gz",
			toolRemoves: []string{
				"pkg/tool/linux_arm64/doc",
				"pkg/tool/linux_arm64/tour",
				"pkg/tool/linux_arm64/test2json",
			},
		},
		{
			os:         "windows",
			arch:       "amd64",
			archiveExt: "zip",
			toolRemoves: []string{
				"pkg/tool/windows_amd64/doc.exe",
				"pkg/tool/windows_amd64/tour.exe",
				"pkg/tool/windows_amd64/test2json.exe",
			},
		},
	}

	for _, platform := range platforms {
		label := fmt.Sprintf("%s/%s", platform.os, platform.arch)
		r.goIndex++
		r.logger.Logf("Downloading Go %s (%d of %d) ...", label, r.goIndex, goTotal)

		archiveName := fmt.Sprintf("go%s.%s-%s.%s", goVersion, platform.os, platform.arch, platform.archiveExt)
		archiveURL := fmt.Sprintf("https://dl.google.com/go/%s", archiveName)
		archivePath := filepath.Join(r.workDir, archiveName)

		if err := r.downloadFile(archiveURL, archivePath); err != nil {
			return err
		}

		r.logger.Logf("Extracting Go %s ...", label)
		goDir := filepath.Join(r.workDir, "go")
		if err := os.RemoveAll(goDir); err != nil {
			return fmt.Errorf("remove previous go dir: %w", err)
		}

		if platform.archiveExt == "zip" {
			if err := extractZip(archivePath, r.workDir); err != nil {
				return err
			}
		} else {
			if err := extractTarGz(archivePath, r.workDir); err != nil {
				return err
			}
		}

		if err := removePaths(goDir, goBloatPaths); err != nil {
			return err
		}

		if platform.includeSrc {
			r.logger.Logf("Compressing src.zip (%s) ...", label)
			srcZipPath := filepath.Join(r.outputDir, "src.zip")
			if err := zipDir(goDir, "src", srcZipPath); err != nil {
				return err
			}
		}

		if err := os.RemoveAll(filepath.Join(goDir, "src")); err != nil {
			return fmt.Errorf("remove src dir: %w", err)
		}

		if err := removePaths(goDir, platform.toolRemoves); err != nil {
			return err
		}

		r.logger.Logf("Compressing Go %s ...", label)
		outputDir := filepath.Join(r.outputDir, platform.os, platform.arch)
		if err := ensureDir(outputDir); err != nil {
			return err
		}
		destZip := filepath.Join(outputDir, "go.zip")
		if err := zipDir(r.workDir, "go", destZip); err != nil {
			return err
		}

		if err := os.RemoveAll(goDir); err != nil {
			return fmt.Errorf("remove go dir: %w", err)
		}
		if err := os.Remove(archivePath); err != nil {
			return fmt.Errorf("remove go archive: %w", err)
		}
	}

	return nil
}

func removePaths(root string, paths []string) error {
	for _, path := range paths {
		path = trimLeadingDot(path)
		if path == "" {
			continue
		}
		full := filepath.Join(root, path)
		if err := os.RemoveAll(full); err != nil {
			return fmt.Errorf("remove %s: %w", full, err)
		}
	}
	return nil
}
