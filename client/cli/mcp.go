package cli

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

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
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	clientmcp "github.com/bishopfox/sliver/client/mcp"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/spf13/cobra"
)

func mcpCmd(con *console.SliverClient) *cobra.Command {
	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start the MCP stdio server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCP(cmd, con)
		},
	}
	mcpCmd.Flags().String("config", "", "path to client config file")
	return mcpCmd
}

func runMCP(cmd *cobra.Command, con *console.SliverClient) error {
	configPath, _ := cmd.Flags().GetString("config")
	config, err := loadMCPClientConfig(configPath)
	if err != nil {
		fmt.Printf("%s\n", err)
		return nil
	}

	rpc, ln, err := transport.MTLSConnect(config)
	if err != nil {
		fmt.Printf("Connection to server failed %s\n", err)
		return nil
	}
	defer ln.Close()

	con.Rpc = rpc
	go handleConnectionLost(ln)

	cfg := clientmcp.DefaultConfig()
	if err := clientmcp.ServeStdio(cfg, con.Rpc); err != nil {
		fmt.Printf("MCP server error: %s\n", err)
		return nil
	}
	return nil
}

func loadMCPClientConfig(path string) (*assets.ClientConfig, error) {
	if path != "" {
		configPath := path
		if !filepath.IsAbs(configPath) {
			if _, err := os.Stat(configPath); err != nil {
				configPath = filepath.Join(assets.GetConfigDir(), configPath)
			}
		}
		return assets.ReadConfig(configPath)
	}

	configs := assets.GetConfigs()
	switch len(configs) {
	case 0:
		return nil, fmt.Errorf("no config files found at %s (use --config)", assets.GetConfigDir())
	case 1:
		for _, config := range configs {
			return config, nil
		}
	default:
		return nil, fmt.Errorf("multiple configs found; use --config to select one")
	}
	return nil, fmt.Errorf("no config files found")
}
