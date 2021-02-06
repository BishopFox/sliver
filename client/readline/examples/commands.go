package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/evilsocket/islazy/fs"
	"github.com/evilsocket/islazy/str"
	"github.com/jessevdk/go-flags"

	"github.com/maxlandon/readline"
)

// This file declares a go-flags parser and a few commands.

var (
	// commandParser - The command parser used by the example console.
	commandParser = flags.NewNamedParser("example", flags.IgnoreUnknown)
)

func bindCommands() (err error) {

	// core console
	// ----------------------------------------------------------------------------------------
	ex, err := commandParser.AddCommand("exit", // Command string
		"Exit from the client/server console", // Description (completions, help usage)
		"",                                    // Long description
		&Exit{})                               // Command implementation
	ex.Aliases = []string{"core"}

	cd, err := commandParser.AddCommand("cd",
		"Change client working directory",
		"",
		&ChangeClientDirectory{})
	cd.Aliases = []string{"core"}

	ls, err := commandParser.AddCommand("ls",
		"List directory contents",
		"",
		&ListClientDirectories{})
	ls.Aliases = []string{"core"}

	// Log
	log, err := commandParser.AddCommand("log",
		"Manage log levels of one or more components",
		"",
		&Log{})
	log.Aliases = []string{"core"}

	// Implant generation
	// ----------------------------------------------------------------------------------------
	g, err := commandParser.AddCommand("generate",
		"Configure and compile an implant (staged or stager)",
		"",
		&Generate{})
	g.Aliases = []string{"builds"}
	g.SubcommandsOptional = true

	_, err = g.AddCommand("stager",
		"Generate a stager shellcode payload using MSFVenom, (to file: --save, to stdout: --format",
		"",
		&GenerateStager{})

	r, err := commandParser.AddCommand("regenerate",
		"Recompile an implant by name, passed as argument (completed)",
		"",
		&Regenerate{})
	r.Aliases = []string{"builds"}

	// Add choices completions (and therefore completions) to some of these commands.
	loadArgumentCompletions(commandParser)

	return
}

// Exit - Kill the current client console
type Exit struct{}

// Execute - Run
func (e *Exit) Execute(args []string) (err error) {

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Confirm exit (Y/y): ")
	text, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(text)

	if (answer == "Y") || (answer == "y") {
		os.Exit(0)
	}

	fmt.Println()
	return
}

// ChangeClientDirectory - Change the working directory of the client console
type ChangeClientDirectory struct {
	Positional struct {
		Path string `description:"local path" required:"1-1"`
	} `positional-args:"yes" required:"yes"`
}

// Execute - Handler for ChangeDirectory
func (cd *ChangeClientDirectory) Execute(args []string) (err error) {

	dir, err := fs.Expand(cd.Positional.Path)

	err = os.Chdir(dir)
	if err != nil {
		fmt.Printf(CommandError+"%s \n", err)
	} else {
		fmt.Printf(Info+"Changed directory to %s \n", dir)
	}

	return
}

// ListClientDirectories - List directory contents
type ListClientDirectories struct {
	Positional struct {
		Path []string `description:"local directory/file"`
	} `positional-args:"yes"`
}

// Execute - Command
func (ls *ListClientDirectories) Execute(args []string) error {

	base := []string{"ls", "--color", "-l"}

	if len(ls.Positional.Path) == 0 {
		ls.Positional.Path = []string{"."}
	}

	fullPaths := []string{}
	for _, path := range ls.Positional.Path {
		full, _ := fs.Expand(path)
		fullPaths = append(fullPaths, full)
	}
	base = append(base, fullPaths...)

	// Print output
	out, err := shellExec(base[0], base[1:])
	if err != nil {
		fmt.Printf(CommandError+"%s \n", err.Error())
		return nil
	}

	// Print output
	fmt.Println(out)

	return nil
}

// shellExec - Execute a program
func shellExec(executable string, args []string) (string, error) {
	path, err := exec.LookPath(executable)
	if err != nil {
		return "", err
	}

	cmd := exec.Command(path, args...)

	// Load OS environment
	cmd.Env = os.Environ()

	out, err := cmd.CombinedOutput()

	if err != nil {
		return "", err
	}
	return str.Trim(string(out)), nil
}

// Generate - Configure and compile an implant
type Generate struct {
	StageOptions // Command makes use of full stage options
}

// StageOptions - All these options, regrouped by area, are used by any command that needs full
// configuration information for a stage Sliver implant.
type StageOptions struct {
	// CoreOptions - All options about OS/arch, files to save, debugs, etc.
	CoreOptions struct {
		OS      string `long:"os" short:"o" description:"target host operating system" default:"windows" value-name:"stage OS"`
		Arch    string `long:"arch" short:"a" description:"target host CPU architecture" default:"amd64" value-name:"stage architectures"`
		Format  string `long:"format" short:"f" description:"output formats (exe, shared (DLL), service (see 'psexec' for info), shellcode (Windows only)" default:"exe" value-name:"stage formats"`
		Profile string `long:"profile-name" description:"implant profile name to use (use with generate-profile)"`
		Name    string `long:"name" short:"N" description:"implant name to use (overrides random name generation)"`
		Save    string `long:"save" short:"s" description:"directory/file where to save binary"`
		Debug   bool   `long:"debug" short:"d" description:"enable debug features (incompatible with obfuscation, and prevailing)"`
	} `group:"core options"`

	// TransportOptions - All options pertaining to transport/RPC matters
	TransportOptions struct {
		MTLS      []string `long:"mtls" short:"m" description:"mTLS C2 domain(s), comma-separated (ex: mtls://host:port)" env-delim:","`
		DNS       []string `long:"dns" short:"n" description:"DNS C2 domain(s), comma-separated (ex: dns://mydomain.com)" env-delim:","`
		HTTP      []string `long:"http" short:"h" description:"HTTP(S) C2 domain(s)" env-delim:","`
		NamedPipe []string `long:"named-pipe" short:"p" description:"Named pipe transport strings, comma-separated" env-delim:","`
		TCPPivot  []string `long:"tcp-pivot" short:"i" description:"TCP pivot transport strings, comma-separated" env-delim:","`
		Reconnect int      `long:"reconnect" short:"j" description:"attempt to reconnect every n second(s)" default:"60"`
		MaxErrors int      `long:"max-errors" short:"k" description:"max number of transport errors" default:"10"`
	} `group:"transport options"`

	// SecurityOptions - All security-oriented options like restrictions.
	SecurityOptions struct {
		LimitDatetime  string `long:"limit-datetime" short:"w" description:"limit execution to before datetime"`
		LimitDomain    bool   `long:"limit-domain-joined" short:"D" description:"limit execution to domain joined machines"`
		LimitUsername  string `long:"limit-username" short:"U" description:"limit execution to specified username"`
		LimitHosname   string `long:"limit-hostname" short:"H" description:"limit execution to specified hostname"`
		LimitFileExits string `long:"limit-file-exists" short:"F" description:"limit execution to hosts with this file in the filesystem"`
	} `group:"security options"`

	// EvasionOptions - All proactive security options (obfuscation, evasion, etc)
	EvasionOptions struct {
		Canary      []string `long:"canary" short:"c" description:"DNS canary domain strings, comma-separated" env-delim:","`
		SkipSymbols bool     `long:"skip-obfuscation" short:"b" description:"skip binary/symbol obfuscation"`
		Evasion     bool     `long:"evasion" short:"e" description:"enable evasion features"`
	} `group:"evasion options"`
}

// Execute - Configure and compile an implant
func (g *Generate) Execute(args []string) (err error) {
	save := g.CoreOptions.Save
	if save == "" {
		save, _ = os.Getwd()
	}

	fmt.Println("Executed 'generate' command. ")
	return
}

// Regenerate - Recompile an implant by name, passed as argument (completed)
type Regenerate struct {
	Positional struct {
		ImplantName string `description:"Name of Sliver implant to recompile" required:"1-1"`
	} `positional-args:"yes" required:"yes"`
	Save string `long:"save" short:"s" description:"Directory/file where to save binary"`
}

// Execute - Recompile an implant with a given profile
func (r *Regenerate) Execute(args []string) (err error) {
	fmt.Println("Executed 'regenerate' command. ")
	return
}

// GenerateStager - Generate a stager payload using MSFVenom
type GenerateStager struct {
	PayloadOptions struct {
		OS       string `long:"os" short:"o" description:"target host operating system" default:"windows" value-name:"stage OS"`
		Arch     string `long:"arch" short:"a" description:"target host CPU architecture" default:"amd64" value-name:"stage architectures"`
		Format   string `long:"msf-format" short:"f" description:"output format (MSF Venom formats). List is auto-completed" default:"raw" value-name:"MSF Venom transform formats"`
		BadChars string `long:"badchars" short:"b" description:"bytes to exclude from stage shellcode"`
		Save     string `long:"save" short:"s" description:"directory to save the generated stager to"`
	} `group:"payload options"`
	TransportOptions struct {
		LHost    string `long:"lhost" short:"l" description:"listening host address" required:"true"`
		LPort    int    `long:"lport" short:"p" description:"listening host port" default:"8443"`
		Protocol string `long:"protocol" short:"P" description:"staging protocol (tcp/http/https)" default:"tcp" value-name:"stager protocol"`
	} `group:"transport options"`
}

// Execute - Generate a stager payload using MSFVenom
func (g *GenerateStager) Execute(args []string) (err error) {
	fmt.Println("Executed 'generate stager' subcommand. ")
	return
}

// Log - Log management commands. Sets log level by default.
type Log struct {
	Positional struct {
		Level      string   `description:"log level to filter by" required:"1-1"`
		Components []string `description:"components on which to apply log filter" required:"1"`
	} `positional-args:"yes" required:"true"`
}

// Execute - Set the log level of one or more components
func (l *Log) Execute(args []string) (err error) {
	fmt.Println("Executed 'log' command. ")
	return
}

var (
	Info    = fmt.Sprintf("%s[-]%s ", readline.BLUE, readline.RESET)
	Warn    = fmt.Sprintf("%s[!]%s ", readline.YELLOW, readline.RESET)
	Error   = fmt.Sprintf("%s[!]%s ", readline.RED, readline.RESET)
	Success = fmt.Sprintf("%s[*]%s ", readline.GREEN, readline.RESET)

	Infof   = fmt.Sprintf("%s[-] ", readline.BLUE)   // Infof - formatted
	Warnf   = fmt.Sprintf("%s[!] ", readline.YELLOW) // Warnf - formatted
	Errorf  = fmt.Sprintf("%s[!] ", readline.RED)    // Errorf - formatted
	Sucessf = fmt.Sprintf("%s[*] ", readline.GREEN)  // Sucessf - formatted

	RPCError     = fmt.Sprintf("%s[RPC Error]%s ", readline.RED, readline.RESET)
	CommandError = fmt.Sprintf("%s[Command Error]%s ", readline.RED, readline.RESET)
	ParserError  = fmt.Sprintf("%s[Parser Error]%s ", readline.RED, readline.RESET)
	DBError      = fmt.Sprintf("%s[DB Error]%s ", readline.RED, readline.RESET)
)
