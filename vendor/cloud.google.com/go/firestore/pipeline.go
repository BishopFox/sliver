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
	"context"
	"fmt"

	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
)

// Pipeline class provides a flexible and expressive framework for building complex data
// transformation and query pipelines for Firestore.
//
// A pipeline takes data sources, such as Firestore collections or collection groups, and applies
// a series of stages that are chained together. Each stage takes the output from the previous stage
// (or the data source) and produces an output for the next stage (or as the final output of the
// pipeline).
//
// Expressions can be used within
// each stages to filter and transform data through the stage.
//
// NOTE: The chained stages do not prescribe exactly how Firestore will execute the pipeline.
// Instead, Firestore only guarantees that the result is the same as if the chained stages were
// executed in order.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type Pipeline struct {
	c               *Client
	stages          []pipelineStage
	readSettings    *readSettings
	executeSettings *executeSettings
	tx              *Transaction
	err             error
}

func newPipeline(client *Client, initialStage pipelineStage) *Pipeline {
	return &Pipeline{
		c:               client,
		stages:          []pipelineStage{initialStage},
		readSettings:    &readSettings{},
		executeSettings: &executeSettings{},
	}
}

// executeSettings holds the options for executing a pipeline.
type executeSettings struct {
	ExplainOptions *executeExplainOptions
	IndexMode      string
}

// ExecuteOption is an option for executing a pipeline query.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type ExecuteOption interface {
	apply(*executeSettings)
}

type funcExecuteOption struct {
	f func(*executeSettings)
}

func (fdo *funcExecuteOption) apply(do *executeSettings) {
	fdo.f(do)
}

func newFuncExecuteOption(f func(*executeSettings)) *funcExecuteOption {
	return &funcExecuteOption{
		f: f,
	}
}

// ExplainMode is the execution mode for pipeline explain.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type ExplainMode string

const (
	// ExplainModeAnalyze both plans and executes the query.
	ExplainModeAnalyze ExplainMode = "analyze"
)

// executeExplainOptions are options for explaining a pipeline execution.
type executeExplainOptions struct {
	Mode ExplainMode
}

// WithExplainMode sets the execution mode for pipeline explain.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func WithExplainMode(mode ExplainMode) ExecuteOption {
	return newFuncExecuteOption(func(eo *executeSettings) {
		eo.ExplainOptions = &executeExplainOptions{Mode: mode}
	})
}

// Execute executes the pipeline and returns a snapshot of the results.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) Execute(ctx context.Context) *PipelineSnapshot {
	ctx = withResourceHeader(ctx, p.c.path())
	ctx = withRequestParamsHeader(ctx, reqParamsHeaderVal(p.c.path()))

	return &PipelineSnapshot{
		iter: &PipelineResultIterator{
			iter: newStreamPipelineResultIterator(ctx, p),
		},
	}
}

func (p *Pipeline) toExecutePipelineRequest() (*pb.ExecutePipelineRequest, error) {
	pipelinePb, err := p.toProto()
	if err != nil {
		return nil, err
	}

	options := make(map[string]*pb.Value)
	if p.executeSettings.ExplainOptions != nil {
		options["explain_options"] = &pb.Value{ValueType: &pb.Value_MapValue{MapValue: &pb.MapValue{
			Fields: map[string]*pb.Value{
				"mode": {ValueType: &pb.Value_StringValue{StringValue: string(p.executeSettings.ExplainOptions.Mode)}},
			},
		}}}
	}
	if p.executeSettings.IndexMode != "" {
		options["index_mode"] = &pb.Value{ValueType: &pb.Value_StringValue{StringValue: p.executeSettings.IndexMode}}
	}

	req := &pb.ExecutePipelineRequest{
		Database: p.c.path(),
		PipelineType: &pb.ExecutePipelineRequest_StructuredPipeline{
			StructuredPipeline: &pb.StructuredPipeline{
				Pipeline: pipelinePb,
				Options:  options,
			},
		},
	}

	// Note that transaction ID and other consistency selectors are mutually exclusive.
	// We respect the transaction first, any read options passed by the caller second,
	// and any read options stored in the client third.
	if p.tx != nil {
		req.ConsistencySelector = &pb.ExecutePipelineRequest_Transaction{Transaction: p.tx.id}
	} else if rt, hasOpts := parseReadTime(p.c, p.readSettings); hasOpts {
		req.ConsistencySelector = &pb.ExecutePipelineRequest_ReadTime{ReadTime: rt}
	}
	return req, nil
}

func (p *Pipeline) toProto() (*pb.Pipeline, error) {
	if p.err != nil {
		return nil, p.err
	}
	protoStages := make([]*pb.Pipeline_Stage, len(p.stages))
	for i, s := range p.stages {
		ps, err := s.toProto()
		if err != nil {
			return nil, fmt.Errorf("firestore: error converting stage %q to proto: %w", s.name(), err)
		}
		protoStages[i] = ps
	}
	return &pb.Pipeline{Stages: protoStages}, nil
}

func (p *Pipeline) copy() *Pipeline {
	newP := &Pipeline{
		c:               p.c,
		stages:          make([]pipelineStage, len(p.stages)),
		readSettings:    &readSettings{},
		executeSettings: &executeSettings{},
		tx:              p.tx,
		err:             p.err,
	}
	copy(newP.stages, p.stages)
	*newP.readSettings = *p.readSettings
	*newP.executeSettings = *p.executeSettings
	return newP
}

// WithReadOptions specifies constraints for accessing documents from the database,
// such as ReadTime.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) WithReadOptions(opts ...ReadOption) *Pipeline {
	newP := p.copy()
	for _, opt := range opts {
		if opt != nil {
			opt.apply(newP.readSettings)
		}
	}
	return newP
}

// WithExecuteOptions specifies options for executing a pipeline.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) WithExecuteOptions(opts ...ExecuteOption) *Pipeline {
	newP := p.copy()
	for _, opt := range opts {
		if opt != nil {
			opt.apply(newP.executeSettings)
		}
	}
	return newP
}

// append creates a new Pipeline by adding a stage to the current one.
func (p *Pipeline) append(s pipelineStage) *Pipeline {
	if p.err != nil {
		return p
	}
	newP := p.copy()
	newP.stages = append(newP.stages, s)
	return newP
}

// Limit limits the maximum number of documents returned by previous stages.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) Limit(limit int) *Pipeline {
	return p.append(newLimitStage(limit))
}

// OrderingDirection is the sort direction for pipeline result ordering.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type OrderingDirection string

const (
	// OrderingAsc sorts results from smallest to largest.
	OrderingAsc OrderingDirection = OrderingDirection("ascending")

	// OrderingDesc sorts results from largest to smallest.
	OrderingDesc OrderingDirection = OrderingDirection("descending")
)

// Ordering specifies the field and direction for sorting.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type Ordering struct {
	Expr      Expression
	Direction OrderingDirection
}

// Ascending creates an Ordering for ascending sort direction.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Ascending(expr Expression) Ordering {
	return Ordering{Expr: expr, Direction: OrderingAsc}
}

// Descending creates an Ordering for descending sort direction.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Descending(expr Expression) Ordering {
	return Ordering{Expr: expr, Direction: OrderingDesc}
}

// Sort sorts the documents by the given fields and directions.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) Sort(orders ...Ordering) *Pipeline {
	return p.append(newSortStage(orders...))
}

// Offset skips the first `offset` number of documents from the results of previous stages.
//
// This stage is useful for implementing pagination in your pipelines, allowing you to retrieve
// results in chunks. It is typically used in conjunction with [*Pipeline.Limit] to control the
// size of each page.
//
// Example:
// Retrieve the second page of 20 results
//
//	  client.Pipeline().Collection("books").
//		  .Offset(20)   // Skip the first 20 results
//		  .Limit(20)    // Take the next 20 results
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) Offset(offset int) *Pipeline {
	return p.append(newOffsetStage(offset))
}

// Select selects or creates a set of fields from the outputs of previous stages.
// The selected fields are defined using field path string, [FieldPath] or [Selectable] expressions.
// [Selectable] expressions can be:
//   - Field: References an existing field.
//   - Function: Represents the result of a function with an assigned alias name using [FunctionExpression.As].
//
// Example:
//
//		client.Pipeline().Collection("users").Select("info.email")
//		client.Pipeline().Collection("users").Select(FieldOf("info.email"))
//		client.Pipeline().Collection("users").Select(FieldOf([]string{"info", "email"}))
//		client.Pipeline().Collection("users").Select(FieldOf([]string{"info", "email"}))
//	 	client.Pipeline().Collection("users").Select(Add("age", 5).As("agePlus5"))
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) Select(fieldpathsOrSelectables ...any) *Pipeline {
	if p.err != nil {
		return p
	}
	stage, err := newSelectStage(fieldpathsOrSelectables...)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// Distinct removes duplicate documents from the outputs of previous stages.
//
// You can optionally specify fields or [Selectable] expressions to determine distinctness.
// If no fields are specified, the entire document is used to determine distinctness.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) Distinct(fieldpathsOrSelectables ...any) *Pipeline {
	if p.err != nil {
		return p
	}
	stage, err := newDistinctStage(fieldpathsOrSelectables...)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// AddFields adds new fields to outputs from previous stages.
//
// This stage allows you to compute values on-the-fly based on existing data from previous
// stages or constants. You can use this to create new fields or overwrite existing ones (if there
// is name overlaps).
//
// The added fields are defined using [Selectable]s
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) AddFields(selectables ...Selectable) *Pipeline {
	if p.err != nil {
		return p
	}
	stage, err := newAddFieldsStage(selectables...)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// RemoveFields removes fields from outputs from previous stages.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) RemoveFields(fieldpaths ...any) *Pipeline {
	if p.err != nil {
		return p
	}
	stage, err := newRemoveFieldsStage(fieldpaths...)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// Where filters the documents from previous stages to only include those matching the specified [BooleanExpression].
//
// This stage allows you to apply conditions to the data, similar to a "WHERE" clause in SQL.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) Where(condition BooleanExpression) *Pipeline {
	if p.err != nil {
		return p
	}
	stage, err := newWhereStage(condition)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// AggregateSpec is used to perform aggregation operations.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type AggregateSpec struct {
	groups     []Selectable
	accTargets []*AliasedAggregate
	err        error
}

// NewAggregateSpec creates a new AggregateSpec with the given accumulator targets.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func NewAggregateSpec(accumulators ...*AliasedAggregate) *AggregateSpec {
	return &AggregateSpec{accTargets: accumulators}
}

// WithGroups sets the grouping keys for the aggregation.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (a *AggregateSpec) WithGroups(fieldpathsOrSelectables ...any) *AggregateSpec {
	a.groups, a.err = fieldsOrSelectablesToSelectables(fieldpathsOrSelectables...)
	return a
}

// Aggregate performs aggregation operations on the documents from previous stages.
// This stage allows you to calculate aggregate values over a set of documents. You define the
// aggregations to perform using [AliasedAggregate] expressions which are typically results of
// calling [AggregateFunction.As] on [AggregateFunction] instances.
// Example:
//
//	client.Pipeline().Collection("users").
//		Aggregate(Sum("age").As("age_sum"))
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) Aggregate(accumulators ...*AliasedAggregate) *Pipeline {
	a := NewAggregateSpec(accumulators...)
	aggStage, err := newAggregateStage(a)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(aggStage)
}

// AggregateWithSpec performs optionally grouped aggregation operations on the documents from previous stages.
// This stage allows you to calculate aggregate values over a set of documents, optionally
// grouped by one or more fields or functions. You can specify:
//   - Grouping Fields or Functions: One or more fields or functions to group the documents
//     by. For each distinct combination of values in these fields, a separate group is created.
//     If no grouping fields are provided, a single group containing all documents is used. Not
//     specifying groups is the same as putting the entire inputs into one group.
//   - Accumulator targets: One or more accumulation operations to perform within each group. These
//     are defined using [AliasedAggregate] expressions which are typically results of calling
//     [AggregateFunction.As] on [AggregateFunction] instances. Each aggregation
//     calculates a value (e.g., sum, average, count) based on the documents within its group.
//
// Example:
//
//		// Calculate the average rating for each genre.
//		client.Pipeline().Collection("books").
//	        AggregateWithSpec(NewAggregateSpec(Average("rating").As("avg_rating")).WithGroups("genre"))
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) AggregateWithSpec(spec *AggregateSpec) *Pipeline {
	aggStage, err := newAggregateStage(spec)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(aggStage)
}

// UnnestOptions holds the configuration for the Unnest stage.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type UnnestOptions struct {
	// IndexField specifies the name of the field to store the array index of the unnested element.
	IndexField any
}

// Unnest produces a document for each element in an array field.
// For each input document, this stage outputs zero or more documents.
// Each output document is a copy of the input document, but the array field is replaced by an element from the array.
// The `field` parameter specifies the array field to unnest. It can be a string representing the field path or a [Selectable] expression.
// The alias of the selectable will be used as the new field name.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) Unnest(field Selectable, opts *UnnestOptions) *Pipeline {
	if p.err != nil {
		return p
	}
	stage, err := newUnnestStageFromSelectable(field, opts)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// UnnestWithAlias produces a document for each element in an array field, with a specified alias for the unnested field.
// It can optionally take UnnestOptions.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) UnnestWithAlias(fieldpath any, alias string, opts *UnnestOptions) *Pipeline {
	if p.err != nil {
		return p
	}

	var fieldExpr Expression
	switch v := fieldpath.(type) {
	case string:
		fieldExpr = FieldOf(v)
	case FieldPath:
		fieldExpr = FieldOf(v)
	default:
		p.err = errInvalidArg(fieldpath, "string", "FieldPath")
		return p
	}

	stage, err := newUnnestStage(fieldExpr, alias, opts)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// Union performs union of all documents from two pipelines, including duplicates.
//
// This stage will pass through documents from previous stage, and also pass through documents
// from previous stage of the other [*Pipeline] given in parameter. The order of documents
// emitted from this stage is undefined.
//
// Example:
//
//	// Emit documents from books collection and magazines collection.
//	client.Pipeline().Collection("books").
//		Union(client.Pipeline().Collection("magazines"))
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) Union(other *Pipeline) *Pipeline {
	if p.err != nil {
		return p
	}
	stage, err := newUnionStage(other)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// SampleMode defines the mode for the sample stage.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type SampleMode string

const (
	// SampleModeDocuments samples a fixed number of documents.
	SampleModeDocuments SampleMode = "documents"
	// SampleModePercent samples a percentage of documents.
	SampleModePercent SampleMode = "percent"
)

// SampleSpec is used to define a sample operation.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type SampleSpec struct {
	Size any
	Mode SampleMode
}

// SampleByDocuments creates a SampleSpec for sampling a fixed number of documents.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func SampleByDocuments(limit int) *SampleSpec {
	return &SampleSpec{Size: limit, Mode: SampleModeDocuments}
}

// Sample performs a pseudo-random sampling of the documents from the previous stage.
//
// This stage will filter documents pseudo-randomly. The behavior is defined by the SampleSpec.
// Use SampleByDocuments or SampleByPercentage to create a SampleSpec.
//
// Example:
//
//	// Sample 10 books, if available.
//	client.Pipeline().Collection("books").Sample(SampleByDocuments(10))
//
//	// Sample 50% of books.
//	client.Pipeline().Collection("books").Sample(&SampleSpec{Size: 0.5, Mode: SampleModePercent})
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) Sample(spec *SampleSpec) *Pipeline {
	if p.err != nil {
		return p
	}
	stage, err := newSampleStage(spec)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// ReplaceWith fully overwrites all fields in a document with those coming from a nested map.
//
// This stage allows you to emit a map value as a document. Each key of the map becomes a field
// on the document that contains the corresponding value.
//
// Example:
//
//	// Input: { "name": "John Doe Jr.", "parents": { "father": "John Doe Sr.", "mother": "Jane Doe" } }
//	// Emit parents as document.
//	client.Pipeline().Collection("people").ReplaceWith("parents")
//	// Output: { "father": "John Doe Sr.", "mother": "Jane Doe" }
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) ReplaceWith(fieldpathOrExpr any) *Pipeline {
	if p.err != nil {
		return p
	}
	stage, err := newReplaceWithStage(fieldpathOrExpr)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// PipelineDistanceMeasure is the distance measure for find_nearest pipeline stage.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type PipelineDistanceMeasure string

const (
	// PipelineDistanceMeasureEuclidean is used to measures the Euclidean distance between the vectors.
	PipelineDistanceMeasureEuclidean PipelineDistanceMeasure = "euclidean"
	// PipelineDistanceMeasureCosine compares vectors based on the angle between them.
	PipelineDistanceMeasureCosine PipelineDistanceMeasure = "cosine"
	// PipelineDistanceMeasureDotProduct is similar to cosine but is affected by the magnitude of the vectors.
	PipelineDistanceMeasureDotProduct PipelineDistanceMeasure = "dot_product"
)

// PipelineFindNearestOptions are options for a FindNearest pipeline stage.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type PipelineFindNearestOptions struct {
	Limit         *int
	DistanceField *string
}

// FindNearest performs vector distance (similarity) search with given parameters to the stage inputs.
//
// This stage adds a "nearest neighbor search" capability to your pipelines. Given a field that
// stores vectors and a target vector, this stage will identify and return the inputs whose vector
// field is closest to the target vector.
//
// The vectorField can be a string, a FieldPath or an Expr.
// The queryVector can be Vector32, Vector64, []float32, or []float64.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) FindNearest(vectorField any, queryVector any, measure PipelineDistanceMeasure, options *PipelineFindNearestOptions) *Pipeline {
	if p.err != nil {
		return p
	}

	stage, err := newFindNearestStage(vectorField, queryVector, measure, options)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// RawStage adds a generic stage to the pipeline.
// This method provides a flexible way to extend the pipeline's functionality by adding custom stages.
//
// Example:
//
//	// Assume we don't have a built-in "where" stage
//	client.Pipeline().Collection("books").
//		RawStage(
//			NewRawStage("where").
//				WithArguments(
//					LessThan(FieldOf("published"), 1900),
//				),
//		).
//		Select("title", "author")
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) RawStage(stage *RawStage) *Pipeline {
	if p.err != nil {
		return p
	}
	return p.append(stage)
}
