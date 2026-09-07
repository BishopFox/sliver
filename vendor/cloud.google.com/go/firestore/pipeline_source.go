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

import "fmt"

// PipelineSource is a factory for creating Pipeline instances.
// It is obtained by calling [Client.Pipeline].
type PipelineSource struct {
	client *Client
}

// WithForceIndex specifies an index to force the query to use.
func WithForceIndex(index string) CollectionSourceOption {
	return newFuncOption(func(options map[string]any) {
		options["force_index"] = index
	})
}

// WithIgnoreIndexFields specifies fields to ignore when selecting an index.
func WithIgnoreIndexFields(fields ...string) CollectionSourceOption {
	return newFuncOption(func(options map[string]any) {
		options["ignore_index_fields"] = fields
	})
}

// CollectionOption is an option for a Collection pipeline stage.
type CollectionOption interface {
	StageOption
	isCollectionOption()
}

// CollectionGroupOption is an option for a CollectionGroup pipeline stage.
type CollectionGroupOption interface {
	StageOption
	isCollectionGroupOption()
}

// CollectionSourceOption is an option that can be applied to both Collection and CollectionGroup pipeline stages.
type CollectionSourceOption interface {
	CollectionOption
	CollectionGroupOption
}

// funcOption wraps a function that modifies an options map
// into an implementation of the CollectionOption and CollectionGroupOption interfaces.
type funcOption struct {
	f func(map[string]any)
}

func (fo *funcOption) applyStage(options map[string]any) {
	fo.f(options)
}

func (*funcOption) isCollectionOption()      {}
func (*funcOption) isCollectionGroupOption() {}

func newFuncOption(f func(map[string]any)) *funcOption {
	return &funcOption{
		f: f,
	}
}

// Collection creates a new [Pipeline] that operates on the specified Firestore collection.
func (ps *PipelineSource) Collection(path string, opts ...CollectionOption) *Pipeline {
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	return newPipeline(ps.client, newInputStageCollection(path, options))
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
func (ps *PipelineSource) CollectionGroup(collectionID string, opts ...CollectionGroupOption) *Pipeline {
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	return newPipeline(ps.client, newInputStageCollectionGroup("", collectionID, options))
}

// DatabaseOption is an option for a Database pipeline stage.
type DatabaseOption interface {
	StageOption
	isDatabaseOption()
}

// Database creates a new [Pipeline] that operates on all documents in the Firestore database.
func (ps *PipelineSource) Database(opts ...DatabaseOption) *Pipeline {
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	return newPipeline(ps.client, newInputStageDatabase(options))
}

// DocumentsOption is an option for a Documents pipeline stage.
type DocumentsOption interface {
	StageOption
	isDocumentsOption()
}

// Documents creates a new [Pipeline] that operates on a specific set of Firestore documents.
func (ps *PipelineSource) Documents(refs []*DocumentRef, opts ...DocumentsOption) *Pipeline {
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	p := newPipeline(ps.client, newInputStageDocuments(refs, options))
	for _, ref := range refs {
		if ref == nil {
			p.err = fmt.Errorf("firestore: Documents() cannot contain nil references")
			break
		}
	}
	return p
}

// CreateFromQuery creates a new [Pipeline] from the given [Queryer]. Under the hood, this will
// translate the query semantics (order by document ID, etc.) to an equivalent pipeline.
func (ps *PipelineSource) CreateFromQuery(query Queryer) *Pipeline {
	return query.query().Pipeline()
}

// CreateFromAggregationQuery creates a new [Pipeline] from the given [AggregationQuery]. Under the hood, this will
// translate the query semantics (order by document ID, etc.) to an equivalent pipeline.
func (ps *PipelineSource) CreateFromAggregationQuery(query *AggregationQuery) *Pipeline {
	return query.Pipeline()
}

// LiteralsOption is an option for a Literals pipeline stage.
type LiteralsOption interface {
	StageOption
	isLiteralsOption()
}

// Literals creates a new [Pipeline] that operates on a fixed set of predefined document objects.
func (ps *PipelineSource) Literals(documents []map[string]any, opts ...LiteralsOption) *Pipeline {
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	return newPipeline(ps.client, newInputStageLiterals(documents, options))
}

// Subcollection creates a new [Pipeline] that operates on a subcollection of the current document.
//
// This method allows you to start a new pipeline that operates on a subcollection of the
// current document. It is intended to be used as a subquery.
//
// Note: A pipeline created with `Subcollection` cannot be executed directly using
// [Pipeline.Execute]. It must be used within a parent pipeline.
//
// Example:
//
//	client.Pipeline().Collection("books").
//		AddFields(Selectables(
//			Subcollection("reviews").
//				Aggregate(Accumulators(Average("rating").As("avg_rating"))).
//				ToScalarExpression().As("average_rating"),
//		))
func Subcollection(path string) *Pipeline {
	return newPipeline(nil, newInputStageSubcollection(path))
}
