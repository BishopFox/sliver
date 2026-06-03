package handlers

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
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/0x90pkt/trigger/pkg/intents"

	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
)

var stopJobLog = log.NamedLogger("c2", "trigger-stop-job")

// jobsLookup is a package-level indirection so tests can supply a
// fake job list. Returns the slice of active jobs.
var jobsLookup = func() []*core.Job { return core.Jobs.All() }

// StopJob is an intents.Handler bound to a specific job name at
// construction. On trigger fire, it locates the first active job
// matching the name and sends a non-blocking shutdown signal on its
// JobCtrl channel.
//
// Name match is exact (case-sensitive). If multiple jobs share the
// name, the first match wins. Operators who need finer control should
// bind multiple StopJob handlers with distinctive task labels, or
// use stopByID instead (not yet implemented).
type StopJob struct {
	intent  string
	jobName string
}

// NewStopJob constructs a StopJob handler.
func NewStopJob(intent, jobName string) (*StopJob, error) {
	if strings.TrimSpace(intent) == "" {
		return nil, errors.New("stop-job: task name must be set")
	}
	if strings.TrimSpace(jobName) == "" {
		return nil, errors.New("stop-job: job name must be set")
	}
	return &StopJob{intent: intent, jobName: jobName}, nil
}

// Name implements intents.Handler.
func (h *StopJob) Name() string { return h.intent }

// Execute implements intents.Handler. Locates the target job, sends a
// non-blocking JobCtrl signal. Returns an error if no matching job is
// active OR if the job's JobCtrl is busy (already shutting down) —
// the non-blocking send prevents the worker goroutine from hanging
// in the case where another goroutine is concurrently draining
// JobCtrl.
func (h *StopJob) Execute(_ context.Context, evt intents.Event) error {
	var target *core.Job
	for _, j := range jobsLookup() {
		if j.Name == h.jobName {
			target = j
			break
		}
	}
	if target == nil {
		return fmt.Errorf("stop-job %s: no active job named %q", h.intent, h.jobName)
	}

	select {
	case target.JobCtrl <- true:
		stopJobLog.Infof("stop-job fired: intent=%s job=%s (id=%d) triggered_by=%s source_ip=%s",
			h.intent, h.jobName, target.ID, evt.ClientID, evt.SourceIP)
		return nil
	default:
		return fmt.Errorf("stop-job %s: job %q (id=%d) JobCtrl is busy (concurrent stop in progress?)",
			h.intent, h.jobName, target.ID)
	}
}
