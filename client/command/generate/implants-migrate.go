package generate

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox
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
	"context"
	"fmt"
	"os"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
)

func ExportImplantCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	name, _ := cmd.Flags().GetString("name")
	all, _ := cmd.Flags().GetBool("all")
	save, _ := cmd.Flags().GetString("save")
	if name == "" && !all {
		con.PrintErrorf("Must specify --name <codename> or --all\n")
		return
	}
	if save == "" {
		con.PrintErrorf("Must specify --save <file.json>\n")
		return
	}
	req := &clientpb.ExportImplantReq{
		Name: name,
		All:  all,
	}
	bundle, err := con.Rpc.ExportImplant(context.Background(), req)
	if err != nil {
		con.PrintErrorf("Export failed: %s\n", err)
		return
	}
	if len(bundle.Bundles) == 0 {
		con.PrintInfof("No implant builds found to export\n")
		return
	}
	marshaller := protojson.MarshalOptions{
		Indent:          "  ",
		EmitUnpopulated: true,
	}
	data, err := marshaller.Marshal(bundle)
	if err != nil {
		con.PrintErrorf("Failed to marshal export data: %s\n", err)
		return
	}
	err = os.WriteFile(save, data, 0600)
	if err != nil {
		con.PrintErrorf("Failed to write file %s: %s\n", save, err)
		return
	}
	con.PrintInfof("Exported %d implant bundle(s) to %s\n", len(bundle.Bundles), save)
	con.PrintInfof("  %d certificate(s), %d key_value(s)\n", len(bundle.Certificates), len(bundle.KeyValues))
	for _, b := range bundle.Bundles {
		if b.Build != nil {
			c2Count := len(b.C2S)
			con.PrintInfof("  → %s (%d C2 endpoint(s))\n", b.Build.Name, c2Count)
		}
	}
}

func ImportImplantCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	filePath, _ := cmd.Flags().GetString("file")
	if filePath == "" {
		con.PrintErrorf("Must specify --file <file.json>\n")
		return
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		con.PrintErrorf("Failed to read file %s: %s\n", filePath, err)
		return
	}
	bundle := &clientpb.ExportImplantBundle{}
	err = protojson.Unmarshal(data, bundle)
	if err != nil {
		con.PrintErrorf("Failed to parse import file: %s\n", err)
		return
	}
	if len(bundle.Bundles) == 0 {
		con.PrintInfof("No bundles found in import file\n")
		return
	}
	con.PrintInfof("Importing %d implant bundles, %d certificates, %d key_values from %s ...\n",
		len(bundle.Bundles), len(bundle.Certificates), len(bundle.KeyValues), filePath)
	for _, b := range bundle.Bundles {
		if b.Build != nil {
			con.PrintInfof("  → %s\n", b.Build.Name)
		}
	}
	_, err = con.Rpc.ImportImplant(context.Background(), bundle)
	if err != nil {
		con.PrintErrorf("Import failed: %s\n", err)
		return
	}
	fmt.Println()
	con.PrintInfof("Import complete. Run 'implants' to verify.\n")
}
