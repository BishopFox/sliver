package e2e

import (
	"testing"

	e2ecoverage "github.com/bishopfox/sliver/test/e2e/coverage"
	"github.com/bishopfox/sliver/test/e2e/shellcodecoverage"
)

func TestShellcodeE2E(t *testing.T) {
	if testOptions.serverPath == "" {
		t.Skip("shellcode E2E requires -server")
	}

	suite, err := newSuite(t, testOptions, false)
	if err != nil {
		t.Fatal(err)
	}
	defer suite.close()

	recorder, err := shellcodecoverage.NewRecorder(e2ecoverage.Target{
		OS:   suite.opts.targetOS,
		Arch: suite.opts.targetArch,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := recorder.Write(suite.opts.resultsDir); err != nil {
			t.Errorf("write shellcode E2E coverage: %v", err)
		}
	}()

	if err := suite.runShellcode(recorder); err != nil {
		t.Fatal(err)
	}
}
