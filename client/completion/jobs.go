package completion

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
	"fmt"
	"strconv"

	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// JobIDs - Completes job IDs along with a description.
func JobIDs() (comps []*readline.CompletionGroup) {
	comp := &readline.CompletionGroup{
		Name:         "jobs",
		Descriptions: map[string]string{},
		DisplayType:  readline.TabDisplayList,
	}

	jobs, err := transport.RPC.GetJobs(context.Background(), &commonpb.Empty{})
	if err != nil {
		return
	}
	for _, job := range jobs.Active {
		jobID := strconv.Itoa(int(job.ID))
		comp.Suggestions = append(comp.Suggestions, jobID)
		comp.Descriptions[jobID] = readline.DIM + job.Name + fmt.Sprintf(" (%s)", job.Description) + readline.RESET
	}

	return []*readline.CompletionGroup{comp}
}
