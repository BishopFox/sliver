package jobs

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

func JobKillCmd(jobID uint32, con *console.SliverConsoleClient) {
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

func killAllJobs(con *console.SliverConsoleClient) {
	jobs, err := con.Rpc.GetJobs(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	for _, job := range jobs.Active {
		JobKillCmd(job.ID, con)
	}
}
