package jobs

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
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// JobsCmd - Manage server jobs (listeners, etc).
func JobsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	if pid, _ := cmd.Flags().GetInt32("kill"); pid != -1 {
		jobKill(uint32(pid), con)
	} else if all, _ := cmd.Flags().GetBool("kill-all"); all {
		killAllJobs(con)
	} else {
		jobs, err := con.Rpc.GetJobs(context.Background(), &commonpb.Empty{})
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		// Convert to a map
		activeJobs := map[uint32]*clientpb.Job{}
		for _, job := range jobs.Active {
			activeJobs[job.ID] = job
		}
		if 0 < len(activeJobs) {
			PrintJobs(activeJobs, con)
		} else {
			con.PrintInfof("No active jobs\n")
		}
	}
}

// PrintJobs - Prints a list of active jobs.
func PrintJobs(jobs map[uint32]*clientpb.Job, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"ID",
		"Name",
		"Protocol",
		"Port",
		"Domains",
	})

	var keys []int
	for _, job := range jobs {
		keys = append(keys, int(job.ID))
	}
	sort.Ints(keys)

	for _, k := range keys {
		job := jobs[uint32(k)]
		tw.AppendRow(table.Row{
			fmt.Sprintf("%d", job.ID),
			job.Name,
			job.Protocol,
			fmt.Sprintf("%d", job.Port),
			strings.Join(job.Domains, ","),
		})
	}
	con.Printf("%s\n", tw.Render())
}

// JobsIDCompleter completes jobs IDs with descriptions.
func JobsIDCompleter(con *console.SliverClient) carapace.Action {
	callback := func(_ carapace.Context) carapace.Action {
		jobs, err := con.Rpc.GetJobs(context.Background(), &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage("No active jobs")
		}

		results := make([]string, 0)

		for _, job := range jobs.Active {
			results = append(results, strconv.Itoa(int(job.ID)))
			desc := fmt.Sprintf("%s  %s %d", job.Protocol, strings.Join(job.Domains, ","), job.Port)
			results = append(results, desc)
		}

		return carapace.ActionValuesDescribed(results...).Tag("jobs")
	}

	return carapace.ActionCallback(callback)
}

func jobKill(jobID uint32, con *console.SliverClient) {
	con.PrintInfof("Killing job #%d ...\n", jobID)
	jobKill, err := con.Rpc.KillJob(context.Background(), &clientpb.KillJobReq{
		ID: jobID,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("Successfully killed job #%d\n", jobKill.ID)
	}
}

func killAllJobs(con *console.SliverClient) {
	jobs, err := con.Rpc.GetJobs(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	for _, job := range jobs.Active {
		jobKill(job.ID, con)
	}
}
