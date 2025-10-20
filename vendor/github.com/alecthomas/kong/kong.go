package kong

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

var (
	callbackReturnSignature = reflect.TypeOf((*error)(nil)).Elem()
)

func failField(parent reflect.Value, field reflect.StructField, format string, args ...interface{}) error {
	name := parent.Type().Name()
	if name == "" {
		name = "<anonymous struct>"
	}
	return fmt.Errorf("%s.%s: %s", name, field.Name, fmt.Sprintf(format, args...))
}

// Must creates a new Parser or panics if there is an error.
func Must(ast interface{}, options ...Option) *Kong {
	k, err := New(ast, options...)
	if err != nil {
		panic(err)
	}
	return k
}

type usageOnError int

const (
	shortUsage usageOnError = iota + 1
	fullUsage
)

// Kong is the main parser type.
type Kong struct {
	// Grammar model.
	Model *Application

	// Termination function (defaults to os.Exit)
	Exit func(int)

	Stdout io.Writer
	Stderr io.Writer

	bindings     bindings
	loader       ConfigurationLoader
	resolvers    []Resolver
	registry     *Registry
	ignoreFields []*regexp.Regexp

	noDefaultHelp bool
	usageOnError  usageOnError
	help          HelpPrinter
	shortHelp     HelpPrinter
	helpFormatter HelpValueFormatter
	helpOptions   HelpOptions
	helpFlag      *Flag
	groups        []Group
	vars          Vars
	flagNamer     func(string) string

	// Set temporarily by Options. These are applied after build().
	postBuildOptions []Option
	embedded         []embedded
	dynamicCommands  []*dynamicCommand
}

// New creates a new Kong parser on grammar.
//
// See the README (https://github.com/alecthomas/kong) for usage instructions.
func New(grammar interface{}, options ...Option) (*Kong, error) {
	k := &Kong{
		Exit:          os.Exit,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		registry:      NewRegistry().RegisterDefaults(),
		vars:          Vars{},
		bindings:      bindings{},
		helpFormatter: DefaultHelpValueFormatter,
		ignoreFields:  make([]*regexp.Regexp, 0),
		flagNamer: func(s string) string {
			return strings.ToLower(dashedString(s))
		},
	}

	options = append(options, Bind(k))

	for _, option := range options {
		if err := option.Apply(k); err != nil {
			return nil, err
		}
	}

	if k.help == nil {
		k.help = DefaultHelpPrinter
	}

	if k.shortHelp == nil {
		k.shortHelp = DefaultShortHelpPrinter
	}

	model, err := build(k, grammar)
	if err != nil {
		return k, err
	}
	model.Name = filepath.Base(os.Args[0])
	k.Model = model
	k.Model.HelpFlag = k.helpFlag

	// Embed any embedded structs.
	for _, embed := range k.embedded {
		tag, err := parseTagString(strings.Join(embed.tags, " ")) //nolint:govet
		if err != nil {
			return nil, err
		}
		tag.Embed = true
		v := reflect.Indirect(reflect.ValueOf(embed.strct))
		node, err := buildNode(k, v, CommandNode, tag, map[string]bool{})
		if err != nil {
			return nil, err
		}
		for _, child := range node.Children {
			child.Parent = k.Model.Node
			k.Model.Children = append(k.Model.Children, child)
		}
		k.Model.Flags = append(k.Model.Flags, node.Flags...)
	}

	// Synthesise command nodes.
	for _, dcmd := range k.dynamicCommands {
		tag, terr := parseTagString(strings.Join(dcmd.tags, " "))
		if terr != nil {
			return nil, terr
		}
		tag.Name = dcmd.name
		tag.Help = dcmd.help
		tag.Group = dcmd.group
		tag.Cmd = true
		v := reflect.Indirect(reflect.ValueOf(dcmd.cmd))
		err = buildChild(k, k.Model.Node, CommandNode, reflect.Value{}, reflect.StructField{
			Name: dcmd.name,
			Type: v.Type(),
		}, v, tag, dcmd.name, map[string]bool{})
		if err != nil {
			return nil, err
		}
	}

	for _, option := range k.postBuildOptions {
		if err = option.Apply(k); err != nil {
			return nil, err
		}
	}
	k.postBuildOptions = nil

	if err = k.interpolate(k.Model.Node); err != nil {
		return nil, err
	}

	k.bindings.add(k.vars)

	return k, nil
}

type varStack []Vars

func (v *varStack) head() Vars { return (*v)[len(*v)-1] }
func (v *varStack) pop()       { *v = (*v)[:len(*v)-1] }
func (v *varStack) push(vars Vars) Vars {
	if len(*v) != 0 {
		vars = (*v)[len(*v)-1].CloneWith(vars)
	}
	*v = append(*v, vars)
	return vars
}

// Interpolate variables into model.
func (k *Kong) interpolate(node *Node) (err error) {
	stack := varStack{}
	return Visit(node, func(node Visitable, next Next) error {
		switch node := node.(type) {
		case *Node:
			vars := stack.push(node.Vars())
			node.Help, err = interpolate(node.Help, vars, nil)
			if err != nil {
				return fmt.Errorf("help for %s: %s", node.Path(), err)
			}
			err = next(nil)
			stack.pop()
			return err

		case *Value:
			return next(k.interpolateValue(node, stack.head()))
		}
		return next(nil)
	})
}

func (k *Kong) interpolateValue(value *Value, vars Vars) (err error) {
	if len(value.Tag.Vars) > 0 {
		vars = vars.CloneWith(value.Tag.Vars)
	}
	if varsContributor, ok := value.Mapper.(VarsContributor); ok {
		vars = vars.CloneWith(varsContributor.Vars(value))
	}

	if value.Enum, err = interpolate(value.Enum, vars, nil); err != nil {
		return fmt.Errorf("enum for %s: %s", value.Summary(), err)
	}

	updatedVars := map[string]string{
		"default": value.Default,
		"enum":    value.Enum,
	}
	if value.Default, err = interpolate(value.Default, vars, nil); err != nil {
		return fmt.Errorf("default value for %s: %s", value.Summary(), err)
	}
	if value.Enum, err = interpolate(value.Enum, vars, nil); err != nil {
		return fmt.Errorf("enum value for %s: %s", value.Summary(), err)
	}
	if value.Flag != nil {
		for i, env := range value.Flag.Envs {
			if value.Flag.Envs[i], err = interpolate(env, vars, nil); err != nil {
				return fmt.Errorf("env value for %s: %s", value.Summary(), err)
			}
		}
		value.Tag.Envs = value.Flag.Envs
		updatedVars["env"] = ""
		if len(value.Flag.Envs) != 0 {
			updatedVars["env"] = value.Flag.Envs[0]
		}
	}
	value.Help, err = interpolate(value.Help, vars, updatedVars)
	if err != nil {
		return fmt.Errorf("help for %s: %s", value.Summary(), err)
	}
	return nil
}

// Provide additional builtin flags, if any.
func (k *Kong) extraFlags() []*Flag {
	if k.noDefaultHelp {
		return nil
	}
	var helpTarget helpValue
	value := reflect.ValueOf(&helpTarget).Elem()
	helpFlag := &Flag{
		Short: 'h',
		Value: &Value{
			Name:         "help",
			Help:         "Show context-sensitive help.",
			OrigHelp:     "Show context-sensitive help.",
			Target:       value,
			Tag:          &Tag{},
			Mapper:       k.registry.ForValue(value),
			DefaultValue: reflect.ValueOf(false),
		},
	}
	helpFlag.Flag = helpFlag
	k.helpFlag = helpFlag
	return []*Flag{helpFlag}
}

// Parse arguments into target.
//
// The return Context can be used to further inspect the parsed command-line, to format help, to find the
// selected command, to run command Run() methods, and so on. See Context and README for more information.
//
// Will return a ParseError if a *semantically* invalid command-line is encountered (as opposed to a syntactically
// invalid one, which will report a normal error).
func (k *Kong) Parse(args []string) (ctx *Context, err error) {
	ctx, err = Trace(k, args)
	if err != nil {
		return nil, err
	}
	if ctx.Error != nil {
		return nil, &ParseError{error: ctx.Error, Context: ctx}
	}
	if err = k.applyHook(ctx, "BeforeReset"); err != nil {
		return nil, &ParseError{error: err, Context: ctx}
	}
	if err = ctx.Reset(); err != nil {
		return nil, &ParseError{error: err, Context: ctx}
	}
	if err = k.applyHook(ctx, "BeforeResolve"); err != nil {
		return nil, &ParseError{error: err, Context: ctx}
	}
	if err = ctx.Resolve(); err != nil {
		return nil, &ParseError{error: err, Context: ctx}
	}
	if err = k.applyHook(ctx, "BeforeApply"); err != nil {
		return nil, &ParseError{error: err, Context: ctx}
	}
	if _, err = ctx.Apply(); err != nil {
		return nil, &ParseError{error: err, Context: ctx}
	}
	if err = ctx.Validate(); err != nil {
		return nil, &ParseError{error: err, Context: ctx}
	}
	if err = k.applyHook(ctx, "AfterApply"); err != nil {
		return nil, &ParseError{error: err, Context: ctx}
	}
	return ctx, nil
}

func (k *Kong) applyHook(ctx *Context, name string) error {
	for _, trace := range ctx.Path {
		var value reflect.Value
		switch {
		case trace.App != nil:
			value = trace.App.Target
		case trace.Argument != nil:
			value = trace.Argument.Target
		case trace.Command != nil:
			value = trace.Command.Target
		case trace.Positional != nil:
			value = trace.Positional.Target
		case trace.Flag != nil:
			value = trace.Flag.Value.Target
		default:
			panic("unsupported Path")
		}
		method := getMethod(value, name)
		if !method.IsValid() {
			continue
		}
		binds := k.bindings.clone()
		binds.add(ctx, trace)
		binds.add(trace.Node().Vars().CloneWith(k.vars))
		binds.merge(ctx.bindings)
		if err := callFunction(method, binds); err != nil {
			return err
		}
	}
	// Path[0] will always be the app root.
	return k.applyHookToDefaultFlags(ctx, ctx.Path[0].Node(), name)
}

// Call hook on any unset flags with default values.
func (k *Kong) applyHookToDefaultFlags(ctx *Context, node *Node, name string) error {
	if node == nil {
		return nil
	}
	return Visit(node, func(n Visitable, next Next) error {
		node, ok := n.(*Node)
		if !ok {
			return next(nil)
		}
		binds := k.bindings.clone().add(ctx).add(node.Vars().CloneWith(k.vars))
		for _, flag := range node.Flags {
			if !flag.HasDefault || ctx.values[flag.Value].IsValid() || !flag.Target.IsValid() {
				continue
			}
			method := getMethod(flag.Target, name)
			if !method.IsValid() {
				continue
			}
			path := &Path{Flag: flag}
			if err := callFunction(method, binds.clone().add(path)); err != nil {
				return next(err)
			}
		}
		return next(nil)
	})
}

func formatMultilineMessage(w io.Writer, leaders []string, format string, args ...interface{}) {
	lines := strings.Split(fmt.Sprintf(format, args...), "\n")
	leader := ""
	for _, l := range leaders {
		if l == "" {
			continue
		}
		leader += l + ": "
	}
	fmt.Fprintf(w, "%s%s\n", leader, lines[0])
	for _, line := range lines[1:] {
		fmt.Fprintf(w, "%*s%s\n", len(leader), " ", line)
	}
}

// Printf writes a message to Kong.Stdout with the application name prefixed.
func (k *Kong) Printf(format string, args ...interface{}) *Kong {
	formatMultilineMessage(k.Stdout, []string{k.Model.Name}, format, args...)
	return k
}

// Errorf writes a message to Kong.Stderr with the application name prefixed.
func (k *Kong) Errorf(format string, args ...interface{}) *Kong {
	formatMultilineMessage(k.Stderr, []string{k.Model.Name, "error"}, format, args...)
	return k
}

// Fatalf writes a message to Kong.Stderr with the application name prefixed then exits with a non-zero status.
func (k *Kong) Fatalf(format string, args ...interface{}) {
	k.Errorf(format, args...)
	k.Exit(1)
}

// FatalIfErrorf terminates with an error message if err != nil.
func (k *Kong) FatalIfErrorf(err error, args ...interface{}) {
	if err == nil {
		return
	}
	msg := err.Error()
	if len(args) > 0 {
		msg = fmt.Sprintf(args[0].(string), args[1:]...) + ": " + err.Error() // nolint
	}
	// Maybe display usage information.
	var parseErr *ParseError
	if errors.As(err, &parseErr) {
		switch k.usageOnError {
		case fullUsage:
			_ = k.help(k.helpOptions, parseErr.Context)
			fmt.Fprintln(k.Stdout)
		case shortUsage:
			_ = k.shortHelp(k.helpOptions, parseErr.Context)
			fmt.Fprintln(k.Stdout)
		}
	}
	k.Fatalf("%s", msg)
}

// LoadConfig from path using the loader configured via Configuration(loader).
//
// "path" will have ~ and any variables expanded.
func (k *Kong) LoadConfig(path string) (Resolver, error) {
	var err error
	path = ExpandPath(path)
	path, err = interpolate(path, k.vars, nil)
	if err != nil {
		return nil, err
	}
	r, err := os.Open(path) // nolint: gas
	if err != nil {
		return nil, err
	}
	defer r.Close() // nolint: gosec

	return k.loader(r)
}
