package generate

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/binject/go-donut/donut"
)

// ShellcodeFromFile returns a Donut shellcode for the given PE file
func ShellcodeFromFile(filePath string, arch string, dotnet bool, params string, className string, method string) (data []byte, err error) {
	pe, err := ioutil.ReadFile(filePath)
	if err != nil {
		return
	}
	var donutType donut.ModuleType
	switch strings.ToLower(filepath.Ext(filePath)) {
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
	return ShellcodeFromPE(pe, arch, dotnet, params, className, method, donutType)
}

// ShellcodeFromPE returns a Donut shellcode for the given PE file
func ShellcodeFromPE(pe []byte, arch string, dotnet bool, params string, className string, method string, donutType donut.ModuleType) (data []byte, err error) {
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

	config := donut.DonutConfig{
		Type:       donutType,
		DotNetMode: dotnet,
		InstType:   donut.DONUT_INSTANCE_PIC,
		Parameters: params,
		Class:      className,
		Method:     method,
		Bypass:     3,         // 1=skip, 2=abort on fail, 3=continue on fail.
		Format:     uint32(1), // 1=raw, 2=base64, 3=c, 4=ruby, 5=python, 6=powershell, 7=C#, 8=hex
		Arch:       donutArch,
		Entropy:    0,         // 1=disable, 2=use random names, 3=random names + symmetric encryption (default)
		Compress:   uint32(1), // 1=disable, 2=LZNT1, 3=Xpress, 4=Xpress Huffman
		Thread:     0,         // start a new thread
		ExitOpt:    1,         // exit thread
		Unicode:    0,
	}
	return getDonut(pe, &config)
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
