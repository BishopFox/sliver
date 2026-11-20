package core

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
)

var (
	// implantBuildID -> *ExternalBuildAssignment
	externalBuildAssignments = &sync.Map{}
	// Orders builder registration changes with assignment creation and teardown.
	externalBuildMu sync.Mutex
)

// ExternalBuildAssignment tracks which remote builder is allowed to access a build.
type ExternalBuildAssignment struct {
	BuildID      string
	BuilderName  string
	OperatorName string
}

// TrackExternalBuildAssignment stores the builder/operator assignment for a build.
func TrackExternalBuildAssignment(buildID string, builderName string, operatorName string) {
	externalBuildMu.Lock()
	defer externalBuildMu.Unlock()
	externalBuildAssignments.Store(buildID, &ExternalBuildAssignment{
		BuildID:      buildID,
		BuilderName:  builderName,
		OperatorName: operatorName,
	})
}

// TrackExternalBuildAssignmentForBuilder stores an assignment only while the
// same builder registration that requested it is still connected. The pointer
// check prevents an old build from being assigned to a new connection that
// reused the builder name during preparation.
func TrackExternalBuildAssignmentForBuilder(buildID string, expected *clientpb.Builder) bool {
	if expected == nil || buildID == "" {
		return false
	}
	externalBuildMu.Lock()
	defer externalBuildMu.Unlock()
	current, ok := builders.Load(expected.Name)
	if !ok || current != expected {
		return false
	}
	externalBuildAssignments.Store(buildID, &ExternalBuildAssignment{
		BuildID: buildID, BuilderName: expected.Name, OperatorName: expected.OperatorName,
	})
	return true
}

// RemoveBuilderAndExternalBuildAssignments disconnects exactly one
// registration and returns its outstanding legacy build IDs for DB cleanup.
// It cannot remove a newer registration that reused the name.
func RemoveBuilderAndExternalBuildAssignments(expected *clientpb.Builder) []string {
	if expected == nil {
		return nil
	}
	externalBuildMu.Lock()
	defer externalBuildMu.Unlock()
	current, ok := builders.Load(expected.Name)
	if !ok || current != expected {
		return nil
	}
	builders.Delete(expected.Name)
	var ids []string
	externalBuildAssignments.Range(func(key, value interface{}) bool {
		assignment := value.(*ExternalBuildAssignment)
		if assignment.BuilderName == expected.Name && assignment.OperatorName == expected.OperatorName {
			externalBuildAssignments.Delete(key)
			ids = append(ids, assignment.BuildID)
		}
		return true
	})
	return ids
}

// GetExternalBuildAssignment returns the assignment for a build, if present.
func GetExternalBuildAssignment(buildID string) *ExternalBuildAssignment {
	assignment, ok := externalBuildAssignments.Load(buildID)
	if !ok {
		return nil
	}
	return assignment.(*ExternalBuildAssignment)
}

// RemoveExternalBuildAssignment clears any assignment for a build.
func RemoveExternalBuildAssignment(buildID string) {
	externalBuildMu.Lock()
	defer externalBuildMu.Unlock()
	externalBuildAssignments.Delete(buildID)
}
