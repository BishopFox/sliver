package cli

import "github.com/spf13/cobra"

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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

var (
	// CATypes - CA types
	CATypes = []string{
		"operator",
		"grpc-server",
		"implant",
		"https",
	}
)

var cmdImportCA = &cobra.Command{
	Use:   "import-ca",
	Short: "Import certificate authority",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

var cmdExportCA = &cobra.Command{
	Use:   "export-ca",
	Short: "Export certificate authority",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

	},
}
