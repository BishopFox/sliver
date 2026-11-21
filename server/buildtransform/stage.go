// Package buildtransform contains installed, typed transforms for local build
// steps. It receives frozen step metadata and bytes, never executor commands.
package buildtransform

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/rc4"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/util"
	utilEncoders "github.com/bishopfox/sliver/util/encoders"
)

const (
	// StageFrameSpec identifies the installed framing transform.
	StageFrameSpec = "builtin/stage-frame"
	// StageCompressSpec identifies the installed compression transform.
	StageCompressSpec = "builtin/stage-compress"
	// StageAESSpec identifies the installed AES transform.
	StageAESSpec = "builtin/stage-aes"
	// StageRC4Spec identifies the installed legacy RC4 transform.
	StageRC4Spec = "builtin/stage-rc4"
	// StageFrameDigest is the digest of the installed framing spec.
	StageFrameDigest = "sha256:a947be3966ef01bc1f287894efdf0a1ca2e08a7f8329eeae83c88a9b5bc5b8d3"
	// StageCompressDigest is the digest of the installed compression spec.
	StageCompressDigest = "sha256:d62eb2ca0688fa18696b9efcd02d5235bf5f482881ce858eb5a78dc6552210df"
	// StageAESDigest is the digest of the installed AES spec.
	StageAESDigest = "sha256:816231a80611e04ad7e6cad1553a703c35a7fb04fc0a82ca3a9709c7bf21bfec"
	// StageRC4Digest is the digest of the installed legacy RC4 spec.
	StageRC4Digest = "sha256:c573ed5628a478b3de79ef66f142d599c7ce3fa3fb4e2e7a2e6d02b095e40a39"
	maxStageBytes  = 512 << 20
)

// ResolveInput retrieves a secret or input artifact by its typed, per-build
// reference. The resolver owns authorization and must never put secret bytes in
// a plan, build record, log, or error string.
type ResolveInput func(kind, name string) ([]byte, error)

// ApplyStageStep executes one frozen stage transform. The supported order is
// frame, compression, then one encryption step; the planner freezes that order
// and this function validates the type of each individual edge.
//
//nolint:gocyclo // Keep transform dispatch and its validation together at the execution boundary.
func ApplyStageStep(ctx context.Context, step buildplan.Step, input []byte, resolve ResolveInput) ([]byte, error) {
	if ctx == nil {
		return nil, errors.New("missing stage transform context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateStageStep(step); err != nil {
		return nil, err
	}
	if len(input) == 0 || len(input) > maxStageBytes {
		return nil, errors.New("stage input is empty or exceeds size limit")
	}
	var (
		output []byte
		err    error
	)
	switch step.Spec.Name {
	case StageFrameSpec:
		prepend, optionErr := stageBoolOption(step.Options, "prependSize", false)
		if optionErr != nil {
			return nil, optionErr
		}
		output, err = frameStage(input, prepend)
	case StageCompressSpec:
		algorithm, optionErr := stageStringOption(step.Options, "algorithm", "zlib")
		if optionErr != nil {
			return nil, optionErr
		}
		output, err = compressStage(input, algorithm)
	case StageAESSpec:
		key, keyErr := stageSecret(step.Options, "key", resolve)
		if keyErr != nil {
			return nil, keyErr
		}
		defer clear(key)
		iv, ivErr := stageSecret(step.Options, "iv", resolve)
		if ivErr != nil {
			return nil, ivErr
		}
		defer clear(iv)
		output, err = encryptStageAES(input, key, iv)
	case StageRC4Spec:
		key, keyErr := stageSecret(step.Options, "key", resolve)
		if keyErr != nil {
			return nil, keyErr
		}
		defer clear(key)
		output, err = encryptStageRC4(input, key)
	default:
		return nil, fmt.Errorf("unsupported stage transform %q", step.Spec.Name)
	}
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(output) == 0 || len(output) > maxStageBytes {
		return nil, errors.New("stage output is empty or exceeds size limit")
	}
	return output, nil
}

//nolint:gocyclo // Each frozen edge is checked alongside its option schema.
func validateStageStep(step buildplan.Step) error {
	if step.Kind != "transform" || step.Provider != "server" || step.Input == nil ||
		step.Toolchain.Driver != "builtin" || len(step.Toolchain.Requirements) != 0 ||
		step.Input.Target != step.Output.Target || step.Output.Kind != "stage" || step.Output.Container != "raw" ||
		step.Output.GoBuildMode != "" || len(step.Output.EntrySymbols) != 0 {
		return errors.New("invalid local stage transform edge")
	}
	if err := stagePinnedReference(step.Spec); err != nil {
		return err
	}
	switch step.Spec.Name {
	case StageFrameSpec:
		if !stageFrameInputAllowed(*step.Input) {
			return errors.New("stage framing requires an installed binary or shellcode input")
		}
		if err := stageOnlyOptions(step.Options, "prependSize"); err != nil {
			return err
		}
	case StageCompressSpec:
		if !stageRawArtifact(*step.Input) {
			return errors.New("stage compression requires raw stage input")
		}
		if err := stageOnlyOptions(step.Options, "algorithm"); err != nil {
			return err
		}
	case StageAESSpec:
		if !stageRawArtifact(*step.Input) {
			return errors.New("stage AES requires raw stage input")
		}
		if err := stageOnlyOptions(step.Options, "key", "iv"); err != nil {
			return err
		}
	case StageRC4Spec:
		if !stageRawArtifact(*step.Input) {
			return errors.New("stage RC4 requires raw stage input")
		}
		if err := stageOnlyOptions(step.Options, "key"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported stage transform %q", step.Spec.Name)
	}
	return nil
}

//nolint:gocyclo // Target and artifact constraints form one compatibility matrix.
func stageFrameInputAllowed(input buildplan.Artifact) bool {
	switch input.Target.OS {
	case "windows":
		if input.Target.Arch != "386" && input.Target.Arch != "amd64" && input.Target.Arch != "arm64" {
			return false
		}
		if input.Kind == "shellcode" {
			return input.Target.Arch != "arm64" && shellcodeArtifact(input)
		}
		return stageFrameBinary(input, "pe", true)
	case "linux":
		if input.Target.Arch != "386" && input.Target.Arch != "amd64" && input.Target.Arch != "arm64" {
			return false
		}
		if input.Kind == "shellcode" {
			return input.Target.Arch != "386" && shellcodeArtifact(input)
		}
		return stageFrameBinary(input, "elf", false)
	case "darwin":
		if input.Target.Arch != "amd64" && input.Target.Arch != "arm64" {
			return false
		}
		if input.Kind == "shellcode" {
			return input.Target.Arch == "arm64" && shellcodeArtifact(input)
		}
		return stageFrameBinary(input, "macho", false)
	default:
		return false
	}
}

func stageFrameBinary(input buildplan.Artifact, container string, service bool) bool {
	switch input.Kind {
	case "executable":
		return input.Container == container && input.GoBuildMode == "default"
	case "service-executable":
		return service && input.Container == container && input.GoBuildMode == "default"
	case "shared-library":
		return input.Container == container && input.GoBuildMode == "c-shared"
	case "c-archive-bundle":
		return input.Container == "zip" && input.GoBuildMode == "c-archive"
	default:
		return false
	}
}

func stageRawArtifact(artifact buildplan.Artifact) bool {
	return artifact.Kind == "stage" && artifact.Container == "raw" && artifact.GoBuildMode == "" &&
		len(artifact.EntrySymbols) == 0 && supportedStageTarget(artifact.Target)
}

func supportedStageTarget(target buildplan.Target) bool {
	switch target.OS {
	case "windows", "linux":
		return target.Arch == "386" || target.Arch == "amd64" || target.Arch == "arm64"
	case "darwin":
		return target.Arch == "amd64" || target.Arch == "arm64"
	default:
		return false
	}
}

func stagePinnedReference(ref buildplan.Reference) error {
	if ref.Version != "1.0.0" {
		return errors.New("stage transform version differs from installed integration")
	}
	var digest string
	switch ref.Name {
	case StageFrameSpec:
		digest = StageFrameDigest
	case StageCompressSpec:
		digest = StageCompressDigest
	case StageAESSpec:
		digest = StageAESDigest
	case StageRC4Spec:
		digest = StageRC4Digest
	case PEMetadataSpec:
		digest = PEMetadataDigest
	default:
		return fmt.Errorf("uninstalled stage transform %q", ref.Name)
	}
	if ref.Digest != digest {
		return errors.New("stage transform digest differs from installed integration")
	}
	return nil
}

func stageOnlyOptions(options []buildplan.Option, allowed ...string) error {
	seen := make(map[string]bool, len(options))
	for _, option := range options {
		if seen[option.Name] {
			return fmt.Errorf("duplicate stage option %q", option.Name)
		}
		seen[option.Name] = true
		valid := false
		for _, name := range allowed {
			if option.Name == name {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("unexpected stage option %q", option.Name)
		}
	}
	return nil
}

func stageOption(options []buildplan.Option, name string) (json.RawMessage, bool) {
	for _, option := range options {
		if option.Name == name {
			return option.ValueJSON, true
		}
	}
	return nil, false
}

func stageBoolOption(options []buildplan.Option, name string, defaultValue bool) (bool, error) {
	raw, exists := stageOption(options, name)
	if !exists {
		return defaultValue, nil
	}
	var value bool
	trimmed := bytes.TrimSpace(raw)
	if err := json.Unmarshal(trimmed, &value); err != nil ||
		!bytes.Equal(trimmed, []byte("true")) && !bytes.Equal(trimmed, []byte("false")) {
		return false, fmt.Errorf("stage option %q must be boolean", name)
	}
	return value, nil
}

func stageStringOption(options []buildplan.Option, name, defaultValue string) (string, error) {
	raw, exists := stageOption(options, name)
	if !exists {
		return defaultValue, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || value == "" {
		return "", fmt.Errorf("stage option %q must be a nonempty string", name)
	}
	return value, nil
}

func stageSecret(options []buildplan.Option, name string, resolve ResolveInput) ([]byte, error) {
	raw, exists := stageOption(options, name)
	if !exists || resolve == nil {
		return nil, fmt.Errorf("stage secret %q is unavailable", name)
	}
	ref, err := stageReference(raw, "secret")
	if err != nil {
		return nil, fmt.Errorf("stage option %q: %w", name, err)
	}
	value, err := resolve(ref.Kind, ref.Name)
	if err != nil {
		return nil, fmt.Errorf("stage secret %q is unavailable", name)
	}
	if len(value) == 0 || len(value) > 256 {
		return nil, fmt.Errorf("stage secret %q has invalid length", name)
	}
	return append([]byte(nil), value...), nil
}

type stageInputReference struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

func stageReference(raw json.RawMessage, expectedKind string) (stageInputReference, error) {
	var ref stageInputReference
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&ref); err != nil {
		return ref, errors.New("invalid typed input reference")
	}
	if err := decoder.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		return ref, errors.New("typed input reference has trailing data")
	}
	if ref.Kind != expectedKind || ref.Name == "" || len(ref.Name) > 128 {
		return ref, errors.New("typed input reference has invalid kind or name")
	}
	return ref, nil
}

func frameStage(input []byte, prepend bool) ([]byte, error) {
	if !prepend {
		return append([]byte(nil), input...), nil
	}
	if uint64(len(input)) > math.MaxUint32 || len(input) > maxStageBytes-4 {
		return nil, errors.New("stage is too large for a four-byte size prefix")
	}
	output := make([]byte, len(input)+4)
	binary.LittleEndian.PutUint32(output[:4], uint32(len(input)))
	copy(output[4:], input)
	return output, nil
}

func compressStage(input []byte, algorithm string) ([]byte, error) {
	switch algorithm {
	case "zlib":
		var output bytes.Buffer
		writer := zlib.NewWriter(&output)
		if _, err := writer.Write(input); err != nil {
			_ = writer.Close()
			return nil, err
		}
		if err := writer.Close(); err != nil {
			return nil, err
		}
		return output.Bytes(), nil
	case "gzip":
		return utilEncoders.GzipBuf(input)
	case "deflate", "deflate9":
		return util.DeflateBuf(input), nil
	default:
		return nil, fmt.Errorf("unsupported stage compression %q", algorithm)
	}
}

func encryptStageAES(input, key, iv []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 32 {
		return nil, errors.New("stage AES key must be 16 or 32 bytes")
	}
	if len(iv) != 16 {
		return nil, errors.New("stage AES IV must be 16 bytes")
	}
	if len(input) > maxStageBytes-32 {
		return nil, errors.New("stage is too large to encrypt")
	}
	output := util.PreludeEncrypt(input, key, iv)
	if len(output) == 0 {
		return nil, errors.New("stage AES encryption failed")
	}
	return output, nil
}

func encryptStageRC4(input, key []byte) ([]byte, error) {
	if len(key) < 1 || len(key) > 256 {
		return nil, errors.New("stage RC4 key must be 1 through 256 bytes")
	}
	cipher, err := rc4.NewCipher(key)
	if err != nil {
		return nil, err
	}
	output := make([]byte, len(input))
	cipher.XORKeyStream(output, input)
	return output, nil
}
