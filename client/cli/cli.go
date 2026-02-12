package cli

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox
	Copyright (C) 2019 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

const (
	logFileName = "sliver-client.log"
)

// Initialize logging.
// 初始化日志。
func initLogging(appDir string) *os.File {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	logFile, err := os.OpenFile(path.Join(appDir, logFileName), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		panic(fmt.Sprintf("[!] Error opening file: %s", err))
	}
	log.SetOutput(logFile)
	return logFile
}

func init() {
	appDir := assets.GetRootAppDir()
	logFile := initLogging(appDir)
	defer logFile.Close()

	rootCmd.TraverseChildren = true
	rootCmd.Flags().String(RCFlagName, "", "path to rc script file")

	// Create the console client, without any RPC or commands bound to it yet.
	// 创建 console client，此时尚未绑定任何 RPC 或命令。
	// This created before anything so that multiple commands can make use of
	// 在其他初始化之前创建它，以便多个命令可以复用
	// the same underlying command/run infrastructure.
	// 相同的底层 command/run 基础设施。
	con := console.NewConsole(false)

	// Import
	// 导入
	rootCmd.AddCommand(importCmd())

	// Version
	// 版本
	rootCmd.AddCommand(cmdVersion)

	// Client console.
	// Client 控制台。
	// All commands and RPC connection are generated WITHIN the command RunE():
	// 所有命令与 RPC 连接都在命令 RunE() 内部生成：
	// that means there should be no redundant command tree/RPC connections with
	// 这意味着不应与下方其他命令树（例如 implant）产生冗余的命令树/RPC 连接，
	// other command trees below, such as the implant one.
	// 例如 implant 命令树。
	rootCmd.AddCommand(consoleCmd(con))

	// MCP stdio server.
	// MCP stdio 服务器。
	rootCmd.AddCommand(mcpCmd(con))

	// Implant.
	// implant 命令。
	// The implant command allows users to run commands on slivers from their
	// implant 命令允许用户从系统 shell 在 sliver 上执行命令。
	// system shell. It makes use of pre-runners for connecting to the server
	// 它使用 pre-runners 连接 server 并绑定 sliver 命令。
	// and binding sliver commands. These same pre-runners are also used for
	// 这些 pre-runners 也用于
	// command completion/filtering purposes.
	// 命令补全/过滤场景。
	rootCmd.AddCommand(implantCmd(con))

	// No subcommand invoked means starting the console.
	// 未调用子命令时，启动 console。
	rootCmd.RunE, rootCmd.PostRunE = consoleRunnerCmd(con, true)

	// Completions
	// 补全
	carapace.Gen(rootCmd)
}

var rootCmd = &cobra.Command{
	Use:   "sliver-client",
	Short: "",
	Long:  ``,
}

// Execute-挂载根命令
// Execute - Execute root command.
// Execute - 执行根命令。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("root command: %s\n", err)
		os.Exit(1)
	}
}
