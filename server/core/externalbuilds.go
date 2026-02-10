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

import "sync"

var (
	// implantBuildID -> *ExternalBuildAssignment
	externalBuildAssignments = &sync.Map{}
)

// ExternalBuildAssignment tracks which remote builder is allowed to access a build.
type ExternalBuildAssignment struct {
	BuildID      string
	BuilderName  string
	OperatorName string
}

// TrackExternalBuildAssignment stores the builder/operator assignment for a build.
func TrackExternalBuildAssignment(buildID string, builderName string, operatorName string) {
	externalBuildAssignments.Store(buildID, &ExternalBuildAssignment{
		BuildID:      buildID,
		BuilderName:  builderName,
		OperatorName: operatorName,
	})
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
	externalBuildAssignments.Delete(buildID)
}
