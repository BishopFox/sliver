package constants

// Meta
const (
	KeepAliveStr = "keepalive"
)

// Events
const (
	EventStr = "event"

	ServerErrorStr = "server"

	// ConnectedEvent - Sliver Connected
	ConnectedEvent = "connected"
	// DisconnectedEvent - Sliver disconnected
	DisconnectedEvent = "disconnected"

	// JoinedEvent - Player joined the game
	JoinedEvent = "joined"
	// LeftEvent - Player left the game
	LeftEvent = "left"

	// CanaryEvent - A DNS canary was triggered
	CanaryEvent = "canary"

	// StartedEvent - Job was started
	StartedEvent = "started"
	// StoppedEvent - Job was stopped
	StoppedEvent = "stopped"
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

	GenerateStr        = "generate"
	RegenerateStr      = "regenerate"
	ProfileGenerateStr = "generate-profile"
	EggGenerate        = "generate-egg"
	ProfilesStr        = "profiles"
	NewProfileStr      = "new-profile"

	ListSliverBuildsStr = "slivers"
	ListCanariesStr     = "canaries"

	JobsStr  = "jobs"
	MtlsStr  = "mtls"
	DnsStr   = "dns"
	HttpStr  = "http"
	HttpsStr = "https"

	MsfStr       = "msf"
	MsfInjectStr = "msf-inject"

	PsStr   = "ps"
	PingStr = "ping"
	KillStr = "kill"

	GetPIDStr = "getpid"
	GetUIDStr = "getuid"
	GetGIDStr = "getgid"
	WhoamiStr = "whoami"

	ShellStr = "shell"

	LsStr               = "ls"
	RmStr               = "rm"
	MkdirStr            = "mkdir"
	CdStr               = "cd"
	PwdStr              = "pwd"
	CatStr              = "cat"
	DownloadStr         = "download"
	UploadStr           = "upload"
	ProcdumpStr         = "procdump"
	ImpersonateStr      = "impersonate"
	ElevateStr          = "elevate"
	GetSystemStr        = "getsystem"
	ExecuteAssemblyStr  = "execute-assembly"
	ExecuteShellcodeStr = "execute-shellcode"
	MigrateStr          = "migrate"

	WebsitesStr = "websites"

	// Groups
	GenericHelpGroup     = "Generic:"
	SliverHelpGroup      = "Sliver:"
	SliverWinHelpGroup   = "Sliver - Windows:"
	MultiplayerHelpGroup = "Multiplayer:"
)
