package carapace

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/carapace-sh/carapace/internal/common"
	"github.com/carapace-sh/carapace/internal/config"
	"github.com/carapace-sh/carapace/internal/env"
	"github.com/carapace-sh/carapace/internal/export"
	"github.com/carapace-sh/carapace/internal/man"
	"github.com/carapace-sh/carapace/pkg/match"
	"github.com/carapace-sh/carapace/pkg/style"
	"github.com/carapace-sh/carapace/pkg/uid"
	"github.com/carapace-sh/carapace/third_party/github.com/acarl005/stripansi"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ActionCallback invokes a go function during completion.
func ActionCallback(callback CompletionCallback) Action {
	return Action{callback: callback}
}

// ActionExecCommand executes an external command.
//
//	carapace.ActionExecCommand("git", "remote")(func(output []byte) carapace.Action {
//	  lines := strings.Split(string(output), "\n")
//	  return carapace.ActionValues(lines[:len(lines)-1]...)
//	})
func ActionExecCommand(name string, arg ...string) func(f func(output []byte) Action) Action {
	return func(f func(output []byte) Action) Action {
		return ActionExecCommandE(name, arg...)(func(output []byte, err error) Action {
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					if firstLine := strings.SplitN(string(exitErr.Stderr), "\n", 2)[0]; strings.TrimSpace(firstLine) != "" {
						err = errors.New(firstLine)
					}
				}
				return ActionMessage(err.Error())
			}
			return f(output)
		})
	}
}

// ActionExecCommandE is like ActionExecCommand but with custom error handling.
//
//	carapace.ActionExecCommandE("supervisorctl", "--configuration", path, "status")(func(output []byte, err error) carapace.Action {
//		if err != nil {
//			const NOT_RUNNING = 3
//			if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != NOT_RUNNING {
//				return carapace.ActionMessage(err.Error())
//			}
//		}
//		return carapace.ActionValues("success")
//	})
func ActionExecCommandE(name string, arg ...string) func(f func(output []byte, err error) Action) Action {
	return func(f func(output []byte, err error) Action) Action {
		return ActionCallback(func(c Context) Action {
			var stdout, stderr bytes.Buffer
			cmd := c.Command(name, arg...)
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitErr.Stderr = stderr.Bytes() // seems this needs to be set manually due to stdout being collected?
				}
				return f(stdout.Bytes(), err)
			}
			return f(stdout.Bytes(), nil)
		})
	}
}

// ActionImport parses the json output from export as Action
//
//	carapace.Gen(rootCmd).PositionalAnyCompletion(
//		carapace.ActionCallback(func(c carapace.Context) carapace.Action {
//			args := []string{"_carapace", "export", ""}
//			args = append(args, c.Args...)
//			args = append(args, c.Value)
//			return carapace.ActionExecCommand("command", args...)(func(output []byte) carapace.Action {
//				return carapace.ActionImport(output)
//			})
//		}),
//	)
func ActionImport(output []byte) Action {
	return ActionCallback(func(c Context) Action {
		var e export.Export
		if err := json.Unmarshal(output, &e); err != nil {
			return ActionMessage(err.Error())
		}
		return Action{
			rawValues: e.Values,
			meta:      e.Meta,
		}
	})
}

// ActionExecute executes completion on an internal command
// TODO example.
func ActionExecute(cmd *cobra.Command) Action {
	return ActionCallback(func(c Context) Action {
		args := []string{"_carapace", "export", cmd.Name()}
		args = append(args, c.Args...)
		args = append(args, c.Value)
		cmd.SetArgs(args)

		Gen(cmd).PreInvoke(func(cmd *cobra.Command, flag *pflag.Flag, action Action) Action {
			return ActionCallback(func(_c Context) Action {
				_c.Env = c.Env
				_c.Dir = c.Dir
				return action.Invoke(_c).ToA()
			})
		})

		var stdout, stderr bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)

		if err := cmd.Execute(); err != nil {
			return ActionMessage(err.Error())
		}
		return ActionImport(stdout.Bytes())
	})
}

// ActionDirectories completes directories.
func ActionDirectories() Action {
	return ActionCallback(func(c Context) Action {
		return actionPath([]string{""}, true).
			MultiParts("/").
			StyleF(style.ForPath).
			UidF(func(s string, uc uid.Context) (*url.URL, error) { // TODO duplicated from ActionFiles
				abs, err := c.Abs(s)
				if err != nil {
					return nil, err
				}
				return url.Parse("file://" + uid.PathEscape(abs))
			}).
			QueryF(func(s string, uc uid.Context) (*url.URL, error) {
				abs, err := c.Abs(c.Dir)
				if err != nil {
					return nil, err
				}
				return url.Parse(fmt.Sprintf("file://%v?directories=true", uid.PathEscape(abs)))
			})
	}).Tag("directories")
}

// ActionFiles completes files with optional suffix filtering.
func ActionFiles(suffix ...string) Action {
	return ActionCallback(func(c Context) Action {
		return actionPath(suffix, false).
			MultiParts("/").
			StyleF(style.ForPath).
			UidF(func(s string, uc uid.Context) (*url.URL, error) {
				abs, err := c.Abs(s)
				if err != nil {
					return nil, err
				}
				return url.Parse("file://" + uid.PathEscape(abs))
			}).
			QueryF(func(s string, uc uid.Context) (*url.URL, error) {
				abs, err := c.Abs(c.Dir)
				if err != nil {
					return nil, err
				}
				query := &url.URL{
					Scheme: "file",
					Path:   abs,
				}
				if len(suffix) > 0 {
					values := query.Query()
					values.Set("suffix", strings.Join(suffix, ","))
					query.RawQuery = values.Encode()
				}
				return query, nil
			})
		// Query("file", "", c.Dir, "suffix", strings.Join(suffix, ","))
	}).Tag("files")
}

// ActionValues completes arbitrary keywords (values).
func ActionValues(values ...string) Action {
	return ActionCallback(func(c Context) Action {
		vals := make([]common.RawValue, 0, len(values))
		for _, val := range values {
			if val != "" {
				vals = append(vals, common.RawValue{Value: val, Display: val})
			}
		}
		return Action{rawValues: vals}
	})
}

// ActionStyledValues is like ActionValues but also accepts a style.
func ActionStyledValues(values ...string) Action {
	return ActionCallback(func(c Context) Action {
		if length := len(values); length%2 != 0 {
			return ActionMessage("invalid amount of arguments [ActionStyledValues]: %v", length)
		}

		vals := make([]common.RawValue, 0, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			vals = append(vals, common.RawValue{Value: values[i], Display: values[i], Style: values[i+1]})
		}
		return Action{rawValues: vals}
	})
}

// ActionValuesDescribed completes arbitrary key (values) with an additional description (value, description pairs).
func ActionValuesDescribed(values ...string) Action {
	return ActionCallback(func(c Context) Action {
		if length := len(values); length%2 != 0 {
			return ActionMessage("invalid amount of arguments [ActionValuesDescribed]: %v", length)
		}

		vals := make([]common.RawValue, 0, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			vals = append(vals, common.RawValue{Value: values[i], Display: values[i], Description: values[i+1]})
		}
		return Action{rawValues: vals}
	})
}

// ActionStyledValuesDescribed is like ActionValues but also accepts a style.
func ActionStyledValuesDescribed(values ...string) Action {
	return ActionCallback(func(c Context) Action {
		if length := len(values); length%3 != 0 {
			return ActionMessage("invalid amount of arguments [ActionStyledValuesDescribed]: %v", length)
		}

		vals := make([]common.RawValue, 0, len(values)/3)
		for i := 0; i < len(values); i += 3 {
			vals = append(vals, common.RawValue{Value: values[i], Display: values[i], Description: values[i+1], Style: values[i+2]})
		}
		return Action{rawValues: vals}
	})
}

// ActionMessage displays a help messages in places where no completions can be generated.
func ActionMessage(msg string, args ...interface{}) Action {
	return ActionCallback(func(c Context) Action {
		if len(args) > 0 {
			msg = fmt.Sprintf(msg, args...)
		}
		a := ActionValues()
		a.meta.Messages.Add(stripansi.Strip(msg))
		return a
	})
}

// ActionMultiParts completes parts of an argument separated by sep.
func ActionMultiParts(sep string, callback func(c Context) Action) Action {
	return ActionMultiPartsN(sep, -1, callback)
}

// ActionMultiPartsN is like ActionMultiParts but limits the number of parts to `n`.
func ActionMultiPartsN(sep string, n int, callback func(c Context) Action) Action {
	return ActionCallback(func(c Context) Action {
		switch n {
		case 0:
			return ActionMessage("invalid value for n [ActionValuesDescribed]: %v", n)
		case 1:
			return callback(c).Invoke(c).ToA()
		}

		splitted := strings.SplitN(c.Value, sep, n)
		prefix := ""
		c.Parts = []string{}

		switch {
		case len(sep) == 0:
			switch {
			case n < 0:
				prefix = c.Value
				c.Value = ""
				c.Parts = splitted
			default:
				prefix = c.Value
				if n-1 < len(prefix) {
					prefix = c.Value[:n-1]
					c.Value = c.Value[n-1:]
				} else {
					c.Value = ""
				}
				c.Parts = strings.Split(prefix, "")
			}
		default:
			if len(splitted) > 1 {
				c.Value = splitted[len(splitted)-1]
				c.Parts = splitted[:len(splitted)-1]
				prefix = strings.Join(c.Parts, sep) + sep
			}
		}

		nospace := '*'
		if runes := []rune(sep); len(runes) > 0 {
			nospace = runes[len(runes)-1]
		}
		return callback(c).Invoke(c).Prefix(prefix).ToA().NoSpace(nospace)
	})
}

// ActionStyleConfig completes style configuration
//
//	carapace.Value=blue
//	carapace.Description=magenta
func ActionStyleConfig() Action {
	return ActionMultiParts("=", func(c Context) Action {
		switch len(c.Parts) {
		case 0:
			return ActionMultiParts(".", func(c Context) Action {
				switch len(c.Parts) {
				case 0:
					return ActionValues(config.GetStyleConfigs()...).Invoke(c).Suffix(".").ToA()

				case 1:
					fields, err := config.GetStyleFields(c.Parts[0])
					if err != nil {
						return ActionMessage(err.Error())
					}
					batch := Batch()
					for _, field := range fields {
						batch = append(batch, ActionStyledValuesDescribed(field.Name, field.Description, field.Style).Tag(field.Tag))
					}
					return batch.Invoke(c).Merge().Suffix("=").ToA()

				default:
					return ActionValues()
				}
			})
		case 1:
			return ActionMultiParts(",", func(c Context) Action {
				return ActionStyles(c.Parts...).Invoke(c).Filter(c.Parts...).ToA().NoSpace()
			})
		default:
			return ActionValues()
		}
	})
}

// Actionstyles completes styles
//
//	blue
//	bg-magenta
func ActionStyles(styles ...string) Action {
	return ActionCallback(func(c Context) Action {
		fg := false
		bg := false

		for _, s := range styles {
			if strings.HasPrefix(s, "bg-") {
				bg = true
			}
			if s == style.Black ||
				s == style.Red ||
				s == style.Green ||
				s == style.Yellow ||
				s == style.Blue ||
				s == style.Magenta ||
				s == style.Cyan ||
				s == style.White ||
				s == style.BrightBlack ||
				s == style.BrightRed ||
				s == style.BrightGreen ||
				s == style.BrightYellow ||
				s == style.BrightBlue ||
				s == style.BrightMagenta ||
				s == style.BrightCyan ||
				s == style.BrightWhite ||
				strings.HasPrefix(s, "#") ||
				strings.HasPrefix(s, "color") ||
				strings.HasPrefix(s, "fg-") {
				fg = true
			}
		}

		batch := Batch()
		_s := func(s string) string {
			return style.Of(append(styles, s)...)
		}

		if !fg {
			batch = append(batch, ActionStyledValues(
				style.Black, _s(style.Black),
				style.Red, _s(style.Red),
				style.Green, _s(style.Green),
				style.Yellow, _s(style.Yellow),
				style.Blue, _s(style.Blue),
				style.Magenta, _s(style.Magenta),
				style.Cyan, _s(style.Cyan),
				style.White, _s(style.White),

				style.BrightBlack, _s(style.BrightBlack),
				style.BrightRed, _s(style.BrightRed),
				style.BrightGreen, _s(style.BrightGreen),
				style.BrightYellow, _s(style.BrightYellow),
				style.BrightBlue, _s(style.BrightBlue),
				style.BrightMagenta, _s(style.BrightMagenta),
				style.BrightCyan, _s(style.BrightCyan),
				style.BrightWhite, _s(style.BrightWhite),
			))

			if strings.HasPrefix(c.Value, "color") {
				for i := 0; i <= 255; i++ {
					batch = append(batch, ActionStyledValues(
						fmt.Sprintf("color%v", i), _s(style.XTerm256Color(uint8(i))),
					))
				}
			} else {
				batch = append(batch, ActionStyledValues("color", style.Of(styles...)))
			}
		}

		if !bg {
			batch = append(batch, ActionStyledValues(
				style.BgBlack, _s(style.BgBlack),
				style.BgRed, _s(style.BgRed),
				style.BgGreen, _s(style.BgGreen),
				style.BgYellow, _s(style.BgYellow),
				style.BgBlue, _s(style.BgBlue),
				style.BgMagenta, _s(style.BgMagenta),
				style.BgCyan, _s(style.BgCyan),
				style.BgWhite, _s(style.BgWhite),

				style.BgBrightBlack, _s(style.BgBrightBlack),
				style.BgBrightRed, _s(style.BgBrightRed),
				style.BgBrightGreen, _s(style.BgBrightGreen),
				style.BgBrightYellow, _s(style.BgBrightYellow),
				style.BgBrightBlue, _s(style.BgBrightBlue),
				style.BgBrightMagenta, _s(style.BgBrightMagenta),
				style.BgBrightCyan, _s(style.BgBrightCyan),
				style.BgBrightWhite, _s(style.BgBrightWhite),
			))

			if strings.HasPrefix(c.Value, "bg-color") {
				for i := 0; i <= 255; i++ {
					batch = append(batch, ActionStyledValues(
						fmt.Sprintf("bg-color%v", i), _s("bg-"+style.XTerm256Color(uint8(i))),
					))
				}
			} else {
				batch = append(batch, ActionStyledValues("bg-color", style.Of(styles...)))
			}
		}

		batch = append(batch, ActionStyledValues(
			style.Bold, _s(style.Bold),
			style.Dim, _s(style.Dim),
			style.Italic, _s(style.Italic),
			style.Underlined, _s(style.Underlined),
			style.Blink, _s(style.Blink),
			style.Inverse, _s(style.Inverse),
		))

		return batch.ToA().NoSpace('r')
	}).Tag("styles")
}

// ActionExecutables completes executables either from PATH or given directories
//
//	nvim
//	chmod
func ActionExecutables(dirs ...string) Action {
	return ActionCallback(func(c Context) Action {
		if len(dirs) == 0 {
			dirs = strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
		}
		manDescriptions := man.Descriptions(c.Value) // TODO allow additional descriptions to be registered somewhere for carapace-bin (key, value,...)
		batch := Batch()
		for i := len(dirs) - 1; i >= 0; i-- {
			batch = append(batch, actionDirectoryExecutables(dirs[i], c.Value, manDescriptions))
		}
		return batch.ToA().
			UidF(func(s string, uc uid.Context) (*url.URL, error) {
				return &url.URL{Scheme: "cmd", Host: uid.PathEscape(s)}, nil
			}).
			Query("cmd", "", "", "directories", strings.Join(dirs, ", "))
	}).Tag("executables")
}

func actionDirectoryExecutables(dir string, prefix string, manDescriptions map[string]string) Action {
	return ActionCallback(func(c Context) Action {
		abs, err := c.Abs(dir)
		if err != nil {
			return ActionMessage(err.Error())
		}

		if files, err := os.ReadDir(abs); err == nil {
			vals := make([]string, 0)
			for _, f := range files {
				if match.HasPrefix(f.Name(), prefix) {
					if info, err := f.Info(); err == nil && !f.IsDir() && isExecAny(info.Mode()) {
						vals = append(vals, f.Name(), manDescriptions[f.Name()], style.ForPath(abs+"/"+f.Name(), c))
					}
				}
			}
			return ActionStyledValuesDescribed(vals...).UidF(func(s string, uc uid.Context) (*url.URL, error) {
				return url.Parse(fmt.Sprintf("file://%v/%v", uid.PathEscape(dir), uid.PathEscape(s))) // TODO trim slash suffix from dir | backslash path possible? (windows)
			})
		}
		return ActionValues()
	})
}

func isExecAny(mode os.FileMode) bool {
	return mode&0o111 != 0
}

// ActionPositional completes positional arguments for given command ignoring `--` (dash).
// TODO: experimental - likely gives issues with preinvoke (does not have the full args)
//
//	carapace.Gen(cmd).DashAnyCompletion(
//		carapace.ActionPositional(cmd),
//	)
func ActionPositional(cmd *cobra.Command) Action {
	return ActionCallback(func(c Context) Action {
		if cmd.ArgsLenAtDash() < 0 {
			return ActionMessage("only allowed for dash arguments [ActionPositional]")
		}

		c.Args = cmd.Flags().Args()
		entry := storage.get(cmd)

		var a Action
		if entry.positionalAny != nil {
			a = *entry.positionalAny
		}

		if index := len(c.Args); index < len(entry.positional) {
			a = entry.positional[len(c.Args)]
		}
		return a.Invoke(c).ToA()
	})
}

// ActionCommands completes (sub)commands of given command.
// `Context.Args` is used to traverse the command tree further down. Use `Action.Shift` to avoid this.
//
//	carapace.Gen(helpCmd).PositionalAnyCompletion(
//		carapace.ActionCommands(rootCmd),
//	)
func ActionCommands(cmd *cobra.Command) Action {
	return ActionCallback(func(c Context) Action {
		if len(c.Args) > 0 {
			for _, subCommand := range cmd.Commands() {
				for _, name := range append(subCommand.Aliases, subCommand.Name()) {
					if name == c.Args[0] { // cmd.Find is too lenient
						return ActionCommands(subCommand).Shift(1)
					}
				}
			}
			return ActionMessage("unknown subcommand %#v for %#v", c.Args[0], cmd.Name())
		}

		batch := Batch()
		for _, subcommand := range cmd.Commands() {
			switch {
			case subcommand.Hidden && subcommand.Name() == "_carapace" && env.Hidden() != env.HIDDEN_INCLUDE_CARAPACE:
				continue // skip `_carapace` subcommand
			case subcommand.Hidden && env.Hidden() == env.HIDDEN_NONE:
				continue // skip all hidden commands
			case subcommand.Deprecated != "":
				continue // skip deprecated commands
			}

			group := common.Group{Cmd: subcommand}
			batch = append(batch, ActionStyledValuesDescribed(subcommand.Name(), subcommand.Short, group.Style()).Tag(group.Tag()))
			for _, alias := range subcommand.Aliases {
				batch = append(batch, ActionStyledValuesDescribed(alias, subcommand.Short, group.Style()).Tag(group.Tag()))
			}
		}
		return batch.ToA().UidF(func(s string, uc uid.Context) (*url.URL, error) {
			uid := uid.Command(cmd)
			if subCommand, _, err := cmd.Find([]string{s}); err == nil {
				s = subCommand.Name() // alias -> actual name
			}

			switch uid.Path {
			case "":
				uid.Path = url.PathEscape(s)
			default:
				uid.Path = uid.Path + "/" + url.PathEscape(s)
			}
			return uid, nil
		})
	})
}

// ActionCora bridges given cobra completion function.
func ActionCobra(f func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)) Action {
	return ActionCallback(func(c Context) Action {
		switch {
		case f == nil:
			return ActionValues()
		case c.cmd == nil: // ensure cmd is never nil even if context does not contain one
			LOG.Print("cmd is nil [ActionCobra]")
			c.cmd = &cobra.Command{Use: "_carapace_actioncobra", Hidden: true, Deprecated: "dummy command for ActionCobra"}
		}

		if !c.cmd.DisableFlagParsing {
			c.Args = c.cmd.Flags().Args()
		}
		values, directive := f(c.cmd, c.Args, c.Value)
		return compDirective(directive).ToA(values...)
	})
}
