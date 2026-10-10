package cli

/*
	Sliver Implant Framework
	Copyright (C) 2020  Bishop Fox

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

import (
	"fmt"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

func importCmd() *cobra.Command {
	cmdImport := &cobra.Command{
		Use:   "import",
		Short: "Import a client configuration file",
		Long:  `import [config files]`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				_, _ = fmt.Fprint(cmd.OutOrStdout(), "Missing config file path, see --help")
				return nil
			}
			return importConfigs(args)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{}, cobra.ShellCompDirectiveDefault
		},
	}

	carapace.Gen(cmdImport).PositionalCompletion(carapace.ActionFiles().Tag("server configuration"))

	return cmdImport
}

func importConfigs(paths []string) error {
	for _, path := range paths {
		conf, err := assets.ReadConfig(path)
		if err != nil {
			return &commandExitError{
				code: 3,
				err:  fmt.Errorf("read import %q: %w", path, err),
			}
		}
		if err := assets.SaveConfig(conf); err != nil {
			return fmt.Errorf("save import %q: %w", path, err)
		}
	}
	return nil
}
