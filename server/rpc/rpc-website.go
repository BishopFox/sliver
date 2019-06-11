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
	"time"

	clientpb "github.com/bishopfox/sliver/protobuf/client"

	"github.com/bishopfox/sliver/server/website"
	"github.com/golang/protobuf/proto"
)

func rpcWebsiteList(_ []byte, _ time.Duration, resp RPCResponse) {
	websiteNames, err := website.ListWebsites()
	if err != nil {
		return
	}
	websites := &clientpb.Websites{Sites: []*clientpb.Website{}}
	for _, name := range websiteNames {
		site, err := website.ListContent(name)
		if err != nil {
			rpcLog.Errorf("Failed to list website content %s", err)
			continue
		}
		websites.Sites = append(websites.Sites, site)
	}
	data, err := proto.Marshal(websites)
	resp(data, err)
}

func rpcWebsiteAddContent(req []byte, _ time.Duration, resp RPCResponse) {
	addWebsite := &clientpb.Website{}
	err := proto.Unmarshal(req, addWebsite)
	if err != nil {
		resp([]byte{}, err)
	}
	for path, content := range addWebsite.Content {
		rpcLog.Infof("Add website content (%s) %s -> %s", addWebsite.Name, path, content.ContentType)
		err := website.AddContent(addWebsite.Name, path, content.ContentType, content.Content)
		if err != nil {
			rpcLog.Errorf("Failed to add website content %s", err)
		}
	}
	resp([]byte{}, nil)
}

func rpcWebsiteRemoveContent(req []byte, _ time.Duration, resp RPCResponse) {
	rmWebsite := &clientpb.Website{}
	err := proto.Unmarshal(req, rmWebsite)
	if err != nil {
		resp([]byte{}, err)
	}
	for webpath := range rmWebsite.Content {
		website.RemoveContent(rmWebsite.Name, webpath)
	}
	resp([]byte{}, nil)
}
