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
	"fmt"
	"reflect"

	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
)

// PipelineSource is a factory for creating Pipeline instances.
// It is obtained by calling [Client.Pipeline()].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type PipelineSource struct {
	client *Client
}

// CollectionHints provides hints to the query planner.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type CollectionHints map[string]any

// WithForceIndex specifies an index to force the query to use.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (ch CollectionHints) WithForceIndex(index string) CollectionHints {
	newCH := make(CollectionHints, len(ch)+1)
	for k, v := range ch {
		newCH[k] = v
	}
	newCH["force_index"] = index
	return newCH
}

// WithIgnoreIndexFields specifies fields to ignore when selecting an index.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (ch CollectionHints) WithIgnoreIndexFields(fields ...string) CollectionHints {
	newCH := make(CollectionHints, len(ch)+1)
	for k, v := range ch {
		newCH[k] = v
	}
	newCH["ignore_index_fields"] = fields
	return newCH
}

func (ch CollectionHints) toProto() (map[string]*pb.Value, error) {
	if ch == nil {
		return nil, nil
	}
	optsMap := make(map[string]*pb.Value)
	for key, val := range ch {
		valPb, _, err := toProtoValue(reflect.ValueOf(val))
		if err != nil {
			return nil, fmt.Errorf("firestore: error converting option %q: %w", key, err)
		}
		optsMap[key] = valPb
	}
	return optsMap, nil
}

// collectionStageSettings provides settings for Collection and CollectionGroup pipeline stages.
type collectionStageSettings struct {
	Hints CollectionHints
}

func (cs *collectionStageSettings) toProto() (map[string]*pb.Value, error) {
	if cs == nil {
		return nil, nil
	}
	return cs.Hints.toProto()
}

// CollectionOption is an option for a Collection pipeline stage.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type CollectionOption interface {
	apply(co *collectionStageSettings)
	isCollectionOption()
}

// CollectionGroupOption is an option for a CollectionGroup pipeline stage.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type CollectionGroupOption interface {
	apply(co *collectionStageSettings)
	isCollectionGroupOption()
}

// funcOption wraps a function that modifies collectionStageSettings
// into an implementation of the CollectionOption and CollectionGroupOption interfaces.
type funcOption struct {
	f func(*collectionStageSettings)
}

func (fo *funcOption) apply(cs *collectionStageSettings) {
	fo.f(cs)
}

func (*funcOption) isCollectionOption() {}

func (*funcOption) isCollectionGroupOption() {}

func newFuncOption(f func(*collectionStageSettings)) *funcOption {
	return &funcOption{
		f: f,
	}
}

// WithCollectionHints specifies hints for the query planner.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func WithCollectionHints(hints CollectionHints) CollectionOption {
	return newFuncOption(func(cs *collectionStageSettings) {
		cs.Hints = hints
	})
}

// WithCollectionGroupHints specifies hints for the query planner.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func WithCollectionGroupHints(hints CollectionHints) CollectionGroupOption {
	return newFuncOption(func(cs *collectionStageSettings) {
		cs.Hints = hints
	})
}

// Collection creates a new [Pipeline] that operates on the specified Firestore collection.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (ps *PipelineSource) Collection(path string, opts ...CollectionOption) *Pipeline {
	cs := &collectionStageSettings{}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(cs)
		}
	}
	return newPipeline(ps.client, newInputStageCollection(path, cs))
}

// CollectionGroup creates a new [Pipeline] that operates on all documents in a group
// of collections that include the given ID, regardless of parent document.
//
// For example, consider:
// Countries/France/Cities/Paris = {population: 100}
// Countries/Canada/Cities/Montreal = {population: 90}
//
// CollectionGroup can be used to query across all "Cities" regardless of
// its parent "Countries".
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (ps *PipelineSource) CollectionGroup(collectionID string, opts ...CollectionGroupOption) *Pipeline {
	cgs := &collectionStageSettings{}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(cgs)
		}
	}
	return newPipeline(ps.client, newInputStageCollectionGroup("", collectionID, cgs))
}

// Database creates a new [Pipeline] that operates on all documents in the Firestore database.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (ps *PipelineSource) Database() *Pipeline {
	return newPipeline(ps.client, newInputStageDatabase())
}

// Documents creates a new [Pipeline] that operates on a specific set of Firestore documents.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (ps *PipelineSource) Documents(refs ...*DocumentRef) *Pipeline {
	return newPipeline(ps.client, newInputStageDocuments(refs...))
}

// CreateFromQuery creates a new [Pipeline] from the given [Queryer]. Under the hood, this will
// translate the query semantics (order by document ID, etc.) to an equivalent pipeline.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (ps *PipelineSource) CreateFromQuery(query Queryer) *Pipeline {
	return query.query().Pipeline()
}

// CreateFromAggregationQuery creates a new [Pipeline] from the given [AggregationQuery]. Under the hood, this will
// translate the query semantics (order by document ID, etc.) to an equivalent pipeline.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (ps *PipelineSource) CreateFromAggregationQuery(query *AggregationQuery) *Pipeline {
	return query.Pipeline()
}
