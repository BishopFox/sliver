package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (rpc *Server) Creds(ctx context.Context, req *commonpb.Empty) (*clientpb.Credentials, error) {
	dbCreds := []*models.Credential{}
	err := db.Session().Where(&models.Credential{}).Find(&dbCreds).Error
	if err != nil {
		return nil, err
	}
	credentials := []*clientpb.Credential{}
	for _, dbCred := range dbCreds {
		credentials = append(credentials, dbCred.ToProtobuf())
	}
	return &clientpb.Credentials{Credentials: credentials}, nil
}

func (rpc *Server) CredsAdd(ctx context.Context, req *clientpb.Credentials) (*commonpb.Empty, error) {
	for _, cred := range req.Credentials {
		err := db.Session().Create(&models.Credential{
			Username:  cred.Username,
			Plaintext: cred.Plaintext,
			Hash:      cred.Hash,
			HashType:  int32(cred.HashType),
			IsCracked: (cred.Plaintext != "" && cred.Hash != ""),
		}).Error
		if err != nil {
			return nil, err
		}
	}
	return &commonpb.Empty{}, nil
}

func (rpc *Server) CredsRm(ctx context.Context, req *clientpb.Credentials) (*commonpb.Empty, error) {
	for _, cred := range req.Credentials {
		credID := uuid.FromStringOrNil(cred.ID)
		if credID == uuid.Nil {
			return nil, status.Error(codes.InvalidArgument, "Invalid credential ID")
		}
		err := db.Session().Create(&models.Credential{
			ID: credID,
		}).Error
		if err != nil {
			return nil, err
		}
	}
	return &commonpb.Empty{}, nil
}

func (rpc *Server) CredsUpdate(ctx context.Context, req *clientpb.Credentials) (*commonpb.Empty, error) {
	for _, cred := range req.Credentials {
		credID := uuid.FromStringOrNil(cred.ID)
		if credID == uuid.Nil {
			return nil, status.Error(codes.InvalidArgument, "Invalid credential ID")
		}
		err := db.Session().Where(&models.Credential{ID: credID}).Updates(&models.Credential{
			Username:  cred.Username,
			Plaintext: cred.Plaintext,
			Hash:      cred.Hash,
			HashType:  int32(cred.HashType),
			IsCracked: cred.IsCracked,
		}).Error
		if err != nil {
			return nil, err
		}
	}
	return &commonpb.Empty{}, nil
}

func (rpc *Server) GetCred(ctx context.Context, req *clientpb.Credential) (*clientpb.Credential, error) {
	credID := uuid.FromStringOrNil(req.ID)
	if credID == uuid.Nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid credential ID")
	}
	dbCred := &models.Credential{}
	err := db.Session().Where(&models.Credential{ID: credID}).First(&dbCred).Error
	if err != nil {
		return nil, err
	}
	return dbCred.ToProtobuf(), nil
}
