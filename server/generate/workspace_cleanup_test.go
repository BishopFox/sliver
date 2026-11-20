package generate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/buildcore/workspace"
)

func TestSliverCoreAdapterConsumesDirectOutput(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SLIVER_ROOT_DIR", root)
	config := &clientpb.ImplantConfig{GOOS: "linux", GOARCH: "amd64", Format: clientpb.OutputFormat_SHELLCODE}
	for _, tc := range []struct {
		name        string
		writeOutput bool
	}{
		{name: "success", writeOutput: true},
		{name: "missing-output"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			work, err := workspace.New(root, config.GOOS, config.GOARCH, "sliver")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := work.WriteInput("sliver-material.json", []byte("secret")); err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(work.BinDir, "sliver")
			if tc.writeOutput {
				if err := os.WriteFile(output, []byte("artifact"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			data, err := ReadAndRemoveSliverBuildOutput("sliver", config, output)
			if tc.writeOutput && (err != nil || string(data) != "artifact") {
				t.Fatalf("consume output: data=%q err=%v", data, err)
			}
			if !tc.writeOutput && err == nil {
				t.Fatal("missing output was accepted")
			}
			if _, err := os.Stat(work.Root); !os.IsNotExist(err) {
				t.Fatalf("private workspace still exists: %v", err)
			}
		})
	}
}
