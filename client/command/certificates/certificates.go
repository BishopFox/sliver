package certificates

import (
	"context"
	"time"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

/*
	Sliver Implant Framework
	Copyright (C) 2024  Bishop Fox
	Copyright (C) 2024 Bishop Fox

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

const (
	timeFormat = "2006-01-02 15:04:05 UTC-0700"
)

// Defining the transport filters
// Defining 传输过滤器
const (
	MTLSTransport uint32 = 1 << iota
	HTTPSTransport
	AllTransports
)

// Defining the role filters
// Defining 角色过滤器
const (
	// Provide some separation between the options so that we do not have duplicate combinations
	// Provide 选项之间有一些分离，这样我们就不会出现重复的组合
	ServerRole uint32 = 8 << iota
	ImplantRole
	AllRoles
)

func CertificateInfoCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	// Since we are sending this value in a protobuf, we will give it a fixed bit size
	// Since 我们在 protobuf 中发送这个值，我们将给它一个固定的位大小
	// 32 is the smallest we can go
	// 32是我们能走的最小的
	var chosenOptions uint32

	if cmd.Flags().Changed("mtls") {
		if cmd.Flags().Changed("https") {
			chosenOptions = AllTransports
		} else {
			chosenOptions = MTLSTransport
		}
	} else if cmd.Flags().Changed("https") {
		chosenOptions = HTTPSTransport
	} else {
		chosenOptions = AllTransports
	}

	if cmd.Flags().Changed("server") {
		if cmd.Flags().Changed("implant") {
			chosenOptions |= AllRoles
		} else {
			chosenOptions |= ServerRole
		}
	} else if cmd.Flags().Changed("implant") {
		chosenOptions |= ImplantRole
	} else {
		chosenOptions |= AllRoles
	}

	request := &clientpb.CertificatesReq{
		CategoryFilters: chosenOptions,
	}

	request.CN, _ = cmd.Flags().GetString("cn")

	// Ask the server for information about certificates
	// Ask 证书信息服务器
	certificateInfo, err := con.Rpc.GetCertificateInfo(context.Background(), request)
	if err != nil {
		con.PrintErrorf("could not get certificate information from database: %s", err.Error())
		return
	}

	printCertificateInfo(con, certificateInfo.Info)
}

func checkCertExpiry(expiryTime time.Time) console.TextStyle {
	if expiryTime.Before(time.Now()) || expiryTime.Equal(time.Now()) {
		return console.StyleBoldRed
	}

	// One week is 168 hours - this is bad
	// One 周是 168 小时 - 这很糟糕
	if expiryTime.Before(time.Now().Add(168 * time.Hour)) {
		return console.StyleBoldRed
	}

	// One month is approximately 730 hours - this is a warning
	// One 月大约为 730 小时 - 这是一个警告
	if expiryTime.Before(time.Now().Add(730 * time.Hour)) {
		return console.StyleBoldOrange
	}

	return console.StyleNormal
}

func printCertificateInfo(con *console.SliverClient, certData []*clientpb.CertificateData) {
	// Get the terminal width
	// Get 端子宽度
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 999
	}

	if len(certData) == 0 {
		con.PrintWarnf("There are no certificates in the database matching the given parameters.\n")
		return
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	wideTermWidth := con.Settings.SmallTermWidth < width

	if wideTermWidth {
		tw.AppendHeader(table.Row{
			"ID",
			"Common Name",
			"Creation Time",
			"Certificate Type",
			"Key Algorithm",
			"Validity Start",
			"Expires",
		})
	} else {
		tw.AppendHeader(table.Row{
			"ID",
			"Common Name",
			"Expires",
		})
	}

	for _, cert := range certData {
		rowStyle := console.StyleNormal

		expiry, err := time.Parse(timeFormat, cert.ValidityExpiry)
		// This should not error out, but if it does, the row will not be colored
		// This 不应出错，但如果出错，该行将不会被着色
		if err == nil {
			rowStyle = checkCertExpiry(expiry)
		}
		if wideTermWidth {
			tw.AppendRow(table.Row{
				rowStyle.Render(cert.ID),
				rowStyle.Render(cert.CN),
				rowStyle.Render(cert.CreationTime),
				rowStyle.Render(cert.Type),
				rowStyle.Render(cert.KeyAlgorithm),
				rowStyle.Render(cert.ValidityStart),
				rowStyle.Render(cert.ValidityExpiry),
			})
		} else {
			tw.AppendRow(table.Row{
				rowStyle.Render(cert.ID),
				rowStyle.Render(cert.CN),
				rowStyle.Render(cert.ValidityExpiry),
			})
		}

	}

	tw.SortBy([]table.SortBy{{Name: "Expires", Mode: table.Dsc}})

	con.Printf("%s\n", tw.Render())
}
