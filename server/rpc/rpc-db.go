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

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/db"
)

// DbSet Set a Key in a Bucket to a Value
func (rpc *Server) DbSet(ctx context.Context, req *sliverpb.DbSetReq) (*sliverpb.DbSet, error) {
	resp := &sliverpb.DbSet{}
	bucket, err := db.GetBucket(req.Bucket)
	if err != nil {
		return nil, err
	}
	err = bucket.Set(req.Key, req.Value)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// DbGet Get the Value of a Key in a Bucket
func (rpc *Server) DbGet(ctx context.Context, req *sliverpb.DbGetReq) (*sliverpb.DbGet, error) {
	resp := &sliverpb.DbGet{}
	bucket, err := db.GetBucket(req.Bucket)
	if err != nil {
		return nil, err
	}
	value, err := bucket.Get(req.Key)
	if err != nil {
		return nil, err
	}
	resp.Value = value
	return resp, nil
}

// DbDelete Delete a Key and Value in a Bucket
func (rpc *Server) DbDelete(ctx context.Context, req *sliverpb.DbDeleteReq) (*sliverpb.DbDelete, error) {
	resp := &sliverpb.DbDelete{}
	bucket, err := db.GetBucket(req.Bucket)
	if err != nil {
		return nil, err
	}
	err = bucket.Delete(req.Key)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
