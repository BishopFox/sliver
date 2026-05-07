package beignet

import (
	"bytes"
	"debug/macho"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/sliverarmory/beignet/internal/aplib"
	"github.com/sliverarmory/beignet/internal/stager"
)

const (
	defaultEntrySymbol = "_StartW"
)

var (
	ErrUnsupportedArch = errors.New("beignet: unsupported architecture (supported: darwin/arm64, darwin/amd64)")
	ErrInvalidMachO    = errors.New("beignet: invalid Mach-O")
)

type Options struct {
	// EntrySymbol is the symbol name to resolve in the loaded module (e.g. "_StartW").
	// If provided without a leading underscore, one is added.
	EntrySymbol string

	// Compress enables aPLib "AP32" safe-packed compression for the staged dylib
	// buffer. This reduces shellcode size and requires the embedded loader.
	Compress bool
}

func DylibFileToShellcode(path string, opts Options) ([]byte, error) {
	payload, err := readMachOSupportedSlice(path)
	if err != nil {
		return nil, err
	}
	return DylibToShellcode(payload, opts)
}

func DylibToShellcode(dylib []byte, opts Options) ([]byte, error) {
	cpu, err := validateMachOSupported(dylib)
	if err != nil {
		return nil, err
	}
	entrySymbol, err := normalizeEntrySymbol(opts.EntrySymbol)
	if err != nil {
		return nil, err
	}

	payload := dylib
	if opts.Compress {
		packed, err := aplib.PackSafe(dylib)
		if err != nil {
			return nil, fmt.Errorf("beignet: aplib compress: %w", err)
		}
		payload = packed
	}

	loaderText, loaderEntryOff, err := stager.LoaderText(cpu)
	if err != nil {
		return nil, err
	}

	bootstrap, err := buildBootstrap(cpu, 0, 0, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("beignet: build bootstrap (%s): %w", cpu, err)
	}
	// The embedded loader is a (mostly) position-independent Mach-O image blob.
	// It expects to start at a page boundary so ADRP-based addressing remains valid.
	//
	// The shellcode buffer itself is mapped at a page boundary by the test runner,
	// so we just need to align the loader start offset.
	const loaderAlign = 0x1000
	loaderStart := alignUp(uint64(len(bootstrap)), loaderAlign)
	loaderPad := int(loaderStart - uint64(len(bootstrap)))
	loaderEntryAbs := loaderStart + loaderEntryOff

	payloadStart := alignUp(loaderStart+uint64(len(loaderText)), 16)
	payloadSize := uint64(len(payload))
	entrySymbolStart := payloadStart + payloadSize

	bootstrap, err = buildBootstrap(cpu, payloadStart, payloadSize, entrySymbolStart, loaderEntryAbs)
	if err != nil {
		return nil, fmt.Errorf("beignet: build bootstrap (%s): %w", cpu, err)
	}

	out := make([]byte, 0, int(entrySymbolStart)+len(entrySymbol)+1)
	out = append(out, bootstrap...)
	if loaderPad > 0 {
		out = append(out, make([]byte, loaderPad)...)
	}
	out = append(out, loaderText...)

	padLen := int(payloadStart - uint64(len(out)))
	if padLen < 0 {
		return nil, fmt.Errorf("beignet: internal alignment error (pad=%d)", padLen)
	}
	if padLen > 0 {
		out = append(out, make([]byte, padLen)...)
	}

	out = append(out, payload...)
	out = append(out, entrySymbol...)
	out = append(out, 0) // NUL-terminate symbol name

	return out, nil
}

func normalizeEntrySymbol(sym string) ([]byte, error) {
	sym = strings.TrimSpace(sym)
	if sym == "" {
		sym = defaultEntrySymbol
	}
	if strings.ContainsRune(sym, '\x00') {
		return nil, fmt.Errorf("beignet: entry symbol contains NUL")
	}
	if !strings.HasPrefix(sym, "_") {
		sym = "_" + sym
	}
	return []byte(sym), nil
}

func validateMachOSupported(b []byte) (macho.Cpu, error) {
	f, err := macho.NewFile(bytes.NewReader(b))
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidMachO, err)
	}
	_ = f.Close()

	if !isSupportedCPU(f.Cpu) {
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedArch, f.Cpu)
	}

	switch f.Type {
	case macho.TypeDylib, macho.TypeBundle:
		return f.Cpu, nil
	default:
		return 0, fmt.Errorf("%w: unexpected file type %v", ErrInvalidMachO, f.Type)
	}
}

func readMachOSupportedSlice(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ff, err := macho.NewFatFile(bytes.NewReader(b))
	if err == nil {
		defer ff.Close()
		for _, cpu := range preferredCPUOrder() {
			for _, arch := range ff.Arches {
				if arch.Cpu != cpu {
					continue
				}
				off := int(arch.Offset)
				size := int(arch.Size)
				if off < 0 || size < 0 || off+size > len(b) {
					return nil, fmt.Errorf("%w: invalid fat arch bounds", ErrInvalidMachO)
				}
				return b[off : off+size], nil
			}
		}
		return nil, fmt.Errorf("%w: no supported slice in fat Mach-O", ErrUnsupportedArch)
	}

	// Not a fat Mach-O; treat as thin.
	return b, nil
}

func buildBootstrap(cpu macho.Cpu, payloadOffset, payloadSize, symbolOffset, loaderEntryOffsetAbs uint64) ([]byte, error) {
	switch cpu {
	case macho.CpuArm64:
		return buildArm64Bootstrap(payloadOffset, payloadSize, symbolOffset, loaderEntryOffsetAbs)
	case macho.CpuAmd64:
		return buildAMD64Bootstrap(payloadOffset, payloadSize, symbolOffset, loaderEntryOffsetAbs)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedArch, cpu)
	}
}

func isSupportedCPU(cpu macho.Cpu) bool {
	switch cpu {
	case macho.CpuArm64, macho.CpuAmd64:
		return true
	default:
		return false
	}
}

func preferredCPUOrder() []macho.Cpu {
	order := []macho.Cpu{}

	switch runtime.GOARCH {
	case "arm64":
		order = append(order, macho.CpuArm64)
	case "amd64":
		order = append(order, macho.CpuAmd64)
	}

	// Keep a deterministic fallback order.
	for _, cpu := range []macho.Cpu{macho.CpuArm64, macho.CpuAmd64} {
		exists := false
		for _, existing := range order {
			if existing == cpu {
				exists = true
				break
			}
		}
		if !exists {
			order = append(order, cpu)
		}
	}
	return order
}

func alignUp(v uint64, align uint64) uint64 {
	if align == 0 {
		return v
	}
	mask := align - 1
	return (v + mask) &^ mask
}
