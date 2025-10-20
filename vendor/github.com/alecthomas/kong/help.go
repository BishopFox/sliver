package kong

import (
	"bytes"
	"fmt"
	"go/doc"
	"io"
	"strings"
)

const (
	defaultIndent        = 2
	defaultColumnPadding = 4
)

// Help flag.
type helpValue bool

func (h helpValue) BeforeReset(ctx *Context) error {
	options := ctx.Kong.helpOptions
	options.Summary = false
	err := ctx.Kong.help(options, ctx)
	if err != nil {
		return err
	}
	ctx.Kong.Exit(0)
	return nil
}

// HelpOptions for HelpPrinters.
type HelpOptions struct {
	// Don't print top-level usage summary.
	NoAppSummary bool

	// Write a one-line summary of the context.
	Summary bool

	// Write help in a more compact, but still fully-specified, form.
	Compact bool

	// Tree writes command chains in a tree structure instead of listing them separately.
	Tree bool

	// Place the flags after the commands listing.
	FlagsLast bool

	// Indenter modulates the given prefix for the next layer in the tree view.
	// The following exported templates can be used: kong.SpaceIndenter, kong.LineIndenter, kong.TreeIndenter
	// The kong.SpaceIndenter will be used by default.
	Indenter HelpIndenter

	// Don't show the help associated with subcommands
	NoExpandSubcommands bool

	// Clamp the help wrap width to a value smaller than the terminal width.
	// If this is set to a non-positive number, the terminal width is used; otherwise,
	// the min of this value or the terminal width is used.
	WrapUpperBound int
}

// Apply options to Kong as a configuration option.
func (h HelpOptions) Apply(k *Kong) error {
	k.helpOptions = h
	return nil
}

// HelpProvider can be implemented by commands/args to provide detailed help.
type HelpProvider interface {
	// This string is formatted by go/doc and thus has the same formatting rules.
	Help() string
}

// PlaceHolderProvider can be implemented by mappers to provide custom placeholder text.
type PlaceHolderProvider interface {
	PlaceHolder(flag *Flag) string
}

// HelpIndenter is used to indent new layers in the help tree.
type HelpIndenter func(prefix string) string

// HelpPrinter is used to print context-sensitive help.
type HelpPrinter func(options HelpOptions, ctx *Context) error

// HelpValueFormatter is used to format the help text of flags and positional arguments.
type HelpValueFormatter func(value *Value) string

// DefaultHelpValueFormatter is the default HelpValueFormatter.
func DefaultHelpValueFormatter(value *Value) string {
	if len(value.Tag.Envs) == 0 || HasInterpolatedVar(value.OrigHelp, "env") {
		return value.Help
	}
	suffix := "(" + formatEnvs(value.Tag.Envs) + ")"
	switch {
	case strings.HasSuffix(value.Help, "."):
		return value.Help[:len(value.Help)-1] + " " + suffix + "."
	case value.Help == "":
		return suffix
	default:
		return value.Help + " " + suffix
	}
}

// DefaultShortHelpPrinter is the default HelpPrinter for short help on error.
func DefaultShortHelpPrinter(options HelpOptions, ctx *Context) error {
	w := newHelpWriter(ctx, options)
	cmd := ctx.Selected()
	app := ctx.Model
	if cmd == nil {
		w.Printf("Usage: %s%s", app.Name, app.Summary())
		w.Printf(`Run "%s --help" for more information.`, app.Name)
	} else {
		w.Printf("Usage: %s %s", app.Name, cmd.Summary())
		w.Printf(`Run "%s --help" for more information.`, cmd.FullPath())
	}
	return w.Write(ctx.Stdout)
}

// DefaultHelpPrinter is the default HelpPrinter.
func DefaultHelpPrinter(options HelpOptions, ctx *Context) error {
	if ctx.Empty() {
		options.Summary = false
	}
	w := newHelpWriter(ctx, options)
	selected := ctx.Selected()
	if selected == nil {
		printApp(w, ctx.Model)
	} else {
		printCommand(w, ctx.Model, selected)
	}
	return w.Write(ctx.Stdout)
}

func printApp(w *helpWriter, app *Application) {
	if !w.NoAppSummary {
		w.Printf("Usage: %s%s", app.Name, app.Summary())
	}
	printNodeDetail(w, app.Node, true)
	cmds := app.Leaves(true)
	if len(cmds) > 0 && app.HelpFlag != nil {
		w.Print("")
		if w.Summary {
			w.Printf(`Run "%s --help" for more information.`, app.Name)
		} else {
			w.Printf(`Run "%s <command> --help" for more information on a command.`, app.Name)
		}
	}
}

func printCommand(w *helpWriter, app *Application, cmd *Command) {
	if !w.NoAppSummary {
		w.Printf("Usage: %s %s", app.Name, cmd.Summary())
	}
	printNodeDetail(w, cmd, true)
	if w.Summary && app.HelpFlag != nil {
		w.Print("")
		w.Printf(`Run "%s --help" for more information.`, cmd.FullPath())
	}
}

func printNodeDetail(w *helpWriter, node *Node, hide bool) {
	if node.Help != "" {
		w.Print("")
		w.Wrap(node.Help)
	}
	if w.Summary {
		return
	}
	if node.Detail != "" {
		w.Print("")
		w.Wrap(node.Detail)
	}
	if len(node.Positional) > 0 {
		w.Print("")
		w.Print("Arguments:")
		writePositionals(w.Indent(), node.Positional)
	}
	printFlags := func() {
		if flags := node.AllFlags(true); len(flags) > 0 {
			groupedFlags := collectFlagGroups(flags)
			for _, group := range groupedFlags {
				w.Print("")
				if group.Metadata.Title != "" {
					w.Wrap(group.Metadata.Title)
				}
				if group.Metadata.Description != "" {
					w.Indent().Wrap(group.Metadata.Description)
					w.Print("")
				}
				writeFlags(w.Indent(), group.Flags)
			}
		}
	}
	if !w.FlagsLast {
		printFlags()
	}
	var cmds []*Node
	if w.NoExpandSubcommands {
		cmds = node.Children
	} else {
		cmds = node.Leaves(hide)
	}
	if len(cmds) > 0 {
		iw := w.Indent()
		if w.Tree {
			w.Print("")
			w.Print("Commands:")
			writeCommandTree(iw, node)
		} else {
			groupedCmds := collectCommandGroups(cmds)
			for _, group := range groupedCmds {
				w.Print("")
				if group.Metadata.Title != "" {
					w.Wrap(group.Metadata.Title)
				}
				if group.Metadata.Description != "" {
					w.Indent().Wrap(group.Metadata.Description)
					w.Print("")
				}

				if w.Compact {
					writeCompactCommandList(group.Commands, iw)
				} else {
					writeCommandList(group.Commands, iw)
				}
			}
		}
	}
	if w.FlagsLast {
		printFlags()
	}
}

func writeCommandList(cmds []*Node, iw *helpWriter) {
	for i, cmd := range cmds {
		if cmd.Hidden {
			continue
		}
		printCommandSummary(iw, cmd)
		if i != len(cmds)-1 {
			iw.Print("")
		}
	}
}

func writeCompactCommandList(cmds []*Node, iw *helpWriter) {
	rows := [][2]string{}
	for _, cmd := range cmds {
		if cmd.Hidden {
			continue
		}
		rows = append(rows, [2]string{cmd.Path(), cmd.Help})
	}
	writeTwoColumns(iw, rows)
}

func writeCommandTree(w *helpWriter, node *Node) {
	rows := make([][2]string, 0, len(node.Children)*2)
	for i, cmd := range node.Children {
		if cmd.Hidden {
			continue
		}
		rows = append(rows, w.CommandTree(cmd, "")...)
		if i != len(node.Children)-1 {
			rows = append(rows, [2]string{"", ""})
		}
	}
	writeTwoColumns(w, rows)
}

type helpFlagGroup struct {
	Metadata *Group
	Flags    [][]*Flag
}

func collectFlagGroups(flags [][]*Flag) []helpFlagGroup {
	// Group keys in order of appearance.
	groups := []*Group{}
	// Flags grouped by their group key.
	flagsByGroup := map[string][][]*Flag{}

	for _, levelFlags := range flags {
		levelFlagsByGroup := map[string][]*Flag{}

		for _, flag := range levelFlags {
			key := ""
			if flag.Group != nil {
				key = flag.Group.Key
				groupAlreadySeen := false
				for _, group := range groups {
					if key == group.Key {
						groupAlreadySeen = true
						break
					}
				}
				if !groupAlreadySeen {
					groups = append(groups, flag.Group)
				}
			}

			levelFlagsByGroup[key] = append(levelFlagsByGroup[key], flag)
		}

		for key, flags := range levelFlagsByGroup {
			flagsByGroup[key] = append(flagsByGroup[key], flags)
		}
	}

	out := []helpFlagGroup{}
	// Ungrouped flags are always displayed first.
	if ungroupedFlags, ok := flagsByGroup[""]; ok {
		out = append(out, helpFlagGroup{
			Metadata: &Group{Title: "Flags:"},
			Flags:    ungroupedFlags,
		})
	}
	for _, group := range groups {
		out = append(out, helpFlagGroup{Metadata: group, Flags: flagsByGroup[group.Key]})
	}
	return out
}

type helpCommandGroup struct {
	Metadata *Group
	Commands []*Node
}

func collectCommandGroups(nodes []*Node) []helpCommandGroup {
	// Groups in order of appearance.
	groups := []*Group{}
	// Nodes grouped by their group key.
	nodesByGroup := map[string][]*Node{}

	for _, node := range nodes {
		key := ""
		if group := node.ClosestGroup(); group != nil {
			key = group.Key
			if _, ok := nodesByGroup[key]; !ok {
				groups = append(groups, group)
			}
		}
		nodesByGroup[key] = append(nodesByGroup[key], node)
	}

	out := []helpCommandGroup{}
	// Ungrouped nodes are always displayed first.
	if ungroupedNodes, ok := nodesByGroup[""]; ok {
		out = append(out, helpCommandGroup{
			Metadata: &Group{Title: "Commands:"},
			Commands: ungroupedNodes,
		})
	}
	for _, group := range groups {
		out = append(out, helpCommandGroup{Metadata: group, Commands: nodesByGroup[group.Key]})
	}
	return out
}

func printCommandSummary(w *helpWriter, cmd *Command) {
	w.Print(cmd.Summary())
	if cmd.Help != "" {
		w.Indent().Wrap(cmd.Help)
	}
}

type helpWriter struct {
	indent        string
	width         int
	lines         *[]string
	helpFormatter HelpValueFormatter
	HelpOptions
}

func newHelpWriter(ctx *Context, options HelpOptions) *helpWriter {
	lines := []string{}
	wrapWidth := guessWidth(ctx.Stdout)
	if options.WrapUpperBound > 0 && wrapWidth > options.WrapUpperBound {
		wrapWidth = options.WrapUpperBound
	}
	w := &helpWriter{
		indent:        "",
		width:         wrapWidth,
		lines:         &lines,
		helpFormatter: ctx.Kong.helpFormatter,
		HelpOptions:   options,
	}
	return w
}

func (h *helpWriter) Printf(format string, args ...interface{}) {
	h.Print(fmt.Sprintf(format, args...))
}

func (h *helpWriter) Print(text string) {
	*h.lines = append(*h.lines, strings.TrimRight(h.indent+text, " "))
}

// Indent returns a new helpWriter indented by two characters.
func (h *helpWriter) Indent() *helpWriter {
	return &helpWriter{indent: h.indent + "  ", lines: h.lines, width: h.width - 2, HelpOptions: h.HelpOptions, helpFormatter: h.helpFormatter}
}

func (h *helpWriter) String() string {
	return strings.Join(*h.lines, "\n")
}

func (h *helpWriter) Write(w io.Writer) error {
	for _, line := range *h.lines {
		_, err := io.WriteString(w, line+"\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *helpWriter) Wrap(text string) {
	w := bytes.NewBuffer(nil)
	doc.ToText(w, strings.TrimSpace(text), "", "    ", h.width)
	for _, line := range strings.Split(strings.TrimSpace(w.String()), "\n") {
		h.Print(line)
	}
}

func writePositionals(w *helpWriter, args []*Positional) {
	rows := [][2]string{}
	for _, arg := range args {
		rows = append(rows, [2]string{arg.Summary(), w.helpFormatter(arg)})
	}
	writeTwoColumns(w, rows)
}

func writeFlags(w *helpWriter, groups [][]*Flag) {
	rows := [][2]string{}
	haveShort := false
	for _, group := range groups {
		for _, flag := range group {
			if flag.Short != 0 {
				haveShort = true
				break
			}
		}
	}
	for i, group := range groups {
		if i > 0 {
			rows = append(rows, [2]string{"", ""})
		}
		for _, flag := range group {
			if !flag.Hidden {
				rows = append(rows, [2]string{formatFlag(haveShort, flag), w.helpFormatter(flag.Value)})
			}
		}
	}
	writeTwoColumns(w, rows)
}

func writeTwoColumns(w *helpWriter, rows [][2]string) {
	maxLeft := 375 * w.width / 1000
	if maxLeft < 30 {
		maxLeft = 30
	}
	// Find size of first column.
	leftSize := 0
	for _, row := range rows {
		if c := len(row[0]); c > leftSize && c < maxLeft {
			leftSize = c
		}
	}

	offsetStr := strings.Repeat(" ", leftSize+defaultColumnPadding)

	for _, row := range rows {
		buf := bytes.NewBuffer(nil)
		doc.ToText(buf, row[1], "", strings.Repeat(" ", defaultIndent), w.width-leftSize-defaultColumnPadding)
		lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")

		line := fmt.Sprintf("%-*s", leftSize, row[0])
		if len(row[0]) < maxLeft {
			line += fmt.Sprintf("%*s%s", defaultColumnPadding, "", lines[0])
			lines = lines[1:]
		}
		w.Print(line)
		for _, line := range lines {
			w.Printf("%s%s", offsetStr, line)
		}
	}
}

// haveShort will be true if there are short flags present at all in the help. Useful for column alignment.
func formatFlag(haveShort bool, flag *Flag) string {
	flagString := ""
	name := flag.Name
	isBool := flag.IsBool()
	if flag.Short != 0 {
		if isBool && flag.Tag.Negatable {
			flagString += fmt.Sprintf("-%c, --[no-]%s", flag.Short, name)
		} else {
			flagString += fmt.Sprintf("-%c, --%s", flag.Short, name)
		}
	} else {
		if isBool && flag.Tag.Negatable {
			if haveShort {
				flagString = fmt.Sprintf("    --[no-]%s", name)
			} else {
				flagString = fmt.Sprintf("--[no-]%s", name)
			}
		} else {
			if haveShort {
				flagString += fmt.Sprintf("    --%s", name)
			} else {
				flagString += fmt.Sprintf("--%s", name)
			}
		}
	}
	if !isBool {
		flagString += fmt.Sprintf("=%s", flag.FormatPlaceHolder())
	}
	return flagString
}

// CommandTree creates a tree with the given node name as root and its children's arguments and sub commands as leaves.
func (h *HelpOptions) CommandTree(node *Node, prefix string) (rows [][2]string) {
	var nodeName string
	switch node.Type {
	default:
		nodeName += prefix + node.Name
		if len(node.Aliases) != 0 {
			nodeName += fmt.Sprintf(" (%s)", strings.Join(node.Aliases, ","))
		}
	case ArgumentNode:
		nodeName += prefix + "<" + node.Name + ">"
	}
	rows = append(rows, [2]string{nodeName, node.Help})
	if h.Indenter == nil {
		prefix = SpaceIndenter(prefix)
	} else {
		prefix = h.Indenter(prefix)
	}
	for _, arg := range node.Positional {
		rows = append(rows, [2]string{prefix + arg.Summary(), arg.Help})
	}
	for _, subCmd := range node.Children {
		if subCmd.Hidden {
			continue
		}
		rows = append(rows, h.CommandTree(subCmd, prefix)...)
	}
	return
}

// SpaceIndenter adds a space indent to the given prefix.
func SpaceIndenter(prefix string) string {
	return prefix + strings.Repeat(" ", defaultIndent)
}

// LineIndenter adds line points to every new indent.
func LineIndenter(prefix string) string {
	if prefix == "" {
		return "- "
	}
	return strings.Repeat(" ", defaultIndent) + prefix
}

// TreeIndenter adds line points to every new indent and vertical lines to every layer.
func TreeIndenter(prefix string) string {
	if prefix == "" {
		return "|- "
	}
	return "|" + strings.Repeat(" ", defaultIndent) + prefix
}

func formatEnvs(envs []string) string {
	formatted := make([]string, len(envs))
	for i := range envs {
		formatted[i] = "$" + envs[i]
	}

	return strings.Join(formatted, ", ")
}
