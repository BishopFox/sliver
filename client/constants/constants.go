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
	SetStr        = "set"

	GenerateStr        = "generate"
	RegenerateStr      = "regenerate"
	ProfileGenerateStr = "generate-profile"
	StagerStr          = "stager"
	ProfilesStr        = "profiles"
	NewProfileStr      = "new-profile"

	ImplantBuildsStr = "implants"
	ListCanariesStr  = "canaries"

	JobsStr        = "jobs"
	MtlsStr        = "mtls"
	DnsStr         = "dns"
	HttpStr        = "http"
	HttpsStr       = "https"
	NamedPipeStr   = "named-pipe"
	TCPListenerStr = "tcp-pivot"

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

	ScreenshotStr = "screenshot"
	PsExecStr     = "psexec"
	BackdoorStr   = "backdoor"
	MakeTokenStr  = "make-token"
	GetEnvStr     = "getenv"

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
