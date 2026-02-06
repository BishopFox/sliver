package beignet

import (
	"bytes"
	"debug/macho"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/sliverarmory/beignet/internal/aplib"
	"github.com/sliverarmory/beignet/internal/stager"
)

const (
	defaultEntrySymbol = "_StartW"
)

var (
	ErrUnsupportedArch = errors.New("beignet: only darwin/arm64 is supported")
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
	payload, err := readMachOArm64Slice(path)
	if err != nil {
		return nil, err
	}
	return DylibToShellcode(payload, opts)
}

func DylibToShellcode(dylib []byte, opts Options) ([]byte, error) {
	if err := validateMachOArm64(dylib); err != nil {
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

	loaderText, loaderEntryOff, err := stager.LoaderText()
	if err != nil {
		return nil, err
	}

	bootstrap, err := buildArm64Bootstrap(0, 0, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("beignet: assemble arm64 bootstrap: %w", err)
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

	bootstrap, err = buildArm64Bootstrap(payloadStart, payloadSize, entrySymbolStart, loaderEntryAbs)
	if err != nil {
		return nil, fmt.Errorf("beignet: assemble arm64 bootstrap: %w", err)
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

func validateMachOArm64(b []byte) error {
	f, err := macho.NewFile(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidMachO, err)
	}
	_ = f.Close()

	if f.Cpu != macho.CpuArm64 {
		return fmt.Errorf("%w: %s", ErrUnsupportedArch, f.Cpu)
	}

	switch f.Type {
	case macho.TypeDylib, macho.TypeBundle:
		return nil
	default:
		return fmt.Errorf("%w: unexpected file type %v", ErrInvalidMachO, f.Type)
	}
}

func readMachOArm64Slice(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ff, err := macho.NewFatFile(bytes.NewReader(b))
	if err == nil {
		defer ff.Close()
		for _, arch := range ff.Arches {
			if arch.Cpu != macho.CpuArm64 {
				continue
			}
			off := int(arch.Offset)
			size := int(arch.Size)
			if off < 0 || size < 0 || off+size > len(b) {
				return nil, fmt.Errorf("%w: invalid fat arch bounds", ErrInvalidMachO)
			}
			return b[off : off+size], nil
		}
		return nil, fmt.Errorf("%w: no arm64 slice in fat Mach-O", ErrUnsupportedArch)
	}

	// Not a fat Mach-O; treat as thin.
	return b, nil
}

func alignUp(v uint64, align uint64) uint64 {
	if align == 0 {
		return v
	}
	mask := align - 1
	return (v + mask) &^ mask
}
