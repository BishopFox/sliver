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
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/util"
	"github.com/desertbit/grumble"
	"github.com/gofrs/uuid"
	"github.com/jedib0t/go-pretty/v6/table"
	"google.golang.org/protobuf/proto"
)

// TrafficEncodersCmd - Generate traffic encoders command implementation
func TrafficEncodersCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	grpcCtx, cancel := con.GrpcContext(ctx)
	defer cancel()
	encoderMap, err := con.Rpc.TrafficEncoderMap(grpcCtx, &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
	DisplayTrafficEncoders(encoderMap, con)
}

// DisplayTrafficEncoders - Display traffic encoders map from server
func DisplayTrafficEncoders(encoderMap *clientpb.TrafficEncoderMap, con *console.SliverConsoleClient) {
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

// TrafficEncodersAddCmd - Add a new traffic encoder to the server
func TrafficEncodersAddCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	grpcCtx, cancel := con.GrpcContext(ctx)
	defer cancel()

	data, err := os.ReadFile(ctx.Args.String("file"))
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
	trafficEncoder := &clientpb.TrafficEncoder{
		Wasm: &commonpb.File{
			Name: filepath.Base(ctx.Args.String("name")),
			Data: data,
		},
	}

	// Spin out a goroutine to display progress
	completed := make(chan interface{}, 1)
	testID, _ := uuid.NewV4()
	go func() {
		displayTrafficEncoderTestProgress(testID.String(), completed, con)
	}()

	// Wait for tests to complete, then display final result
	tests, err := con.Rpc.TrafficEncoderAdd(grpcCtx, trafficEncoder)
	completed <- nil
	<-completed
	close(completed)
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
	displayTrafficEncoderTests(tests, con)
	con.Println()
	if !testsWereSuccessful(tests) {
		con.PrintErrorf("Failed to add traffic encoder: %s\n", trafficEncoder.Wasm.Name)
		return
	}
	con.PrintInfof("Successfully added traffic encoder: %s\n", trafficEncoder.Wasm.Name)
}

func testsWereSuccessful(tests *clientpb.TrafficEncoderTests) bool {
	for _, test := range tests.Tests {
		if !test.Success {
			return false
		}
	}
	return true
}

func displayTrafficEncoderTestProgress(testID string, completed chan interface{}, con *console.SliverConsoleClient) {
	listenerID, events := con.CreateEventListener()
	defer con.RemoveEventListener(listenerID)
	lineCount := 0
	for {
		select {
		case event := <-events:
			if event.EventType == consts.TrafficEncoderTestProgressEvent {
				tests := &clientpb.TrafficEncoderTests{}
				if tests.TestID != testID {
					continue
				}
				proto.Unmarshal(event.Data, tests)
				clearLines(lineCount, con)
				lineCount = displayTrafficEncoderTests(tests, con)
			}
		case <-completed:
			completed <- nil
			return
		}
	}
}

func clearLines(count int, con *console.SliverConsoleClient) {
	for i := 0; i < count; i++ {
		con.Printf(console.Clearln + "\r")
		con.Printf(console.UpN, 1)
	}
}

// displayTrafficEncoderTests - Display the results of traffic encoder tests, return number of lines written
func displayTrafficEncoderTests(tests *clientpb.TrafficEncoderTests, con *console.SliverConsoleClient) int {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Test",
		"Succeeded",
		"Duration",
	})
	tw.SortBy([]table.SortBy{
		{Name: "Test", Mode: table.Asc},
	})
	for _, test := range tests.Tests {
		tw.AppendRow(table.Row{
			test.Name,
			test.Success,
			time.Duration(test.Duration),
		})
	}
	tableText := tw.Render()
	con.Printf(console.Clearln+"\r%s\r", tableText)
	return len(strings.Split(tableText, "\n"))
}

// TrafficEncodersRemoveCmd - Remove a traffic encoder
func TrafficEncodersRemoveCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	_, cancel := con.GrpcContext(ctx)
	defer cancel()

}
