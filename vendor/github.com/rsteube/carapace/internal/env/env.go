package env

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/rsteube/carapace/internal/common"
)

const (
	CARAPACE_COVERDIR      = "CARAPACE_COVERDIR"      // coverage directory for sandbox tests
	CARAPACE_HIDDEN        = "CARAPACE_HIDDEN"        // show hidden commands/flags
	CARAPACE_LENIENT       = "CARAPACE_LENIENT"       // allow unknown flags
	CARAPACE_LOG           = "CARAPACE_LOG"           // enable logging
	CARAPACE_MATCH         = "CARAPACE_MATCH"         // match case insensitive
	CARAPACE_SANDBOX       = "CARAPACE_SANDBOX"       // mock context for sandbox tests
	CARAPACE_ZSH_HASH_DIRS = "CARAPACE_ZSH_HASH_DIRS" // zsh hash directories
	CLICOLOR               = "CLICOLOR"               // disable color
	NO_COLOR               = "NO_COLOR"               // disable color
)

func ColorDisabled() bool {
	return os.Getenv(NO_COLOR) != "" || os.Getenv(CLICOLOR) == "0"
}

func Lenient() bool {
	return os.Getenv(CARAPACE_LENIENT) != ""
}

func Hashdirs() string {
	return os.Getenv(CARAPACE_ZSH_HASH_DIRS)
}

func Sandbox() (m *common.Mock, err error) {
	sandbox := os.Getenv(CARAPACE_SANDBOX)
	if sandbox == "" || !isGoRun() {
		return nil, errors.New("no sandbox")
	}

	err = json.Unmarshal([]byte(sandbox), &m)
	return
}

func Log() bool {
	return os.Getenv(CARAPACE_LOG) != ""
}

func Hidden() bool {
	return os.Getenv(CARAPACE_HIDDEN) != ""
}

func CoverDir() string {
	return os.Getenv(CARAPACE_COVERDIR) // custom env for GOCOVERDIR so that it works together with `-coverprofile`
}

func isGoRun() bool { return strings.HasPrefix(os.Args[0], os.TempDir()+"/go-build") }

func Match() string { // see match.Match
	return os.Getenv(CARAPACE_MATCH)
}
