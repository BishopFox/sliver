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

	"github.com/bishopfox/sliver/client/credentials"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	credsRpcLog = log.NamedLogger("rpc", "creds")

	ErrInvalidCredID       = status.Errorf(codes.InvalidArgument, "Invalid credential ID")
	ErrCredNotFound        = status.Error(codes.NotFound, "Credential not found")
	ErrCredOperationFailed = status.Error(codes.Internal, "Credential operation failed")
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
			Collection: cred.Collection,
			Username:   cred.Username,
			Plaintext:  cred.Plaintext,
			Hash:       cred.Hash,
			HashType:   int32(cred.HashType),
			IsCracked:  (cred.Plaintext != "" && cred.Hash != ""),
		}).Error
		if err != nil {
			credsRpcLog.Errorf("Failed to add credential: %s", err)
			return nil, ErrCredOperationFailed
		}
	}
	return &commonpb.Empty{}, nil
}

func (rpc *Server) CredsRm(ctx context.Context, req *clientpb.Credentials) (*commonpb.Empty, error) {
	for _, cred := range req.Credentials {
		dbCred, err := db.CredentialByID(cred.ID)
		if err != nil {
			credsRpcLog.Errorf("Failed to get credential: %s", err)
			return nil, ErrCredNotFound
		}
		credsRpcLog.Infof("got cred: %#v", dbCred)
		err = db.Session().Delete(dbCred).Error
		if err != nil {
			credsRpcLog.Errorf("Failed to remove credential: %s", err)
			return nil, ErrCredOperationFailed
		}
	}
	return &commonpb.Empty{}, nil
}

func (rpc *Server) CredsUpdate(ctx context.Context, req *clientpb.Credentials) (*commonpb.Empty, error) {
	for _, cred := range req.Credentials {
		credID := uuid.FromStringOrNil(cred.ID)
		if credID == uuid.Nil {
			return nil, ErrInvalidCredID
		}
		err := db.Session().Where(&models.Credential{ID: credID}).Updates(&models.Credential{
			Collection: cred.Collection,
			Username:   cred.Username,
			Plaintext:  cred.Plaintext,
			Hash:       cred.Hash,
			HashType:   int32(cred.HashType),
			IsCracked:  cred.IsCracked,
		}).Error
		if err != nil {
			credsRpcLog.Errorf("Failed to update credential: %s", err)
			return nil, ErrCredOperationFailed
		}
	}
	return &commonpb.Empty{}, nil
}

func (rpc *Server) GetCredByID(ctx context.Context, req *clientpb.Credential) (*clientpb.Credential, error) {
	dbCred, err := db.CredentialByID(req.ID)
	if err != nil {
		credsRpcLog.Errorf("Failed to get credential: %s", err)
		return nil, ErrCredNotFound
	}
	return dbCred, nil
}

func (rpc *Server) GetCredsByHashType(ctx context.Context, req *clientpb.Credential) (*clientpb.Credentials, error) {
	dbCreds, err := db.CredentialsByHashType(req.HashType)
	if err != nil {
		credsRpcLog.Errorf("Failed to get credential: %s", err)
		return nil, ErrCredOperationFailed
	}
	credentials := []*clientpb.Credential{}
	for _, dbCred := range dbCreds {
		credentials = append(credentials, dbCred)
	}
	return &clientpb.Credentials{Credentials: credentials}, nil
}

func (rpc *Server) CredsSniffHashType(ctx context.Context, req *clientpb.Credential) (*clientpb.Credential, error) {
	return &clientpb.Credential{HashType: credentials.SniffHashType(req.Hash)}, nil
}

func (rpc *Server) GetPlaintextCredsByHashType(ctx context.Context, req *clientpb.Credential) (*clientpb.Credentials, error) {
	dbCreds, err := db.PlaintextCredentialsByHashType(req.HashType)
	if err != nil {
		credsRpcLog.Errorf("Failed to get credential: %s", err)
		return nil, ErrCredOperationFailed
	}
	credentials := []*clientpb.Credential{}
	for _, dbCred := range dbCreds {
		credentials = append(credentials, dbCred)
	}
	return &clientpb.Credentials{Credentials: credentials}, nil
}
