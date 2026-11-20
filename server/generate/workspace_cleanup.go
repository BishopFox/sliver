package generate

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/buildcore/artifactaccept"
	"github.com/bishopfox/sliver/server/buildcore/workspace"
)

// ReadAndRemoveSliverBuildOutput consumes an output from a direct Sliver build
// and removes its private workspace, including the staged key material. Direct
// callers use this instead of retaining the compiler's output path.
func ReadAndRemoveSliverBuildOutput(name string, config *clientpb.ImplantConfig, output string) ([]byte, error) {
	if config == nil {
		return nil, errors.New("missing implant config")
	}
	data, readErr := func() ([]byte, error) {
		if err := writeSliverResult(output); err != nil {
			return nil, err
		}
		expected, err := sliverOutputArtifact(config)
		if err != nil {
			return nil, err
		}
		artifact, _, err := artifactaccept.Accept(filepath.Dir(output), expected)
		return artifact, err
	}()
	cleanupErr := workspace.RemoveOutputWorkspace(assets.GetRootAppDir(), config.GOOS, config.GOARCH, name, output)
	if cleanupErr != nil {
		cleanupErr = fmt.Errorf("remove private Sliver build workspace: %w", cleanupErr)
	}
	if err := errors.Join(readErr, cleanupErr); err != nil {
		return nil, err
	}
	return data, nil
}
