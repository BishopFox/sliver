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

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
)

// GetOperators - Get a list of operators
func (s *Server) GetOperators(ctx context.Context, _ *commonpb.Empty) (*clientpb.Operators, error) {
	operators := &clientpb.Operators{Operators: []*clientpb.Operator{}}
	dbOperators, err := db.OperatorAll()
	if err != nil {
		return nil, ErrDatabaseFailure
	}
	for _, dbOperator := range dbOperators {
		operators.Operators = append(operators.Operators, &clientpb.Operator{
			Name:   dbOperator.Name,
			Online: isOperatorOnline(dbOperator.Name),
		})
	}
	return operators, nil
}

func isOperatorOnline(commonName string) bool {
	for _, operator := range core.Clients.ActiveOperators() {
		if commonName == operator {
			return true
		}
	}
	return false
}
