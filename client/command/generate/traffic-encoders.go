package generate

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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/util"
	"github.com/gofrs/uuid"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/protobuf/proto"
)

// TrafficEncodersCmd - Generate traffic encoders command implementation.
// TrafficEncodersCmd - Generate 流量编码器命令 implementation.
func TrafficEncodersCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	grpcCtx, cancel := con.GrpcContext(cmd)
	defer cancel()
	encoderMap, err := con.Rpc.TrafficEncoderMap(grpcCtx, &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
	DisplayTrafficEncoders(encoderMap, con)
}

// DisplayTrafficEncoders - Display traffic encoders map from server.
// DisplayTrafficEncoders - Display 流量编码器映射自 server.
func DisplayTrafficEncoders(encoderMap *clientpb.TrafficEncoderMap, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"ID",
		"Name",
		"Size (Uncompressed)",
	})
	tw.SortBy([]table.SortBy{
		{Name: "Name", Mode: table.Asc},
	})
	allIDs := []uint64{}
	for _, encoder := range encoderMap.Encoders {
		isDuplicate := util.Contains(allIDs, encoder.ID)
		allIDs = append(allIDs, encoder.ID)
		name := encoder.Wasm.Name
		if isDuplicate {
			name = console.StyleBoldRed.Render(fmt.Sprintf("%s (duplicate id)", name))
		}
		tw.AppendRow(table.Row{
			encoder.ID,
			name,
			util.ByteCountBinary(int64(len(encoder.Wasm.Data))),
		})
	}
	con.Println(tw.Render())
}

// TrafficEncodersAddCmd - Add a new traffic encoder to the server.
// TrafficEncodersAddCmd - Add 到 server. 的新流量编码器
func TrafficEncodersAddCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	grpcCtx, cancel := con.GrpcContext(cmd)
	defer cancel()

	data, err := os.ReadFile(args[0])
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}

	skipTests, _ := cmd.Flags().GetBool("skip-tests")
	testID := uuid.Must(uuid.NewV4()).String()
	trafficEncoder := &clientpb.TrafficEncoder{
		Wasm: &commonpb.File{
			Name: filepath.Base(args[0]),
			Data: data,
		},
		SkipTests: skipTests,
		TestID:    testID,
	}

	// Spin out a goroutine to display progress
	// Spin 出一个 goroutine 来显示进度
	completed := make(chan interface{})
	go func() {
		displayTrafficEncoderTestProgress(testID, completed, con)
	}()

	// Wait for tests to complete, then display final result
	// Wait 用于完成测试，然后显示最终结果
	tests, err := con.Rpc.TrafficEncoderAdd(grpcCtx, trafficEncoder)
	completed <- nil
	<-completed
	close(completed)
	if err != nil {
		con.PrintErrorf("Failed to add traffic encoder %s", err)
		con.Println()
		return
	}
	displayTrafficEncoderTests(false, tests, con)
	con.Println()
	if !allTestsPassed(tests) {
		con.Println()
		con.PrintErrorf("%s failed tests!\n", trafficEncoder.Wasm.Name)
		return
	} else {
		for _, test := range tests.Tests {
			if !test.Success && test.Sample != nil {
				saveFailedSample(trafficEncoder.Wasm.Name, test)
			}
		}
	}
	con.Println()
	con.PrintInfof("Successfully added traffic encoder: %s\n", trafficEncoder.Wasm.Name)
}

// saveFailedSample - Save the sample the encoder failed to properly encode/decode.
// saveFailedSample - Save 编码器未能正确采样 encode/decode.
func saveFailedSample(encoderName string, test *clientpb.TrafficEncoderTest) {
	confirm := false
	_ = forms.Confirm(fmt.Sprintf("Failed to add traffic encoder %s, save failed sample to disk?", encoderName), &confirm)
	if !confirm {
		return
	}
	sampleFileName := fmt.Sprintf("sample-failed_%s_%s.bin", time.Now().Format("2006-01-02-15-04-05"), filepath.Base(encoderName))
	err := os.WriteFile(sampleFileName, test.Sample, 0o644)
	if err != nil {
		fmt.Printf("Failed to save failed sample to disk: %s", err)
		return
	}
}

// allTestsPassed - Check if all tests passed.
// allTestsPassed - Check 如果所有测试都是 passed.
func allTestsPassed(tests *clientpb.TrafficEncoderTests) bool {
	for _, test := range tests.Tests {
		if !test.Success {
			return false
		}
	}
	return true
}

// displayTrafficEncoderTests - Display traffic encoder tests in real time.
// displayTrafficEncoderTests - Display 流量编码器实际测试 time.
func displayTrafficEncoderTestProgress(testID string, completed chan interface{}, con *console.SliverClient) {
	listenerID, events := con.CreateEventListener()
	defer con.RemoveEventListener(listenerID)
	lineCount := 0
	for {
		select {
		case event := <-events:
			if event.EventType == consts.TrafficEncoderTestProgressEvent {
				tests := &clientpb.TrafficEncoderTests{}
				proto.Unmarshal(event.Data, tests)
				if tests.Encoder.TestID != testID {
					continue
				}
				clearLines(lineCount-1, con)
				lineCount = displayTrafficEncoderTests(true, tests, con)
			}
		case <-completed:
			clearLines(lineCount-1, con)
			completed <- nil
			return
		}
	}
}

// clearLines - Clear a number of lines from the console.
// clearLines - Clear 来自 console. 的多行
func clearLines(count int, con *console.SliverClient) {
	for i := 0; i < count; i++ {
		con.Printf(console.Clearln + "\r")
		con.Printf(console.UpN, 1)
	}
}

// displayTrafficEncoderTests - Display the results of traffic encoder tests, return number of lines written.
// displayTrafficEncoderTests - Display 流量编码器测试结果，返回行数 written.
func displayTrafficEncoderTests(running bool, tests *clientpb.TrafficEncoderTests, con *console.SliverClient) int {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Test",
		"Result",
		"Duration",
		"Error",
	})
	tw.SortBy([]table.SortBy{
		{Name: "Test", Mode: table.Asc},
	})
	titleCase := cases.Title(language.AmericanEnglish)
	for _, test := range tests.Tests {
		var success string
		if test.Success {
			success = console.StyleBoldGreen.Render("Passed")
		} else {
			success = console.StyleBoldRed.Render("Failed!")
		}
		errorMsg := "N/A"
		if test.Err != "" {
			errorMsg = titleCase.String(test.Err)
		}
		tw.AppendRow(table.Row{
			test.Name,
			success,
			time.Duration(test.Duration),
			errorMsg,
		})
	}
	tableText := tw.Render()
	con.Printf(console.Clearln+"\r%s\r", tableText)
	lineCount := len(strings.Split(tableText, "\n"))

	if running {
		con.Println()
		con.Println()
		con.Printf("%s\r", console.StyleBold.Render(fmt.Sprintf("  >>> Running test %d of %d please wait ...", len(tests.Tests), tests.TotalTests)))
		lineCount += 2
	}

	return lineCount
}

// TrafficEncodersRemoveCmd - Remove a traffic encoder.
// TrafficEncodersRemoveCmd - Remove 交通 encoder.
func TrafficEncodersRemoveCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	_, cancel := con.GrpcContext(cmd)
	defer cancel()

	var name string
	if len(args) > 0 {
		name = args[0]
	}
	if name == "" {
		name = SelectTrafficEncoder(con)
	}
	grpcCtx, cancel := con.GrpcContext(cmd)
	defer cancel()
	_, err := con.Rpc.TrafficEncoderRm(grpcCtx, &clientpb.TrafficEncoder{
		Wasm: &commonpb.File{
			Name: name,
		},
	})
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
	con.Println()
	con.PrintInfof("Successfully removed traffic encoder: %s\n", name)
}

// SelectTrafficEncoder - Select a traffic encoder from a list.
// SelectTrafficEncoder - Select 来自 list. 的流量编码器
func SelectTrafficEncoder(con *console.SliverClient) string {
	grpcCtx, cancel := con.GrpcContext(nil)
	defer cancel()
	encoders, err := con.Rpc.TrafficEncoderMap(grpcCtx, &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s", err)
		return ""
	}
	var encoderNames []string
	for _, encoder := range encoders.Encoders {
		encoderNames = append(encoderNames, encoder.Wasm.Name)
	}
	sort.Strings(encoderNames)
	var selectedEncoder string
	err = forms.Select("Select a traffic encoder:", encoderNames, &selectedEncoder)
	if err != nil {
		con.PrintErrorf("%s", err)
		return ""
	}
	return selectedEncoder
}
