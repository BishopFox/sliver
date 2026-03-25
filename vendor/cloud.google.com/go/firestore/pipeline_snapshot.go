// Copyright 2026 Google LLC
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
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// PipelineSnapshot contains zero or more [PipelineResult] objects
// representing the documents returned by a pipeline query. It provides methods
// to iterate over the documents and access metadata about the query results.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type PipelineSnapshot struct {
	iter *PipelineResultIterator
}

var errExecutionTimeBeforeEnd = errors.New("firestore: ExecutionTime is available only after the iterator reaches the end")

// Results returns an iterator over the query results.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (ps *PipelineSnapshot) Results() *PipelineResultIterator {
	return ps.iter
}

// ExecutionTime returns the time at which the pipeline was executed.
// It is available only after the iterator reaches the end.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (ps *PipelineSnapshot) ExecutionTime() (*time.Time, error) {
	if ps == nil {
		return nil, errors.New("firestore: PipelineSnapshot is nil")
	}
	if ps.iter == nil {
		return nil, errors.New("firestore: PipelineResultIterator is nil")
	}
	if ps.iter.err != iterator.Done {
		return nil, errExecutionTimeBeforeEnd
	}
	return ps.iter.iter.getExecutionTime()
}

// ExplainStats returns stats from query explain.
// If [WithExplainMode] was set to [ExplainModeExplain] or left unset, then no stats will be available.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (ps *PipelineSnapshot) ExplainStats() *ExplainStats {
	if ps == nil {
		return &ExplainStats{err: errors.New("firestore: PipelineSnapshot is nil")}
	}
	if ps.iter == nil {
		return &ExplainStats{err: errors.New("firestore: PipelineResultIterator is nil")}
	}
	if ps.iter.err == nil || ps.iter.err != iterator.Done {
		return &ExplainStats{err: errStatsBeforeEnd}
	}
	statsPb, statsErr := ps.iter.iter.getExplainStats()
	return &ExplainStats{statsPb: statsPb, err: statsErr}
}

// ExplainStats is query explain stats.
//
// Contains all metadata related to pipeline planning and execution, specific
// contents depend on the supplied pipeline options.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type ExplainStats struct {
	statsPb *pb.ExplainStats
	err     error
}

// RawData returns the explain stats in an encoded proto format, as returned from the Firestore backend.
// The caller is responsible for unpacking this proto message.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (es *ExplainStats) RawData() (*anypb.Any, error) {
	if es.err != nil {
		return nil, es.err
	}
	if es.statsPb == nil {
		return nil, nil
	}

	return es.statsPb.GetData(), nil
}

// Text returns the explain stats as a string from the Firestore backend.
// If explain stats were requested with `outputFormat = 'text'`, the string is
// returned verbatim. If `outputFormat = 'json'`, this returns the explain stats
// as stringified JSON.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (es *ExplainStats) Text() (string, error) {
	if es.err != nil {
		return "", es.err
	}
	if es.statsPb == nil || es.statsPb.GetData() == nil {
		return "", nil
	}

	var data wrapperspb.StringValue
	if err := es.statsPb.GetData().UnmarshalTo(&data); err != nil {
		return "", fmt.Errorf("firestore: failed to unmarshal Any to wrapperspb.StringValue: %w", err)
	}

	return data.GetValue(), nil
}
