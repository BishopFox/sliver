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
	"errors"
	"fmt"
	"reflect"

	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
)

var (
	// ErrPipelineWithoutDatabase is returned when a pipeline is executed without a database such as a subcollection pipeline.
	ErrPipelineWithoutDatabase = errors.New("firestore: pipeline without a database cannot be executed directly, only as part of another pipeline")
	// ErrRelativeScopeUnionUnsupported is returned when a union is used with a relative scope pipeline.
	ErrRelativeScopeUnionUnsupported = errors.New("firestore: union only supports combining root pipelines; relative scope pipelines (like subcollection pipelines) are not supported")
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
	RawOptions     map[string]any
}

// ExecuteOption is an option for executing a pipeline query.
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
func WithExplainMode(mode ExplainMode) ExecuteOption {
	return newFuncExecuteOption(func(eo *executeSettings) {
		eo.ExplainOptions = &executeExplainOptions{Mode: mode}
	})
}

// StageOption is an option for configuring a pipeline stage.
type StageOption interface {
	applyStage(options map[string]any)
}

// RawOptions specifies raw options to be passed to the Firestore backend.
// These options are not validated by the SDK and are passed directly to the backend.
// Options specified here will take precedence over any options with the same name set by the SDK.
type RawOptions map[string]any

func (r RawOptions) applyStage(options map[string]any) {
	for k, v := range r {
		options[k] = v
	}
}

func (r RawOptions) applyAggregate(options map[string]any) {
	for k, v := range r {
		options[k] = v
	}
}

func (RawOptions) isLimitOption() {}

func (RawOptions) isSortOption() {}

func (RawOptions) isOffsetOption() {}

func (RawOptions) isSelectOption() {}

func (RawOptions) isDistinctOption() {}

func (RawOptions) isAddFieldsOption() {}

func (RawOptions) isRemoveFieldsOption() {}

func (RawOptions) isWhereOption() {}

func (RawOptions) isAggregateOption() {}

func (RawOptions) isUnnestOption() {}

func (RawOptions) isUnionOption() {}

func (RawOptions) isSampleOption() {}

func (RawOptions) isReplaceWithOption() {}

func (RawOptions) isFindNearestOption() {}

func (RawOptions) isUpdateOption() {}

func (RawOptions) isDeleteOption() {}

func (RawOptions) isCollectionOption() {}

func (RawOptions) isCollectionGroupOption() {}

func (RawOptions) isDatabaseOption() {}

func (RawOptions) isDocumentsOption() {}

func (RawOptions) isLiteralsOption() {}

func (RawOptions) isDefineOption() {}

func (RawOptions) isSearchOption() {}

func (r RawOptions) apply(eo *executeSettings) {
	if eo.RawOptions == nil {
		eo.RawOptions = make(map[string]any)
	}
	for k, v := range r {
		eo.RawOptions[k] = v
	}
}

// Fields is a helper function that returns its arguments as a slice of any.
// It is used to provide variadic-like ergonomics for pipeline stages that accept a slice of fields or expressions.
func Fields(f ...any) []any {
	return []any(f)
}

// Orders is a helper function that returns its arguments as a slice of Ordering.
// It is used to provide variadic-like ergonomics for the Sort pipeline stage.
func Orders(o ...Ordering) []Ordering {
	return []Ordering(o)
}

// Accumulators is a helper function that returns its arguments as a slice of *AliasedAggregate.
// It is used to provide variadic-like ergonomics for the Aggregate pipeline stage.
func Accumulators(a ...*AliasedAggregate) []*AliasedAggregate {
	return []*AliasedAggregate(a)
}

// Selectables is a helper function that returns its arguments as a slice of Selectable.
// It is used to provide variadic-like ergonomics for pipeline stages that accept a slice of Selectable expressions.
func Selectables(s ...Selectable) []Selectable {
	return []Selectable(s)
}

// AliasedExpressions is a helper function that returns its arguments as a slice of *AliasedExpression.
// It is used to provide variadic-like ergonomics for the [Pipeline.Define] pipeline stage.
func AliasedExpressions(v ...*AliasedExpression) []*AliasedExpression {
	return v
}

// Execute executes the pipeline and returns a snapshot of the results.
func (p *Pipeline) Execute(ctx context.Context, opts ...ExecuteOption) *PipelineSnapshot {
	newP := p
	if len(opts) > 0 {
		newP = p.copy()
		for _, opt := range opts {
			if opt != nil {
				opt.apply(newP.executeSettings)
			}
		}
	}

	if newP.c == nil {
		newP.err = ErrPipelineWithoutDatabase
		return &PipelineSnapshot{
			iter: &PipelineResultIterator{
				err: newP.err,
			},
		}
	}

	ctx = withResourceHeader(ctx, newP.c.path())
	ctx = withRequestParamsHeader(ctx, reqParamsHeaderVal(newP.c.path()))

	return &PipelineSnapshot{
		iter: &PipelineResultIterator{
			iter: newStreamPipelineResultIterator(ctx, newP),
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

	for k, v := range p.executeSettings.RawOptions {
		pbVal, _, err := toProtoValue(reflect.ValueOf(v))
		if err != nil {
			return nil, fmt.Errorf("firestore: error converting raw option %q: %w", k, err)
		}
		options[k] = pbVal
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
	if p.executeSettings.RawOptions != nil {
		newRawOptions := make(map[string]any, len(p.executeSettings.RawOptions))
		for k, v := range p.executeSettings.RawOptions {
			newRawOptions[k] = v
		}
		newP.executeSettings.RawOptions = newRawOptions
	}
	if p.executeSettings.ExplainOptions != nil {
		newExplainOpts := *p.executeSettings.ExplainOptions
		newP.executeSettings.ExplainOptions = &newExplainOpts
	}
	return newP
}

// WithReadOptions specifies constraints for accessing documents from the database,
// such as ReadTime.
func (p *Pipeline) WithReadOptions(opts ...ReadOption) *Pipeline {
	newP := p.copy()
	for _, opt := range opts {
		if opt != nil {
			opt.apply(newP.readSettings)
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

// LimitOption is an option for a Limit pipeline stage.
type LimitOption interface {
	StageOption
	isLimitOption()
}

// Limit limits the maximum number of documents returned by previous stages.
func (p *Pipeline) Limit(limit int, opts ...LimitOption) *Pipeline {
	if p.err != nil {
		return p
	}
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	return p.append(newLimitStage(limit, options))
}

// OrderingDirection is the sort direction for pipeline result ordering.
type OrderingDirection string

const (
	// OrderingAsc sorts results from smallest to largest.
	OrderingAsc OrderingDirection = OrderingDirection("ascending")

	// OrderingDesc sorts results from largest to smallest.
	OrderingDesc OrderingDirection = OrderingDirection("descending")
)

// Ordering specifies the field and direction for sorting.
type Ordering struct {
	Expr      Expression
	Direction OrderingDirection
}

// Ascending creates an Ordering for ascending sort direction.
func Ascending(expr Expression) Ordering {
	return Ordering{Expr: expr, Direction: OrderingAsc}
}

// Descending creates an Ordering for descending sort direction.
func Descending(expr Expression) Ordering {
	return Ordering{Expr: expr, Direction: OrderingDesc}
}

// SortOption is an option for a Sort pipeline stage.
type SortOption interface {
	StageOption
	isSortOption()
}

// Sort sorts the documents by the given fields and directions.
// Use [Orders] to provide variadic-like ergonomics for the orders argument.
func (p *Pipeline) Sort(orders []Ordering, opts ...SortOption) *Pipeline {
	if p.err != nil {
		return p
	}
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	return p.append(newSortStage(orders, options))
}

// OffsetOption is an option for an Offset pipeline stage.
type OffsetOption interface {
	StageOption
	isOffsetOption()
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
func (p *Pipeline) Offset(offset int, opts ...OffsetOption) *Pipeline {
	if p.err != nil {
		return p
	}
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	return p.append(newOffsetStage(offset, options))
}

// SelectOption is an option for a Select pipeline stage.
type SelectOption interface {
	StageOption
	isSelectOption()
}

// Select selects or creates a set of fields from the outputs of previous stages.
// The selected fields are defined using field path string, [FieldPath] or [Selectable] expressions.
// [Selectable] expressions can be:
//   - Field: References an existing field.
//   - Function: Represents the result of a function with an assigned alias name using [FunctionExpression.As].
//
// Use [Fields] to provide variadic-like ergonomics for the fields argument.
//
// Example:
//
//		client.Pipeline().Collection("users").Select(Fields("info.email"))
//		client.Pipeline().Collection("users").Select(Fields(FieldOf("info.email")))
//		client.Pipeline().Collection("users").Select(Fields(FieldOf([]string{"info", "email"})))
//		client.Pipeline().Collection("users").Select([]any{"info.email", "name"})
//	 	client.Pipeline().Collection("users").Select(Fields(Add("age", 5).As("agePlus5")))
func (p *Pipeline) Select(fields []any, opts ...SelectOption) *Pipeline {
	if p.err != nil {
		return p
	}
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	stage, err := newSelectStage(fields, options)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// DistinctOption is an option for a Distinct pipeline stage.
type DistinctOption interface {
	StageOption
	isDistinctOption()
}

// Distinct removes duplicate documents from the outputs of previous stages.
//
// You can optionally specify fields or [Selectable] expressions to determine distinctness.
// If no fields are specified, the entire document is used to determine distinctness.
// Use [Fields] to provide variadic-like ergonomics for the fields argument.
func (p *Pipeline) Distinct(fields []any, opts ...DistinctOption) *Pipeline {
	if p.err != nil {
		return p
	}
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	stage, err := newDistinctStage(fields, options)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// AddFieldsOption is an option for an AddFields pipeline stage.
type AddFieldsOption interface {
	StageOption
	isAddFieldsOption()
}

// AddFields adds new fields to outputs from previous stages.
//
// This stage allows you to compute values on-the-fly based on existing data from previous
// stages or constants. You can use this to create new fields or overwrite existing ones (if there
// is name overlaps).
//
// The added fields are defined using [Selectable]'s.
// Use [Selectables] to provide variadic-like ergonomics for the fields argument.
func (p *Pipeline) AddFields(fields []Selectable, opts ...AddFieldsOption) *Pipeline {
	if p.err != nil {
		return p
	}
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	stage, err := newAddFieldsStage(fields, options)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// RemoveFieldsOption is an option for an RemoveFields pipeline stage.
type RemoveFieldsOption interface {
	StageOption
	isRemoveFieldsOption()
}

// RemoveFields removes fields from outputs from previous stages.
// fieldpaths can be a string or a [FieldPath] or an expression obtained by calling [FieldOf].
// Use [Fields] to provide variadic-like ergonomics for the fields argument.
func (p *Pipeline) RemoveFields(fields []any, opts ...RemoveFieldsOption) *Pipeline {
	if p.err != nil {
		return p
	}
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	stage, err := newRemoveFieldsStage(fields, options)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// WhereOption is an option for a Where pipeline stage.
type WhereOption interface {
	StageOption
	isWhereOption()
}

// Where filters the documents from previous stages to only include those matching the specified [BooleanExpression].
//
// This stage allows you to apply conditions to the data, similar to a "WHERE" clause in SQL.
func (p *Pipeline) Where(condition BooleanExpression, opts ...WhereOption) *Pipeline {
	if p.err != nil {
		return p
	}
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	stage, err := newWhereStage(condition, options)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// AggregateOption is an option for executing a pipeline aggregation stage.
type AggregateOption interface {
	StageOption
	applyAggregate(options map[string]any)
	isAggregateOption()
}

type funcAggregateOption struct {
	f func(map[string]any)
}

func (fao *funcAggregateOption) applyAggregate(ao map[string]any) {
	fao.f(ao)
}

func (fao *funcAggregateOption) applyStage(ao map[string]any) {
	fao.f(ao)
}

func (*funcAggregateOption) isAggregateOption() {}

func newFuncAggregateOption(f func(map[string]any)) *funcAggregateOption {
	return &funcAggregateOption{
		f: f,
	}
}

// WithAggregateGroups specifies the fields or expressions to group the documents by.
// Each of the grouping keys can be a string field path, a [FieldPath], or a [Selectable] expression.
func WithAggregateGroups(groups ...any) AggregateOption {
	return newFuncAggregateOption(func(ao map[string]any) {
		g, ok := ao["groups"].([]any)
		if !ok {
			g = []any{}
		}
		ao["groups"] = append(g, groups...)
	})
}

// Aggregate performs aggregation operations on the documents from previous stages.
// This stage allows you to calculate aggregate values over a set of documents. You define the
// aggregations to perform using [AliasedAggregate] expressions which are typically results of
// calling AggregateFunction.As on [AggregateFunction] instances.
// Use [Accumulators] to provide variadic-like ergonomics for the accumulators argument.
//
// Example:
//
//	client.Pipeline().Collection("users").
//		Aggregate(Accumulators(Sum("age").As("age_sum")))
func (p *Pipeline) Aggregate(accumulators []*AliasedAggregate, opts ...AggregateOption) *Pipeline {
	if p.err != nil {
		return p
	}
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyAggregate(options)
		}
	}
	aggStage, err := newAggregateStage(accumulators, options)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(aggStage)
}

// UnnestOption is an option for executing a pipeline unnest stage.
type UnnestOption interface {
	StageOption
	isUnnestOption()
}

type funcUnnestOption struct {
	f func(map[string]any)
}

func (fuo *funcUnnestOption) applyStage(uo map[string]any) {
	fuo.f(uo)
}

func (*funcUnnestOption) isUnnestOption() {}

func newFuncUnnestOption(f func(map[string]any)) *funcUnnestOption {
	return &funcUnnestOption{
		f: f,
	}
}

// WithUnnestIndexField specifies the name of the field to store the array index of the unnested element.
func WithUnnestIndexField(indexField any) UnnestOption {
	return newFuncUnnestOption(func(uo map[string]any) {
		uo["index_field"] = indexField
	})
}

// Unnest produces a document for each element in an array field.
// For each input document, this stage outputs zero or more documents.
// Each output document is a copy of the input document, but the array field is replaced by an element from the array.
// The `field` parameter specifies the array field to unnest. It can be a string representing the field path or a [Selectable] expression.
// The alias of the selectable will be used as the new field name.
func (p *Pipeline) Unnest(field Selectable, opts ...UnnestOption) *Pipeline {
	if p.err != nil {
		return p
	}
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	stage, err := newUnnestStage("Unnest", field, options)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// UnnestWithAlias produces a document for each element in an array field, with a specified alias for the unnested field.
// It can optionally take UnnestOptions.
func (p *Pipeline) UnnestWithAlias(fieldpath any, alias string, opts ...UnnestOption) *Pipeline {
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
		p.err = errInvalidArg("UnnestWithAlias", fieldpath, "string", "FieldPath")
		return p
	}

	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	stage, err := newUnnestStage("UnnestWithAlias", fieldExpr.As(alias), options)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// UnionOption is an option for a Union pipeline stage.
type UnionOption interface {
	StageOption
	isUnionOption()
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
func (p *Pipeline) Union(other *Pipeline, opts ...UnionOption) *Pipeline {
	if p.err != nil {
		return p
	}
	if other.c == nil {
		p.err = ErrRelativeScopeUnionUnsupported
		return p
	}
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	stage, err := newUnionStage(other, options)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// SampleMode defines the mode for the sample stage.
type SampleMode string

const (
	// SampleModeDocuments samples a fixed number of documents.
	SampleModeDocuments SampleMode = "documents"
	// SampleModePercent samples a percentage of documents.
	SampleModePercent SampleMode = "percent"
)

// Sampler is used to define a sample operation.
type Sampler struct {
	Size any
	Mode SampleMode
}

// WithDocLimit creates a Sampler for sampling a fixed number of documents.
func WithDocLimit(limit int) *Sampler {
	return &Sampler{Size: limit, Mode: SampleModeDocuments}
}

// WithPercentage creates a Sampler for sampling a percentage of documents.
func WithPercentage(percentage float64) *Sampler {
	return &Sampler{Size: percentage, Mode: SampleModePercent}
}

// SampleOption is an option for a Sample pipeline stage.
type SampleOption interface {
	StageOption
	isSampleOption()
}

// Sample performs a pseudo-random sampling of the documents from the previous stage.
//
// This stage will filter documents pseudo-randomly. The behavior is defined by the Sampler.
// Use WithDocLimit or WithPercentage to create a Sampler.
//
// Example:
//
//	// Sample 10 books, if available.
//	client.Pipeline().Collection("books").Sample(WithDocLimit(10))
//
//	// Sample 50% of books.
//	client.Pipeline().Collection("books").Sample(WithPercentage(0.5))
func (p *Pipeline) Sample(sampler *Sampler, opts ...SampleOption) *Pipeline {
	if p.err != nil {
		return p
	}
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	stage, err := newSampleStage(sampler, options)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// ReplaceWithOption is an option for a ReplaceWith pipeline stage.
type ReplaceWithOption interface {
	StageOption
	isReplaceWithOption()
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
func (p *Pipeline) ReplaceWith(fieldpathOrExpr any, opts ...ReplaceWithOption) *Pipeline {
	if p.err != nil {
		return p
	}
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	stage, err := newReplaceWithStage(fieldpathOrExpr, options)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// PipelineDistanceMeasure is the distance measure for find_nearest pipeline stage.
type PipelineDistanceMeasure string

const (
	// PipelineDistanceMeasureEuclidean is used to measures the Euclidean distance between the vectors.
	PipelineDistanceMeasureEuclidean PipelineDistanceMeasure = "euclidean"
	// PipelineDistanceMeasureCosine compares vectors based on the angle between them.
	PipelineDistanceMeasureCosine PipelineDistanceMeasure = "cosine"
	// PipelineDistanceMeasureDotProduct is similar to cosine but is affected by the magnitude of the vectors.
	PipelineDistanceMeasureDotProduct PipelineDistanceMeasure = "dot_product"
)

// FindNearestOption is an option for a FindNearest pipeline stage.
type FindNearestOption interface {
	StageOption
	isFindNearestOption()
}

type funcFindNearestOption struct {
	f func(map[string]any)
}

func (fao *funcFindNearestOption) applyStage(ao map[string]any) {
	fao.f(ao)
}

func (*funcFindNearestOption) isFindNearestOption() {}

func newFuncFindNearestOption(f func(map[string]any)) *funcFindNearestOption {
	return &funcFindNearestOption{
		f: f,
	}
}

// WithFindNearestLimit specifies the maximum number of nearest neighbors to return.
func WithFindNearestLimit(limit int) FindNearestOption {
	return newFuncFindNearestOption(func(ao map[string]any) {
		ao["limit"] = limit
	})
}

// WithFindNearestDistanceField specifies the name of the field to store the calculated distance.
func WithFindNearestDistanceField(field string) FindNearestOption {
	return newFuncFindNearestOption(func(ao map[string]any) {
		ao["distance_field"] = field
	})
}

// FindNearest performs vector distance (similarity) search with given parameters to the stage inputs.
//
// This stage adds a "nearest neighbor search" capability to your pipelines. Given a field that
// stores vectors and a target vector, this stage will identify and return the inputs whose vector
// field is closest to the target vector.
//
// The vectorField can be a string, a FieldPath or an Expr.
// The queryVector can be Vector32, Vector64, []float32, or []float64.
func (p *Pipeline) FindNearest(vectorField any, queryVector any, measure PipelineDistanceMeasure, opts ...FindNearestOption) *Pipeline {
	if p.err != nil {
		return p
	}
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	stage, err := newFindNearestStage(vectorField, queryVector, measure, options)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// SearchOption is an option for a Search pipeline stage.
//
// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type SearchOption interface {
	StageOption
	isSearchOption()
}

type funcSearchOption struct {
	f func(map[string]any)
}

func (fso *funcSearchOption) applyStage(so map[string]any) {
	fso.f(so)
}

func (*funcSearchOption) isSearchOption() {}

func newFuncSearchOption(f func(map[string]any)) *funcSearchOption {
	return &funcSearchOption{
		f: f,
	}
}

// WithSearchQuery specifies the search query that will be used to query and score documents by the search stage.
// It can be a string (automatically wrapped in DocumentMatches) or a BooleanExpression.
//
// Example:
//
//	client.Pipeline().Collection("restaurants").Search(
//		WithSearchQuery("waffles"),
//	)
//
// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func WithSearchQuery(query any) SearchOption {
	return newFuncSearchOption(func(so map[string]any) {
		so["query"] = query
	})
}

// WithSearchSort specifies how the returned documents are sorted. One or more ordering are required.
//
// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func WithSearchSort(orders ...Ordering) SearchOption {
	return newFuncSearchOption(func(so map[string]any) {
		t, ok := so["sort"].([]Ordering)
		if !ok {
			t = []Ordering{}
		}
		so["sort"] = append(t, orders...)
	})
}

// WithSearchAddFields specifies the fields to add to each document.
//
// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func WithSearchAddFields(fields ...Selectable) SearchOption {
	return newFuncSearchOption(func(so map[string]any) {
		t, ok := so["add_fields"].([]Selectable)
		if !ok {
			t = []Selectable{}
		}
		so["add_fields"] = append(t, fields...)
	})
}

// WithSearchRetrievalDepth specifies the maximum number of documents to retrieve. Documents will be retrieved in the
// pre-sort order specified by the search index.
//
// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func WithSearchRetrievalDepth(depth int64) SearchOption {
	return newFuncSearchOption(func(so map[string]any) {
		so["retrieval_depth"] = depth
	})
}

// WithSearchLimit specifies the maximum number of documents to return from the search stage.
// The limit is applied after documents are scored and sorted.
//
// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func WithSearchLimit(limit int64) SearchOption {
	return newFuncSearchOption(func(so map[string]any) {
		so["limit"] = limit
	})
}

// WithSearchOffset specifies the number of documents to skip from the beginning of the search result set.
//
// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func WithSearchOffset(offset int64) SearchOption {
	return newFuncSearchOption(func(so map[string]any) {
		so["offset"] = offset
	})
}

// WithSearchLanguageCode specifies the BCP-47 language code of text in the search query, such as "en" or "sr".
//
// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func WithSearchLanguageCode(languageCode string) SearchOption {
	return newFuncSearchOption(func(so map[string]any) {
		so["language_code"] = languageCode
	})
}

// Search adds a search stage to the Pipeline.
// This must be the first stage of the pipeline.
// A limited set of expressions are supported in the search stage.
// Use [WithSearchQuery] to specify the search query.
//
// Example:
//
//	client.Pipeline().Collection("restaurants").Search(
//		WithSearchQuery(DocumentMatches("waffles OR pancakes")),
//		WithSearchSort(Descending(Score())),
//		WithSearchRetrievalDepth(10),
//	)
//
// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) Search(opts ...SearchOption) *Pipeline {
	if p.err != nil {
		return p
	}
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	stage, err := newSearchStage(options)
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
//		RawStage("where", []any{LessThan(FieldOf("published"), 1900)}).
//		Select(Fields("title", "author"))
func (p *Pipeline) RawStage(name string, args []any, opts ...StageOption) *Pipeline {
	if p.err != nil {
		return p
	}

	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}

	stage := &rawStage{
		stageName: name,
		args:      args,
		options:   options,
	}
	return p.append(stage)
}

// UpdateOption is an option for an Update pipeline stage.
//
// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type UpdateOption interface {
	StageOption
	isUpdateOption()
}

type funcUpdateOption struct {
	f func(map[string]any)
}

func (fuo *funcUpdateOption) applyStage(uo map[string]any) {
	fuo.f(uo)
}

func (*funcUpdateOption) isUpdateOption() {}

func newFuncUpdateOption(f func(map[string]any)) *funcUpdateOption {
	return &funcUpdateOption{
		f: f,
	}
}

// WithUpdateTransformations specifies the list of field transformations to apply in an update operation.
//
// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func WithUpdateTransformations(field Selectable, additionalFields ...Selectable) UpdateOption {
	return newFuncUpdateOption(func(uo map[string]any) {
		t, ok := uo["transformations"].([]Selectable)
		if !ok {
			t = []Selectable{}
		}
		uo["transformations"] = append(t, append([]Selectable{field}, additionalFields...)...)
	})
}

// Update performs an update operation using documents from previous stages.
//
// This method updates the documents in the database based on the data flowing through the pipeline.
// You can optionally specify a list of [Selectable] field transformations using [WithUpdateTransformations].
// If no transformations are provided, the entire document flowing from the previous stage is used as the update payload.
//
// Example:
//
//	// In-place update
//	client.Pipeline().Literals(updateData).Update()
//
//	// Update with transformations
//	client.Pipeline().Collection("books").
//		Where(GreaterThan("price", 50)).
//		Update(WithUpdateTransformations(ConstantOf("Discounted").As("status")))
//
// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) Update(opts ...UpdateOption) *Pipeline {
	if p.err != nil {
		return p
	}

	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}

	stage, err := newUpdateStage(options)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}

// DeleteOption is an option for a Delete pipeline stage.
//
// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type DeleteOption interface {
	StageOption
	isDeleteOption()
}

// Delete deletes the documents from previous stages.
//
// Example:
//
//	client.Pipeline().Collection("logs").
//		Where(Equal("status", "archived")).
//		Delete()
//
// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (p *Pipeline) Delete(opts ...DeleteOption) *Pipeline {
	if p.err != nil {
		return p
	}
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	stage := newDeleteStage(options)
	return p.append(stage)
}

// ToScalarExpression converts this Pipeline into an expression that evaluates to a single scalar result.
// Used for 1:1 lookups or Aggregations when the subquery is expected to return a single value or object.
//
// Example:
//
//	// Calculate average rating for each restaurant using a subquery
//	client.Pipeline().Collection("restaurants").
//		AddFields(Selectables(
//			Subcollection("reviews").
//				Aggregate(Accumulators(Average("rating").As("avg_score"))).
//				ToScalarExpression().As("stats"),
//		))
//	// Output format:
//	// [
//	//   {
//	//     "name": "The Burger Joint",
//	//     "stats": {
//	//       "avg_score": 4.8,
//	//       "review_count": 120
//	//     }
//	//   }
//	// ]
func (p *Pipeline) ToScalarExpression() Expression {
	return newBaseFunction("scalar", []Expression{newPipelineValueExpression(p)})
}

// ToArrayExpression converts this Pipeline into an expression that evaluates to an array result.
//
// Example:
//
//	// Embed a subcollection of reviews as an array into each restaurant document
//	client.Pipeline().Collection("restaurants").
//		AddFields(Selectables(
//			Subcollection("reviews").
//				Select(Fields("reviewer", "rating")).
//				ToArrayExpression().As("reviews"),
//		))
//	// Output format:
//	// [
//	//   {
//	//     "name": "The Burger Joint",
//	//     "reviews": [
//	//       { "reviewer": "Alice", "rating": 5 },
//	//       { "reviewer": "Bob", "rating": 4 }
//	//     ]
//	//   }
//	// ]
func (p *Pipeline) ToArrayExpression() Expression {
	return newBaseFunction("array", []Expression{newPipelineValueExpression(p)})
}

// DefineOption is an option for a Define pipeline stage.
type DefineOption interface {
	StageOption
	isDefineOption()
}

// Define defines one or more variables in the pipeline's scope. `Define` is used to bind a value to a
// variable for internal reuse within the pipeline body (accessed via the [Variable] function).
//
// This stage is useful for declaring reusable values or intermediate calculations that can be
// referenced multiple times in later parts of the pipeline, improving readability and
// maintainability.
//
// Each variable is defined using an [AliasedExpression], which pairs an expression with
// a name (alias).
//
// Example:
//
//	// Define a variable and use it in a filter
//	client.Pipeline().Collection("products").
//		Define(AliasedExpressions(
//			Multiply("price", 0.9).As("discountedPrice"),
//			Add("stock", 10).As("newStock"),
//		)).
//		Where(LessThan(Variable("discountedPrice"), 100)).
//		Select(Fields("name", Variable("newStock")))
func (p *Pipeline) Define(variables []*AliasedExpression, opts ...DefineOption) *Pipeline {
	if p.err != nil {
		return p
	}
	options := make(map[string]any)
	for _, opt := range opts {
		if opt != nil {
			opt.applyStage(options)
		}
	}
	stage, err := newDefineStage(variables, options)
	if err != nil {
		p.err = err
		return p
	}
	return p.append(stage)
}
