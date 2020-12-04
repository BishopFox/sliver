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
	"encoding/json"
	"errors"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
)

func userSave(user models.User) error {
	err := db.Session().Save(&user).Error
	if err != nil {
		return err
	}

	return nil
}

func findUser(uuid string, uid string) (*models.User, error) {
	user := models.User{}
	err := db.Session().Where(&models.User{UUID: uuid, UID: uid}).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, err
}

func (rpc *Server) UserAttributeSet(ctx context.Context, req *clientpb.UserAttributeSetReq) (*clientpb.UserAttributeSet, error) {
	resp := &clientpb.UserAttributeSet{}
	ua, err := findUser(req.UUID, req.UID)
	if err != nil {
		attributes := make(map[string]string)
		attributes[req.Attribute] = req.Value
		marshal, err := json.Marshal(attributes)
		if err != nil {
			return nil, err
		}
		userSave(models.User{
			UUID:       req.UUID,
			UID:        req.UID,
			Attributes: string(marshal),
		})
		ua, err = findUser(req.UUID, req.UID)
		if err != nil {
			return nil, err
		}
	}
	var attributes map[string]string
	err = json.Unmarshal([]byte(ua.Attributes), &attributes)
	if err != nil {
		return nil, err
	}
	attributes[req.Attribute] = req.Value
	marshal, err := json.Marshal(attributes)
	if err != nil {
		return nil, err
	}
	ua.Attributes = string(marshal)
	err = db.Session().Save(ua).Error
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (rpc *Server) UserAttributeGet(ctx context.Context, req *clientpb.UserAttributeGetReq) (*clientpb.UserAttributeGet, error) {
	resp := &clientpb.UserAttributeGet{}
	ua, err := findUser(req.UUID, req.UID)
	if err != nil {
		return nil, err
	}
	var attributes map[string]string
	err = json.Unmarshal([]byte(ua.Attributes), &attributes)
	if err != nil {
		return nil, err
	}
	value, ok := attributes[req.Attribute]
	if !ok {
		return nil, errors.New("Attribute Not Found")
	}
	resp.Value = value
	return resp, nil
}

func (rpc *Server) UserAttributeDelete(ctx context.Context, req *clientpb.UserAttributeDeleteReq) (*clientpb.UserAttributeDelete, error) {
	resp := &clientpb.UserAttributeDelete{}
	ua, err := findUser(req.UUID, req.UID)
	if err != nil {
		return nil, err
	}
	var attributes map[string]string
	err = json.Unmarshal([]byte(ua.Attributes), &attributes)
	if err != nil {
		return nil, err
	}
	delete(attributes, req.Attribute)
	marshal, err := json.Marshal(attributes)
	if err != nil {
		return nil, err
	}
	ua.Attributes = string(marshal)
	err = db.Session().Save(ua).Error
	if err != nil {
		return nil, err
	}
	return resp, nil
}
