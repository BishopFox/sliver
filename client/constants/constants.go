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
	KeepAliveStr = "keepalive"
)

// Events
const (
	UpdateStr  = "update"
	VersionStr = "version"
	ExitStr    = "exit"

	EventStr = "event"

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
)

// Commands
const (
	NewPlayerStr       = "new-player"
	PlayersStr         = "players"
	KickPlayerStr      = "kick-player"
	MultiplayerModeStr = "multiplayer"

	SessionsStr   = "sessions"
	BackgroundStr = "background"
	InfoStr       = "info"
	UseStr        = "use"
	InteractStr   = "interact"
	SetStr        = "set"
	HelpStr       = "help"

	GenerateStr        = "generate"
	StageStr           = "stage"
	StagerStr          = "stager"
	RegenerateStr      = "regenerate"
	ProfileGenerateStr = "generate-profile"
	ProfilesStr        = "profiles"
	ProfilesDeleteStr  = "delete"
	NewProfileStr      = "new-profile"

	ImplantBuildsStr = "implants"
	ListCanariesStr  = "canaries"

	JobsStr        = "jobs"
	JobsKillStr    = "kill"
	JobsKillAllStr = "kill-all"
	MtlsStr        = "mtls"
	DnsStr         = "dns"
	HttpStr        = "http"
	HttpsStr       = "https"
	NamedPipeStr   = "named-pipe"
	TCPListenerStr = "tcp-pivot"

	ConfigStr             = "config"
	ConfigSaveStr         = "save"
	ConfigPromptServerStr = "prompt-server"
	ConfigPromptSliverStr = "prompt-sliver"
	ConfigVimStr          = "vim"
	ConfigEmacsStr        = "emacs"
	ConfigHintsStr        = "hints"

	LogStr = "log"

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
	RmStr       = "rm"
	MkdirStr    = "mkdir"
	CdStr       = "cd"
	LcdStr      = "lcd"
	PwdStr      = "pwd"
	CatStr      = "cat"
	DownloadStr = "download"
	UploadStr   = "upload"
	IfconfigStr = "ifconfig"
	NetstatStr  = "netstat"

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
	LoadExtensionStr    = "load-extension"
	StageListenerStr    = "stage-listener"

	WebsitesStr       = "websites"
	RmWebContentStr   = "rm-content"
	AddWebContentStr  = "add-content"
	WebContentTypeStr = "content-type"
	WebUpdateStr      = "update"

	ScreenshotStr         = "screenshot"
	PsExecStr             = "psexec"
	BackdoorStr           = "backdoor"
	MakeTokenStr          = "make-token"
	GetEnvStr             = "getenv"
	SetEnvStr             = "setenv"
	RegistryStr           = "registry"
	RegistryReadStr       = "read"
	RegistryWriteStr      = "write"
	RegistryListSubStr    = "list-subkeys"
	RegistryListValuesStr = "list-values"
	RegistryCreateKeyStr  = "create"

	RouteStr       = "route"
	RoutePrintStr  = "print"
	RouteAddStr    = "add"
	RouteRemoveStr = "remove"

	PortfwdStr      = "portfwd"
	PortfwdPrintStr = "print"
	PortfwdOpenStr  = "open"
	PortfwdCloseStr = "close"

	LicensesStr = "licenses"
)

// Groups
const (
	GenericHelpGroup     = "Generic:"
	SliverHelpGroup      = "Sliver:"
	SliverWinHelpGroup   = "Sliver - Windows:"
	MultiplayerHelpGroup = "Multiplayer:"
	ExtensionHelpGroup   = "Sliver - 3rd Party extensions:"
)

// Command categories
const (
	AdminGroup      = "admin"
	CoreServerGroup = "core (server)"
	BuildsGroup     = "implants"
	TransportsGroup = "transports"
	SessionsGroup   = "sessions"
	CommGroup       = "comm"

	// Session only
	CoreSessionGroup = "core (session)"
	FilesystemGroup  = "filesystem"
	InfoGroup        = "information"
	ProcGroup        = "process"
	PrivGroup        = "priv"
	ExecuteGroup     = "execution"
	PersistenceGroup = "persistence"
	ExtensionsGroup  = "extensions"
)

// C2 default values
const (
	DefaultMTLSLPort    = 8888
	DefaultHTTPLPort    = 80
	DefaultHTTPSLPort   = 443
	DefaultDNSLPort     = 53
	DefaultTCPLPort     = 4444
	DefaultTCPPivotPort = 9898

	DefaultReconnect = 60
	DefaultMaxErrors = 1000

	DefaultTimeout = 60
)
