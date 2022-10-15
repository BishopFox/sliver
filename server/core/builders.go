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
	"errors"
	"sync"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

var (
	// ClientID -> *clientpb.Builder
	builders = &sync.Map{}

	ErrDuplicateExternalBuilderName = errors.New("builder name must be unique, this name is already in use")
)

func AddBuilder(builder *clientpb.Builder) error {
	_, loaded := builders.LoadOrStore(builder.Name, builder)
	if loaded {
		return ErrDuplicateExternalBuilderName
	}
	return nil
}

func GetBuilder(builderName string) *clientpb.Builder {
	builder, _ := builders.Load(builderName)
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

func RemoveBuilder(builderName string) {
	builders.Delete(builderName)
}
