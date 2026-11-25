package buildtransform

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/encoders/shellcode"
)

const maxEncoderInputBytes = 128 << 20

type encoderBinding struct {
	name   string
	digest string
}

var encoderBindings = map[string]encoderBinding{
	"builtin/shikata-ga-nai": {
		name: "shikata_ga_nai", digest: "sha256:630d2624378a7559a7c0fd1dbda0431f8ad3e61071e2756758ce12c93731739a",
	},
	"builtin/xor": {
		name: "xor", digest: "sha256:40caa733df52db78843b43abd479e5e818fd61257557f0bc0c67bd4307f72268",
	},
	"builtin/xor-dynamic": {
		name: "xor_dynamic", digest: "sha256:858e892a666672e46841117f0713f3e4917e660144ae669d50510096c9dd99ac",
	},
}

// SupportsEncoder reports whether the installed encoder can execute a
// catalog-declared shellcode transform for the target. The operating-system
// matrix matches the immutable built-in v1 transform specs.
func SupportsEncoder(specName string, target buildplan.Target) bool {
	binding, ok := encoderBindings[specName]
	if !ok {
		return false
	}
	if target.OS != "windows" && target.OS != "linux" && target.OS != "darwin" {
		return false
	}
	switch binding.name {
	case "shikata_ga_nai":
		if target.Arch == "386" {
			return target.OS == "windows" || target.OS == "linux"
		}
		return target.Arch == "amd64"
	case "xor", "xor_dynamic":
		return target.Arch == "amd64" || target.Arch == "arm64"
	default:
		return false
	}
}

// EncodeShellcode executes one typed, locally installed encoder step. The
// planner resolves its immutable spec and artifact edge before this call; the
// checks here prevent a direct caller from bypassing the installed matrix.
// Built-in v1 encoders use the legacy one-pass behavior with no bad chars.
//
//nolint:gocyclo // Keep the local executor checks and their output limits at the call boundary.
func EncodeShellcode(ctx context.Context, step buildplan.Step, input []byte) ([]byte, error) {
	if ctx == nil {
		return nil, errors.New("missing encoder context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	binding, ok := encoderBindings[step.Spec.Name]
	if !ok || step.Spec.Version != "1.0.0" || step.Spec.Digest != binding.digest {
		return nil, fmt.Errorf("unsupported installed encoder spec %q", step.Spec.Name)
	}
	if step.Kind != "transform" || step.Provider != "server" || step.Toolchain.Driver != "builtin" ||
		step.Input == nil || len(step.Options) != 0 || len(step.Toolchain.Requirements) != 0 {
		return nil, errors.New("invalid local encoder step")
	}
	if !shellcodeArtifact(*step.Input) || !shellcodeArtifact(step.Output) ||
		step.Input.Target != step.Output.Target || !SupportsEncoder(step.Spec.Name, step.Output.Target) {
		return nil, errors.New("encoder artifact or target is incompatible with installed capability")
	}
	if len(input) == 0 || len(input) > maxEncoderInputBytes {
		return nil, errors.New("encoder input is empty or exceeds size limit")
	}
	output, err := EncodeShellcodeCompat(ctx, binding.name, step.Output.Target.Arch, 1, nil, input)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(output) == 0 || len(output) > maxEncoderInputBytes {
		return nil, errors.New("encoder output is empty or exceeds size limit")
	}
	return output, nil
}

// EncodeShellcodeCompat is the existing encoder RPC's adapter to the same
// installed implementations used by typed build steps. The RPC supplies its
// established architecture normalization and default-selection behavior.
// Unlike v1 plan steps, compatibility calls may specify iterations/bad chars.
func EncodeShellcodeCompat(ctx context.Context, name, arch string, iterations int, badChars, input []byte) ([]byte, error) {
	if ctx == nil {
		return nil, errors.New("missing encoder context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	encoder := shellcode.ShellcodeEncoders[arch][name]
	if encoder == nil {
		return nil, fmt.Errorf("encoder %s is not installed for architecture %s", name, arch)
	}
	if iterations <= 0 {
		iterations = 1
	}
	output, err := encoder.Encode(bytes.Clone(input), shellcode.ShellcodeEncoderArgs{
		Iterations: iterations, BadChars: bytes.Clone(badChars),
	})
	if err != nil {
		return nil, fmt.Errorf("encode shellcode: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return output, nil
}

func shellcodeArtifact(artifact buildplan.Artifact) bool {
	return artifact.Kind == "shellcode" && artifact.Container == "raw" && artifact.GoBuildMode == "" &&
		len(artifact.EntrySymbols) == 0
}
