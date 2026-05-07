package wasmdonut

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/emscripten"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const (
	DonutArchX86 = 1
	DonutArchX64 = 2
	DonutArchX84 = 3

	DonutEntropyNone    = 1
	DonutEntropyRandom  = 2
	DonutEntropyDefault = 3

	DonutCompressNone          = 1
	DonutCompressAplib         = 2
	DonutCompressLZNT1         = 3
	DonutCompressXpress        = 4
	DonutCompressXpressHuffman = 5

	DonutExitThread  = 1
	DonutExitProcess = 2
	DonutExitBlock   = 3

	DonutBypassNone     = 1
	DonutBypassAbort    = 2
	DonutBypassContinue = 3

	DonutHeadersOverwrite = 1
	DonutHeadersKeep      = 2

	DonutFormatBinary     = 1
	DonutFormatBase64     = 2
	DonutFormatC          = 3
	DonutFormatRuby       = 4
	DonutFormatPython     = 5
	DonutFormatPowershell = 6
	DonutFormatCSharp     = 7
	DonutFormatHex        = 8
	DonutFormatUUID       = 9
)

//go:embed donut/wasm/donut.wasm
var embeddedWasm []byte

// GenerateOptions configures Donut shellcode generation.
type GenerateOptions struct {
	Ext      string
	Args     string
	Class    string
	Method   string
	Domain   string
	Runtime  string
	Decoy    string
	Server   string
	ModName  string
	Arch     int
	Bypass   int
	Headers  int
	Entropy  int
	Compress int
	ExitOpt  int
	Thread   bool
	Unicode  bool
	OEP      uint32
	Format   int
}

// GenerateResult contains the output of a wasm Donut generation.
type GenerateResult struct {
	Loader     []byte
	Module     []byte
	ModuleName string
}

// Generate runs the embedded Donut wasm module and returns shellcode bytes.
func Generate(ctx context.Context, input []byte, ext string, opts GenerateOptions) (GenerateResult, error) {
	var result GenerateResult
	if len(input) == 0 {
		return result, errors.New("input is empty")
	}
	if ext == "" {
		return result, errors.New("extension is required")
	}
	if len(embeddedWasm) == 0 {
		return result, errors.New("embedded wasm is empty; run donut/wasm/build.sh")
	}

	arch := opts.Arch
	if arch == 0 {
		arch = DonutArchX84
	}
	bypass := opts.Bypass
	if bypass == 0 {
		bypass = DonutBypassContinue
	}
	headers := opts.Headers
	if headers == 0 {
		headers = DonutHeadersOverwrite
	}
	entropy := opts.Entropy
	if entropy == 0 {
		entropy = DonutEntropyDefault
	}
	compress := opts.Compress
	if compress == 0 {
		compress = DonutCompressNone
	}
	exitOpt := opts.ExitOpt
	if exitOpt == 0 {
		exitOpt = DonutExitThread
	}

	thread := uint32(0)
	if opts.Thread {
		thread = 1
	}
	unicode := uint32(0)
	if opts.Unicode {
		unicode = 1
	}

	runtime := wazero.NewRuntime(ctx)
	defer func() {
		_ = runtime.Close(ctx)
	}()

	wasi_snapshot_preview1.MustInstantiate(ctx, runtime)

	compiled, err := runtime.CompileModule(ctx, embeddedWasm)
	if err != nil {
		return result, fmt.Errorf("compile wasm: %w", err)
	}

	exporter, err := emscripten.NewFunctionExporterForModule(compiled)
	if err != nil {
		return result, fmt.Errorf("create emscripten exporter: %w", err)
	}

	envBuilder := runtime.NewHostModuleBuilder("env")
	envBuilder.NewFunctionBuilder().WithFunc(func(ctx context.Context, mod api.Module, ptr, length uint32) uint32 {
		if length == 0 {
			return 1
		}
		buf := make([]byte, length)
		if _, err := rand.Read(buf); err != nil {
			return 0
		}
		if ok := mod.Memory().Write(ptr, buf); !ok {
			return 0
		}
		return 1
	}).Export("donut_random")
	exporter.ExportFunctions(envBuilder)

	envCloser, err := envBuilder.Instantiate(ctx)
	if err != nil {
		return result, fmt.Errorf("instantiate env: %w", err)
	}
	defer func() {
		_ = envCloser.Close(ctx)
	}()

	module, err := runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	if err != nil {
		return result, fmt.Errorf("instantiate wasm: %w", err)
	}
	defer func() {
		_ = module.Close(ctx)
	}()

	mallocFn, err := exportedFunc(module, "malloc", "_malloc")
	if err != nil {
		return result, fmt.Errorf("malloc export: %w", err)
	}
	freeFn, err := exportedFunc(module, "free", "_free")
	if err != nil {
		return result, fmt.Errorf("free export: %w", err)
	}
	genFn, err := exportedFunc(module, "donut_generate", "_donut_generate")
	if err != nil {
		return result, fmt.Errorf("donut_generate export: %w", err)
	}

	mem := module.Memory()
	if mem == nil {
		return result, errors.New("module has no exported memory")
	}

	inPtr, err := allocBytes(ctx, mallocFn, mem, input)
	if err != nil {
		return result, err
	}
	defer callFree(ctx, freeFn, inPtr)

	extPtr, err := allocCString(ctx, mallocFn, mem, ext)
	if err != nil {
		return result, err
	}
	defer callFree(ctx, freeFn, extPtr)

	argsPtr := uint32(0)
	if opts.Args != "" {
		argsPtr, err = allocCString(ctx, mallocFn, mem, opts.Args)
		if err != nil {
			return result, err
		}
		defer callFree(ctx, freeFn, argsPtr)
	}

	clsPtr := uint32(0)
	if opts.Class != "" {
		clsPtr, err = allocCString(ctx, mallocFn, mem, opts.Class)
		if err != nil {
			return result, err
		}
		defer callFree(ctx, freeFn, clsPtr)
	}

	methodPtr := uint32(0)
	if opts.Method != "" {
		methodPtr, err = allocCString(ctx, mallocFn, mem, opts.Method)
		if err != nil {
			return result, err
		}
		defer callFree(ctx, freeFn, methodPtr)
	}

	domainPtr := uint32(0)
	if opts.Domain != "" {
		domainPtr, err = allocCString(ctx, mallocFn, mem, opts.Domain)
		if err != nil {
			return result, err
		}
		defer callFree(ctx, freeFn, domainPtr)
	}

	runtimePtr := uint32(0)
	if opts.Runtime != "" {
		runtimePtr, err = allocCString(ctx, mallocFn, mem, opts.Runtime)
		if err != nil {
			return result, err
		}
		defer callFree(ctx, freeFn, runtimePtr)
	}

	decoyPtr := uint32(0)
	if opts.Decoy != "" {
		decoyPtr, err = allocCString(ctx, mallocFn, mem, opts.Decoy)
		if err != nil {
			return result, err
		}
		defer callFree(ctx, freeFn, decoyPtr)
	}

	serverPtr := uint32(0)
	if opts.Server != "" {
		serverPtr, err = allocCString(ctx, mallocFn, mem, opts.Server)
		if err != nil {
			return result, err
		}
		defer callFree(ctx, freeFn, serverPtr)
	}

	modnamePtr := uint32(0)
	if opts.ModName != "" {
		modnamePtr, err = allocCString(ctx, mallocFn, mem, opts.ModName)
		if err != nil {
			return result, err
		}
		defer callFree(ctx, freeFn, modnamePtr)
	}

	const resultSize = 28
	resultPtr, err := allocBytes(ctx, mallocFn, mem, make([]byte, resultSize))
	if err != nil {
		return result, err
	}
	defer callFree(ctx, freeFn, resultPtr)

	callArgs := []uint64{
		uint64(inPtr),
		uint64(len(input)),
		uint64(extPtr),
		uint64(argsPtr),
		uint64(clsPtr),
		uint64(methodPtr),
		uint64(domainPtr),
		uint64(runtimePtr),
		uint64(decoyPtr),
		uint64(serverPtr),
		uint64(modnamePtr),
		uint64(arch),
		uint64(bypass),
		uint64(headers),
		uint64(entropy),
		uint64(compress),
		uint64(exitOpt),
		uint64(thread),
		uint64(unicode),
		uint64(opts.OEP),
		uint64(resultPtr),
	}

	res, err := genFn.Call(ctx, callArgs...)
	if err != nil {
		return result, fmt.Errorf("donut_generate call: %w", err)
	}
	if len(res) > 0 && res[0] != 0 {
		return result, fmt.Errorf("donut_generate returned error code: %d", res[0])
	}

	resultBytes, ok := mem.Read(resultPtr, resultSize)
	if !ok {
		return result, errors.New("read result struct")
	}

	loaderPtr := binary.LittleEndian.Uint32(resultBytes[0:4])
	loaderLen := binary.LittleEndian.Uint32(resultBytes[4:8])
	modulePtr := binary.LittleEndian.Uint32(resultBytes[8:12])
	moduleLen := binary.LittleEndian.Uint32(resultBytes[12:16])
	modnamePtrOut := binary.LittleEndian.Uint32(resultBytes[16:20])
	modnameLen := binary.LittleEndian.Uint32(resultBytes[20:24])
	outErr := int32(binary.LittleEndian.Uint32(resultBytes[24:28]))

	if outErr != 0 {
		return result, fmt.Errorf("donut_generate failed with error code: %d", outErr)
	}
	if loaderPtr == 0 || loaderLen == 0 {
		return result, errors.New("donut_generate returned empty output")
	}

	loaderBytes, ok := mem.Read(loaderPtr, loaderLen)
	if !ok {
		return result, errors.New("read output bytes")
	}
	result.Loader = append([]byte(nil), loaderBytes...)
	callFree(ctx, freeFn, loaderPtr)

	if modulePtr != 0 && moduleLen != 0 {
		moduleBytes, ok := mem.Read(modulePtr, moduleLen)
		if !ok {
			return result, errors.New("read module bytes")
		}
		result.Module = append([]byte(nil), moduleBytes...)
		callFree(ctx, freeFn, modulePtr)
	}

	if modnamePtrOut != 0 && modnameLen != 0 {
		nameBytes, ok := mem.Read(modnamePtrOut, modnameLen)
		if !ok {
			return result, errors.New("read modname bytes")
		}
		result.ModuleName = string(nameBytes)
		callFree(ctx, freeFn, modnamePtrOut)
	}

	return result, nil
}

// GenerateFromFile loads input from disk and returns generated shellcode.
func GenerateFromFile(ctx context.Context, path string, opts GenerateOptions) (GenerateResult, error) {
	if path == "" {
		return GenerateResult{}, errors.New("input path is required")
	}

	input, err := os.ReadFile(path)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("read input: %w", err)
	}

	ext := opts.Ext
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(path))
		if ext == "" {
			return GenerateResult{}, errors.New("could not infer extension; pass Ext in options")
		}
	}

	return Generate(ctx, input, ext, opts)
}

// GenerateToFile writes shellcode generated from the input path to outPath.
func GenerateToFile(ctx context.Context, inPath, outPath string, opts GenerateOptions) error {
	if inPath == "" {
		return errors.New("input path is required")
	}

	result, err := GenerateFromFile(ctx, inPath, opts)
	if err != nil {
		return err
	}

	format := opts.Format
	if format == 0 {
		format = DonutFormatBinary
	}

	if outPath == "" {
		outPath = defaultOutputPath(format)
	}

	outputBytes, err := formatOutput(result.Loader, format)
	if err != nil {
		return err
	}

	if err := os.WriteFile(outPath, outputBytes, 0644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	if opts.Server != "" {
		if len(result.Module) == 0 {
			return errors.New("module bytes missing for HTTP staging")
		}
		modName := result.ModuleName
		if modName == "" {
			modName = opts.ModName
		}
		if modName == "" {
			return errors.New("module name missing for HTTP staging")
		}
		if err := os.WriteFile(modName, result.Module, 0644); err != nil {
			return fmt.Errorf("write module: %w", err)
		}
	}

	return nil
}

func defaultOutputPath(format int) string {
	switch format {
	case DonutFormatBinary:
		return "loader.bin"
	case DonutFormatBase64:
		return "loader.b64"
	case DonutFormatRuby:
		return "loader.rb"
	case DonutFormatC:
		return "loader.c"
	case DonutFormatPython:
		return "loader.py"
	case DonutFormatPowershell:
		return "loader.ps1"
	case DonutFormatCSharp:
		return "loader.cs"
	case DonutFormatHex:
		return "loader.hex"
	case DonutFormatUUID:
		return "loader.uuid"
	default:
		return "loader.bin"
	}
}

func formatOutput(data []byte, format int) ([]byte, error) {
	switch format {
	case DonutFormatBinary:
		return data, nil
	case DonutFormatBase64:
		return []byte(base64.StdEncoding.EncodeToString(data)), nil
	case DonutFormatC, DonutFormatRuby:
		return []byte(formatCRuby(data)), nil
	case DonutFormatPython:
		return []byte(formatPython(data)), nil
	case DonutFormatPowershell:
		return []byte(formatPowerShell(data)), nil
	case DonutFormatCSharp:
		return []byte(formatCSharp(data)), nil
	case DonutFormatHex:
		return []byte(formatHex(data)), nil
	case DonutFormatUUID:
		return []byte(formatUUID(data)), nil
	default:
		return nil, fmt.Errorf("invalid output format: %d", format)
	}
}

func formatCRuby(data []byte) string {
	var b strings.Builder
	b.WriteString("unsigned char buf[] = \n")
	for i, v := range data {
		if i%16 == 0 {
			b.WriteByte('"')
		}
		fmt.Fprintf(&b, "\\x%02x", v)
		if i%16 == 15 && i+1 < len(data) {
			b.WriteString("\"\n")
		}
	}
	b.WriteString("\";\n")
	return b.String()
}

func formatPython(data []byte) string {
	var b strings.Builder
	b.WriteString("buf   = \"\"\n")
	for i, v := range data {
		if i%16 == 0 {
			b.WriteString("buff += \"")
		}
		fmt.Fprintf(&b, "\\x%02x", v)
		if i%16 == 15 {
			b.WriteString("\"\n")
		}
	}
	if len(data)%16 != 0 {
		b.WriteByte('"')
	}
	return b.String()
}

func formatPowerShell(data []byte) string {
	var b strings.Builder
	b.WriteString("[Byte[]] $buf = ")
	for i, v := range data {
		fmt.Fprintf(&b, "0x%02x", v)
		if i < len(data)-1 {
			b.WriteByte(',')
		}
	}
	return b.String()
}

func formatCSharp(data []byte) string {
	var b strings.Builder
	fmt.Fprintf(&b, "byte[] my_buf = new byte[%d] {\n", len(data))
	for i, v := range data {
		fmt.Fprintf(&b, "0x%02x", v)
		if i < len(data)-1 {
			b.WriteByte(',')
		}
	}
	b.WriteString("};")
	return b.String()
}

func formatHex(data []byte) string {
	var b strings.Builder
	for _, v := range data {
		fmt.Fprintf(&b, "\\x%02x", v)
	}
	return b.String()
}

func formatUUID(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	rem := len(data) % 16
	if rem != 0 {
		pad := make([]byte, rem)
		data = append(append([]byte(nil), data...), pad...)
	}
	var b strings.Builder
	for i := 0; i < len(data); i += 16 {
		chunk := data[i : i+16]
		fmt.Fprintf(&b, "%02x%02x%02x%02x-", chunk[3], chunk[2], chunk[1], chunk[0])
		fmt.Fprintf(&b, "%02x%02x-", chunk[5], chunk[4])
		fmt.Fprintf(&b, "%02x%02x-", chunk[7], chunk[6])
		fmt.Fprintf(&b, "%02x%02x-", chunk[8], chunk[9])
		fmt.Fprintf(&b, "%02x%02x%02x%02x%02x%02x\n", chunk[10], chunk[11], chunk[12], chunk[13], chunk[14], chunk[15])
	}
	return b.String()
}

func exportedFunc(module api.Module, names ...string) (api.Function, error) {
	for _, name := range names {
		if fn := module.ExportedFunction(name); fn != nil {
			return fn, nil
		}
	}
	return nil, fmt.Errorf("export not found (tried %s)", strings.Join(names, ", "))
}

func allocBytes(ctx context.Context, malloc api.Function, mem api.Memory, data []byte) (uint32, error) {
	res, err := malloc.Call(ctx, uint64(len(data)))
	if err != nil {
		return 0, fmt.Errorf("malloc: %w", err)
	}
	ptr := uint32(res[0])
	if ptr == 0 {
		return 0, errors.New("malloc returned null")
	}
	if len(data) == 0 {
		return ptr, nil
	}
	if ok := mem.Write(ptr, data); !ok {
		return 0, errors.New("memory write failed")
	}
	return ptr, nil
}

func allocCString(ctx context.Context, malloc api.Function, mem api.Memory, s string) (uint32, error) {
	if strings.ContainsRune(s, '\x00') {
		return 0, errors.New("string contains NUL byte")
	}
	data := append([]byte(s), 0)
	return allocBytes(ctx, malloc, mem, data)
}

func callFree(ctx context.Context, free api.Function, ptr uint32) {
	if ptr == 0 {
		return
	}
	_, _ = free.Call(ctx, uint64(ptr))
}
