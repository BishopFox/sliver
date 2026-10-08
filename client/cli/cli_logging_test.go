package cli

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/client/assets"
)

func TestPackageLoggerRemainsOpenAfterInit(t *testing.T) {
	logFile, ok := log.Writer().(*os.File)
	if !ok {
		t.Fatalf("standard logger writer = %T, want *os.File", log.Writer())
	}
	if want := filepath.Join(assets.GetRootAppDir(), logFileName); filepath.Clean(logFile.Name()) != want {
		t.Fatalf("standard logger file = %q, want %q", logFile.Name(), want)
	}
	if _, err := logFile.Stat(); err != nil {
		t.Fatalf("standard logger file is closed after package initialization: %v", err)
	}
}
