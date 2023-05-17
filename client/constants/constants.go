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

// Meta
const (
	// KeepAliveStr - Keep alive constant
	KeepAliveStr = "keepalive"
)

const (
	// LastUpdateCheckFileName - Last update check file name
	LastUpdateCheckFileName = "last_update_check"
)

// Events
const (
	// UpdateStr - "update"
	UpdateStr = "update"
	// VersionStr - "version"
	VersionStr = "version"

	// EventStr - "event"
	EventStr = "event"

	// ServersStr - "server-error"
	ServerErrorStr = "server-error"

	// ConnectedEvent - Sliver Connected
	SessionOpenedEvent = "session-connected"
	// DisconnectedEvent - Sliver disconnected
	SessionClosedEvent = "session-disconnected"
	// UpdateEvent - Sliver updated
	SessionUpdateEvent = "session-updated"

	// JoinedEvent - Player joined the game
	JoinedEvent = "client-joined"
	// LeftEvent - Player left the game
	LeftEvent = "client-left"

	// CanaryEvent - A DNS canary was triggered
	CanaryEvent = "canary"

	// WatchtowerEvent - An implant hash has been identified on a threat intel platform
	WatchtowerEvent = "watchtower"

	// StartedEvent - Job was started
	JobStartedEvent = "job-started"
	// StoppedEvent - Job was stopped
	JobStoppedEvent = "job-stopped"

	// BuildEvent - Fires on change to builds
	BuildEvent = "build"

	// BuildCompletedEvent - Fires when a build completes
	BuildCompletedEvent = "build-completed"

	// ProfileEvent - Fires whenever there's a change to profiles
	ProfileEvent = "profile"

	// WebsiteEvent - Fires whenever there's a change to websites
	WebsiteEvent = "website"

	// LootAdded
	LootAddedEvent = "loot-added"

	// LootRemoved
	LootRemovedEvent = "loot-removed"

	// BeaconRegisteredEvent - First connection from a new beacon
	BeaconRegisteredEvent = "beacon-registered"

	// BeaconTaskResult - Beacon task completed with a result
	BeaconTaskResultEvent = "beacon-taskresult"

	// ExternalBuildEvent
	ExternalBuildEvent          = "external-build"
	AcknowledgeBuildEvent       = "external-acknowledge"
	ExternalBuildFailedEvent    = "external-build-failed"
	ExternalBuildCompletedEvent = "external-build-completed"

	// WireGuardNewPeer - New Wireguard peer added
	WireGuardNewPeer = "wireguard-newpeer"
)

// Commands
const (
	OperatorsStr       = "operators"
	NewOperatorStr     = "new-operator"
	KickOperatorStr    = "kick-operator"
	MultiplayerModeStr = "multiplayer"

	SessionsStr     = "sessions"
	BackgroundStr   = "background"
	InfoStr         = "info"
	UseStr          = "use"
	ReconfigStr     = "reconfig"
	PruneStr        = "prune"
	TasksStr        = "tasks"
	CancelStr       = "cancel"
	GenerateStr     = "generate"
	RegenerateStr   = "regenerate"
	CompilerInfoStr = "info"
	StagerStr       = "stager"
	ProfilesStr     = "profiles"
	BeaconStr       = "beacon"
	BeaconsStr      = "beacons"
	WatchStr        = "watch"
	SettingsStr     = "settings"
	SearchStr       = "search"

	// Generic

	// NewStr - "new"
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

	LootStr       = "loot"
	LootLocalStr  = "local"
	LootRemoteStr = "remote"
	FetchStr      = "fetch"
	LootCredsStr  = "creds"

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

	MsfStr       = "msf"
	MsfInjectStr = "msf-inject"

	PsStr        = "ps"
	PingStr      = "ping"
	KillStr      = "kill"
	TerminateStr = "terminate"

	GetPIDStr = "getpid"
	GetUIDStr = "getuid"
	GetGIDStr = "getgid"
	WhoamiStr = "whoami"

	ShellStr   = "shell"
	ExecuteStr = "execute"

	LsStr       = "ls"
	MvStr       = "mv"
	RmStr       = "rm"
	MkdirStr    = "mkdir"
	CdStr       = "cd"
	PwdStr      = "pwd"
	CatStr      = "cat"
	DownloadStr = "download"
	UploadStr   = "upload"
	IfconfigStr = "ifconfig"
	NetstatStr  = "netstat"
	ChmodStr    = "chmod"
	ChownStr    = "chown"
	ChtimesStr  = "chtimes"

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
	PivotsStr             = "pivots"
	WgConfigStr           = "wg-config"
	WgSocksStr            = "wg-socks"
	WgPortFwdStr          = "wg-portfwd"
	MonitorStr            = "monitor"
	SSHStr                = "ssh"
	DLLHijackStr          = "dllhijack"
	InteractiveStr        = "interactive"
	CloseStr              = "close"

	PortfwdStr  = "portfwd"
	Socks5Str   = "socks5"
	RportfwdStr = "rportfwd"

	ReactionStr = "reaction"

	HostsStr = "hosts"
	IOCStr   = "ioc"

	LicensesStr = "licenses"

	GetPrivsStr        = "getprivs"
	PreludeOperatorStr = "prelude-operator"
	ConnectStr         = "connect"

	ShikataGaNai = "shikata-ga-nai"

	Cursed         = "cursed"
	CursedChrome   = "chrome"
	CursedConsole  = "console"
	CursedElectron = "electron"
	CursedEdge     = "edge"
	CursedCookies  = "cookies"

	BuildersStr = "builders"
)

// Groups
const (
	GenericHelpGroup     = "Generic:"
	SliverHelpGroup      = "Sliver:"
	SliverWinHelpGroup   = "Sliver - Windows:"
	MultiplayerHelpGroup = "Multiplayer:"
	AliasHelpGroup       = "Sliver - 3rd Party macros:"
	ExtensionHelpGroup   = "Sliver - 3rd Party extensions:"
)
