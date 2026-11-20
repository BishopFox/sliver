// Package mage provides the core functionality for the mage build tool.
package mage

import (
	"context"
	"crypto/sha256"
	_ "embed" // so we can use //go:embed for the magefile template and colors
	"errors"
	"flag"
	"fmt"
	"go/build"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	dbg "runtime/debug"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/magefile/mage/internal"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/parse"
	"github.com/magefile/mage/sh"
)

// magicRebuildKey is used when hashing the output binary to ensure that we get
// a new binary even if nothing in the input files or generated mainfile has
// changed. This can be used when we change how we parse files, or otherwise
// change the inputs to the compiling process.
const magicRebuildKey = "v0.3"

// (Aaaa)(Bbbb) -> aaaaBbbb.
var firstWordRx = regexp.MustCompile(`^([[:upper:]][^[:upper:]]+)([[:upper:]].*)$`)

// (AAAA)(Bbbb) -> aaaaBbbb.
var firstAbbrevRx = regexp.MustCompile(`^([[:upper:]]+)([[:upper:]][^[:upper:]].*)$`)

func lowerFirstWord(s string) string {
	if match := firstWordRx.FindStringSubmatch(s); match != nil {
		return strings.ToLower(match[1]) + match[2]
	}
	if match := firstAbbrevRx.FindStringSubmatch(s); match != nil {
		return strings.ToLower(match[1]) + match[2]
	}
	return strings.ToLower(s)
}

var mainfileTemplate = template.Must(template.New("").Funcs(map[string]interface{}{
	"lower": strings.ToLower,
	"lowerFirst": func(s string) string {
		parts := strings.Split(s, ":")
		for i, t := range parts {
			parts[i] = lowerFirstWord(t)
		}
		return strings.Join(parts, ":")
	},
}).Parse(mageMainfileTplString))
var initOutput = template.Must(template.New("").Parse(mageTpl))

const (
	mainfile = "mage_output_file.go"
	initFile = "magefile.go"
)

var debug = log.New(io.Discard, "DEBUG: ", log.Ltime|log.Lmicroseconds)

//go:generate stringer -type=Command

// Command tracks invocations of mage that run without targets or other flags.
type Command int

// The various command types.
const (
	None          Command = iota
	Version               // report the current version of mage
	Init                  // create a starting template for mage
	Clean                 // clean out old compiled mage binaries from the cache
	CompileStatic         // compile a static binary of the current directory
	Install               // install shell tab completion
)

// Main is the entrypoint for running mage.  It exists external to mage's main
// function to allow it to be used from other programs, specifically so you can
// go run a simple file that run's mage's Main.
func Main() int {
	return ParseAndRun(os.Stdout, os.Stderr, os.Stdin, os.Args[1:])
}

// Invocation contains the args for invoking a run of Mage.
type Invocation struct {
	Debug        bool          // turn on debug messages
	Dir          string        // directory to read magefiles from
	WorkDir      string        // directory where magefiles will run
	Force        bool          // forces recreation of the compiled binary
	Verbose      bool          // tells the magefile to print out log statements
	List         bool          // tells the magefile to print out a list of targets
	Help         bool          // tells the magefile to print out help for a specific target
	Keep         bool          // tells mage to keep the generated main file after compiling
	Timeout      time.Duration // tells mage to set a timeout to running the targets
	CompileOut   string        // tells mage to compile a static binary to this path, but not execute
	GOOS         string        // sets the GOOS when producing a binary with -compileout
	GOARCH       string        // sets the GOARCH when producing a binary with -compileout
	Ldflags      string        // sets the ldflags when producing a binary with -compileout
	Stdout       io.Writer     // writer to write stdout messages to
	Stderr       io.Writer     // writer to write stderr messages to
	Stdin        io.Reader     // reader to read stdin from
	Args         []string      // args to pass to the compiled binary
	GoCmd        string        // the go binary command to run
	CacheDir     string        // the directory where we should store compiled binaries
	HashFast     bool          // don't rely on GOCACHE, just hash the magefiles
	Multiline    bool          // whether to retain line returns in help text for the generated main file
	Autocomplete bool          // parse magefiles and print target names for shell completion
	InstallShell string        // shell to install tab completion for (bash, zsh, fish, powershell/pwsh)
}

// MagefilesDirName is the name of the default folder to look for if no directory was specified,
// if this folder exists it will be assumed mage package lives inside it.
const MagefilesDirName = "magefiles"

// UsesMagefiles returns true if we are getting our mage files from a magefiles directory.
func (i Invocation) UsesMagefiles() bool {
	return filepath.Base(i.Dir) == MagefilesDirName
}

// ParseAndRun parses the command line, and then compiles and runs the mage
// files in the given directory with the given args (do not include the command
// name in the args).
func ParseAndRun(stdout, stderr io.Writer, stdin io.Reader, args []string) int {
	errlog := log.New(stderr, "", 0)
	out := log.New(stdout, "", 0)
	inv, cmd, err := Parse(stderr, stdout, args)
	inv.Stderr = stderr
	inv.Stdin = stdin
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		errlog.Println("Error:", err)
		return 2
	}

	switch cmd {
	case Version:
		doVersion(out)
		return 0
	case Init:
		if err := generateInit(inv.Dir); err != nil {
			errlog.Println("Error:", err)
			return 1
		}
		out.Println(initFile, "created")
		return 0
	case Clean:
		if err := removeContents(inv.CacheDir); err != nil {
			out.Println("Error:", err)
			return 1
		}
		out.Println(inv.CacheDir, "cleaned")
		return 0
	case Install:
		if err := installCompletion(stdout, inv.InstallShell); err != nil {
			errlog.Println("Error:", err)
			return 1
		}
		return 0
	case CompileStatic, None:
		return Invoke(inv)
	default:
		errlog.Printf("unknown command type: %v", cmd)
		return 1
	}
}

func doVersion(out *log.Logger) {
	var (
		commitHash = "<not set>"
		timestamp  = "<not set>"
		gitTag     = ""
	)

	info, ok := dbg.ReadBuildInfo()
	if ok {
		if info.Main.Version != "" {
			gitTag = info.Main.Version
		}
		for _, kv := range info.Settings {
			switch kv.Key {
			case "vcs.revision":
				commitHash = kv.Value
			case "vcs.time":
				timestamp = kv.Value
			default:
				continue
			}
		}
	}
	out.Println("Mage Build Tool", gitTag)
	out.Println("Build Date:", timestamp)
	out.Println("Commit:", commitHash)
	out.Println("built with:", runtime.Version())
}

// Parse parses the given args and returns structured data.  If parse returns
// flag.ErrHelp, the calling process should exit with code 0.
func Parse(stderr, stdout io.Writer, args []string) (inv Invocation, cmd Command, err error) {
	inv.Stdout = stdout
	fs := flag.FlagSet{}
	fs.SetOutput(stdout)

	// options flags

	fs.BoolVar(&inv.Force, "f", false, "force recreation of compiled magefile")
	fs.BoolVar(&inv.Debug, "debug", mg.Debug(), "turn on debug messages")
	fs.BoolVar(&inv.Verbose, "v", mg.Verbose(), "show verbose output when running mage targets")
	fs.BoolVar(&inv.Multiline, "multiline", mg.Multiline(), "retain line returns in help text")
	fs.BoolVar(&inv.Help, "h", false, "show this help")
	fs.DurationVar(&inv.Timeout, "t", 0, "timeout in duration parsable format (e.g. 5m30s)")
	fs.BoolVar(&inv.Keep, "keep", false, "keep intermediate mage files around after running")
	fs.StringVar(&inv.Dir, "d", "", "directory to read magefiles from")
	fs.StringVar(&inv.WorkDir, "w", "", "working directory where magefiles will run")
	fs.StringVar(&inv.GoCmd, "gocmd", mg.GoCmd(), "use the given go binary to compile the output")
	fs.StringVar(&inv.GOOS, "goos", "", "set GOOS for binary produced with -compile")
	fs.StringVar(&inv.GOARCH, "goarch", "", "set GOARCH for binary produced with -compile")
	fs.StringVar(&inv.Ldflags, "ldflags", "", "set ldflags for binary produced with -compile")
	fs.BoolVar(&inv.Autocomplete, "autocomplete", false, "print target names for shell completion, without compiling")

	// commands below

	fs.BoolVar(&inv.List, "l", false, "list mage targets in this directory")
	var showVersion bool
	fs.BoolVar(&showVersion, "version", false, "show version info for the mage binary")
	var mageInit bool
	fs.BoolVar(&mageInit, "init", false, "create a starting template if no mage files exist")
	var clean bool
	fs.BoolVar(&clean, "clean", false, "clean out old generated binaries from CACHE_DIR")
	var compileOutPath string
	fs.StringVar(&compileOutPath, "compile", "", "output a static binary to the given path")
	var installShell string
	fs.StringVar(&installShell, "install", "", "install shell tab completion (bash, zsh, fish, powershell/pwsh)")

	fs.Usage = func() {
		_, _ = fmt.Fprint(stdout, `
mage [options] [target]

Mage is a make-like command runner.  See https://magefile.org for full docs.

Commands:
  -autocomplete
            print target names for shell completion, without compiling
  -clean    clean out old generated binaries from CACHE_DIR
  -compile <string>
            output a static binary to the given path
  -h        show this help
  -init     create a starting template if no mage files exist
  -install <string>
            install shell tab completion (bash, zsh, fish, powershell/pwsh)
  -l        list mage targets in this directory
  -version  show version info for the mage binary

Options:
  -d <string> 
              directory to read magefiles from (default "." or "magefiles" if exists)
  -debug      turn on debug messages
  -f          force recreation of compiled magefile
  -goarch     sets the GOARCH for the binary created by -compile (default: current arch)
  -gocmd <string>
		      use the given go binary to compile the output (default: "go")
  -goos       sets the GOOS for the binary created by -compile (default: current OS)
  -ldflags    sets the ldflags for the binary created by -compile (default: "")
  -multiline  retain line returns in help docs (default: convert to spaces)
  -h          show description of a target
  -keep       keep intermediate mage files around after running
  -t <string>
              timeout in duration parsable format (e.g. 5m30s)
  -v          show verbose output when running mage targets
  -w <string>
              working directory where magefiles will run (default -d value)
`[1:])
	}
	err = fs.Parse(args)
	if errors.Is(err, flag.ErrHelp) {
		// parse will have already called fs.Usage()
		return inv, cmd, err
	}
	if err == nil && inv.Help && len(fs.Args()) == 0 {
		fs.Usage()
		// tell upstream, to just exit
		return inv, cmd, flag.ErrHelp
	}

	numCommands := 0
	switch {
	case mageInit:
		numCommands++
		cmd = Init
	case compileOutPath != "":
		numCommands++
		cmd = CompileStatic
		inv.CompileOut = compileOutPath
		inv.Force = true
	case installShell != "":
		numCommands++
		cmd = Install
		inv.InstallShell = installShell
	case showVersion:
		numCommands++
		cmd = Version
	case clean:
		numCommands++
		cmd = Clean
		if fs.NArg() > 0 {
			return inv, cmd, errors.New("-h, -init, -clean, -compile, -install, -autocomplete and -version cannot be used simultaneously")
		}
	default:
		// no command flags set
	}
	if inv.Help {
		numCommands++
	}
	if inv.Autocomplete {
		numCommands++
	}

	if inv.Debug {
		debug.SetOutput(stderr)
	}

	inv.CacheDir = mg.CacheDir()

	if numCommands > 1 {
		debug.Printf("%d commands defined", numCommands)
		return inv, cmd, errors.New("-h, -init, -clean, -compile, -install, -autocomplete and -version cannot be used simultaneously")
	}

	if cmd != CompileStatic && (inv.GOARCH != "" || inv.GOOS != "") {
		return inv, cmd, errors.New("-goos and -goarch only apply when running with -compile")
	}

	inv.Args = fs.Args()
	if inv.Help && len(inv.Args) > 1 {
		return inv, cmd, errors.New("-h can only show help for a single target")
	}

	if len(inv.Args) > 0 && cmd != None {
		return inv, cmd, fmt.Errorf("unexpected arguments to command: %q", inv.Args)
	}
	inv.HashFast = mg.HashFast()
	return inv, cmd, err
}

const dotDirectory = "."

//go:embed colors.go
var colorsFile string

// Invoke runs Mage with the given arguments.
func Invoke(inv Invocation) int {
	errlog := log.New(inv.Stderr, "", 0)
	if inv.GoCmd == "" {
		inv.GoCmd = "go"
	}
	if inv.Dir == "" {
		inv.Dir = dotDirectory
	}
	if inv.WorkDir == "" {
		inv.WorkDir = inv.Dir
	}
	magefilesDir := filepath.Join(inv.Dir, MagefilesDirName)
	// . will be default unless we find a mage folder.
	mfSt, err := os.Stat(magefilesDir)
	if err == nil && mfSt.IsDir() {
		originalDir := inv.Dir
		inv.Dir = magefilesDir // preemptive assignment
		// TODO: Remove this fallback and the above Magefiles invocation when the bw compatibility is removed.
		existingFiles, mfErr := Magefiles(originalDir, inv.GOOS, inv.GOARCH, inv.Debug)
		if mfErr == nil && len(existingFiles) != 0 {
			errlog.Println("[WARNING] You have both a magefiles directory and mage files in the " +
				"current directory, in future versions the files will be ignored in favor of the directory")
			inv.Dir = originalDir
		}
	}

	if inv.CacheDir == "" {
		inv.CacheDir = mg.CacheDir()
	}

	files, err := Magefiles(inv.Dir, inv.GOOS, inv.GOARCH, inv.UsesMagefiles())
	if err != nil {
		errlog.Println("Error determining list of magefiles:", err)
		return 1
	}

	if len(files) == 0 {
		errlog.Println("No .go files marked with the mage build tag in this directory.")
		return 1
	}
	debug.Printf("found magefiles: %s", strings.Join(files, ", "))
	exePath := inv.CompileOut
	if inv.CompileOut == "" {
		exePath, err = ExeName(inv.GoCmd, inv.CacheDir, files)
		if err != nil {
			errlog.Println("Error getting exe name:", err)
			return 1
		}
	}
	debug.Println("output exe is ", exePath)

	useCache := false
	if inv.HashFast {
		debug.Println("user has set MAGEFILE_HASHFAST, so we'll ignore GOCACHE")
	} else {
		gocache, gocacheErr := internal.OutputDebug(inv.GoCmd, "env", "GOCACHE")
		if gocacheErr != nil {
			errlog.Printf("failed to run %s env GOCACHE: %s", inv.GoCmd, gocacheErr)
			return 1
		}

		// if GOCACHE exists, always rebuild, so we catch transitive
		// dependencies that have changed.
		if gocache != "" {
			debug.Println("go build cache exists, will ignore any compiled binary")
			useCache = true
		}
	}

	if !useCache {
		_, err = os.Stat(exePath)
		switch {
		case err == nil:
			if !inv.Force {
				debug.Println("Running existing exe")
				return RunCompiled(inv, exePath, errlog)
			}
			debug.Println("ignoring existing executable")
		case os.IsNotExist(err):
			debug.Println("no existing exe, creating new")
		default:
			debug.Printf("error reading existing exe at %v: %v", exePath, err)
			debug.Println("creating new exe")
		}
	}

	// parse wants dir + filenames... arg
	fnames := make([]string, 0, len(files))
	for i := range files {
		fnames = append(fnames, filepath.Base(files[i]))
	}
	if inv.Debug {
		parse.EnableDebug()
	}
	debug.Println("parsing files")
	info, err := parse.PrimaryPackage(inv.GoCmd, inv.Dir, fnames, inv.Multiline)
	if err != nil {
		errlog.Println("Error parsing magefiles:", err)
		return 1
	}

	if inv.Autocomplete {
		return printAutocompleteTargets(inv.Stdout, info)
	}

	// reproducible output for deterministic builds
	sort.Sort(info.Funcs)
	sort.Sort(info.Imports)

	binaryName := "mage"
	if inv.CompileOut != "" {
		binaryName = filepath.Base(inv.CompileOut)
	}

	data := mainfileTemplateData{
		Description: info.Description,
		Funcs:       info.Funcs,
		Aliases:     info.Aliases,
		Imports:     info.Imports,
		BinaryName:  binaryName,
	}

	if info.DefaultFunc != nil {
		data.DefaultFunc = *info.DefaultFunc
	}

	if inv.List {
		_, _ = fmt.Fprint(inv.Stdout, mageListOutput(data, info))
		return 0
	}

	if inv.Help {
		if len(inv.Args) < 1 {
			_, _ = fmt.Fprintln(inv.Stderr, "no target specified")
			return 2
		}
		output, code := mageHelpOutput(data, inv.Args[0])
		if code != 0 {
			_, _ = fmt.Fprint(inv.Stderr, output)
		} else {
			_, _ = fmt.Fprint(inv.Stdout, output)
		}
		return code
	}

	// ensure we use the same color output code in the generated mainfile as we do in mage's own output.
	idx := strings.Index(colorsFile, "var printName =")
	if idx == -1 {
		panic(errors.New("unable to find printName func in colorsFile colors.go"))
	}
	data.PrintNameFunc = colorsFile[idx:]

	main := filepath.Join(inv.Dir, mainfile)
	err = GenerateMainfile(data, main)
	if err != nil {
		errlog.Println("Error:", err)
		return 1
	}
	if !inv.Keep {
		defer func() { _ = os.RemoveAll(main) }()
	}
	files = append(files, main)
	if err := Compile(inv.GOOS, inv.GOARCH, inv.Ldflags, inv.Dir, inv.GoCmd, exePath, files, inv.Debug, inv.Stderr, inv.Stdout); err != nil {
		errlog.Println("Error:", err)
		return 1
	}
	if !inv.Keep {
		// move aside this file before we run the compiled version, in case the
		// compiled file screws things up.  Yes this doubles up with the above
		// defer, that's ok.
		_ = os.RemoveAll(main)
	} else {
		debug.Print("keeping mainfile")
	}

	if inv.CompileOut != "" {
		return 0
	}

	return RunCompiled(inv, exePath, errlog)
}

func mageListOutput(data mainfileTemplateData, info *parse.PkgInfo) string {
	list := strings.Builder{}

	lowerFirst := func(s string) string {
		parts := strings.Split(s, ":")
		for i, t := range parts {
			parts[i] = lowerFirstWord(t)
		}
		return strings.Join(parts, ":")
	}

	var defaultFunc parse.Function
	if info.DefaultFunc != nil {
		defaultFunc = *info.DefaultFunc
	}

	if data.Description != "" {
		_, _ = fmt.Fprintf(&list, "%s\n\n", data.Description)
	}

	targets := map[string]string{}
	for _, f := range data.Funcs {
		name := lowerFirst(f.TargetName())
		if f.Name == defaultFunc.Name && f.Receiver == defaultFunc.Receiver {
			name += "*"
		}
		targets[name] = f.Synopsis
	}
	for _, imp := range data.Imports {
		for _, f := range imp.Info.Funcs {
			name := lowerFirst(f.TargetName())
			if f.Name == defaultFunc.Name && f.Receiver == defaultFunc.Receiver {
				name += "*"
			}
			targets[name] = f.Synopsis
		}
	}

	keys := make([]string, 0, len(targets))
	for name := range targets {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	_, _ = fmt.Fprintln(&list, "Targets:")
	w := tabwriter.NewWriter(&list, 0, 4, 4, ' ', 0)
	for _, name := range keys {
		_, _ = fmt.Fprintf(w, "  %v\t%v\n", printName(name), targets[name])
	}
	_ = w.Flush()
	if defaultFunc.Name != "" {
		_, _ = fmt.Fprintln(&list, "\n* default target")
	}
	return list.String()
}

// mageHelpOutput generates help text for a single target without compiling.
// It returns the formatted output string and an exit code (0 for success, 2 for errors).
// The output matches what a compiled binary produces for -h.
func mageHelpOutput(data mainfileTemplateData, target string) (output string, code int) {
	target = strings.ToLower(target)

	// Collect all functions from main package and imports.
	var allFuncs []*parse.Function
	allFuncs = append(allFuncs, data.Funcs...)
	for _, imp := range data.Imports {
		allFuncs = append(allFuncs, imp.Info.Funcs...)
	}

	// Find the matching function.
	var fn *parse.Function
	for _, f := range allFuncs {
		if strings.ToLower(f.TargetName()) == target {
			fn = f
			break
		}
	}
	if fn == nil {
		return fmt.Sprintf("Unknown target: %q\n", target), 2
	}

	var buf strings.Builder

	if fn.Comment != "" {
		_, _ = fmt.Fprintln(&buf, fn.Comment)
		_, _ = fmt.Fprintln(&buf)
	}

	// Build usage line matching template format.
	_, _ = fmt.Fprintf(&buf, "Usage:\n\n\t%s %s", data.BinaryName, strings.ToLower(fn.TargetName()))
	for _, a := range fn.RequiredArgs() {
		_, _ = fmt.Fprintf(&buf, " <%s>", a.Name)
	}
	if fn.MultipleOptionalArgs() {
		_, _ = fmt.Fprint(&buf, " [<flags>]")
	} else {
		for _, a := range fn.OptionalArgs() {
			_, _ = fmt.Fprintf(&buf, " [-%s=<%s>]", a.Name, a.Type)
		}
	}
	_, _ = fmt.Fprint(&buf, "\n\n")

	if fn.ShowFlagDocs() {
		_, _ = fmt.Fprint(&buf, fn.FlagDocsString())
	}

	// Collect and sort aliases for deterministic output.
	var aliases []string
	for alias, af := range data.Aliases {
		if af.Name == fn.Name && af.Receiver == fn.Receiver {
			aliases = append(aliases, alias)
		}
	}
	if len(aliases) > 0 {
		sort.Strings(aliases)
		_, _ = fmt.Fprintf(&buf, "Aliases: %s\n\n", strings.Join(aliases, ", "))
	}

	return buf.String(), 0
}

// printAutocompleteTargets outputs target names one per line for shell completion.
func printAutocompleteTargets(stdout io.Writer, info *parse.PkgInfo) int {
	names := map[string]struct{}{}

	for _, f := range info.Funcs {
		names[strings.ToLower(f.TargetName())] = struct{}{}
	}
	for _, imp := range info.Imports {
		for _, f := range imp.Info.Funcs {
			names[strings.ToLower(f.TargetName())] = struct{}{}
		}
	}
	for alias := range info.Aliases {
		names[strings.ToLower(alias)] = struct{}{}
	}

	sorted := make([]string, 0, len(names))
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)

	for _, name := range sorted {
		_, _ = fmt.Fprintln(stdout, name)
	}
	return 0
}

type mainfileTemplateData struct {
	Description   string
	Funcs         []*parse.Function
	DefaultFunc   parse.Function
	Aliases       map[string]*parse.Function
	Imports       []*parse.Import
	BinaryName    string
	PrintNameFunc string
}

// listGoFiles returns a list of all .go files in a given directory,
// matching the provided tag.
func listGoFiles(magePath, tag string, envStr []string) ([]string, error) {
	origMagePath := magePath
	if !filepath.IsAbs(magePath) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("can't get current working directory: %w", err)
		}
		magePath = filepath.Join(cwd, magePath)
	}

	env, err := internal.SplitEnv(envStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing environment variables: %w", err)
	}

	bctx := build.Default
	bctx.BuildTags = []string{tag}

	if _, ok := env["GOOS"]; ok {
		bctx.GOOS = env["GOOS"]
	}

	if _, ok := env["GOARCH"]; ok {
		bctx.GOARCH = env["GOARCH"]
	}

	pkg, err := bctx.Import(".", magePath, 0)
	if err != nil {
		var noGoErr *build.NoGoError
		if errors.As(err, &noGoErr) {
			return []string{}, nil
		}

		// Allow multiple packages in the same directory.
		var multiplePkgErr *build.MultiplePackageError
		if !errors.As(err, &multiplePkgErr) {
			return nil, fmt.Errorf("failed to parse go source files: %w", err)
		}
	}

	goFiles := make([]string, len(pkg.GoFiles))
	for i := range pkg.GoFiles {
		goFiles[i] = filepath.Join(origMagePath, pkg.GoFiles[i])
	}

	debug.Printf("found %d go files with build tag %s (files: %v)", len(goFiles), tag, goFiles)
	return goFiles, nil
}

// Magefiles returns the list of magefiles in dir.
func Magefiles(magePath, goos, goarch string, isMagefilesDirectory bool) ([]string, error) {
	start := time.Now()
	defer func() {
		debug.Println("time to scan for Magefiles:", time.Since(start))
	}()

	env, err := internal.EnvWithGOOS(goos, goarch)
	if err != nil {
		return nil, err
	}

	debug.Println("getting all files including those with mage tag in", magePath)
	mageFiles, err := listGoFiles(magePath, "mage", env)
	if err != nil {
		return nil, fmt.Errorf("listing mage files: %w", err)
	}

	if isMagefilesDirectory {
		// For the magefiles directory, we always use all go files, both with
		// and without the mage tag, as per normal go build tag rules.
		debug.Println("using all go files in magefiles directory", magePath)
		return mageFiles, nil
	}

	// For folders other than the magefiles directory, we only consider files
	// that have the mage build tag and ignore those that don't.

	debug.Println("getting all files without mage tag in", magePath)
	nonMageFiles, err := listGoFiles(magePath, "", env)
	if err != nil {
		return nil, fmt.Errorf("listing non-mage files: %w", err)
	}

	// convert non-Mage list to a map of files to exclude.
	exclude := map[string]bool{}
	for _, f := range nonMageFiles {
		if f != "" {
			debug.Printf("marked file as non-mage: %q", f)
			exclude[f] = true
		}
	}

	// filter out the non-mage files from the mage files.
	var files []string
	for _, f := range mageFiles {
		if f != "" && !exclude[f] {
			files = append(files, f)
		}
	}

	return files, nil
}

// Compile uses the go tool to compile the files into an executable at path.
func Compile(goos, goarch, ldflags, magePath, goCmd, compileTo string, gofiles []string, isDebug bool, stderr, stdout io.Writer) error {
	debug.Println("compiling to", compileTo)
	debug.Println("compiling using gocmd:", goCmd)
	if isDebug {
		_ = internal.RunDebug(goCmd, "version")
		_ = internal.RunDebug(goCmd, "env")
	}
	environ, err := internal.EnvWithGOOS(goos, goarch)
	if err != nil {
		return err
	}
	// strip off the path since we're setting the path in the build command
	for i := range gofiles {
		gofiles[i] = filepath.Base(gofiles[i])
	}
	buildArgs := []string{"build", "-o", compileTo}
	if ldflags != "" {
		buildArgs = append(buildArgs, "-ldflags", ldflags)
	}
	args := append([]string{}, buildArgs...)
	args = append(args, gofiles...)

	debug.Printf("running %s %s", goCmd, strings.Join(args, " "))
	c := exec.CommandContext(context.Background(), goCmd, args...)
	c.Env = environ
	c.Stderr = stderr
	c.Stdout = stdout
	c.Dir = magePath
	start := time.Now()
	err = c.Run()
	debug.Println("time to compile Magefile:", time.Since(start))
	if err != nil {
		return errors.New("error compiling magefiles")
	}
	return nil
}

// GenerateMainfile generates the mage mainfile at path.
func GenerateMainfile(data mainfileTemplateData, path string) error {
	debug.Println("Creating mainfile at", path)

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating generated mainfile: %w", err)
	}
	defer func() { _ = f.Close() }()

	debug.Println("writing new file at", path)
	if err := mainfileTemplate.Execute(f, data); err != nil {
		return fmt.Errorf("can't execute mainfile template: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("error closing generated mainfile: %w", err)
	}
	// we set an old modtime on the generated mainfile so that the go tool
	// won't think it has changed more recently than the compiled binary.
	longAgo := time.Now().Add(-time.Hour * 24 * 365 * 10)
	if err := os.Chtimes(path, longAgo, longAgo); err != nil {
		return fmt.Errorf("error setting old modtime on generated mainfile: %w", err)
	}
	return nil
}

// ExeName reports the executable filename that this version of Mage would
// create for the given magefiles.
func ExeName(goCmd, cacheDir string, files []string) (string, error) {
	var hashes []string
	for _, s := range files {
		h, err := hashFile(s)
		if err != nil {
			return "", err
		}
		hashes = append(hashes, h)
	}
	// hash the mainfile template to ensure if it gets updated, we make a new
	// binary.
	hashes = append(hashes, fmt.Sprintf("%x", sha256.Sum256([]byte(mageMainfileTplString))))
	sort.Strings(hashes)
	ver, err := internal.OutputDebug(goCmd, "version")
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(strings.Join(hashes, "") + magicRebuildKey + ver))
	filename := fmt.Sprintf("%x", hash)

	out := filepath.Join(cacheDir, filename)
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	return out, nil
}

func hashFile(fn string) (string, error) {
	f, err := os.Open(fn)
	if err != nil {
		return "", fmt.Errorf("can't open input file for hashing: %w", err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("can't write data to hash: %w", err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func generateInit(dir string) error {
	debug.Println("generating default magefile in", dir)
	f, err := os.Create(filepath.Join(dir, initFile))
	if err != nil {
		return fmt.Errorf("could not create mage template: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := initOutput.Execute(f, nil); err != nil {
		return fmt.Errorf("can't execute magefile template: %w", err)
	}

	return nil
}

// RunCompiled runs an already-compiled mage command with the given args.
func RunCompiled(inv Invocation, exePath string, errlog *log.Logger) int {
	debug.Println("running binary", exePath)
	c := exec.CommandContext(context.Background(), exePath, inv.Args...)
	c.Stderr = inv.Stderr
	c.Stdout = inv.Stdout
	c.Stdin = inv.Stdin
	c.Dir = inv.Dir
	if inv.WorkDir != inv.Dir {
		c.Dir = inv.WorkDir
	}
	// intentionally pass through unaltered os.Environ here.. your magefile has
	// to deal with it.
	c.Env = os.Environ()
	if inv.Verbose {
		c.Env = append(c.Env, "MAGEFILE_VERBOSE=1")
	}
	if inv.List {
		c.Env = append(c.Env, "MAGEFILE_LIST=1")
	}
	if inv.Help {
		c.Env = append(c.Env, "MAGEFILE_HELP=1")
	}
	if inv.Debug {
		c.Env = append(c.Env, "MAGEFILE_DEBUG=1")
	}
	if inv.GoCmd != "" {
		c.Env = append(c.Env, fmt.Sprintf("MAGEFILE_GOCMD=%s", inv.GoCmd))
	}
	if inv.Timeout > 0 {
		c.Env = append(c.Env, fmt.Sprintf("MAGEFILE_TIMEOUT=%s", inv.Timeout.String()))
	}
	debug.Print("running magefile with mage vars:\n", strings.Join(filter(c.Env, "MAGEFILE"), "\n"))
	// catch SIGINT to allow magefile to handle them
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	defer signal.Stop(sigCh)
	err := c.Run()
	if !sh.CmdRan(err) {
		errlog.Printf("failed to run compiled magefile: %v", err)
	}
	return sh.ExitStatus(err)
}

func filter(list []string, prefix string) []string {
	var out []string
	for _, s := range list {
		if strings.HasPrefix(s, prefix) {
			out = append(out, s)
		}
	}
	return out
}

// removeContents removes all files but not any subdirectories in the given
// directory.
func removeContents(dir string) error {
	debug.Println("removing all files in", dir)
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		err = os.Remove(filepath.Join(dir, f.Name()))
		if err != nil {
			return err
		}
	}
	return nil
}
