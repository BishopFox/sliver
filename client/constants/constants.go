package constants

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

// Meta.
const (
	// KeepAliveStr - Keep alive constant.
	KeepAliveStr = "keepalive"
)

const (
	// LastUpdateCheckFileName - Last update check file name.
	LastUpdateCheckFileName = "last_update_check"
)

// Console.
const (
	ImplantMenu = "implant"
	ServerMenu  = ""
)

// Events.
const (
	// UpdateStr - "update".
	UpdateStr = "update"
	// VersionStr - "version".
	VersionStr = "version"

	// EventStr - "event".
	EventStr = "event"

	// ServersStr - "server-error".
	ServerErrorStr = "server-error"

	// ConnectedEvent - Sliver Connected.
	SessionOpenedEvent = "session-connected"
	// DisconnectedEvent - Sliver disconnected.
	SessionClosedEvent = "session-disconnected"
	// UpdateEvent - Sliver updated.
	SessionUpdateEvent = "session-updated"

	// JoinedEvent - Player joined the game.
	JoinedEvent = "client-joined"
	// LeftEvent - Player left the game.
	LeftEvent = "client-left"

	// CanaryEvent - A DNS canary was triggered.
	CanaryEvent = "canary"

	// WatchtowerEvent - An implant hash has been identified on a threat intel platform.
	WatchtowerEvent = "watchtower"

	// StartedEvent - Job was started.
	JobStartedEvent = "job-started"
	// StoppedEvent - Job was stopped.
	JobStoppedEvent = "job-stopped"

	// BuildEvent - Fires on change to builds.
	BuildEvent = "build"

	// BuildCompletedEvent - Fires when a build completes.
	BuildCompletedEvent = "build-completed"

	// ProfileEvent - Fires whenever there's a change to profiles.
	ProfileEvent = "profile"

	// WebsiteEvent - Fires whenever there's a change to websites.
	WebsiteEvent = "website"

	// LootAdded.
	LootAddedEvent = "loot-added"

	// LootRemoved.
	LootRemovedEvent = "loot-removed"

	// BeaconRegisteredEvent - First connection from a new beacon.
	BeaconRegisteredEvent = "beacon-registered"

	// BeaconTaskResult - Beacon task completed with a result.
	BeaconTaskResultEvent = "beacon-taskresult"

	// ExternalBuildEvent.
	ExternalBuildEvent          = "external-build"
	AcknowledgeBuildEvent       = "external-acknowledge"
	ExternalBuildFailedEvent    = "external-build-failed"
	ExternalBuildCompletedEvent = "external-build-completed"

	// TrafficEncoder Events.
	TrafficEncoderTestProgressEvent = "traffic-encoder-test-progress"

	// Crackstation Events.
	CrackstationConnected    = "crackstation-connected"
	CrackstationDisconnected = "crackstation-disconnected"

	// Crack Events - Events consumed by crackstations.
	CrackBenchmark   = "crack-benchmark"
	CrackStatusEvent = "crack-status"

	// WireGuardNewPeer - New Wireguard peer added.
	WireGuardNewPeer = "wireguard-newpeer"
)

// Commands.
const (
	OperatorsStr       = "operators"
	NewOperatorStr     = "new-operator"
	KickOperatorStr    = "kick-operator"
	MultiplayerModeStr = "multiplayer"

	SessionsStr        = "sessions"
	BackgroundStr      = "background"
	InfoStr            = "info"
	UseStr             = "use"
	TaskmanyStr        = "taskmany"
	ReconfigStr        = "reconfig"
	PruneStr           = "prune"
	TasksStr           = "tasks"
	CancelStr          = "cancel"
	GenerateStr        = "generate"
	C2GenerateStr      = "generate"
	RegenerateStr      = "regenerate"
	CompilerInfoStr    = "info"
	ProfilesStr        = "profiles"
	BeaconStr          = "beacon"
	BeaconsStr         = "beacons"
	WatchStr           = "watch"
	SettingsStr        = "settings"
	SearchStr          = "search"
	TrafficEncodersStr = "traffic-encoders"
	C2ProfileStr       = "c2profiles"
	ImportC2ProfileStr = "import"
	ExportC2ProfileStr = "export"
	CertificatesStr    = "certificates"

	// Generic.

	// NewStr - "new".
	NewStr     = "new"
	AddStr     = "add"
	StartStr   = "start"
	StopStr    = "stop"
	SetStr     = "set"
	UnsetStr   = "unset"
	SaveStr    = "save"
	ReloadStr  = "reload"
	LoadStr    = "load"
	TablesStr  = "tables"
	DetailsStr = "details"
	GraphStr   = "graph"
	EnableStr  = "enable"
	DisableStr = "disable"
	ModifyStr  = "modify"
	RefreshStr = "refresh"
	ResetStr   = "reset"

	LootStr       = "loot"
	LootLocalStr  = "local"
	LootRemoteStr = "remote"
	FetchStr      = "fetch"
	CredsStr      = "creds"
	FileStr       = "file"

	RenameStr = "rename"

	ImplantBuildsStr = "implants"
	CanariesStr      = "canaries"

	JobsStr        = "jobs"
	MtlsStr        = "mtls"
	WGStr          = "wg"
	DnsStr         = "dns"
	HttpStr        = "http"
	HttpsStr       = "https"
	NamedPipeStr   = "named-pipe"
	TCPListenerStr = "tcp"
	UDPListenerStr = "udp"

	MsfStr       = "msf"
	MsfInjectStr = "msf-inject"

	PsStr            = "ps"
	PingStr          = "ping"
	KillStr          = "kill"
	TerminateStr     = "terminate"
	ServicesStr      = "services"
	ServicesInfoStr  = "info"
	ServicesStopStr  = "stop"
	ServicesStartStr = "start"

	GetPIDStr = "getpid"
	GetUIDStr = "getuid"
	GetGIDStr = "getgid"
	WhoamiStr = "whoami"

	ShellStr   = "shell"
	ExecuteStr = "execute"

	LsStr       = "ls"
	MvStr       = "mv"
	CpStr       = "cp"
	RmStr       = "rm"
	StageStr    = "stage"
	MkdirStr    = "mkdir"
	CdStr       = "cd"
	PwdStr      = "pwd"
	CatStr      = "cat"
	DownloadStr = "download"
	HeadStr     = "head"
	TailStr     = "tail"
	GrepStr     = "grep"
	UploadStr   = "upload"
	IfconfigStr = "ifconfig"
	NetstatStr  = "netstat"
	ChmodStr    = "chmod"
	ChownStr    = "chown"
	ChtimesStr  = "chtimes"
	MountStr    = "mount"

	MemfilesStr = "memfiles"

	ProcdumpStr         = "procdump"
	ImpersonateStr      = "impersonate"
	RunAsStr            = "runas"
	ElevateStr          = "elevate"
	GetSystemStr        = "getsystem"
	RevToSelfStr        = "rev2self"
	ExecuteAssemblyStr  = "execute-assembly"
	ExecuteShellcodeStr = "execute-shellcode"
	MigrateStr          = "migrate"
	SideloadStr         = "sideload"
	SpawnDllStr         = "spawndll"
	ExtensionsStr       = "extensions"
	InstallStr          = "install"
	ListStr             = "list"
	ArmoryStr           = "armory"
	AliasesStr          = "aliases"
	StageListenerStr    = "stage-listener"

	WebsitesStr       = "websites"
	RmWebContentStr   = "rm-content"
	AddWebContentStr  = "add-content"
	WebContentTypeStr = "content-type"

	ScreenshotStr         = "screenshot"
	PsExecStr             = "psexec"
	BackdoorStr           = "backdoor"
	MakeTokenStr          = "make-token"
	EnvStr                = "env"
	RegistryStr           = "registry"
	RegistryReadStr       = "read"
	RegistryWriteStr      = "write"
	RegistryListSubStr    = "list-subkeys"
	RegistryListValuesStr = "list-values"
	RegistryCreateKeyStr  = "create"
	RegistryDeleteKeyStr  = "delete"
	RegistryReadHiveStr   = "hive"
	PivotsStr             = "pivots"
	WgConfigStr           = "wg-config"
	WgSocksStr            = "wg-socks"
	WgPortFwdStr          = "wg-portfwd"
	MonitorStr            = "monitor"
	MonitorConfigStr      = "config"
	SSHStr                = "ssh"
	DLLHijackStr          = "dllhijack"
	InteractiveStr        = "interactive"
	CloseStr              = "close"
	WasmStr               = "wasm"

	PortfwdStr  = "portfwd"
	Socks5Str   = "socks5"
	RportfwdStr = "rportfwd"

	ReactionStr = "reaction"

	HostsStr = "hosts"
	IOCStr   = "ioc"

	LicensesStr = "licenses"

	GetPrivsStr = "getprivs"
	ConnectStr  = "connect"

	ShikataGaNai = "shikata-ga-nai"

	Cursed         = "cursed"
	CursedChrome   = "chrome"
	CursedConsole  = "console"
	CursedElectron = "electron"
	CursedEdge     = "edge"
	CursedCookies  = "cookies"

	BuildersStr = "builders"

	CrackStr         = "crack"
	StationsStr      = "stations"
	WordlistsStr     = "wordlists"
	RulesStr         = "rules"
	Hcstat2Str       = "hcstat2"
	DefaultC2Profile = "default"
)

// Groups.
const (
	// Server commands =====================.
	GenericHelpGroup  = "Generic"
	NetworkHelpGroup  = "Network"
	PayloadsHelpGroup = "Payload"
	DataHelpGroup     = "Data"
	SliverHelpGroup   = "Sliver"

	// Sliver commands =====================.
	SliverCoreHelpGroup = "Core"
	InfoHelpGroup       = "Info"
	FilesystemHelpGroup = "Filesystem"
	ExecutionHelpGroup  = "Execution"
	PrivilegesHelpGroup = "Privileges"
	ProcessHelpGroup    = "Process"

	AliasHelpGroup     = "Sliver - 3rd Party macros"
	ExtensionHelpGroup = "Sliver - 3rd Party extensions"

	// Useless.
	SliverWinHelpGroup   = "Sliver - Windows"
	MultiplayerHelpGroup = "Multiplayer"
)

// Command types / filters (per OS/type/C2/etc)
// Should not be changed: extension.json artifact file (architecture/OS) rely on some of the values below,.
const (
	SessionCmdsFilter   = "session"
	BeaconCmdsFilter    = "beacon"
	WindowsCmdsFilter   = "windows"
	WireguardCmdsFilter = "wireguard"
)

// Creds (needed here to avoid recursive imports).
const (
	UserColonHashNewlineFormat = "user:hash" // username:hash\n
	HashNewlineFormat          = "hash"      // hash\n
	CSVFormat                  = "csv"       // username,hash\n
)
