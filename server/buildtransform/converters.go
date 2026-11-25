// Package buildtransform executes installed, typed artifact transforms.
package buildtransform

import (
	"bytes"
	"context"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/sliverarmory/beignet"
	"github.com/sliverarmory/malasada"
	wasmdonut "github.com/sliverarmory/wasm-donut"
)

const maxConverterBytes = 512 << 20

var converterDigests = map[string]string{
	"builtin/donut":    "sha256:eda20801b552f474a32bb5157fa0260261213286dd46f094fc8bae58547781c3",
	"builtin/beignet":  "sha256:0e07ef6cfc047b6b87760a1fcf316f8fc5ffc8d8613e7c613917f10ced4bf971",
	"builtin/malasada": "sha256:c021df62cb59723569038fd13a9b29eeaddb0ed6a244435d9adbc388954e8b1f",
}

// Convert applies one frozen shellcode converter step to accepted core bytes.
// workDir is the private build directory used for Malasada's file-based input;
// the temporary shared object is removed before this function returns.
//
//nolint:gocyclo // Keep each converter dispatch together with its input and output checks.
func Convert(ctx context.Context, step buildplan.Step, input []byte, workDir string) ([]byte, error) {
	if ctx == nil {
		return nil, errors.New("missing converter context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(input) == 0 || len(input) > maxConverterBytes {
		return nil, errors.New("converter input is empty or exceeds size limit")
	}
	if err := validateConverterStep(step); err != nil {
		return nil, err
	}
	options, err := converterOptions(step)
	if err != nil {
		return nil, err
	}
	var output []byte
	switch step.Spec.Name {
	case "builtin/donut":
		if err := validateConverterPE(input, step.Input.Target.Arch); err != nil {
			return nil, fmt.Errorf("invalid Donut input: %w", err)
		}
		arch := wasmdonut.DonutArchX86
		if step.Input.Target.Arch == "amd64" {
			arch = wasmdonut.DonutArchX64
		}
		compress := wasmdonut.DonutCompressNone
		if options.compress {
			compress = wasmdonut.DonutCompressAplib
		}
		result, err := wasmdonut.Generate(ctx, input, ".exe", wasmdonut.GenerateOptions{
			Ext: ".exe", Arch: arch, Entropy: int(options.entropy),
			Compress: compress, ExitOpt: int(options.exitOption),
			Bypass: int(options.bypass), Headers: int(options.headers),
			Thread: options.thread, Unicode: options.unicode,
			OEP: options.originalEntryPoint,
		})
		if err != nil {
			return nil, fmt.Errorf("convert with Donut: %w", err)
		}
		// Preserve the legacy Sliver loader's stack alignment prelude exactly.
		output = append([]byte{0x48, 0x83, 0xe4, 0xf0, 0x48, 0x83, 0xc4, 0x08}, result.Loader...)
	case "builtin/beignet":
		if err := validateConverterMachO(input); err != nil {
			return nil, fmt.Errorf("invalid Beignet input: %w", err)
		}
		output, err = beignet.DylibToShellcode(input, beignet.Options{
			EntrySymbol: step.Input.EntrySymbols[0], Compress: options.compress,
		})
		if err != nil {
			return nil, fmt.Errorf("convert with Beignet: %w", err)
		}
	case "builtin/malasada":
		if err := validateConverterELF(input, step.Input.Target.Arch); err != nil {
			return nil, fmt.Errorf("invalid Malasada input: %w", err)
		}
		output, err = convertMalasada(ctx, input, step.Input.EntrySymbols[0], options.compress, workDir)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("no installed converter for %q", step.Spec.Name)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(output) == 0 || len(output) > maxConverterBytes {
		return nil, errors.New("converter output is empty or exceeds size limit")
	}
	return output, nil
}

//nolint:gocyclo // The installed spec and artifact constraints are checked as one predicate for each converter.
func validateConverterStep(step buildplan.Step) error {
	if step.Spec.Version != "1.0.0" || step.Spec.Digest != converterDigests[step.Spec.Name] {
		return errors.New("converter step does not match an installed immutable spec")
	}
	if step.Kind != "transform" || step.Provider != "server" || step.Input == nil ||
		step.Output.Kind != "shellcode" || step.Output.Container != "raw" ||
		step.Output.GoBuildMode != "" || step.Input.Target != step.Output.Target ||
		step.Toolchain.Driver != "builtin" {
		return errors.New("converter step has incompatible input, output, or provider")
	}
	input := step.Input
	switch step.Spec.Name {
	case "builtin/donut":
		if input.Target.OS != "windows" || input.Target.Arch != "386" && input.Target.Arch != "amd64" ||
			input.Kind != "executable" || input.Container != "pe" || input.GoBuildMode != "pie" {
			return errors.New("converter Donut requires a Windows PIE executable for 386 or amd64")
		}
	case "builtin/beignet":
		if input.Target.OS != "darwin" || input.Target.Arch != "arm64" ||
			input.Kind != "shared-library" || input.Container != "macho" || input.GoBuildMode != "c-shared" ||
			!validConverterEntry(input.EntrySymbols) {
			return errors.New("converter Beignet requires a Darwin arm64 shared library and entry symbol")
		}
	case "builtin/malasada":
		if input.Target.OS != "linux" || input.Target.Arch != "amd64" && input.Target.Arch != "arm64" ||
			input.Kind != "shared-library" || input.Container != "elf" || input.GoBuildMode != "c-shared" ||
			!validConverterEntry(input.EntrySymbols) {
			return errors.New("converter Malasada requires a Linux amd64 or arm64 shared library and entry symbol")
		}
	default:
		return fmt.Errorf("no installed converter for %q", step.Spec.Name)
	}
	return nil
}

func validConverterEntry(entries []string) bool {
	return len(entries) > 0 && entries[0] != "" && strings.TrimSpace(entries[0]) == entries[0] &&
		!strings.ContainsRune(entries[0], '\x00') && len(entries[0]) <= 256
}

type parsedConverterOptions struct {
	compress           bool
	entropy            uint32
	exitOption         uint32
	bypass             uint32
	headers            uint32
	thread             bool
	unicode            bool
	originalEntryPoint uint32
}

//nolint:gocyclo // Keep the installed option types and bounds together with their defaults.
func converterOptions(step buildplan.Step) (parsedConverterOptions, error) {
	options := parsedConverterOptions{entropy: 1, exitOption: 1, bypass: 3, headers: 1}
	seen := make(map[string]bool, len(step.Options))
	for _, option := range step.Options {
		if seen[option.Name] {
			return options, fmt.Errorf("duplicate converter option %q", option.Name)
		}
		seen[option.Name] = true
		switch option.Name {
		case "compress", "thread", "unicode":
			if step.Spec.Name != "builtin/donut" && option.Name != "compress" {
				return options, fmt.Errorf("option %q is only valid for Donut", option.Name)
			}
			var value bool
			if err := json.Unmarshal(option.ValueJSON, &value); err != nil {
				return options, fmt.Errorf("invalid converter option %q: %w", option.Name, err)
			}
			switch option.Name {
			case "compress":
				options.compress = value
			case "thread":
				options.thread = value
			case "unicode":
				options.unicode = value
			}
		case "entropy", "exitOption", "bypass", "headers", "originalEntryPoint":
			if step.Spec.Name != "builtin/donut" {
				return options, fmt.Errorf("option %q is only valid for Donut", option.Name)
			}
			var value uint32
			if err := json.Unmarshal(option.ValueJSON, &value); err != nil {
				return options, fmt.Errorf("invalid converter option %q: %w", option.Name, err)
			}
			switch option.Name {
			case "entropy":
				if value < 1 || value > 3 {
					return options, errors.New("entropy for Donut is outside 1..3")
				}
				options.entropy = value
			case "exitOption":
				if value < 1 || value > 3 {
					return options, errors.New("exit option for Donut is outside 1..3")
				}
				options.exitOption = value
			case "bypass":
				if value < 1 || value > 3 {
					return options, errors.New("bypass for Donut is outside 1..3")
				}
				options.bypass = value
			case "headers":
				if value < 1 || value > 2 {
					return options, errors.New("headers for Donut is outside 1..2")
				}
				options.headers = value
			case "originalEntryPoint":
				options.originalEntryPoint = value
			}
		default:
			return options, fmt.Errorf("unknown converter option %q", option.Name)
		}
	}
	return options, nil
}

func validateConverterPE(input []byte, arch string) error {
	file, err := pe.NewFile(bytes.NewReader(input))
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	want := map[string]uint16{"386": 0x014c, "amd64": 0x8664}[arch]
	if want == 0 || file.Machine != want || file.Characteristics&0x2000 != 0 {
		return errors.New("PE machine or executable type does not match the frozen target")
	}
	if arch == "386" {
		if _, ok := file.OptionalHeader.(*pe.OptionalHeader32); !ok {
			return errors.New("PE optional header is not 32-bit")
		}
	} else if _, ok := file.OptionalHeader.(*pe.OptionalHeader64); !ok {
		return errors.New("PE optional header is not 64-bit")
	}
	return nil
}

func validateConverterMachO(input []byte) error {
	file, err := macho.NewFile(bytes.NewReader(input))
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	if file.Cpu != macho.CpuArm64 || file.Type != macho.TypeDylib {
		return errors.New("Mach-O target or shared-library type does not match the frozen step")
	}
	return nil
}

func validateConverterELF(input []byte, arch string) error {
	file, err := elf.NewFile(bytes.NewReader(input))
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	want := map[string]elf.Machine{"amd64": elf.EM_X86_64, "arm64": elf.EM_AARCH64}[arch]
	if want == 0 || file.Machine != want || file.Type != elf.ET_DYN {
		return errors.New("ELF target or shared-library type does not match the frozen step")
	}
	return nil
}

func convertMalasada(ctx context.Context, input []byte, entry string, compress bool, workDir string) ([]byte, error) {
	if !filepath.IsAbs(workDir) {
		return nil, errors.New("absolute private work directory required for Malasada")
	}
	info, err := os.Lstat(workDir)
	if err != nil || !info.IsDir() {
		return nil, errors.New("work directory for Malasada is unavailable")
	}
	file, err := os.CreateTemp(workDir, "malasada-*.so")
	if err != nil {
		return nil, fmt.Errorf("stage Malasada shared object: %w", err)
	}
	defer func() { _ = os.Remove(file.Name()) }()
	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		return nil, err
	}
	if _, err := file.Write(input); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("write Malasada shared object: %w", err)
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close Malasada shared object: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	output, err := malasada.ConvertSharedObject(file.Name(), entry, compress)
	if err != nil {
		return nil, fmt.Errorf("convert with Malasada: %w", err)
	}
	return output, nil
}
