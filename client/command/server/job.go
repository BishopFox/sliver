package server

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

	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// Jobs - Root jobs command.
type Jobs struct{}

// Execute - Root jobs command.
func (j *Jobs) Execute(args []string) (err error) {

	jobs, err := transport.RPC.GetJobs(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.RPCError+"%s", err)
		return
	}
	// Convert to a map
	activeJobs := map[uint32]*clientpb.Job{}
	for _, job := range jobs.Active {
		activeJobs[job.ID] = job
	}
	if 0 < len(activeJobs) {
		printJobs(activeJobs)
	} else {
		fmt.Printf(util.Info + "No active jobs\n")
	}
	return
}

// JobsKill - Kill a job given an ID
type JobsKill struct {
	Positional struct {
		JobID []uint32 `description:"active job ID" required:"1"`
	} `positional-args:"yes" required:"true"`
}

// Execute - Kill a job given an ID
func (j *JobsKill) Execute(args []string) (err error) {
	for _, jobID := range j.Positional.JobID {
		fmt.Printf(util.Info+"Killing job #%d ... \n", jobID)
		_, err := transport.RPC.KillJob(context.Background(), &clientpb.KillJobReq{
			ID: jobID,
		})
		if err != nil {
			fmt.Printf(util.Error+"%s\n", err)
		}
	}
	return
}

// JobsKillAll - Kill all active server jobs
type JobsKillAll struct{}

// Execute - Kill all active server jobs
func (j *JobsKillAll) Execute(args []string) (err error) {

	jobs, err := transport.RPC.GetJobs(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.RPCError+"%s\n", err)
		return
	}
	for _, job := range jobs.Active {
		killJob(job.ID)
	}
	return
}

func killJob(jobID uint32) {
	fmt.Printf(util.Info+"Killing job #%d ...\n", jobID)
	jobKill, err := transport.RPC.KillJob(context.Background(), &clientpb.KillJobReq{
		ID: jobID,
	})
	if err != nil {
		fmt.Printf(util.RPCError+"%s\n", err)
	} else {
		fmt.Printf(util.Info+"Successfully killed job #%d\n", jobKill.ID)
	}
}

func printJobs(jobs map[uint32]*clientpb.Job) {

	// Filter jobs based on their types.
	var listeners = map[uint32]*clientpb.Job{} // C2 Handlers
	var servers = map[uint32]*clientpb.Job{}   // gRPC servers
	var others = map[uint32]*clientpb.Job{}    // Others
	var next bool                              // Controls new lines

	// Sort keys
	var keys []int
	for _, job := range jobs {
		keys = append(keys, int(job.ID))
	}
	sort.Ints(keys)

	for _, k := range keys {
		job := jobs[uint32(k)]
		if job.Name == "grpc" {
			servers[job.ID] = job
			continue
		}

		listeners[job.ID] = job

		// if strings.Contains(job.Name, "server") || strings.Contains(job.Name, "via") {
		//         handlers[job.ID] = job
		//         continue
		// }
		//
		// others[job.ID] = job
	}

	// Print handler jobs
	if len(listeners) > 0 {
		// Sort keys
		var keys []int
		for _, job := range listeners {
			keys = append(keys, int(job.ID))
		}
		sort.Ints(keys)

		table := util.NewTable(readline.Bold(readline.Yellow("Listeners")))
		headers := []string{"ID", "Protocol", "Domain(s)", "Port", "Description"}
		headLen := []int{2, 10, 0, 5, 0}
		table.SetColumns(headers, headLen)

		for _, k := range keys {
			job := listeners[uint32(k)]

			// Some host address might be scattered
			// between names and domain values.
			var domains string
			if len(job.Domains) != 0 {
				strings.Join(job.Domains, ",")
			} else {
				domains = job.Name
			}

			// Append elements
			table.AppendRow([]string{
				strconv.Itoa(int(job.ID)),
				job.Protocol,
				domains,
				strconv.Itoa(int(job.Port)),
				job.Description,
			})
		}

		// Print table
		table.Output()
		next = true
	}

	// Print server jobs
	if len(servers) > 0 {
		if next {
			fmt.Println()
		}

		// Sort keys
		var keys []int
		for _, job := range servers {
			keys = append(keys, int(job.ID))
		}
		sort.Ints(keys)

		table := util.NewTable(readline.Bold(readline.Yellow("gRPC servers")))
		headers := []string{"ID", "Domain", "Port"}
		headLen := []int{2, 10, 5}
		table.SetColumns(headers, headLen)

		for _, k := range keys {
			job := servers[uint32(k)]

			// Some host address might be scattered
			// between names and domain values.
			var domains string
			if len(job.Domains) != 0 {
				strings.Join(job.Domains, ",")
			} else {
				domains = job.Name
			}

			// Append elements
			table.AppendRow([]string{
				strconv.Itoa(int(job.ID)),
				domains,
				strconv.Itoa(int(job.Port)),
			})
		}

		// Print table
		table.Output()
		next = true
	}

	// Print other jobs
	if len(others) > 0 {
		if next {
			fmt.Println()
		}

		// Sort keys
		var keys []int
		for _, job := range others {
			keys = append(keys, int(job.ID))
		}
		sort.Ints(keys)

		table := util.NewTable(readline.Bold(readline.Yellow("Others")))
		headers := []string{"ID", "Name", "Description"}
		headLen := []int{2, 0, 0}
		table.SetColumns(headers, headLen)

		for _, k := range keys {
			job := others[uint32(k)]

			// Append elements
			table.AppendRow([]string{
				strconv.Itoa(int(job.ID)),
				job.Name,
				job.Description,
			})
		}

		// Print table
		table.Output()
		next = true
	}
}
