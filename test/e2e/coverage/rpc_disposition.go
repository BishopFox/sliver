package coverage

// RPCDispositionClass describes how a SliverRPC method relates to an implant.
type RPCDispositionClass string

const (
	// RPCServerOnly is a control-plane method which does not exchange a command
	// with an implant.
	RPCServerOnly RPCDispositionClass = "server-only"
	// RPCCommandCovered is a finite implant command exercised by the
	// comprehensive E2E command catalog.
	RPCCommandCovered RPCDispositionClass = "implant-command-covered"
	// RPCCommandDeferred is a finite implant command which is deliberately not
	// yet exercised by the comprehensive E2E command catalog.
	RPCCommandDeferred RPCDispositionClass = "implant-command-deferred"
	// RPCImplantLifecycle changes the implant process or connection lifecycle.
	RPCImplantLifecycle RPCDispositionClass = "implant-lifecycle"
	// RPCTunnelInteractive belongs to a pivot, tunnel, SOCKS, port-forward, or
	// interactive stream protocol rather than the finite command matrix.
	RPCTunnelInteractive RPCDispositionClass = "tunnel-or-interactive"
)

// RPCDisposition classifies one method in the generated SliverRPC service.
// ImplantTraffic is false for server-only methods and for tunnel allocation
// methods which create only server-side state.
type RPCDisposition struct {
	Method         string              `json:"method"`
	Class          RPCDispositionClass `json:"class"`
	ImplantTraffic bool                `json:"implant_traffic"`
	Reason         string              `json:"reason"`
}

// ComprehensiveRPCDispositions returns the exhaustive classification of the
// generated SliverRPC service surface. Tests compare this registry against the
// protobuf service descriptor, so a newly added RPC cannot remain unclassified.
func ComprehensiveRPCDispositions() []RPCDisposition {
	dispositions := []RPCDisposition{}
	add := func(class RPCDispositionClass, implantTraffic bool, reason string, methods ...string) {
		for _, method := range methods {
			dispositions = append(dispositions, RPCDisposition{
				Method: method, Class: class, ImplantTraffic: implantTraffic, Reason: reason,
			})
		}
	}

	add(RPCServerOnly, false, "server control-plane operation; no implant command is dispatched",
		"GetVersion",
		"ClientLog",
		"GetOperators",
		"Rename",
		"GetSessions",
		"MonitorStart",
		"MonitorStop",
		"MonitorListConfig",
		"MonitorAddConfig",
		"MonitorDelConfig",
		"GetAIProviders",
		"GetAIConversations",
		"GetAIConversation",
		"SaveAIConversation",
		"DeleteAIConversation",
		"GetAIConversationMessages",
		"SaveAIConversationMessage",
		"StartMTLSListener",
		"StartWGListener",
		"StartDNSListener",
		"StartHTTPSListener",
		"StartHTTPListener",
		"GetBeacons",
		"GetBeacon",
		"RmBeacon",
		"GetBeaconTasks",
		"GetBeaconTaskContent",
		"CancelBeaconTask",
		"UpdateBeaconIntegrityInformation",
		"GetJobs",
		"KillJob",
		"RestartJobs",
		"StartTCPStagerListener",
		"LootAdd",
		"LootRm",
		"LootUpdate",
		"LootContent",
		"LootAll",
		"Creds",
		"CredsAdd",
		"CredsRm",
		"CredsUpdate",
		"GetCredByID",
		"GetCredsByHashType",
		"GetPlaintextCredsByHashType",
		"CredsSniffHashType",
		"Hosts",
		"Host",
		"HostRm",
		"HostIOCRm",
		"Generate",
		"GenerateSpoofMetadata",
		"GenerateExternal",
		"GenerateExternalSaveBuild",
		"GenerateExternalGetBuildConfig",
		"GenerateStage",
		"StageImplantBuild",
		"GetHTTPC2Profiles",
		"GetHTTPC2ProfileByName",
		"SaveHTTPC2Profile",
		"BuilderRegister",
		"BuilderTrigger",
		"Builders",
		"GetCertificateInfo",
		"GetCertificateAuthorityInfo",
		"Crack",
		"CrackstationRegister",
		"CrackstationTrigger",
		"CrackstationBenchmark",
		"Crackstations",
		"CrackTaskByID",
		"CrackTaskUpdate",
		"CrackFilesList",
		"CrackFileCreate",
		"CrackFileChunkUpload",
		"CrackFileChunkDownload",
		"CrackFileComplete",
		"CrackFileDelete",
		"Regenerate",
		"ImplantBuilds",
		"DeleteImplantBuild",
		"Canaries",
		"GenerateWGClientConfig",
		"GenerateUniqueIP",
		"ImplantProfiles",
		"DeleteImplantProfile",
		"SaveImplantProfile",
		"ShellcodeRDI",
		"GetCompiler",
		"ShellcodeEncoder",
		"ShellcodeEncoderMap",
		"TrafficEncoderMap",
		"TrafficEncoderAdd",
		"TrafficEncoderRm",
		"Websites",
		"Website",
		"WebsiteRemove",
		"WebsiteAddContent",
		"WebsiteUpdateContent",
		"WebsiteRemoveContent",
		"PivotGraph",
		"Events",
	)

	add(RPCCommandCovered, true, "exercised by ComprehensiveCatalog",
		"Ping",
		"Ps",
		"Terminate",
		"Ifconfig",
		"Netstat",
		"Ls",
		"Cd",
		"Pwd",
		"Mv",
		"Cp",
		"Rm",
		"Mkdir",
		"Download",
		"Upload",
		"Grep",
		"Chmod",
		"Chown",
		"Chtimes",
		"MemfilesList",
		"MemfilesAdd",
		"MemfilesRm",
		"Mount",
		"Execute",
		"ExecuteChildren",
		"CurrentTokenOwner",
		"Services",
		"ServiceDetail",
		"GetEnv",
		"SetEnv",
		"UnsetEnv",
		"GetPrivs",
		"RegistryRead",
		"RegistryWrite",
		"RegistryCreateKey",
		"RegistryDeleteKey",
		"RegistryListSubKeys",
		"RegistryListValues",
		"RegisterExtension",
		"CallExtension",
		"ListExtensions",
		"ListWasmExtensions",
	)

	deferred := []RPCDisposition{
		{Method: "Reconfigure", Reason: "changes callback behavior; reconnect transition assertions are not yet isolated"},
		{Method: "ProcessDump", Reason: "captures process memory and has no bounded disposable fixture"},
		{Method: "RunAs", Reason: "requires managed credentials and platform privileges"},
		{Method: "Impersonate", Reason: "requires a managed token source and identity rollback"},
		{Method: "RevToSelf", Reason: "requires a paired isolated impersonation fixture"},
		{Method: "GetSystem", Reason: "performs privilege escalation and generated payload execution"},
		{Method: "Task", Reason: "executes arbitrary shellcode"},
		{Method: "Msf", Reason: "requires external msfvenom and executes generated shellcode"},
		{Method: "MsfRemote", Reason: "requires external msfvenom and remote-process injection"},
		{Method: "ExecuteAssembly", Reason: "executes an arbitrary managed assembly"},
		{Method: "Migrate", Reason: "injects a new implant and changes the connection lifecycle"},
		{Method: "ExecuteWindows", Reason: "Windows token-specific execution fixture is not yet implemented"},
		{Method: "Sideload", Reason: "loads an arbitrary native library in a remote process"},
		{Method: "SpawnDll", Reason: "injects and executes an arbitrary DLL"},
		{Method: "Screenshot", Reason: "captures desktop contents and needs a controlled display fixture"},
		{Method: "StartServiceByName", Reason: "changes host service state"},
		{Method: "StartService", Reason: "creates and starts a host service"},
		{Method: "StopService", Reason: "changes host service state"},
		{Method: "RemoveService", Reason: "deletes host service configuration"},
		{Method: "MakeToken", Reason: "requires managed credentials and changes the active token"},
		{Method: "Backdoor", Reason: "rewrites a remote executable"},
		{Method: "RegistryReadHive", Reason: "reads a registry hive artifact and has no disposable fixture"},
		{Method: "RunSSHCommand", Reason: "requires an isolated SSH endpoint and managed credentials"},
		{Method: "HijackDLL", Reason: "rewrites and uploads a remote DLL"},
		{Method: "RegisterWasmExtension", Reason: "no pinned signed cross-platform WASM fixture is selected"},
		{Method: "ExecWasmExtension", Reason: "no pinned signed cross-platform WASM fixture is selected"},
	}
	for _, disposition := range deferred {
		disposition.Class = RPCCommandDeferred
		disposition.ImplantTraffic = true
		dispositions = append(dispositions, disposition)
	}

	lifecycle := []RPCDisposition{
		{Method: "Kill", Reason: "terminates the E2E implant process"},
		{Method: "OpenSession", Reason: "beacon-only transition into a new interactive session"},
		{Method: "CloseSession", Reason: "closes the active session connection"},
	}
	for _, disposition := range lifecycle {
		disposition.Class = RPCImplantLifecycle
		disposition.ImplantTraffic = true
		dispositions = append(dispositions, disposition)
	}

	add(RPCTunnelInteractive, true, "finite implant control for a pivot, forwarder, or interactive channel",
		"PivotStartListener",
		"PivotStopListener",
		"PivotSessionListeners",
		"StartRportFwdListener",
		"GetRportFwdListeners",
		"StopRportFwdListener",
		"WGStartPortForward",
		"WGStopPortForward",
		"WGStartSocks",
		"WGStopSocks",
		"WGListForwarders",
		"WGListSocksServers",
		"Shell",
		"ShellResize",
		"Portfwd",
		"CloseSocks",
		"SocksProxy",
		"CloseTunnel",
		"TunnelData",
	)
	add(RPCTunnelInteractive, false, "allocates server-side tunnel state; subsequent stream RPCs exchange implant data",
		"CreateSocks",
		"CreateTunnel",
	)

	return dispositions
}
