package carapace

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/carapace-sh/carapace/internal/env"
	"github.com/carapace-sh/carapace/internal/pflagfork"
	"github.com/carapace-sh/carapace/pkg/style"
	"github.com/carapace-sh/carapace/pkg/uid"
	"github.com/carapace-sh/carapace/pkg/util"
	"github.com/spf13/cobra"
)

func actionPath(fileSuffixes []string, dirOnly bool) Action {
	return ActionCallback(func(c Context) Action {
		if len(c.Value) == 2 && util.HasVolumePrefix(c.Value) {
			// TODO should be fixed in Abs or wherever this is happening
			return ActionValues(c.Value + "/") // prevent `C:` -> `C:.`
		}

		abs, err := c.Abs(c.Value)
		if err != nil {
			return ActionMessage(err.Error())
		}

		displayFolder := filepath.ToSlash(filepath.Dir(c.Value))
		if displayFolder == "." {
			displayFolder = ""
		} else if !strings.HasSuffix(displayFolder, "/") {
			displayFolder = displayFolder + "/"
		}

		actualFolder := filepath.ToSlash(filepath.Dir(abs))
		files, err := os.ReadDir(actualFolder)
		if err != nil {
			return ActionMessage(err.Error())
		}

		showHidden := !strings.HasSuffix(abs, "/") && strings.HasPrefix(filepath.Base(abs), ".")

		vals := make([]string, 0, len(files)*2)
		for _, file := range files {
			if !showHidden && strings.HasPrefix(file.Name(), ".") {
				continue
			}

			info, err := file.Info()
			if err != nil {
				return ActionMessage(err.Error())
			}
			symlinkedDir := false
			if evaluatedPath, err := filepath.EvalSymlinks(actualFolder + "/" + file.Name()); err == nil {
				if evaluatedInfo, err := os.Stat(evaluatedPath); err == nil {
					symlinkedDir = evaluatedInfo.IsDir()
				}
			}

			switch {
			case info.IsDir():
				vals = append(vals, displayFolder+file.Name()+"/", style.ForPath(filepath.Clean(actualFolder+"/"+file.Name()+"/"), c))
			case symlinkedDir:
				vals = append(vals, displayFolder+file.Name()+"/", style.ForPath(filepath.Clean(actualFolder+"/"+file.Name()), c)) // TODO colorist not returning the symlink color
			case !dirOnly:
				if len(fileSuffixes) == 0 {
					fileSuffixes = []string{""}
				}
				for _, suffix := range fileSuffixes {
					if strings.HasSuffix(file.Name(), suffix) {
						vals = append(vals, displayFolder+file.Name(), style.ForPath(filepath.Clean(actualFolder+"/"+file.Name()), c))
						break
					}
				}
			}
		}
		if strings.HasPrefix(c.Value, "./") {
			return ActionStyledValues(vals...).Invoke(Context{}).Prefix("./").ToA()
		}
		return ActionStyledValues(vals...)
	})
}

func actionFlags(cmd *cobra.Command) Action {
	return ActionCallback(func(c Context) Action {
		cmd.InitDefaultHelpFlag()
		cmd.InitDefaultVersionFlag()

		flagSet := pflagfork.FlagSet{FlagSet: cmd.Flags()}
		isShorthandSeries := flagSet.IsShorthandSeries(c.Value)

		nospace := make([]rune, 0)
		batch := Batch()
		flagSet.VisitAll(func(f *pflagfork.Flag) {
			switch {
			case f.Hidden && env.Hidden() == env.HIDDEN_NONE:
				return // skip hidden flags
			case f.Deprecated != "":
				return // skip deprecated flags
			case f.Changed && !f.IsRepeatable():
				return // don't repeat flag
			case flagSet.IsMutuallyExclusive(f.Flag):
				return // skip flag of group already set
			}

			if isShorthandSeries {
				if f.Shorthand != "" && f.ShorthandDeprecated == "" {
					for _, shorthand := range c.Value[1:] {
						if shorthandFlag := cmd.Flags().ShorthandLookup(string(shorthand)); shorthandFlag != nil && shorthandFlag.Value.Type() != "bool" && shorthandFlag.Value.Type() != "count" && shorthandFlag.NoOptDefVal == "" {
							return // abort shorthand flag series if a previous one is not bool or count and requires an argument (no default value)
						}
					}
					batch = append(batch, ActionStyledValuesDescribed(f.Shorthand, f.Usage, f.Style()).Tag("shorthand flags").
						UidF(func(s string, uc uid.Context) (*url.URL, error) { return uid.Flag(cmd, f), nil }))
					if f.IsOptarg() {
						nospace = append(nospace, []rune(f.Shorthand)[0])
					}
				}
			} else {
				switch f.Mode() {
				case pflagfork.NameAsShorthand:
					batch = append(batch, ActionStyledValuesDescribed("-"+f.Name, f.Usage, f.Style()).Tag("longhand flags").
						UidF(func(s string, uc uid.Context) (*url.URL, error) { return uid.Flag(cmd, f), nil }))
				case pflagfork.Default:
					batch = append(batch, ActionStyledValuesDescribed("--"+f.Name, f.Usage, f.Style()).Tag("longhand flags").
						UidF(func(s string, uc uid.Context) (*url.URL, error) { return uid.Flag(cmd, f), nil }))
				}

				if f.Shorthand != "" && f.ShorthandDeprecated == "" {
					batch = append(batch, ActionStyledValuesDescribed("-"+f.Shorthand, f.Usage, f.Style()).Tag("shorthand flags").
						UidF(func(s string, uc uid.Context) (*url.URL, error) { return uid.Flag(cmd, f), nil }))
				}
			}
		})

		if isShorthandSeries {
			if len(nospace) > 0 {
				return batch.ToA().Prefix(c.Value).NoSpace(nospace...)
			}
			return batch.ToA().Prefix(c.Value)
		}
		return batch.ToA().MultiParts(".") // multiparts completion for flags grouped with `.`
	})
}

func initHelpCompletion(cmd *cobra.Command) {
	helpCmd, _, err := cmd.Find([]string{"help"})
	if err != nil {
		return
	}

	if helpCmd.Name() != "help" ||
		helpCmd.Short != "Help about any command" ||
		!strings.HasPrefix(helpCmd.Long, `Help provides help for any command in the application.`) {
		return
	}

	Gen(helpCmd).PositionalAnyCompletion(
		ActionCommands(cmd),
	)
}
