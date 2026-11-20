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
	builders   = &sync.Map{}
	buildersMu sync.Mutex

	// ErrDuplicateBuilderName indicates that a builder name is already registered.
	ErrDuplicateBuilderName = errors.New("builder name must be unique, this name is already in use")
)

func AddBuilder(builder *clientpb.Builder) error {
	buildersMu.Lock()
	defer buildersMu.Unlock()
	_, loaded := builders.LoadOrStore(builder.Name, builder)
	if loaded {
		return ErrDuplicateBuilderName
	}
	return nil
}

func GetBuilder(builderName string) *clientpb.Builder {
	builder, ok := builders.Load(builderName)
	if !ok {
		return nil
	}
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
	buildersMu.Lock()
	defer buildersMu.Unlock()
	builders.Delete(builderName)
}

// RemoveBuilderIfCurrent tears down only the registration represented by
// expected. A stale disconnect cannot remove a replacement with the same name.
func RemoveBuilderIfCurrent(expected *clientpb.Builder) bool {
	if expected == nil {
		return false
	}
	buildersMu.Lock()
	defer buildersMu.Unlock()
	current, ok := builders.Load(expected.Name)
	if !ok || current != expected {
		return false
	}
	builders.Delete(expected.Name)
	return true
}
