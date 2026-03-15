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
	"strings"

	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
)

// baseStage is an internal helper to reduce repetition in pipelineStage
// implementations.
type baseStage struct {
	stageName string
	stagePb   *pb.Pipeline_Stage
}

func (s *baseStage) name() string                         { return s.stageName }
func (s *baseStage) toProto() (*pb.Pipeline_Stage, error) { return s.stagePb, nil }

func errInvalidArg(v any, expected ...string) error {
	return fmt.Errorf("firestore: invalid argument type: %T, expected one of: [%s]", v, strings.Join(expected, ", "))
}

const (
	stageNameAddFields       = "add_fields"
	stageNameAggregate       = "aggregate"
	stageNameCollection      = "collection"
	stageNameCollectionGroup = "collection_group"
	stageNameDatabase        = "database"
	stageNameDistinct        = "distinct"
	stageNameDocuments       = "documents"
	stageNameFindNearest     = "find_nearest"
	stageNameRemoveFields    = "remove_fields"
	stageNameReplaceWith     = "replace_with"
	stageNameSample          = "sample"
	stageNameSelect          = "select"
	stageNameUnion           = "union"
	stageNameUnnest          = "unnest"
	stageNameWhere           = "where"
)

// internal interface for pipeline stages.
type pipelineStage interface {
	toProto() (*pb.Pipeline_Stage, error)
	name() string // For identification, logging, and potential validation
}

// inputStageCollection returns all documents from the entire collection.
type inputStageCollection struct {
	path    string
	options *collectionStageSettings
}

func newInputStageCollection(path string, options *collectionStageSettings) *inputStageCollection {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return &inputStageCollection{path: path, options: options}
}
func (s *inputStageCollection) name() string { return stageNameCollection }
func (s *inputStageCollection) toProto() (*pb.Pipeline_Stage, error) {
	optionsPb, err := s.options.toProto()
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    []*pb.Value{{ValueType: &pb.Value_ReferenceValue{ReferenceValue: s.path}}},
		Options: optionsPb,
	}, nil
}

// inputStageCollection returns all documents from the entire collection.
type inputStageCollectionGroup struct {
	collectionID string
	ancestor     string
	options      *collectionStageSettings
}

func newInputStageCollectionGroup(ancestor, collectionID string, options *collectionStageSettings) *inputStageCollectionGroup {
	return &inputStageCollectionGroup{ancestor: ancestor, collectionID: collectionID, options: options}
}
func (s *inputStageCollectionGroup) name() string { return stageNameCollectionGroup }
func (s *inputStageCollectionGroup) toProto() (*pb.Pipeline_Stage, error) {
	optionsPb, err := s.options.toProto()
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name: s.name(),
		Args: []*pb.Value{
			{ValueType: &pb.Value_ReferenceValue{ReferenceValue: s.ancestor}},
			{ValueType: &pb.Value_StringValue{StringValue: s.collectionID}},
		},
		Options: optionsPb,
	}, nil
}

// inputStageDatabase returns all documents from the entire database.
type inputStageDatabase struct{}

func newInputStageDatabase() *inputStageDatabase {
	return &inputStageDatabase{}
}
func (s *inputStageDatabase) name() string { return stageNameDatabase }
func (s *inputStageDatabase) toProto() (*pb.Pipeline_Stage, error) {
	return &pb.Pipeline_Stage{
		Name: s.name(),
	}, nil
}

type inputStageDocuments struct {
	baseStage
}

func newInputStageDocuments(refs ...*DocumentRef) *inputStageDocuments {
	args := make([]*pb.Value, len(refs))
	for i, ref := range refs {
		args[i] = &pb.Value{ValueType: &pb.Value_ReferenceValue{ReferenceValue: "/" + ref.shortPath}}
	}
	return &inputStageDocuments{baseStage{
		stageName: stageNameDocuments,
		stagePb: &pb.Pipeline_Stage{
			Name: stageNameDocuments,
			Args: args,
		},
	}}
}

// addFieldsStage is the internal representation of a AddFields stage.
type addFieldsStage struct {
	baseStage
}

func newAddFieldsStage(selectables ...Selectable) (*addFieldsStage, error) {
	mapVal, err := projectionsToMapValue(selectables)
	if err != nil {
		return nil, err
	}
	stagePb := newUnaryStage(stageNameAddFields, mapVal)
	return &addFieldsStage{baseStage{
		stageName: stageNameAddFields,
		stagePb:   stagePb,
	}}, nil
}

type aggregateStage struct {
	baseStage
}

func newAggregateStage(a *AggregateSpec) (*aggregateStage, error) {
	if a.err != nil {
		return nil, a.err
	}
	targetsPb, err := aliasedAggregatesToMapValue(a.accTargets)
	if err != nil {
		return nil, err
	}
	groupsPb, err := projectionsToMapValue(a.groups)
	if err != nil {
		return nil, err
	}
	return &aggregateStage{baseStage{
		stageName: stageNameAggregate,
		stagePb: &pb.Pipeline_Stage{
			Name: stageNameAggregate,
			Args: []*pb.Value{
				targetsPb,
				groupsPb,
			},
		},
	}}, nil
}

type distinctStage struct {
	baseStage
}

// newProjectionStage is a helper for creating pipeline stages that take a
// projection as an argument.
func newProjectionStage(name string, fieldsOrSelectables ...any) (*pb.Pipeline_Stage, error) {
	selectables, err := fieldsOrSelectablesToSelectables(fieldsOrSelectables...)
	if err != nil {
		return nil, err
	}
	mapVal, err := projectionsToMapValue(selectables)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name: name,
		Args: []*pb.Value{mapVal},
	}, nil
}

func newDistinctStage(fieldsOrSelectables ...any) (*distinctStage, error) {
	stagePb, err := newProjectionStage(stageNameDistinct, fieldsOrSelectables...)
	if err != nil {
		return nil, err
	}
	return &distinctStage{baseStage{stageName: stageNameDistinct, stagePb: stagePb}}, nil
}

type findNearestStage struct {
	baseStage
}

func newFindNearestStage(vectorField any, queryVector any, measure PipelineDistanceMeasure, options *PipelineFindNearestOptions) (*findNearestStage, error) {
	var propertyExpr Expression
	switch v := vectorField.(type) {
	case string:
		propertyExpr = FieldOf(v)
	case FieldPath:
		propertyExpr = FieldOf(v)
	case Expression:
		propertyExpr = v
	default:
		return nil, errInvalidArg(vectorField, "string", "FieldPath", "Expression")
	}
	propPb, err := propertyExpr.toProto()
	if err != nil {
		return nil, err
	}
	var vectorPb *pb.Value
	switch v := queryVector.(type) {
	case Vector32:
		vectorPb = vectorToProtoValue([]float32(v))
	case []float32:
		vectorPb = vectorToProtoValue(v)
	case Vector64:
		vectorPb = vectorToProtoValue([]float64(v))
	case []float64:
		vectorPb = vectorToProtoValue(v)
	default:
		return nil, errInvalidVector
	}
	measurePb := &pb.Value{ValueType: &pb.Value_StringValue{StringValue: string(measure)}}
	var optionsPb map[string]*pb.Value
	if options != nil {
		optionsPb = make(map[string]*pb.Value)
		if options.Limit != nil {
			optionsPb["limit"] = &pb.Value{ValueType: &pb.Value_IntegerValue{IntegerValue: int64(*options.Limit)}}
		}
		if options.DistanceField != nil {
			optionsPb["distance_field"] = &pb.Value{ValueType: &pb.Value_FieldReferenceValue{FieldReferenceValue: *options.DistanceField}}
		}
	}
	return &findNearestStage{baseStage{
		stageName: stageNameFindNearest,
		stagePb: &pb.Pipeline_Stage{
			Name:    stageNameFindNearest,
			Args:    []*pb.Value{propPb, vectorPb, measurePb},
			Options: optionsPb,
		},
	}}, nil
}

type limitStage struct {
	limit int
}

func newLimitStage(limit int) *limitStage {
	return &limitStage{limit: limit}
}
func (s *limitStage) name() string { return "limit" }
func (s *limitStage) toProto() (*pb.Pipeline_Stage, error) {
	arg := &pb.Value{ValueType: &pb.Value_IntegerValue{IntegerValue: int64(s.limit)}}
	return &pb.Pipeline_Stage{
		Name: s.name(),
		Args: []*pb.Value{arg},
	}, nil
}

type offsetStage struct {
	offset int
}

func newOffsetStage(offset int) *offsetStage {
	return &offsetStage{offset: offset}
}
func (s *offsetStage) name() string { return "offset" }
func (s *offsetStage) toProto() (*pb.Pipeline_Stage, error) {
	arg := &pb.Value{ValueType: &pb.Value_IntegerValue{IntegerValue: int64(s.offset)}}
	return &pb.Pipeline_Stage{
		Name: s.name(),
		Args: []*pb.Value{arg},
	}, nil
}

type removeFieldsStage struct {
	baseStage
}

func newRemoveFieldsStage(fieldpaths ...any) (*removeFieldsStage, error) {
	fields := make([]Expression, len(fieldpaths))
	for i, fp := range fieldpaths {
		switch v := fp.(type) {
		case string:
			fields[i] = FieldOf(v)
		case FieldPath:
			fields[i] = FieldOf(v)
		default:
			return nil, errInvalidArg(fp, "string", "FieldPath")
		}
	}
	args := make([]*pb.Value, len(fields))
	for i, f := range fields {
		pb, err := f.toProto()
		if err != nil {
			return nil, err
		}
		args[i] = pb
	}
	return &removeFieldsStage{baseStage{
		stageName: stageNameRemoveFields,
		stagePb: &pb.Pipeline_Stage{
			Name: stageNameRemoveFields,
			Args: args,
		},
	}}, nil
}

type replaceWithStage struct {
	baseStage
}

func newReplaceWithStage(fieldpathOrExpr any) (*replaceWithStage, error) {
	var expr Expression
	switch v := fieldpathOrExpr.(type) {
	case string:
		expr = FieldOf(v)
	case FieldPath:
		expr = FieldOf(v)
	case Expression:
		expr = v
	default:
		return nil, errInvalidArg(fieldpathOrExpr, "string", "FieldPath", "Expression")
	}
	exprPb, err := expr.toProto()
	if err != nil {
		return nil, err
	}
	return &replaceWithStage{baseStage{
		stageName: stageNameReplaceWith,
		stagePb: &pb.Pipeline_Stage{
			Name: stageNameReplaceWith,
			Args: []*pb.Value{exprPb, {ValueType: &pb.Value_StringValue{StringValue: "full_replace"}}},
		},
	}}, nil
}

type sampleStage struct {
	baseStage
}

func newSampleStage(spec *SampleSpec) (*sampleStage, error) {
	var sizePb *pb.Value
	switch v := spec.Size.(type) {
	case int:
		sizePb = &pb.Value{ValueType: &pb.Value_IntegerValue{IntegerValue: int64(v)}}
	case int64:
		sizePb = &pb.Value{ValueType: &pb.Value_IntegerValue{IntegerValue: v}}
	case float64:
		sizePb = &pb.Value{ValueType: &pb.Value_DoubleValue{DoubleValue: v}}
	default:
		return nil, fmt.Errorf("firestore: invalid type for sample size: %T", spec.Size)
	}
	modePb := &pb.Value{ValueType: &pb.Value_StringValue{StringValue: string(spec.Mode)}}
	return &sampleStage{baseStage{
		stageName: stageNameSample,
		stagePb: &pb.Pipeline_Stage{
			Name: stageNameSample,
			Args: []*pb.Value{sizePb, modePb},
		},
	}}, nil
}

type selectStage struct {
	baseStage
}

func newSelectStage(fieldsOrSelectables ...any) (*selectStage, error) {
	stagePb, err := newProjectionStage(stageNameSelect, fieldsOrSelectables...)
	if err != nil {
		return nil, err
	}
	return &selectStage{baseStage{stageName: stageNameSelect, stagePb: stagePb}}, nil
}

type sortStage struct {
	orders []Ordering
}

func newSortStage(orders ...Ordering) *sortStage {
	return &sortStage{orders: orders}
}
func (s *sortStage) name() string { return "sort" }
func (s *sortStage) toProto() (*pb.Pipeline_Stage, error) {
	sortOrders := make([]*pb.Value, len(s.orders))
	for i, so := range s.orders {
		fieldPb, err := so.Expr.toProto()
		if err != nil {
			return nil, err
		}
		sortOrders[i] = &pb.Value{
			ValueType: &pb.Value_MapValue{
				MapValue: &pb.MapValue{
					Fields: map[string]*pb.Value{
						"direction": {
							ValueType: &pb.Value_StringValue{
								StringValue: string(so.Direction),
							},
						},
						"expression": fieldPb,
					},
				},
			},
		}
	}
	return &pb.Pipeline_Stage{
		Name: s.name(),
		Args: sortOrders,
	}, nil
}

type unionStage struct {
	baseStage
}

func newUnionStage(other *Pipeline) (*unionStage, error) {
	otherPb, err := other.toProto()
	if err != nil {
		return nil, err
	}
	return &unionStage{baseStage{
		stageName: stageNameUnion,
		stagePb: &pb.Pipeline_Stage{
			Name: stageNameUnion,
			Args: []*pb.Value{
				{ValueType: &pb.Value_PipelineValue{PipelineValue: otherPb}},
			},
		},
	}}, nil
}

type unnestStage struct {
	baseStage
}

func newUnnestStage(fieldExpr Expression, alias string, opts *UnnestOptions) (*unnestStage, error) {
	exprPb, err := fieldExpr.toProto()
	if err != nil {
		return nil, err
	}
	aliasPb, err := FieldOf(alias).toProto()
	if err != nil {
		return nil, err
	}
	var optionsPb map[string]*pb.Value
	if opts != nil && opts.IndexField != nil {
		var indexFieldExpr Expression
		switch v := opts.IndexField.(type) {
		case FieldPath:
			indexFieldExpr = FieldOf(v)
		case string:
			indexFieldExpr = FieldOf(v)
		default:
			return nil, errInvalidArg(opts.IndexField, "string", "FieldPath")
		}
		indexPb, err := indexFieldExpr.toProto()
		if err != nil {
			return nil, err
		}
		optionsPb = make(map[string]*pb.Value)
		optionsPb["index_field"] = indexPb
	}
	return &unnestStage{baseStage{
		stageName: stageNameUnnest,
		stagePb: &pb.Pipeline_Stage{
			Name:    stageNameUnnest,
			Args:    []*pb.Value{exprPb, aliasPb},
			Options: optionsPb,
		},
	}}, nil
}

func newUnnestStageFromSelectable(field Selectable, opts *UnnestOptions) (*unnestStage, error) {
	alias, expr := field.getSelectionDetails()
	return newUnnestStage(expr, alias, opts)
}

type whereStage struct {
	baseStage
}

// newUnaryStage is a helper for creating pipeline stages that take a single
// proto as an argument.
func newUnaryStage(name string, val *pb.Value) *pb.Pipeline_Stage {
	return &pb.Pipeline_Stage{
		Name: name,
		Args: []*pb.Value{val},
	}
}

func newWhereStage(condition BooleanExpression) (*whereStage, error) {
	argsPb, err := condition.toProto()
	if err != nil {
		return nil, err
	}
	return &whereStage{baseStage{
		stageName: stageNameWhere,
		stagePb:   newUnaryStage(stageNameWhere, argsPb),
	}}, nil
}

// RawStageOptions holds the options for a RawStage.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type RawStageOptions map[string]any

// RawStage is a generic stage in the pipeline.
// It provides a flexible way to extend the pipeline's functionality by adding custom
// stages. It also allows the users to call the stages that are supported by the Firestore backend
// but not yet available in the current SDK version.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type RawStage struct {
	stageName string
	args      []any
	options   RawStageOptions
}

// NewRawStage creates a new RawStage with the given name.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func NewRawStage(name string) *RawStage {
	return &RawStage{stageName: name}
}

// WithArguments sets the arguments for the RawStage.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (s *RawStage) WithArguments(args ...any) *RawStage {
	s.args = args
	return s
}

// WithOptions sets the options for the RawStage.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (s *RawStage) WithOptions(options RawStageOptions) *RawStage {
	s.options = options
	return s
}

func (s *RawStage) name() string { return s.stageName }

func (s *RawStage) toProto() (*pb.Pipeline_Stage, error) {
	argsPb := make([]*pb.Value, len(s.args))
	for i, arg := range s.args {
		val, _, err := toProtoValue(reflect.ValueOf(arg))
		if err != nil {
			return nil, fmt.Errorf("firestore: error converting raw stage argument %d: %w", i, err)
		}
		argsPb[i] = val
	}

	optionsPb := make(map[string]*pb.Value, len(s.options))
	for key, val := range s.options {
		valPb, _, err := toProtoValue(reflect.ValueOf(val))
		if err != nil {
			return nil, fmt.Errorf("firestore: error converting raw stage option %q: %w", key, err)
		}
		optionsPb[key] = valPb
	}

	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    argsPb,
		Options: optionsPb,
	}, nil
}
