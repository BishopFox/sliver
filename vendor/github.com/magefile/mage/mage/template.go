package mage

// this template uses the "data"

// var only for tests.
var mageMainfileTplString = `//go:build ignore
// +build ignore

package main

import (
	_context "context"
	_flag "flag"
	_fmt "fmt"
	_io "io"
	_log "log"
	_os "os"
	_signal "os/signal"
	_filepath "path/filepath"
	_sort "sort"
	_strconv "strconv"
	_strings "strings"
	_syscall "syscall"
	_tabwriter "text/tabwriter"
	_time "time"
	{{range .Imports}}{{.UniqueName}} "{{.Path}}"
	{{end}}
)

func main() {
	// Use local types and functions in order to avoid name conflicts with additional magefiles.
	type arguments struct {
		Verbose       bool          // print out log statements
		List          bool          // print out a list of targets
		Help          bool          // print out help for a specific target
		Timeout       _time.Duration // set a timeout to running the targets
		Args          []string      // args contain the non-flag command-line arguments
	}

	parseBool := func(env string) bool {
		val := _os.Getenv(env)
		if val == "" {
			return false
		}		
		b, err := _strconv.ParseBool(val)
		if err != nil {
			_log.Printf("warning: environment variable %s is not a valid bool value: %v", env, val)
			return false
		}
		return b
	}

	parseDuration := func(env string) _time.Duration {
		val := _os.Getenv(env)
		if val == "" {
			return 0
		}		
		d, err := _time.ParseDuration(val)
		if err != nil {
			_log.Printf("warning: environment variable %s is not a valid duration value: %v", env, val)
			return 0
		}
		return d
	}
	args := arguments{}
	fs := _flag.FlagSet{}
	fs.SetOutput(_os.Stdout)

	// default flag set with ExitOnError and auto generated PrintDefaults should be sufficient
	fs.BoolVar(&args.Verbose, "v", parseBool("MAGEFILE_VERBOSE"), "show verbose output when running targets")
	fs.BoolVar(&args.List, "l", parseBool("MAGEFILE_LIST"), "list targets for this binary")
	fs.BoolVar(&args.Help, "h", parseBool("MAGEFILE_HELP"), "print out help for a specific target")
	fs.DurationVar(&args.Timeout, "t", parseDuration("MAGEFILE_TIMEOUT"), "timeout in duration parsable format (e.g. 5m30s)")
	fs.Usage = func() {
		_fmt.Fprintf(_os.Stdout, ` + "`" + `
%s [options] [target]

Commands:
  -l    list targets in this binary
  -h    show this help

Options:
  -h    show description of a target
  -t <string>
        timeout in duration parsable format (e.g. 5m30s)
  -v    show verbose output when running targets
 ` + "`" + `[1:], _filepath.Base(_os.Args[0]))
	}
	if err := fs.Parse(_os.Args[1:]); err != nil {
		// flag will have printed out an error already.
		return
	}
	args.Args = fs.Args()
	if args.Help && len(args.Args) == 0 {
		fs.Usage()
		return
	}
		
	{{.PrintNameFunc}}

	list := func() error {
		{{with .Description}}_fmt.Println(` + "`{{.}}\n`" + `)
		{{- end}}
		{{- $default := .DefaultFunc}}
		targets := map[string]string{
		{{- range .Funcs}}
			"{{lowerFirst .TargetName}}{{if and (eq .Name $default.Name) (eq .Receiver $default.Receiver)}}*{{end}}": {{printf "%q" .Synopsis}},
		{{- end}}
		{{- range .Imports}}{{$imp := .}}
			{{- range .Info.Funcs}}
			"{{lowerFirst .TargetName}}{{if and (eq .Name $default.Name) (eq .Receiver $default.Receiver)}}*{{end}}": {{printf "%q" .Synopsis}},
			{{- end}}
		{{- end}}
		}

		keys := make([]string, 0, len(targets))
		for name := range targets {
			keys = append(keys, name)
		}
		_sort.Strings(keys)

		_fmt.Println("Targets:")
		w := _tabwriter.NewWriter(_os.Stdout, 0, 4, 4, ' ', 0)
		for _, name := range keys {
			_fmt.Fprintf(w, "  %v\t%v\n", printName(name), targets[name])
		}
		err := w.Flush()
		{{- if .DefaultFunc.Name}}
			if err == nil {
				_fmt.Println("\n* default target")
			}
		{{- end}}
		return err
	}

	var ctx _context.Context
	ctxCancel := func(){}

	// by deferring in a closure, we let the cancel function get replaced
	// by the getContext function.
	defer func() {
		ctxCancel()
	}()

	getContext := func() (_context.Context, func()) {
		if ctx == nil {
			if args.Timeout != 0 {
				ctx, ctxCancel = _context.WithTimeout(_context.Background(), args.Timeout)
			} else {
				ctx, ctxCancel = _context.WithCancel(_context.Background())
			}
		}

		return ctx, ctxCancel
	}

	runTarget := func(logger *_log.Logger, fn func(_context.Context) error) interface{} {
		var err interface{}
		ctx, cancel := getContext()
		d := make(chan interface{})
		go func() {
			defer func() {
				err := recover()
				d <- err
			}()
			err := fn(ctx)
			d <- err
		}()
		sigCh := make(chan _os.Signal, 1)
		_signal.Notify(sigCh, _syscall.SIGINT)
		select {
		case <-sigCh:
			logger.Println("cancelling mage targets, waiting up to 5 seconds for cleanup...")
			cancel()
			cleanupCh := _time.After(5 * _time.Second)

			select {
			// target exited by itself
			case err = <-d:
				return err
			// cleanup timeout exceeded
			case <-cleanupCh:
				return _fmt.Errorf("cleanup timeout exceeded")
			// second SIGINT received
			case <-sigCh:
				logger.Println("exiting mage")
				return _fmt.Errorf("exit forced")
			}
		case <-ctx.Done():
			cancel()
			e := ctx.Err()
			_fmt.Printf("ctx err: %v\n", e)
			return e
		case err = <-d:
			// we intentionally don't cancel the context here, because
			// the next target will need to run with the same _context.
			return err
		}
	}
	// This is necessary in case there aren't any targets, to avoid an unused
	// variable error.
	_ = runTarget

	handleError := func(logger *_log.Logger, err interface{}) {
		if err != nil {
			logger.Printf("Error: %+v\n", err)
			type code interface {
				ExitStatus() int
			}
			if c, ok := err.(code); ok {
				_os.Exit(c.ExitStatus())
			}
			_os.Exit(1)
		}
	}
	_ = handleError

	// Set MAGEFILE_VERBOSE so mg.Verbose() reflects the flag value.
	if args.Verbose {
		_os.Setenv("MAGEFILE_VERBOSE", "1")
	} else {
		_os.Setenv("MAGEFILE_VERBOSE", "0")
	}

	_log.SetFlags(0)
	if !args.Verbose {
		_log.SetOutput(_io.Discard)
	}
	logger := _log.New(_os.Stderr, "", 0)
	if args.List {
		if err := list(); err != nil {
			_log.Println(err)
			_os.Exit(1)
		}
		return
	}

	if args.Help {
		if len(args.Args) < 1 {
			logger.Println("no target specified")
			_os.Exit(2)
		}
		switch _strings.ToLower(args.Args[0]) {
			{{range .Funcs -}}
			case "{{lower .TargetName}}":
				{{if ne .Comment "" -}}
				_fmt.Println({{printf "%q" .Comment}})
				_fmt.Println()
				{{end}}
				_fmt.Print("Usage:\n\n\t{{$.BinaryName}} {{lower .TargetName}}{{range .RequiredArgs}} <{{.Name}}>{{end}}{{if .MultipleOptionalArgs}} [<flags>]{{else}}{{range .OptionalArgs}} [-{{.Name}}=<{{.Type}}>]{{end}}{{end}}\n\n")
				{{if .ShowFlagDocs}}_fmt.Print({{printf "%q" .FlagDocsString}})
				{{end -}}
				var aliases []string
				{{- $name := .Name -}}
				{{- $recv := .Receiver -}}
				{{range $alias, $func := $.Aliases}}
				{{if and (eq $name $func.Name) (eq $recv $func.Receiver)}}aliases = append(aliases, "{{$alias}}"){{end -}}
				{{- end}}
				if len(aliases) > 0 {
					_sort.Strings(aliases)
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
			{{end -}}
			{{range .Imports -}}
				{{range .Info.Funcs -}}
			case "{{lower .TargetName}}":
				{{if ne .Comment "" -}}
				_fmt.Println({{printf "%q" .Comment}})
				_fmt.Println()
				{{end}}
				_fmt.Print("Usage:\n\n\t{{$.BinaryName}} {{lower .TargetName}}{{range .RequiredArgs}} <{{.Name}}>{{end}}{{if .MultipleOptionalArgs}} [<flags>]{{else}}{{range .OptionalArgs}} [-{{.Name}}=<{{.Type}}>]{{end}}{{end}}\n\n")
				{{if .ShowFlagDocs}}_fmt.Print({{printf "%q" .FlagDocsString}})
				{{end -}}
				var aliases []string
				{{- $name := .Name -}}
				{{- $recv := .Receiver -}}
				{{range $alias, $func := $.Aliases}}
				{{if and (eq $name $func.Name) (eq $recv $func.Receiver)}}aliases = append(aliases, "{{$alias}}"){{end -}}
				{{- end}}
				if len(aliases) > 0 {
					_sort.Strings(aliases)
					_fmt.Printf("Aliases: %s\n\n", _strings.Join(aliases, ", "))
				}
				return
				{{end -}}
			{{end -}}
			default:
				logger.Printf("Unknown target: %q\n", args.Args[0])
				_os.Exit(2)
		}
	}
	if len(args.Args) < 1 {
	{{- if .DefaultFunc.Name}}
		ignoreDefault, _ := _strconv.ParseBool(_os.Getenv("MAGEFILE_IGNOREDEFAULT"))
		if ignoreDefault {
			if err := list(); err != nil {
				logger.Println("Error:", err)
				_os.Exit(1)
			}
			return
		}
		{{.DefaultFunc.ExecCode}}
		handleError(logger, ret)
		return
	{{- else}}
		if err := list(); err != nil {
			logger.Println("Error:", err)
			_os.Exit(1)
		}
		return
	{{- end}}
	}
	for x := 0; x < len(args.Args); {
		target := args.Args[x]
		x++

		// resolve aliases
		switch _strings.ToLower(target) {
		{{range $alias, $func := .Aliases}}
			case "{{lower $alias}}":
				target = "{{$func.TargetName}}"
		{{- end}}
		}

		switch _strings.ToLower(target) {
		{{range .Funcs }}
			case "{{lower .TargetName}}":
				expected := x + {{.NumRequiredArgs}}
				if expected > len(args.Args) {
					// note that expected and args at this point include the arg for the target itself
					// so we subtract 1 here to show the number of args without the target.
					logger.Printf("not enough arguments for target \"{{.TargetName}}\", expected %v, got %v\n", expected-1, len(args.Args)-1)
					_os.Exit(2)
				}
				if args.Verbose {
					logger.Println("Running target:", "{{.TargetName}}")
				}
				{{.ExecCode}}
				handleError(logger, ret)
		{{- end}}
		{{range .Imports}}
		{{$imp := .}}
			{{range .Info.Funcs }}
				case "{{lower .TargetName}}":
					expected := x + {{.NumRequiredArgs}}
					if expected > len(args.Args) {
						// note that expected and args at this point include the arg for the target itself
						// so we subtract 1 here to show the number of args without the target.
						logger.Printf("not enough arguments for target \"{{.TargetName}}\", expected %v, got %v\n", expected-1, len(args.Args)-1)
						_os.Exit(2)
					}
					if args.Verbose {
						logger.Println("Running target:", "{{.TargetName}}")
					}
					{{.ExecCode}}
					handleError(logger, ret)
			{{- end}}
		{{- end}}
		default:
			logger.Printf("Unknown target specified: %q\n", target)
			_os.Exit(2)
		}
	}
}




`
