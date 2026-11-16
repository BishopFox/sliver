package generate

/*
 * Sliver Implant Framework
 * Copyright (C) 2026  Bishop Fox
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

const (
	UPXLevelNormal     = 1
	UPXLevelBestLZMA   = 2
	UPXLevelUltraBrute = 3
)

func compressWithUPX(outputPath string, level uint32, config *clientpb.ImplantConfig, con *console.SliverClient) error {
	if config.Format == clientpb.OutputFormat_SHELLCODE {
		con.PrintWarnf("UPX compression is not applicable to shellcode output, skipping.\n")
		return nil
	}
	if config.Format == clientpb.OutputFormat_GO_ARCHIVE {
		con.PrintWarnf("UPX compression is not applicable to Go archive output, skipping.\n")
		return nil
	}

	upxPath, err := exec.LookPath("upx")
	if err != nil {
		con.PrintWarnf("UPX not found in PATH. Install it to use --upx:\n")
		con.PrintWarnf("  Debian/Ubuntu: sudo apt install upx-ucl\n")
		con.PrintWarnf("  Arch:          sudo pacman -S upx\n")
		con.PrintWarnf("  macOS:         brew install upx\n")
		return nil
	}

	originalInfo, err := os.Stat(outputPath)
	if err != nil {
		return fmt.Errorf("failed to stat output file: %w", err)
	}
	originalSize := originalInfo.Size()

	var upxArgs []string
	var levelName string
	switch level {
		case UPXLevelNormal:
			upxArgs = []string{"-q", outputPath}
			levelName = "normal"
		case UPXLevelBestLZMA:
			upxArgs = []string{"-q", "--best", "--lzma", outputPath}
			levelName = "best + LZMA (recommended)"
		case UPXLevelUltraBrute:
			upxArgs = []string{"-q", "--ultra-brute", outputPath}
			levelName = "ultra-brute"
		default:
			upxArgs = []string{"-q", "--best", "--lzma", outputPath}
			levelName = "best + LZMA (recommended)"
	}

	con.PrintWarnf("UPX may increase AV false-positive detection rates. Use with caution.\n")
	con.PrintInfof("Compressing with UPX (%s) ... ", levelName)

	cmd := exec.Command(upxPath, upxArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		con.Printf("failed\n")
		return fmt.Errorf("upx failed: %w: %s", err, string(output))
	}
	con.Printf("done\n")

	newInfo, err := os.Stat(outputPath)
	if err != nil {
		return fmt.Errorf("failed to stat compressed file: %w", err)
	}
	newSize := newInfo.Size()

	ratio := float64(newSize) / float64(originalSize) * 100
	con.PrintInfof("Size: %s → %s (%.1f%% of original, saved %s)\n",
		       humanSize(originalSize),
		       humanSize(newSize),
		       ratio,
		humanSize(originalSize-newSize),
	)

	return nil
}

func humanSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
	)
	switch {
		case bytes >= MB:
			return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
		case bytes >= KB:
			return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
		default:
			return fmt.Sprintf("%d B", bytes)
	}
}
