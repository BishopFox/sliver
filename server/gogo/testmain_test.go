package gogo

import (
	"fmt"
	"os"
	"testing"

	"github.com/bishopfox/sliver/server/testutils"
)

func TestMain(m *testing.M) {
	envState, err := testutils.SetupTestEnv(testutils.EnvOptions{
		ForceFreshServerRoot: true,
		ForceFreshClientRoot: true,
		UnpackServerAssets:   true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup test env: %v\n", err)
		os.Exit(1)
	}

	exitCode := m.Run()
	if cleanupErr := envState.Cleanup(); cleanupErr != nil {
		fmt.Fprintf(os.Stderr, "%v\n", cleanupErr)
		if exitCode == 0 {
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}
