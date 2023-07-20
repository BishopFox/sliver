package generate

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

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
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
			name = fmt.Sprintf(console.Bold+console.Red+"%s (duplicate id)"+console.Normal, name)
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
	completed := make(chan interface{})
	go func() {
		displayTrafficEncoderTestProgress(testID, completed, con)
	}()

	// Wait for tests to complete, then display final result
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
func saveFailedSample(encoderName string, test *clientpb.TrafficEncoderTest) {
	confirm := false
	prompt := &survey.Confirm{
		Message: fmt.Sprintf("Failed to add traffic encoder %s, save failed sample to disk?", encoderName),
	}
	survey.AskOne(prompt, &confirm)
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
func allTestsPassed(tests *clientpb.TrafficEncoderTests) bool {
	for _, test := range tests.Tests {
		if !test.Success {
			return false
		}
	}
	return true
}

// displayTrafficEncoderTests - Display traffic encoder tests in real time.
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
func clearLines(count int, con *console.SliverClient) {
	for i := 0; i < count; i++ {
		con.Printf(console.Clearln + "\r")
		con.Printf(console.UpN, 1)
	}
}

// displayTrafficEncoderTests - Display the results of traffic encoder tests, return number of lines written.
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
			success = console.Bold + console.Green + "Passed" + console.Normal
		} else {
			success = console.Bold + console.Red + "Failed!" + console.Normal
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
		con.Printf(console.Bold+"  >>> Running test %d of %d please wait ...\r"+console.Normal, len(tests.Tests), tests.TotalTests)
		lineCount += 2
	}

	return lineCount
}

// TrafficEncodersRemoveCmd - Remove a traffic encoder.
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
	prompt := &survey.Select{
		Message: "Select a traffic encoder:",
		Options: encoderNames,
	}
	survey.AskOne(prompt, &selectedEncoder)
	return selectedEncoder
}
