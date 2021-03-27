package rpc

import (
	"github.com/golang/protobuf/proto"
	grpcConn "github.com/mitchellh/go-grpc-net-conn"

	"github.com/bishopfox/sliver/protobuf/commpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/comm"
)

// Channel - The client requested to setup its Comm system and wire it with the server.
func (s *Server) InitComm(stream rpcpb.SliverRPC_InitCommServer) error {

	// Get an ID/operator name for this client, so that the Comms system knows
	// where to route back connections that are meant for this client proxy/portfwd utilities.
	commonName := s.getClientCommonName(stream.Context())

	// We need to create a callback so the conn knows how to decode/encode
	// arbitrary byte slices for our proto type.
	fieldFunc := func(msg proto.Message) *[]byte {
		return &msg.(*commpb.Bytes).Data
	}

	// Wrap our conn around the response.
	conn := &grpcConn.Conn{
		Stream:   stream,
		Request:  &commpb.Bytes{},
		Response: &commpb.Bytes{},
		Encode:   grpcConn.SimpleEncoder(fieldFunc),
		Decode:   grpcConn.SimpleDecoder(fieldFunc),
	}

	// The connection is a valid net.Conn upon which we can setup SSH.
	// We pass the commonName for SSH public key fingerprinting.
	commClient, err := comm.InitClient(conn, commonName)
	if err != nil {
		return err
	}

	// Serve the Comm client (blocking)
	commClient.ServeClient()

	// We get here because the client disconnected:
	// We shutdown this Comm (closes as gracefully as possible)
	err = commClient.ShutdownClient()
	if err != nil {
		rpcLog.Errorf("Error shuting down Comm for client (%s): %v", commonName, err)
	}
	rpcLog.Infof("Comm client has shutdown gracefully (%s)", commonName)

	return nil
}
