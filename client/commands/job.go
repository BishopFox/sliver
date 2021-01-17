package commands

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
	"os"
	"sort"
	"strings"
	"text/tabwriter"

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
		JobID uint32 `description:"Active job ID" `
	} `positional-args:"yes" required:"true"`
}

// Execute - Kill a job given an ID
func (j *JobsKill) Execute(args []string) (err error) {
	jobID := j.Positional.JobID
	fmt.Printf(util.Info+"Killing job #%d ...", jobID)
	jobKill, err := transport.RPC.KillJob(context.Background(), &clientpb.KillJobReq{
		ID: jobID,
	})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
	} else {
		fmt.Printf(util.Info+"Successfully killed job #%d\n", jobKill.ID)
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
	table := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(table, "ID\tName\tProtocol\tPort\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("ID")),
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("Protocol")),
		strings.Repeat("=", len("Port")))

	var keys []int
	for _, job := range jobs {
		keys = append(keys, int(job.ID))
	}
	sort.Ints(keys) // Fucking Go can't sort int32's, so we convert to/from int's

	for _, k := range keys {
		job := jobs[uint32(k)]
		fmt.Fprintf(table, "%d\t%s\t%s\t%d\t\n", job.ID, job.Name, job.Protocol, job.Port)
	}
	table.Flush()
}
