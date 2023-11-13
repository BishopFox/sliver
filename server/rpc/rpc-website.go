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
	"mime"
	"path/filepath"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/website"
)

var (
	rpcWebsiteLog = log.NamedLogger("rpc", "website")
)

// Websites - List existing websites
func (rpc *Server) Websites(ctx context.Context, _ *commonpb.Empty) (*clientpb.Websites, error) {
	websiteNames, err := website.Names()
	if err != nil {
		rpcWebsiteLog.Warnf("Failed to find website %s", err)
		return nil, err
	}
	websites := &clientpb.Websites{Websites: []*clientpb.Website{}}
	for _, name := range websiteNames {
		siteContent, err := website.MapContent(name, false)
		if err != nil {
			rpcWebsiteLog.Warnf("Failed to list website content %s", err)
			continue
		}
		websites.Websites = append(websites.Websites, siteContent)
	}
	return websites, nil
}

// WebsiteRemove - Delete an entire website
func (rpc *Server) WebsiteRemove(ctx context.Context, req *clientpb.Website) (*commonpb.Empty, error) {
	web, err := website.MapContent(req.Name, false)
	if err != nil {
		return nil, err
	}
	for path := range web.Contents {
		err := website.RemoveContent(req.Name, path)
		if err != nil {
			rpcWebsiteLog.Errorf("Failed to remove content %s", err)
			return nil, err
		}
	}

	dbWebsite, err := website.WebsiteByName(req.Name)
	if err != nil {
		return nil, err
	}

	err = db.RemoveWebSite(dbWebsite.ID)
	if err != nil {
		return nil, err
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.WebsiteEvent,
		Data:      []byte(req.Name),
	})

	return &commonpb.Empty{}, nil
}

// Website - Get one website
func (rpc *Server) Website(ctx context.Context, req *clientpb.Website) (*clientpb.Website, error) {
	return website.MapContent(req.Name, false)
}

// WebsiteAddContent - Add content to a website, the website is created if `name` does not exist
func (rpc *Server) WebsiteAddContent(ctx context.Context, req *clientpb.WebsiteAddContent) (*clientpb.Website, error) {

	if 0 < len(req.Contents) {
		for _, content := range req.Contents {
			// If no content-type was specified by the client we try to detect the mime based on path ext
			if content.ContentType == "" {
				content.ContentType = mime.TypeByExtension(filepath.Ext(content.Path))
				if content.ContentType == "" {
					content.ContentType = "text/html; charset=utf-8" // Default mime
				}
			}

			content.Size = uint64(len(content.Content))
			rpcLog.Infof("Add website content (%s) %s -> %s", req.Name, content.Path, content.ContentType)
			err := website.AddContent(req.Name, content)
			if err != nil {
				rpcWebsiteLog.Errorf("Failed to add content %s", err)
				return nil, err
			}
		}
	} else {
		_, err := website.AddWebsite(req.Name)
		if err != nil {
			return nil, err
		}
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.WebsiteEvent,
		Data:      []byte(req.Name),
	})

	return website.MapContent(req.Name, true)
}

// WebsiteUpdateContent - Update specific content from a website, currently you can only the update Content-type field
func (rpc *Server) WebsiteUpdateContent(ctx context.Context, req *clientpb.WebsiteAddContent) (*clientpb.Website, error) {
	dbWebsite, err := website.WebsiteByName(req.Name)
	if err != nil {
		return nil, err
	}
	for _, content := range req.Contents {
		website.AddContent(dbWebsite.Name, content)
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.WebsiteEvent,
		Data:      []byte(req.Name),
	})

	return website.MapContent(req.Name, false)
}

// WebsiteRemoveContent - Remove specific content from a website
func (rpc *Server) WebsiteRemoveContent(ctx context.Context, req *clientpb.WebsiteRemoveContent) (*clientpb.Website, error) {
	for _, path := range req.Paths {
		err := website.RemoveContent(req.Name, path)
		if err != nil {
			rpcWebsiteLog.Errorf("Failed to remove content %s", err)
			return nil, err
		}
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.WebsiteEvent,
		Data:      []byte(req.Name),
	})

	return website.MapContent(req.Name, false)
}
