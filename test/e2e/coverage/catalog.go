package coverage

// CommandExpectation declares one required command/scenario row and the exact
// targets on which it is supported. SupportedTargets must be explicit and
// nonempty so additions to the target matrix cannot silently gain coverage.
type CommandExpectation struct {
	Command
	SupportedTargets  []Target
	UnsupportedReason string
}

// ComprehensiveCatalog returns the static command/scenario catalog exercised
// by the comprehensive E2E harness. Keeping this independent of observed
// reports ensures a command omitted everywhere is still rendered as NOT RUN.
func ComprehensiveCatalog() []CommandExpectation {
	allTargets := comprehensiveTargets()
	unixTargets := []Target{
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "linux", Arch: "386"},
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
	}
	linuxTargets := []Target{
		{OS: "linux", Arch: "386"},
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
	}
	windowsTargets := []Target{
		{OS: "windows", Arch: "386"},
		{OS: "windows", Arch: "amd64"},
		{OS: "windows", Arch: "arm64"},
	}
	armoryTargets := []Target{
		{OS: "windows", Arch: "386"},
		{OS: "windows", Arch: "amd64"},
	}
	opforAMD64Targets := []Target{
		{OS: "windows", Arch: "amd64"},
	}

	portable := func(method, scenario string) CommandExpectation {
		return CommandExpectation{
			Command:          Command{GRPCMethod: method, Scenario: scenario},
			SupportedTargets: append([]Target(nil), allTargets...),
		}
	}
	restricted := func(method, scenario, detail string, targets []Target) CommandExpectation {
		return CommandExpectation{
			Command:           Command{GRPCMethod: method, Scenario: scenario},
			SupportedTargets:  append([]Target(nil), targets...),
			UnsupportedReason: detail,
		}
	}

	catalog := []CommandExpectation{
		portable("Ping", "exact nonce round trip"),
		portable("Pwd", "initial working directory"),
		portable("Mkdir", "recursive nested directory"),
		portable("Ls", "directory and wildcard metadata"),
		portable("Cd", "relative, parent, and rejected missing path"),
		portable("Upload", "gzip file and overwrite"),
		portable("Upload", "tar directory recursive overwrite truncation"),
		portable("Download", "file, byte/line limits, and recursive directory"),
		portable("Grep", "context and recursive regex"),
		portable("Cp", "copy exact bytes"),
		portable("Mv", "rename within test root"),
		portable("Chtimes", "exact access and modification time"),
		portable("Rm", "file then recursive directory with force"),
		portable("Ifconfig", "loopback interface and parseable addresses"),
		portable("Netstat", "TCP IPv4 listening"),
		portable("Netstat", "TCP IPv4 established"),
		portable("Netstat", "UDP-only IPv4"),
		portable("GetEnv", "full inherited environment"),
		portable("SetEnv", "set unique process variable"),
		portable("GetEnv", "named variable after set"),
		portable("UnsetEnv", "unset unique process variable"),
		portable("Ps", "FullInfo=false and implant PID"),
		portable("Ps", "FullInfo=true and implant PID"),
		portable("Execute", "captured stdout stderr status and explicit environment"),
		portable("Execute", "tracked background child"),
		portable("ExecuteChildren", "find tracked live child"),
		portable("Terminate", "kill only tracked test child"),
		portable("Mount", "nonempty read-only mount inventory"),
		portable("ListWasmExtensions", "empty initial extension inventory"),

		restricted("Chmod", "recursive mode change inside test root", "supported only on Darwin and Linux", unixTargets),
		restricted("Chown", "recursive no-op to current owner", "supported only on Darwin and Linux", unixTargets),
		restricted("MemfilesAdd", "create anonymous memfd", "supported only on Linux", linuxTargets),
		restricted("MemfilesList", "list exact anonymous memfd", "supported only on Linux", linuxTargets),
		restricted("MemfilesRm", "close only created anonymous memfd", "supported only on Linux", linuxTargets),

		restricted("CurrentTokenOwner", "nonempty current token identity", "supported only on Windows", windowsTargets),
		restricted("GetPrivs", "read-only process privilege inventory", "supported only on Windows", windowsTargets),
		restricted("Services", "read-only local service inventory", "supported only on Windows", windowsTargets),
		restricted("ServiceDetail", "exact detail for inventoried service", "supported only on Windows", windowsTargets),
		restricted("RegistryCreateKey", "unique HKCU subtree and child", "supported only on Windows", windowsTargets),
		restricted("RegistryWrite", "string binary DWORD and QWORD values", "supported only on Windows", windowsTargets),
		restricted("RegistryRead", "exact typed value round trips", "supported only on Windows", windowsTargets),
		restricted("RegistryListSubKeys", "exact disposable child inventory", "supported only on Windows", windowsTargets),
		restricted("RegistryListValues", "exact typed value inventory", "supported only on Windows", windowsTargets),
		restricted("RegistryDeleteKey", "remove only disposable child and root", "supported only on Windows", windowsTargets),

		restricted("RegisterExtension", "signed COFFLoader exact target", "signed Armory packages have no exact target for this OS/architecture", armoryTargets),
		restricted("ListExtensions", "registered COFFLoader digest", "signed Armory packages have no exact target for this OS/architecture", armoryTargets),
		restricted("CallExtension", "signed sa-env BOF through COFFLoader", "signed Armory packages have no exact target for this OS/architecture", armoryTargets),
		restricted("CallExtension", "signed sa-whoami BOF through COFFLoader", "signed Armory packages have no exact target for this OS/architecture", armoryTargets),

		restricted("CallExtension", "OPFOR Cat CNA reads isolated test file", "OPFOR CNA BOFs are Windows-only and the OPFOR provider does not support windows/arm64", armoryTargets),
		restricted("CallExtension", "OPFOR FirefoxDump CNA finds no host profiles", "OPFOR CNA BOFs are Windows-only and the OPFOR provider does not support windows/arm64", armoryTargets),
		restricted("CallExtension", "OPFOR FindDotnet CNA read-only process inventory", "the pinned OperatorsKit BOF is windows/amd64-only; OPFOR CNA BOFs are Windows-only and the provider does not support windows/arm64", opforAMD64Targets),
		restricted("CallExtension", "OPFOR FindSysmon CNA read-only registry probe", "the pinned OperatorsKit BOF is windows/amd64-only; OPFOR CNA BOFs are Windows-only and the provider does not support windows/arm64", opforAMD64Targets),
		restricted("CallExtension", "OPFOR callback preserves ordered typed binary channels", "the OPFOR synthetic BOF fixtures are Windows-only and the provider does not support windows/arm64", armoryTargets),
		restricted("CallExtension", "typed BOF partial output retained on callback error", "the OPFOR synthetic BOF fixtures are Windows-only and the provider does not support windows/arm64", armoryTargets),
		restricted("CallExtension", "malformed BOF returns bounded loader error", "the OPFOR synthetic BOF fixtures are Windows-only and the provider does not support windows/arm64", armoryTargets),
		restricted("CallExtension", "finite BOF deadline returns and target recovers", "the OPFOR synthetic BOF fixtures are Windows-only and the provider does not support windows/arm64", armoryTargets),
	}
	return catalog
}

func (expectation CommandExpectation) supports(target Target) bool {
	for _, supported := range expectation.SupportedTargets {
		if supported == target {
			return true
		}
	}
	return false
}

func comprehensiveTargets() []Target {
	return []Target{
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "windows", Arch: "386"},
		{OS: "windows", Arch: "amd64"},
		{OS: "windows", Arch: "arm64"},
		{OS: "linux", Arch: "386"},
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
	}
}
