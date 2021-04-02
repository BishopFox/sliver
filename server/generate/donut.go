package generate

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/binject/go-donut/donut"
)

// DonutShellcodeFromFile returns a Donut shellcode for the given PE file
func DonutShellcodeFromFile(filePath string, arch string, dotnet bool, params string, className string, method string) (data []byte, err error) {
	pe, err := ioutil.ReadFile(filePath)
	if err != nil {
		return
	}
	donutType := getDonutType(filepath.Ext(filePath), dotnet)
	return DonutShellcodeFromPE(pe, arch, dotnet, params, className, method, donutType)
}

// DonutShellcodeFromPE returns a Donut shellcode for the given PE file
func DonutShellcodeFromPE(pe []byte, arch string, dotnet bool, params string, className string, method string, donutType donut.ModuleType) (data []byte, err error) {
	donutArch := getDonutArch(arch)
	config := donut.DonutConfig{
		Type:       donutType,
		InstType:   donut.DONUT_INSTANCE_PIC,
		Parameters: params,
		Class:      className,
		Method:     method,
		Bypass:     3,         // 1=skip, 2=abort on fail, 3=continue on fail.
		Format:     uint32(1), // 1=raw, 2=base64, 3=c, 4=ruby, 5=python, 6=powershell, 7=C#, 8=hex
		Arch:       donutArch,
		Entropy:    0,         // 1=disable, 2=use random names, 3=random names + symmetric encryption (default)
		Compress:   uint32(1), // 1=disable, 2=LZNT1, 3=Xpress, 4=Xpress Huffman
		Thread:     1,         // start a new thread
		ExitOpt:    1,         // exit thread
		Unicode:    0,
	}
	return getDonut(pe, &config)
}

// DonutFromAssembly - Generate a donut shellcode from a .NET assembly
func DonutFromAssembly(assembly []byte, isDLL bool, arch string, params string, method string, className string, appDomain string) ([]byte, error) {
	ext := ".exe"
	if isDLL {
		ext = ".dll"
	}
	donutArch := getDonutArch(arch)
	config := donut.DefaultConfig()
	config.Bypass = 3
	config.Runtime = "v4.0.30319" // we might want to make this configurable
	config.Format = 1
	config.Arch = donutArch
	config.Class = className
	config.Parameters = params
	config.Domain = appDomain
	config.Method = method
	config.Entropy = 3
	config.Unicode = 0
	config.Type = getDonutType(ext, true)
	return getDonut(assembly, config)
}

func getDonut(data []byte, config *donut.DonutConfig) (shellcode []byte, err error) {
	buf := bytes.NewBuffer(data)
	res, err := donut.ShellcodeFromBytes(buf, config)
	if err != nil {
		return
	}
	shellcode = res.Bytes()
	return
}

func getDonutArch(arch string) donut.DonutArch {
	var donutArch donut.DonutArch
	switch strings.ToLower(arch) {
	case "x32", "386":
		donutArch = donut.X32
	case "x64", "amd64":
		donutArch = donut.X64
	case "x84":
		donutArch = donut.X84
	default:
		donutArch = donut.X84
	}
	return donutArch
}

func getDonutType(ext string, dotnet bool) donut.ModuleType {
	var donutType donut.ModuleType
	switch strings.ToLower(filepath.Ext(ext)) {
	case ".exe", ".bin":
		if dotnet {
			donutType = donut.DONUT_MODULE_NET_EXE
		} else {
			donutType = donut.DONUT_MODULE_EXE
		}
	case ".dll":
		if dotnet {
			donutType = donut.DONUT_MODULE_NET_DLL
		} else {
			donutType = donut.DONUT_MODULE_DLL
		}
	case ".xsl":
		donutType = donut.DONUT_MODULE_XSL
	case ".js":
		donutType = donut.DONUT_MODULE_JS
	case ".vbs":
		donutType = donut.DONUT_MODULE_VBS
	}
	return donutType
}
