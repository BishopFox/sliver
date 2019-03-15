package rpc

import (
	"bytes"
	"crypto/x509"
	clientpb "sliver/protobuf/client"
	"sliver/server/assets"
	"sliver/server/certs"
	"sliver/server/core"

	"github.com/golang/protobuf/proto"
)

func rpcPlayers(_ []byte, resp RPCResponse) {

	clientCerts := certs.GetClientCertificates(assets.GetRootAppDir())

	players := &clientpb.Players{Players: []*clientpb.Player{}}
	for _, cert := range clientCerts {
		players.Players = append(players.Players, &clientpb.Player{
			Client: &clientpb.Client{
				Operator: cert.Subject.CommonName,
			},
			Online: isPlayerOnline(cert),
		})
	}

	data, err := proto.Marshal(players)
	if err != nil {
		rpcLog.Errorf("Error encoding rpc response %v", err)
	}
	resp(data, err)
}

// isPlayerOnline - Is a player connected using a given certificate
func isPlayerOnline(cert *x509.Certificate) bool {
	for _, client := range *core.Clients.Connections {
		if client.Certificate == nil {
			continue // Server certificate is nil
		}
		if bytes.Equal(cert.Raw, client.Certificate.Raw) {
			return true
		}
	}
	return false
}
