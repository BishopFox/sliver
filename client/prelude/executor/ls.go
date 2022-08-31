package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bishopfox/sliver/client/prelude/bridge"
	"github.com/bishopfox/sliver/client/prelude/config"
	"github.com/bishopfox/sliver/client/prelude/implant"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"google.golang.org/protobuf/proto"
)

func LsHandler(arguments interface{}, _ []byte, impBridge *bridge.OperatorImplantBridge, cb func(string, int, int), outputFormat string) (string, int, int) {
	path, ok := arguments.(map[string]interface{})["Path"].(string)
	if !ok {
		return sendError(errors.New("invalid path"))
	}
	lsResp, err := impBridge.RPC.Ls(context.Background(), &sliverpb.LsReq{
		Path:    path,
		Request: implant.MakeRequest(impBridge.Implant),
	})
	if err != nil {
		return sendError(err)
	}
	// Async callback
	if lsResp.Response != nil && lsResp.Response.Async {
		impBridge.BeaconCallback(lsResp.Response.TaskID, func(task *clientpb.BeaconTask) {
			err := proto.Unmarshal(task.Response, lsResp)
			if err != nil {
				cb(sendError(err))
				return
			}
			cb(handleLsOutput(lsResp, int(impBridge.Implant.GetPID()), outputFormat))
		})
		return "", config.SuccessExitStatus, int(impBridge.Implant.GetPID())
	}
	// Sync response
	if lsResp.Response != nil && lsResp.Response.Err != "" {
		return sendError(errors.New(lsResp.Response.Err))
	}
	return handleLsOutput(lsResp, int(impBridge.Implant.GetPID()), outputFormat)
}

func handleLsOutput(lsResp *sliverpb.Ls, pid int, format string) (string, int, int) {
	if format == "json" {
		return JSONFormatter(lsResp, pid)
	}
	return string(PrintLs(lsResp)), config.SuccessExitStatus, pid
}

func PrintLs(ls *sliverpb.Ls) []byte {
	out := strings.Builder{}
	// Generate metadata to print with the path
	numberOfFiles := len(ls.Files)
	var totalSize int64 = 0
	var pathInfo string

	for _, fileInfo := range ls.Files {
		totalSize += fileInfo.Size
	}

	if numberOfFiles == 1 {
		pathInfo = fmt.Sprintf("%s (%d item, %s)", ls.Path, numberOfFiles, util.ByteCountBinary(totalSize))
	} else {
		pathInfo = fmt.Sprintf("%s (%d items, %s)", ls.Path, numberOfFiles, util.ByteCountBinary(totalSize))
	}

	out.WriteString(fmt.Sprintf("%s\n", pathInfo))
	out.WriteString(fmt.Sprintf("%s\n", strings.Repeat("=", len(pathInfo))))

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	sort.SliceStable(ls.Files, func(i, j int) bool {
		return strings.ToLower(ls.Files[i].Name) < strings.ToLower(ls.Files[j].Name)
	})

	sort.SliceStable(ls.Files, func(i, j int) bool {
		return ls.Files[i].ModTime < ls.Files[j].ModTime
	})

	for _, fileInfo := range ls.Files {
		modTime := time.Unix(fileInfo.ModTime, 0)
		implantLocation := time.FixedZone(ls.Timezone, int(ls.TimezoneOffset))
		modTime = modTime.In(implantLocation)

		if fileInfo.IsDir {
			fmt.Fprintf(table, "%s\t%s\t<dir>\t%s\n", fileInfo.Mode, fileInfo.Name, modTime.Format(time.RubyDate))
		} else if fileInfo.Link != "" {
			fmt.Fprintf(table, "%s\t%s -> %s\t%s\t%s\n", fileInfo.Mode, fileInfo.Name, fileInfo.Link, util.ByteCountBinary(fileInfo.Size), modTime.Format(time.RubyDate))
		} else {
			fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", fileInfo.Mode, fileInfo.Name, util.ByteCountBinary(fileInfo.Size), modTime.Format(time.RubyDate))
		}
	}
	table.Flush()
	out.WriteString(outputBuf.String())
	return []byte(out.String())
}
