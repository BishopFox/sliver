package builder

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var vmPageSize = regexp.MustCompile(`page size of ([0-9]+) bytes`)

// vm_stat reports free, inactive, and speculative pages in the host kernel.
// This command is available on supported macOS builder hosts without CGO.
func availableMemoryBytes() uint64 {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "/usr/bin/vm_stat").Output()
	if err != nil {
		return 0
	}
	text := string(output)
	match := vmPageSize.FindStringSubmatch(text)
	if len(match) != 2 {
		return 0
	}
	pageSize, err := strconv.ParseUint(match[1], 10, 64)
	if err != nil || pageSize == 0 {
		return 0
	}
	var pages uint64
	for _, line := range strings.Split(text, "\n") {
		for _, prefix := range []string{"Pages free:", "Pages inactive:", "Pages speculative:"} {
			if !strings.HasPrefix(line, prefix) {
				continue
			}
			value := strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(line, prefix)), ".")
			count, err := strconv.ParseUint(value, 10, 64)
			if err != nil || count > ^uint64(0)-pages {
				return 0
			}
			pages += count
		}
	}
	if pages > ^uint64(0)/pageSize {
		return 0
	}
	return pages * pageSize
}
