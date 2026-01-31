package generate

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	wasmdonut "github.com/sliverarmory/wasm-donut"
)

const (
	defaultDonutEntropy  = wasmdonut.DonutEntropyNone
	defaultDonutCompress = wasmdonut.DonutCompressNone
	defaultDonutExitOpt  = wasmdonut.DonutExitThread
)

// DonutShellcodeFromFile returns a Donut shellcode for the given PE file
func DonutShellcodeFromFile(filePath string, arch string, dotnet bool, params string, className string, method string, donutConfig *clientpb.DonutConfig) (data []byte, err error) {
	pe, err := os.ReadFile(filePath)
	if err != nil {
		return
	}
	isDLL := (filepath.Ext(filePath) == ".dll")
	return DonutShellcodeFromPE(pe, arch, dotnet, params, className, method, isDLL, false, true, donutConfig)
}

// DonutShellcodeFromPE returns a Donut shellcode for the given PE file
func DonutShellcodeFromPE(pe []byte, arch string, dotnet bool, params string, className string, method string, isDLL bool, isUnicode bool, createNewThread bool, donutConfig *clientpb.DonutConfig) (data []byte, err error) {
	ext := ".exe"
	if isDLL {
		ext = ".dll"
	}
	// wasm-donut does not expose Unicode or thread flags; keep params for compatibility.
	_ = dotnet
	_ = isUnicode
	_ = createNewThread

	entropy, compress, exitOpt := normalizeDonutConfig(donutConfig)
	donutArch := getDonutArch(arch)

	opts := wasmdonut.GenerateOptions{
		Ext:      ext,
		Args:     params,
		Class:    className,
		Method:   method,
		Arch:     donutArch,
		Entropy:  entropy,
		Compress: compress,
		ExitOpt:  exitOpt,
	}

	shellcode, err := wasmdonut.Generate(context.Background(), pe, ext, opts)
	if err != nil {
		return nil, err
	}
	return addStackCheck(shellcode), nil
}

func normalizeDonutConfig(config *clientpb.DonutConfig) (int, int, int) {
	entropy := defaultDonutEntropy
	compress := defaultDonutCompress
	exitOpt := defaultDonutExitOpt
	if config == nil {
		return entropy, compress, exitOpt
	}
	if config.Entropy >= 1 && config.Entropy <= 3 {
		entropy = int(config.Entropy)
	}
	if config.Compress >= 1 && config.Compress <= 2 {
		compress = int(config.Compress)
	}
	if config.ExitOpt >= 1 && config.ExitOpt <= 3 {
		exitOpt = int(config.ExitOpt)
	}
	return entropy, compress, exitOpt
}

// DonutFromAssembly - Generate a donut shellcode from a .NET assembly
func DonutFromAssembly(assembly []byte, isDLL bool, arch string, params string, method string, className string, appDomain string) ([]byte, error) {
	ext := ".exe"
	if isDLL {
		ext = ".dll"
	}
	donutArch := getDonutArch(arch)
	_ = appDomain

	opts := wasmdonut.GenerateOptions{
		Ext:      ext,
		Args:     params,
		Class:    className,
		Method:   method,
		Arch:     donutArch,
		Entropy:  wasmdonut.DonutEntropyDefault,
		Compress: defaultDonutCompress,
		ExitOpt:  defaultDonutExitOpt,
	}
	shellcode, err := wasmdonut.Generate(context.Background(), assembly, ext, opts)
	if err != nil {
		return nil, err
	}
	return addStackCheck(shellcode), nil
}

func getDonutArch(arch string) int {
	donutArch := wasmdonut.DonutArchX84
	switch strings.ToLower(arch) {
	case "x32", "386":
		donutArch = wasmdonut.DonutArchX86
	case "x64", "amd64":
		donutArch = wasmdonut.DonutArchX64
	case "x84":
		donutArch = wasmdonut.DonutArchX84
	}
	return donutArch
}

func addStackCheck(shellcode []byte) []byte {
	stackCheckPrologue := []byte{
		// Check stack is 8 byte but not 16 byte aligned or else errors in LoadLibrary
		0x48, 0x83, 0xE4, 0xF0, // and rsp,0xfffffffffffffff0
		0x48, 0x83, 0xC4, 0x08, // add rsp,0x8
	}
	return append(stackCheckPrologue, shellcode...)
}
