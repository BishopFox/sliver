package rpc

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
	"crypto/x509"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/certs"
)

// GetOperators - Get a list of operators
func (s *Server) GetOperators(ctx context.Context, _ *commonpb.Empty) (*clientpb.Operators, error) {
	operatorCerts := certs.OperatorClientListCertificates()
	operators := &clientpb.Operators{
		Operators: []*clientpb.Operator{},
	}
	for _, cert := range operatorCerts {
		operators.Operators = append(operators.Operators, &clientpb.Operator{
			Name:   cert.Subject.CommonName,
			Online: isOperatorOnline(cert),
		})
	}
	return operators, nil
}

// isOperatorOnline - Is a player connected using a given certificate
func isOperatorOnline(cert *x509.Certificate) bool {
	// for _, client := range *core.Clients.Connections {
	// 	if client.Certificate == nil {
	// 		continue // Server certificate is nil
	// 	}
	// 	if bytes.Equal(cert.Raw, client.Certificate.Raw) {
	// 		return true
	// 	}
	// }
	return false
}
