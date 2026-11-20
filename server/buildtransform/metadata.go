package buildtransform

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/generate"
	"google.golang.org/protobuf/proto"
)

const (
	PEMetadataSpec         = "builtin/pe-metadata"
	PEMetadataDigest       = "sha256:b7205339fd8b8aa8ac4b14bc386dd4a1953b74a4b4444077a0a0dae2ea6c9bad"
	metadataDigestPrefix   = "metadata-sha256:"
	maxMetadataConfigBytes = 128 << 20
)

// ApplyPEMetadata clones supported metadata from a per-build donor into one
// PE artifact. The existing Sliver metadata mutator operates on a file, so the
// transform uses a private, short-lived scratch directory and returns only the
// resulting bytes to the step executor.
func ApplyPEMetadata(ctx context.Context, step buildplan.Step, input, donor []byte) ([]byte, error) {
	config := &generate.SpoofMetadataConfig{PE: &generate.PESpoofMetadataConfig{
		Source: &generate.SpoofMetadataFile{Data: donor},
	}}
	return applyPEMetadata(ctx, step, input, donor, config)
}

// ApplyPEMetadataConfig applies all supported legacy PE override fields as
// one typed build step. The ref name binds the frozen plan to the exact
// per-build config digest; the config and donor bytes themselves stay out of
// the plan and record.
func ApplyPEMetadataConfig(ctx context.Context, step buildplan.Step, input, donor, configBytes []byte) ([]byte, error) {
	if len(configBytes) == 0 || len(configBytes) > maxMetadataConfigBytes {
		return nil, errors.New("PE metadata configuration is empty or exceeds size limit")
	}
	ref, err := peMetadataDonorReference(step)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(ref.Name, metadataDigestPrefix) {
		return nil, errors.New("PE metadata configuration has no frozen digest reference")
	}
	hash := sha256.Sum256(configBytes)
	if ref.Name != metadataDigestPrefix+hex.EncodeToString(hash[:]) {
		return nil, errors.New("PE metadata configuration differs from frozen reference")
	}
	var wire clientpb.SpoofMetadataConfig
	if err := proto.Unmarshal(configBytes, &wire); err != nil {
		return nil, errors.New("invalid PE metadata configuration")
	}
	canonical, err := proto.MarshalOptions{Deterministic: true}.Marshal(&wire)
	if err != nil || !bytes.Equal(canonical, configBytes) {
		return nil, errors.New("PE metadata configuration has noncanonical encoding")
	}
	config, err := generate.SpoofMetadataConfigFromProto(&wire)
	if err != nil {
		return nil, fmt.Errorf("invalid PE metadata configuration: %w", err)
	}
	if config.PE == nil || config.PE.Source == nil || !bytes.Equal(config.PE.Source.Data, donor) {
		return nil, errors.New("PE metadata donor differs from configuration")
	}
	if config.PE.Icon != nil && len(config.PE.Icon.Data) > maxStageBytes {
		return nil, errors.New("PE metadata icon exceeds size limit")
	}
	return applyPEMetadata(ctx, step, input, donor, config)
}

func applyPEMetadata(ctx context.Context, step buildplan.Step, input, donor []byte,
	config *generate.SpoofMetadataConfig) (result []byte, resultErr error) {
	if ctx == nil {
		return nil, errors.New("missing PE metadata transform context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validatePEMetadataStep(step); err != nil {
		return nil, err
	}
	if len(input) == 0 || len(input) > maxStageBytes || len(donor) == 0 || len(donor) > maxStageBytes {
		return nil, errors.New("PE or donor input is empty or exceeds size limit")
	}
	wantMachine := map[string]uint16{"386": pe.IMAGE_FILE_MACHINE_I386, "amd64": pe.IMAGE_FILE_MACHINE_AMD64, "arm64": pe.IMAGE_FILE_MACHINE_ARM64}[step.Output.Target.Arch]
	if wantMachine == 0 {
		return nil, errors.New("unsupported PE metadata target architecture")
	}
	wantDLL := step.Output.Kind == "shared-library"
	for _, item := range []struct {
		label string
		data  []byte
	}{{"target", input}, {"donor", donor}} {
		label, data := item.label, item.data
		file, err := pe.NewFile(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("parse %s PE: %w", label, err)
		}
		machine := file.Machine
		dll := file.Characteristics&0x2000 != 0
		_ = file.Close()
		if machine != wantMachine {
			return nil, fmt.Errorf("%s PE architecture differs from frozen target", label)
		}
		if dll != wantDLL {
			return nil, fmt.Errorf("%s PE executable type differs from frozen artifact", label)
		}
	}
	scratch, err := os.MkdirTemp("", "sliver-pe-metadata-*")
	if err != nil {
		return nil, fmt.Errorf("create private metadata scratch: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(scratch); err != nil {
			result = nil
			resultErr = errors.Join(resultErr, fmt.Errorf("remove private metadata scratch: %w", err))
		}
	}()
	targetPath := filepath.Join(scratch, "target.exe")
	if err := os.WriteFile(targetPath, input, 0600); err != nil {
		return nil, fmt.Errorf("stage PE metadata input: %w", err)
	}
	if err := generate.SpoofMetadata(targetPath, config); err != nil {
		return nil, fmt.Errorf("apply PE metadata: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := os.Lstat(targetPath)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxStageBytes {
		return nil, errors.New("PE metadata output is missing, nonregular, or too large")
	}
	output, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, fmt.Errorf("read PE metadata output: %w", err)
	}
	if len(output) == 0 || len(output) > maxStageBytes {
		return nil, errors.New("PE metadata output exceeds size limit")
	}
	file, err := pe.NewFile(bytes.NewReader(output))
	if err != nil {
		return nil, fmt.Errorf("metadata output is not PE: %w", err)
	}
	machine := file.Machine
	dll := file.Characteristics&0x2000 != 0
	_ = file.Close()
	if machine != wantMachine || dll != wantDLL {
		return nil, errors.New("metadata output type or architecture differs from frozen target")
	}
	return output, nil
}

func peMetadataDonorReference(step buildplan.Step) (stageInputReference, error) {
	for _, option := range step.Options {
		if option.Name == "donorArtifact" {
			return stageReference(option.ValueJSON, "input-artifact")
		}
	}
	return stageInputReference{}, errors.New("PE metadata requires a donor artifact reference")
}

func validatePEMetadataStep(step buildplan.Step) error {
	if step.Kind != "transform" || step.Provider != "server" || step.Spec.Name != PEMetadataSpec ||
		step.Toolchain.Driver != "builtin" || len(step.Toolchain.Requirements) != 0 ||
		step.Input == nil || step.Input.Target.OS != "windows" || step.Output.Target.OS != "windows" ||
		step.Input.Container != "pe" || step.Output.Container != "pe" ||
		!sameMetadataArtifact(*step.Input, step.Output) {
		return errors.New("invalid PE metadata transform edge")
	}
	if err := stagePinnedReference(step.Spec); err != nil {
		return err
	}
	switch step.Input.Kind {
	case "executable", "service-executable":
		if step.Input.GoBuildMode != "default" {
			return errors.New("PE metadata executable requires default Go build mode")
		}
	case "shared-library":
		if step.Input.GoBuildMode != "c-shared" {
			return errors.New("PE metadata library requires c-shared Go build mode")
		}
	default:
		return errors.New("PE metadata requires an executable or shared library")
	}
	if !supportedStageTarget(step.Input.Target) {
		return errors.New("PE metadata target is not installed")
	}
	if err := stageOnlyOptions(step.Options, "donorArtifact"); err != nil {
		return err
	}
	raw, exists := stageOption(step.Options, "donorArtifact")
	if !exists {
		return errors.New("PE metadata requires a donor artifact reference")
	}
	if _, err := stageReference(raw, "input-artifact"); err != nil {
		return fmt.Errorf("PE metadata donor: %w", err)
	}
	return nil
}

func sameMetadataArtifact(left, right buildplan.Artifact) bool {
	return left.Kind == right.Kind && left.Container == right.Container && left.Target == right.Target &&
		left.GoBuildMode == right.GoBuildMode && slices.Equal(left.EntrySymbols, right.EntrySymbols)
}
