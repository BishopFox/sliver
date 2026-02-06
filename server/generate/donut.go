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
	defaultDonutBypass   = wasmdonut.DonutBypassContinue
	defaultDonutHeaders  = wasmdonut.DonutHeadersOverwrite
)

// DonutShellcodeFromFile returns a Donut shellcode for the given PE file
func DonutShellcodeFromFile(filePath string, arch string, dotnet bool, params string, className string, method string, shellcodeConfig *clientpb.ShellcodeConfig) (data []byte, err error) {
	pe, err := os.ReadFile(filePath)
	if err != nil {
		return
	}
	isDLL := (filepath.Ext(filePath) == ".dll")
	return DonutShellcodeFromPE(pe, arch, dotnet, params, className, method, isDLL, false, true, shellcodeConfig)
}

// DonutShellcodeFromPE returns a Donut shellcode for the given PE file
func DonutShellcodeFromPE(pe []byte, arch string, dotnet bool, params string, className string, method string, isDLL bool, isUnicode bool, createNewThread bool, shellcodeConfig *clientpb.ShellcodeConfig) (data []byte, err error) {
	ext := ".exe"
	if isDLL {
		ext = ".dll"
	}
	_ = dotnet

	donutOpts := normalizeDonutConfig(shellcodeConfig, createNewThread, isUnicode)
	donutArch := getDonutArch(arch)

	opts := wasmdonut.GenerateOptions{
		Ext:      ext,
		Args:     params,
		Class:    className,
		Method:   method,
		Arch:     donutArch,
		Bypass:   donutOpts.bypass,
		Headers:  donutOpts.headers,
		Entropy:  donutOpts.entropy,
		Compress: donutOpts.compress,
		ExitOpt:  donutOpts.exitOpt,
		Thread:   donutOpts.thread,
		Unicode:  donutOpts.unicode,
		OEP:      donutOpts.oep,
	}

	result, err := wasmdonut.Generate(context.Background(), pe, ext, opts)
	if err != nil {
		return nil, err
	}
	return addStackCheck(result.Loader), nil
}

type donutOptions struct {
	entropy  int
	compress int
	exitOpt  int
	bypass   int
	headers  int
	thread   bool
	unicode  bool
	oep      uint32
}

func normalizeDonutConfig(config *clientpb.ShellcodeConfig, fallbackThread bool, fallbackUnicode bool) donutOptions {
	opts := donutOptions{
		entropy:  defaultDonutEntropy,
		compress: defaultDonutCompress,
		exitOpt:  defaultDonutExitOpt,
		bypass:   defaultDonutBypass,
		headers:  defaultDonutHeaders,
		thread:   fallbackThread,
		unicode:  fallbackUnicode,
	}
	if config == nil {
		return opts
	}
	if config.Entropy >= 1 && config.Entropy <= 3 {
		opts.entropy = int(config.Entropy)
	}
	if config.Compress >= 1 && config.Compress <= 2 {
		opts.compress = int(config.Compress)
	}
	if config.ExitOpt >= 1 && config.ExitOpt <= 3 {
		opts.exitOpt = int(config.ExitOpt)
	}
	if config.Bypass >= 1 && config.Bypass <= 3 {
		opts.bypass = int(config.Bypass)
	}
	if config.Headers >= 1 && config.Headers <= 2 {
		opts.headers = int(config.Headers)
	}
	opts.thread = config.Thread
	opts.unicode = config.Unicode
	if config.OEP > 0 {
		opts.oep = config.OEP
	}
	return opts
}

// DonutFromAssembly - Generate a donut shellcode from a .NET assembly
func DonutFromAssembly(assembly []byte, isDLL bool, arch string, params string, method string, className string, appDomain string, runtime string) ([]byte, error) {
	ext := ".exe"
	if isDLL {
		ext = ".dll"
	}
	donutArch := getDonutArch(arch)

	opts := wasmdonut.GenerateOptions{
		Ext:      ext,
		Args:     params,
		Class:    className,
		Method:   method,
		Domain:   appDomain,
		Runtime:  runtime,
		Arch:     donutArch,
		Entropy:  wasmdonut.DonutEntropyDefault,
		Compress: defaultDonutCompress,
		ExitOpt:  defaultDonutExitOpt,
	}
	result, err := wasmdonut.Generate(context.Background(), assembly, ext, opts)
	if err != nil {
		return nil, err
	}
	return addStackCheck(result.Loader), nil
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
