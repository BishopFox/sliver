package rpc

import (
	"log"
	pb "sliver/protobuf/client"
	"sliver/server/c2"
	"sliver/server/core"

	"github.com/golang/protobuf/proto"
)

func rpcJobs(_ []byte, resp RPCResponse) {
	jobs := &pb.Jobs{
		Active: []*pb.Job{},
	}
	for _, job := range *core.Jobs.Active {
		jobs.Active = append(jobs.Active, &pb.Job{
			ID:          int32(job.ID),
			Name:        job.Name,
			Description: job.Description,
			Protocol:    job.Protocol,
			Port:        int32(job.Port),
		})
	}
	data, err := proto.Marshal(jobs)
	if err != nil {
		log.Printf("Error encoding rpc response %v", err)
		resp([]byte{}, err)
		return
	}
	resp(data, err)
}

func rpcStartMTLSListener(data []byte, resp RPCResponse) {
	mtlsReq := &pb.MTLSReq{}
	err := proto.Unmarshal(data, mtlsReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	jobID, err := jobStartMTLSListener(mtlsReq.Server, uint16(mtlsReq.LPort))
	if err != nil {
		resp([]byte{}, err)
		return
	}
	data, err = proto.Marshal(&pb.MTLS{JobID: int32(jobID)})
	resp(data, err)
}

func jobStartMTLSListener(bindIface string, port uint16) (int, error) {

	ln, err := c2.StartMutualTLSListener(bindIface, port)
	if err != nil {
		return -1, err // If we fail to bind don't setup the Job
	}

	job := &core.Job{
		ID:          core.GetJobID(),
		Name:        "mTLS",
		Description: "mutual tls",
		Protocol:    "tcp",
		Port:        port,
		JobCtrl:     make(chan bool),
	}

	go func() {
		<-job.JobCtrl
		log.Printf("Stopping mTLS listener (%d) ...", job.ID)
		ln.Close() // Kills listener GoRoutines in startMutualTLSListener() but NOT connections

		core.Jobs.RemoveJob(job)

		core.Events <- core.Event{EventType: "stopped", Job: job}
	}()

	core.Jobs.AddJob(job)

	return job.ID, nil
}

func rpcStartDNSListener(data []byte, resp RPCResponse) {

}
