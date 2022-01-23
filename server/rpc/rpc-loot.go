package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/loot"
)

// var (
// 	lootRPCLog = log.NamedLogger("rpc", "loot")
// )

// LootAdd - Add loot
func (rpc *Server) LootAdd(ctx context.Context, lootReq *clientpb.Loot) (*clientpb.Loot, error) {
	loot, err := loot.GetLootStore().Add(lootReq)
	if err != nil {
		return nil, err
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.LootAddedEvent,
		Data:      []byte(loot.LootID),
	})
	return loot, nil
}

// LootRm - Remove loot
func (rpc *Server) LootRm(ctx context.Context, lootReq *clientpb.Loot) (*commonpb.Empty, error) {
	err := loot.GetLootStore().Rm(lootReq.LootID)
	if err != nil {
		return nil, err
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.LootRemovedEvent,
	})
	return &commonpb.Empty{}, err
}

// LootUpdate - Update loot metadata
func (rpc *Server) LootUpdate(ctx context.Context, lootReq *clientpb.Loot) (*clientpb.Loot, error) {
	loot, err := loot.GetLootStore().Update(lootReq)
	if err != nil {
		return nil, err
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.LootAddedEvent,
	})
	return loot, err
}

// LootContent - Get a list of all loot of a specific type
func (rpc *Server) LootContent(ctx context.Context, lootReq *clientpb.Loot) (*clientpb.Loot, error) {
	return loot.GetLootStore().GetContent(lootReq.LootID, true)
}

// LootAll - Get a list of all loot
func (rpc *Server) LootAll(ctx context.Context, _ *commonpb.Empty) (*clientpb.AllLoot, error) {
	return loot.GetLootStore().All(), nil
}

// LootAllOf - Get a list of all loot of a specific type
func (rpc *Server) LootAllOf(ctx context.Context, lootReq *clientpb.Loot) (*clientpb.AllLoot, error) {
	allLoot := loot.GetLootStore().AllOf(lootReq.Type)
	return allLoot, nil
}
