// Copyright 2017 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package firestore

import (
	"errors"
	"fmt"
	"time"

	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// A Precondition modifies a Firestore update or delete operation.
type Precondition interface {
	// Returns the corresponding Precondition proto.
	preconditionProto() (*pb.Precondition, error)
}

// Exists is a Precondition that checks for the existence of a resource before
// writing to it. If the check fails, the write does not occur.
var Exists Precondition

func init() {
	// Initialize here so godoc doesn't show the internal value.
	Exists = exists(true)
}

type exists bool

func (e exists) preconditionProto() (*pb.Precondition, error) {
	return &pb.Precondition{
		ConditionType: &pb.Precondition_Exists{Exists: bool(e)},
	}, nil
}

func (e exists) String() string {
	if e {
		return "Exists"
	}
	return "DoesNotExist"
}

// LastUpdateTime returns a Precondition that checks that a resource must exist and
// must have last been updated at the given time. If the check fails, the write
// does not occur.
func LastUpdateTime(t time.Time) Precondition { return lastUpdateTime(t) }

type lastUpdateTime time.Time

func (u lastUpdateTime) preconditionProto() (*pb.Precondition, error) {
	ts := timestamppb.New(time.Time(u))
	return &pb.Precondition{
		ConditionType: &pb.Precondition_UpdateTime{UpdateTime: ts},
	}, ts.CheckValid()
}

func (u lastUpdateTime) String() string { return fmt.Sprintf("LastUpdateTime(%s)", time.Time(u)) }

func processPreconditionsForDelete(preconds []Precondition) (*pb.Precondition, error) {
	// At most one option permitted.
	switch len(preconds) {
	case 0:
		return nil, nil
	case 1:
		return preconds[0].preconditionProto()
	default:
		return nil, fmt.Errorf("firestore: conflicting preconditions: %+v", preconds)
	}
}

func processPreconditionsForUpdate(preconds []Precondition) (*pb.Precondition, error) {
	// At most one option permitted, and it cannot be Exists.
	switch len(preconds) {
	case 0:
		// If the user doesn't provide any options, default to Exists(true).
		return exists(true).preconditionProto()
	case 1:
		if _, ok := preconds[0].(exists); ok {
			return nil, errors.New("cannot use Exists with Update")
		}
		return preconds[0].preconditionProto()
	default:
		return nil, fmt.Errorf("firestore: conflicting preconditions: %+v", preconds)
	}
}

func processPreconditionsForVerify(preconds []Precondition) (*pb.Precondition, error) {
	// At most one option permitted.
	switch len(preconds) {
	case 0:
		return nil, nil
	case 1:
		return preconds[0].preconditionProto()
	default:
		return nil, fmt.Errorf("firestore: conflicting preconditions: %+v", preconds)
	}
}

// A SetOption modifies a Firestore set operation.
type SetOption interface {
	fieldPaths() (fps []FieldPath, all bool, err error)
}

// MergeAll is a SetOption that causes all the field paths given in the data argument
// to Set to be overwritten. It is not supported for struct data.
var MergeAll SetOption = merge{all: true}

// Merge returns a SetOption that causes only the given field paths to be
// overwritten. Other fields on the existing document will be untouched. It is an
// error if a provided field path does not refer to a value in the data passed to
// Set.
func Merge(fps ...FieldPath) SetOption {
	for _, fp := range fps {
		if err := fp.validate(); err != nil {
			return merge{err: err}
		}
	}
	return merge{paths: fps}
}

type merge struct {
	all   bool
	paths []FieldPath
	err   error
}

func (m merge) String() string {
	if m.err != nil {
		return fmt.Sprintf("<Merge error: %v>", m.err)
	}
	if m.all {
		return "MergeAll"
	}
	return fmt.Sprintf("Merge(%+v)", m.paths)
}

func (m merge) fieldPaths() (fps []FieldPath, all bool, err error) {
	if m.err != nil {
		return nil, false, m.err
	}
	if err := checkNoDupOrPrefix(m.paths); err != nil {
		return nil, false, err
	}
	if m.all {
		return nil, true, nil
	}
	return m.paths, false, nil
}

func processSetOptions(opts []SetOption) (fps []FieldPath, all bool, err error) {
	switch len(opts) {
	case 0:
		return nil, false, nil
	case 1:
		return opts[0].fieldPaths()
	default:
		return nil, false, fmt.Errorf("conflicting options: %+v", opts)
	}
}

type runQuerySettings struct {
	// Explain options for the query. If set, additional query
	// statistics will be returned. If not, only query results will be returned.
	explainOptions *pb.ExplainOptions
}

// newRunQuerySettings creates a runQuerySettings with a given RunOption slice.
func newRunQuerySettings(opts []RunOption) (*runQuerySettings, error) {
	s := &runQuerySettings{}
	for _, o := range opts {
		if o == nil {
			return nil, errors.New("firestore: RunOption cannot be nil")
		}
		err := o.apply(s)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

// RunOption are options used while running a query
type RunOption interface {
	apply(*runQuerySettings) error
}

// ExplainOptions are Query Explain options.
// Query Explain allows you to submit Cloud Firestore queries to the backend and
// receive detailed performance statistics on backend query execution in return.
type ExplainOptions struct {
	// When false (the default), Query Explain plans the query, but skips over the
	// execution stage. This will return planner stage information.
	//
	// When true, Query Explain both plans and executes the query. This returns all
	// the planner information along with statistics from the query execution runtime.
	// This will include the billing information of the query along with system-level
	// insights into the query execution.
	Analyze bool
}

func (e ExplainOptions) apply(s *runQuerySettings) error {
	if s.explainOptions != nil {
		return errors.New("firestore: ExplainOptions can be specified only once")
	}
	pbExplainOptions := pb.ExplainOptions{Analyze: e.Analyze}
	s.explainOptions = &pbExplainOptions
	return nil
}
