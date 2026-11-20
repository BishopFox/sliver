package buildtransform

import (
	"errors"
	"fmt"

	"github.com/bishopfox/sliver/server/buildplan"
)

// ValidateInstalledStep checks that a frozen transform step can run on the
// server before the core is compiled. It checks the installed version, digest,
// artifact edge, toolchain, and typed options without reading input bytes or
// resolving per-build material.
func ValidateInstalledStep(step buildplan.Step) error {
	if step.Kind != "transform" || step.Provider != "server" || step.Toolchain.Driver != "builtin" ||
		len(step.Toolchain.Requirements) != 0 {
		return errors.New("transform is not an installed local step")
	}
	switch step.Spec.Name {
	case "builtin/donut", "builtin/beignet", "builtin/malasada":
		if err := validateConverterStep(step); err != nil {
			return err
		}
		_, err := converterOptions(step)
		return err
	case "builtin/shikata-ga-nai", "builtin/xor", "builtin/xor-dynamic":
		return validateInstalledEncoderStep(step)
	case StageFrameSpec, StageCompressSpec, StageAESSpec, StageRC4Spec:
		if err := validateStageStep(step); err != nil {
			return err
		}
		return validateStageOptions(step)
	case PEMetadataSpec:
		return validatePEMetadataStep(step)
	default:
		return fmt.Errorf("transform %q is not installed on the server", step.Spec.Name)
	}
}

func validateInstalledEncoderStep(step buildplan.Step) error {
	binding, ok := encoderBindings[step.Spec.Name]
	if !ok || step.Spec.Version != "1.0.0" || step.Spec.Digest != binding.digest {
		return fmt.Errorf("unsupported installed encoder spec %q", step.Spec.Name)
	}
	if step.Input == nil || len(step.Options) != 0 ||
		!shellcodeArtifact(*step.Input) || !shellcodeArtifact(step.Output) ||
		step.Input.Target != step.Output.Target || !SupportsEncoder(step.Spec.Name, step.Output.Target) {
		return errors.New("encoder artifact or target is incompatible with installed capability")
	}
	return nil
}

func validateStageOptions(step buildplan.Step) error {
	switch step.Spec.Name {
	case StageFrameSpec:
		_, err := stageBoolOption(step.Options, "prependSize", false)
		return err
	case StageCompressSpec:
		algorithm, err := stageStringOption(step.Options, "algorithm", "zlib")
		if err != nil {
			return err
		}
		switch algorithm {
		case "zlib", "gzip", "deflate", "deflate9":
			return nil
		default:
			return fmt.Errorf("unsupported stage compression %q", algorithm)
		}
	case StageAESSpec:
		for _, name := range []string{"key", "iv"} {
			raw, exists := stageOption(step.Options, name)
			if !exists {
				return fmt.Errorf("missing stage secret reference %q", name)
			}
			if _, err := stageReference(raw, "secret"); err != nil {
				return fmt.Errorf("stage option %q: %w", name, err)
			}
		}
		return nil
	case StageRC4Spec:
		raw, exists := stageOption(step.Options, "key")
		if !exists {
			return errors.New("missing stage secret reference \"key\"")
		}
		_, err := stageReference(raw, "secret")
		return err
	default:
		return fmt.Errorf("unsupported stage transform %q", step.Spec.Name)
	}
}
