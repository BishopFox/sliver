// Package uid provides unique identifiers
package uid

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// Command creates a uid for given command.
func Command(cmd *cobra.Command) string {
	names := make([]string, 0)
	current := cmd
	for {
		names = append(names, current.Name())
		current = current.Parent()
		if current == nil {
			break
		}
	}

	reverse := make([]string, len(names))
	for i, entry := range names {
		reverse[len(names)-i-1] = entry
	}

	return "_" + strings.Join(reverse, "__")
}

// Executable returns the name of the executable.
func Executable() string {
	if executable, err := os.Executable(); err != nil {
		return "echo" // safe fallback that should never happen
	} else if filepath.Base(executable) == "cmd.test" {
		return "example" // for `go test -v ./...`
	} else {
		return filepath.Base(executable)
	}
}
