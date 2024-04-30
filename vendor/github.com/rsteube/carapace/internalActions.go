package carapace

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/rsteube/carapace/internal/env"
	"github.com/rsteube/carapace/internal/pflagfork"
	"github.com/rsteube/carapace/pkg/style"
	"github.com/rsteube/carapace/pkg/util"
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
		files, err := ioutil.ReadDir(actualFolder)
		if err != nil {
			return ActionMessage(err.Error())
		}

		showHidden := !strings.HasSuffix(abs, "/") && strings.HasPrefix(filepath.Base(abs), ".")

		vals := make([]string, 0, len(files)*2)
		for _, file := range files {
			if !showHidden && strings.HasPrefix(file.Name(), ".") {
				continue
			}

			resolvedFile := file
			if resolved, err := filepath.EvalSymlinks(actualFolder + file.Name()); err == nil {
				if stat, err := os.Stat(resolved); err == nil {
					resolvedFile = stat
				}
			}

			if resolvedFile.IsDir() {
				vals = append(vals, displayFolder+file.Name()+"/", style.ForPath(filepath.Clean(actualFolder+"/"+file.Name()+"/"), c))
			} else if !dirOnly {
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
		vals := make([]string, 0)
		flagSet.VisitAll(func(f *pflagfork.Flag) {
			switch {
			case f.Hidden && !env.Hidden():
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
					vals = append(vals, f.Shorthand, f.Usage, f.Style())
					if f.IsOptarg() {
						nospace = append(nospace, []rune(f.Shorthand)[0])
					}
				}
			} else {
				switch f.Mode() {
				case pflagfork.NameAsShorthand:
					vals = append(vals, "-"+f.Name, f.Usage, f.Style())
				case pflagfork.Default:
					vals = append(vals, "--"+f.Name, f.Usage, f.Style())
				}

				if f.Shorthand != "" && f.ShorthandDeprecated == "" {
					vals = append(vals, "-"+f.Shorthand, f.Usage, f.Style())
				}
			}
		})

		if isShorthandSeries {
			if len(nospace) > 0 {
				return ActionStyledValuesDescribed(vals...).Prefix(c.Value).NoSpace(nospace...)
			}
			return ActionStyledValuesDescribed(vals...).Prefix(c.Value)
		}
		return ActionStyledValuesDescribed(vals...).MultiParts(".") // multiparts completion for flags grouped with `.`
	}).Tag("flags")
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
