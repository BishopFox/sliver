package core

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
	"sync"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/gofrs/uuid"
)

var (
	// ClientID -> *clientpb.Builder
	builders = &sync.Map{}
)

func AddBuilder(builder *clientpb.Builder) string {
	builderID, _ := uuid.NewV4()
	builders.Store(builderID.String(), builder)
	return builderID.String()
}

func GetBuilder(builderID string) *clientpb.Builder {
	builder, _ := builders.Load(builderID)
	return builder.(*clientpb.Builder)
}

func AllBuilders() []*clientpb.Builder {
	externalBuilders := []*clientpb.Builder{}
	builders.Range(func(key, value interface{}) bool {
		externalBuilders = append(externalBuilders, value.(*clientpb.Builder))
		return true
	})
	return externalBuilders
}

func RemoveBuilder(builderID string) {
	builders.Delete(builderID)
}
