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

func errInvalidArg(stageName string, v any, expected ...string) error {
	return fmt.Errorf("firestore: invalid argument type for stage %s: %T, expected one of: [%s]", stageName, v, strings.Join(expected, ", "))
}

const (
	stageNameAddFields       = "add_fields"
	stageNameAggregate       = "aggregate"
	stageNameCollection      = "collection"
	stageNameCollectionGroup = "collection_group"
	stageNameDatabase        = "database"
	stageNameDelete          = "delete"
	stageNameDistinct        = "distinct"
	stageNameDocuments       = "documents"
	stageNameFindNearest     = "find_nearest"
	stageNameLimit           = "limit"
	stageNameLiterals        = "literals"
	stageNameOffset          = "offset"
	stageNameRemoveFields    = "remove_fields"
	stageNameReplaceWith     = "replace_with"
	stageNameSample          = "sample"
	stageNameSearch          = "search"
	stageNameSelect          = "select"
	stageNameSort            = "sort"
	stageNameUnion           = "union"
	stageNameUnnest          = "unnest"
	stageNameUpdate          = "update"
	stageNameWhere           = "where"
	stageNameDefine          = "let"
)

// internal interface for pipeline stages.
type pipelineStage interface {
	toProto() (*pb.Pipeline_Stage, error)
	name() string // For identification, logging, and potential validation
}

func stageOptionsToProto(options map[string]any) (map[string]*pb.Value, error) {
	if len(options) == 0 {
		return nil, nil
	}
	optsPb := make(map[string]*pb.Value)
	for k, v := range options {
		valPb, _, err := toProtoValue(reflect.ValueOf(v))
		if err != nil {
			return nil, fmt.Errorf("firestore: error converting stage option %q: %w", k, err)
		}
		optsPb[k] = valPb
	}
	return optsPb, nil
}

// inputStageCollection returns all documents from the entire collection.
type inputStageCollection struct {
	path    string
	options map[string]any
}

func newInputStageCollection(path string, options map[string]any) *inputStageCollection {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return &inputStageCollection{path: path, options: options}
}
func (s *inputStageCollection) name() string { return stageNameCollection }
func (s *inputStageCollection) toProto() (*pb.Pipeline_Stage, error) {
	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    []*pb.Value{{ValueType: &pb.Value_ReferenceValue{ReferenceValue: s.path}}},
		Options: optionsPb,
	}, nil
}

// inputStageCollectionGroup returns all documents from a group of collections.
type inputStageCollectionGroup struct {
	collectionID string
	ancestor     string
	options      map[string]any
}

func newInputStageCollectionGroup(ancestor, collectionID string, options map[string]any) *inputStageCollectionGroup {
	return &inputStageCollectionGroup{ancestor: ancestor, collectionID: collectionID, options: options}
}
func (s *inputStageCollectionGroup) name() string { return stageNameCollectionGroup }
func (s *inputStageCollectionGroup) toProto() (*pb.Pipeline_Stage, error) {
	optionsPb, err := stageOptionsToProto(s.options)
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
type inputStageDatabase struct {
	options map[string]any
}

func newInputStageDatabase(options map[string]any) *inputStageDatabase {
	return &inputStageDatabase{options: options}
}
func (s *inputStageDatabase) name() string { return stageNameDatabase }
func (s *inputStageDatabase) toProto() (*pb.Pipeline_Stage, error) {
	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Options: optionsPb,
	}, nil
}

// inputStageDocuments returns all documents from the specific references.
type inputStageDocuments struct {
	refs    []*DocumentRef
	options map[string]any
}

func newInputStageDocuments(refs []*DocumentRef, options map[string]any) *inputStageDocuments {
	return &inputStageDocuments{refs: refs, options: options}
}
func (s *inputStageDocuments) name() string { return stageNameDocuments }
func (s *inputStageDocuments) toProto() (*pb.Pipeline_Stage, error) {
	args := make([]*pb.Value, len(s.refs))
	for i, ref := range s.refs {
		if ref == nil {
			return nil, fmt.Errorf("firestore: inputStageDocuments contains nil reference")
		}
		args[i] = &pb.Value{ValueType: &pb.Value_ReferenceValue{ReferenceValue: "/" + ref.shortPath}}
	}
	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    args,
		Options: optionsPb,
	}, nil
}

// inputStageSubcollection returns a pipeline starting from a subcollection of the current document.
type inputStageSubcollection struct {
	path string
}

func newInputStageSubcollection(path string) *inputStageSubcollection {
	return &inputStageSubcollection{path: path}
}
func (s *inputStageSubcollection) name() string { return "subcollection" }
func (s *inputStageSubcollection) toProto() (*pb.Pipeline_Stage, error) {
	return &pb.Pipeline_Stage{
		Name: s.name(),
		Args: []*pb.Value{{ValueType: &pb.Value_StringValue{StringValue: s.path}}},
	}, nil
}

// inputStageLiterals returns a fixed set of documents.
type inputStageLiterals struct {
	documents []map[string]any
	options   map[string]any
}

func newInputStageLiterals(documents []map[string]any, options map[string]any) *inputStageLiterals {
	return &inputStageLiterals{documents: documents, options: options}
}
func (s *inputStageLiterals) name() string { return stageNameLiterals }
func (s *inputStageLiterals) toProto() (*pb.Pipeline_Stage, error) {
	args := make([]*pb.Value, len(s.documents))
	for i, doc := range s.documents {
		val, _, err := toProtoValue(reflect.ValueOf(doc))
		if err != nil {
			return nil, err
		}
		args[i] = val
	}
	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    args,
		Options: optionsPb,
	}, nil
}

type defineStage struct {
	variables []*AliasedExpression
	options   map[string]any
}

func newDefineStage(variables []*AliasedExpression, options map[string]any) (*defineStage, error) {
	return &defineStage{variables: variables, options: options}, nil
}
func (s *defineStage) name() string { return stageNameDefine }
func (s *defineStage) toProto() (*pb.Pipeline_Stage, error) {
	selectables := make([]Selectable, len(s.variables))
	for i, v := range s.variables {
		selectables[i] = v
	}
	mapVal, err := projectionsToMapValue(selectables)
	if err != nil {
		return nil, err
	}
	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    []*pb.Value{mapVal},
		Options: optionsPb,
	}, nil
}

// addFieldsStage is the internal representation of an AddFields stage.
type addFieldsStage struct {
	fields  []Selectable
	options map[string]any
}

func newAddFieldsStage(fields []Selectable, options map[string]any) (*addFieldsStage, error) {
	return &addFieldsStage{fields: fields, options: options}, nil
}
func (s *addFieldsStage) name() string { return stageNameAddFields }
func (s *addFieldsStage) toProto() (*pb.Pipeline_Stage, error) {
	mapVal, err := projectionsToMapValue(s.fields)
	if err != nil {
		return nil, err
	}
	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    []*pb.Value{mapVal},
		Options: optionsPb,
	}, nil
}

type aggregateStage struct {
	accumulators []*AliasedAggregate
	options      map[string]any
}

func newAggregateStage(accumulators []*AliasedAggregate, options map[string]any) (*aggregateStage, error) {
	if len(accumulators) == 0 {
		return nil, fmt.Errorf("firestore: the 'aggregate' stage requires at least one accumulator")
	}
	return &aggregateStage{accumulators: accumulators, options: options}, nil
}
func (s *aggregateStage) name() string { return stageNameAggregate }
func (s *aggregateStage) toProto() (*pb.Pipeline_Stage, error) {
	targetsPb, err := aliasedAggregatesToMapValue(s.accumulators)
	if err != nil {
		return nil, err
	}

	var groups []any
	if g, ok := s.options["groups"].([]any); ok {
		groups = g
	}
	selectables, err := fieldsOrSelectablesToSelectables(groups...)
	if err != nil {
		return nil, err
	}
	groupsPb, err := projectionsToMapValue(selectables)
	if err != nil {
		return nil, err
	}

	// Filter out 'groups' from options before converting to proto
	filteredOptions := make(map[string]any)
	for k, v := range s.options {
		if k != "groups" {
			filteredOptions[k] = v
		}
	}

	optionsPb, err := stageOptionsToProto(filteredOptions)
	if err != nil {
		return nil, err
	}

	return &pb.Pipeline_Stage{
		Name: s.name(),
		Args: []*pb.Value{
			targetsPb,
			groupsPb,
		},
		Options: optionsPb,
	}, nil
}

type distinctStage struct {
	fields  []any
	options map[string]any
}

func newDistinctStage(fields []any, options map[string]any) (*distinctStage, error) {
	return &distinctStage{fields: fields, options: options}, nil
}
func (s *distinctStage) name() string { return stageNameDistinct }
func (s *distinctStage) toProto() (*pb.Pipeline_Stage, error) {
	selectables, err := fieldsOrSelectablesToSelectables(s.fields...)
	if err != nil {
		return nil, err
	}
	mapVal, err := projectionsToMapValue(selectables)
	if err != nil {
		return nil, err
	}
	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    []*pb.Value{mapVal},
		Options: optionsPb,
	}, nil
}

type findNearestStage struct {
	vectorField any
	queryVector any
	measure     PipelineDistanceMeasure
	options     map[string]any
}

func newFindNearestStage(vectorField any, queryVector any, measure PipelineDistanceMeasure, options map[string]any) (*findNearestStage, error) {
	return &findNearestStage{
		vectorField: vectorField,
		queryVector: queryVector,
		measure:     measure,
		options:     options,
	}, nil
}
func (s *findNearestStage) name() string { return stageNameFindNearest }
func (s *findNearestStage) toProto() (*pb.Pipeline_Stage, error) {
	var propertyExpr Expression
	switch v := s.vectorField.(type) {
	case string:
		propertyExpr = FieldOf(v)
	case FieldPath:
		propertyExpr = FieldOf(v)
	case Expression:
		propertyExpr = v
	default:
		return nil, errInvalidArg("FindNearest", s.vectorField, "string", "FieldPath", "Expression")
	}
	if propertyExpr == nil {
		return nil, fmt.Errorf("firestore: internal error: findNearestStage vectorField resolved to nil expression")
	}
	propPb, err := propertyExpr.toProto()
	if err != nil {
		return nil, err
	}
	var vectorPb *pb.Value
	switch v := s.queryVector.(type) {
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
	measurePb := &pb.Value{ValueType: &pb.Value_StringValue{StringValue: string(s.measure)}}

	optsCopy := make(map[string]any)
	for k, v := range s.options {
		optsCopy[k] = v
	}

	optionsPb, err := stageOptionsToProto(optsCopy)
	if err != nil {
		return nil, err
	}

	// Correctly encode distance_field as FieldReferenceValue if it's a string
	if df, ok := optsCopy["distance_field"].(string); ok {
		if optionsPb == nil {
			optionsPb = make(map[string]*pb.Value)
		}
		optionsPb["distance_field"] = &pb.Value{ValueType: &pb.Value_FieldReferenceValue{FieldReferenceValue: df}}
	}

	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    []*pb.Value{propPb, vectorPb, measurePb},
		Options: optionsPb,
	}, nil
}

type limitStage struct {
	limit   int
	options map[string]any
}

func newLimitStage(limit int, options map[string]any) *limitStage {
	return &limitStage{limit: limit, options: options}
}
func (s *limitStage) name() string { return stageNameLimit }
func (s *limitStage) toProto() (*pb.Pipeline_Stage, error) {
	arg := &pb.Value{ValueType: &pb.Value_IntegerValue{IntegerValue: int64(s.limit)}}
	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    []*pb.Value{arg},
		Options: optionsPb,
	}, nil
}

type offsetStage struct {
	offset  int
	options map[string]any
}

func newOffsetStage(offset int, options map[string]any) *offsetStage {
	return &offsetStage{offset: offset, options: options}
}
func (s *offsetStage) name() string { return stageNameOffset }
func (s *offsetStage) toProto() (*pb.Pipeline_Stage, error) {
	arg := &pb.Value{ValueType: &pb.Value_IntegerValue{IntegerValue: int64(s.offset)}}
	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    []*pb.Value{arg},
		Options: optionsPb,
	}, nil
}

type removeFieldsStage struct {
	fields  []any
	options map[string]any
}

func newRemoveFieldsStage(fields []any, options map[string]any) (*removeFieldsStage, error) {
	return &removeFieldsStage{fields: fields, options: options}, nil
}
func (s *removeFieldsStage) name() string { return stageNameRemoveFields }
func (s *removeFieldsStage) toProto() (*pb.Pipeline_Stage, error) {
	fields := make([]Expression, len(s.fields))
	for i, fp := range s.fields {
		switch v := fp.(type) {
		case string:
			fields[i] = FieldOf(v)
		case FieldPath:
			fields[i] = FieldOf(v)
		case *field:
			fields[i] = v
		default:
			return nil, errInvalidArg("RemoveFields", fp, "string", "FieldPath", "expression obtained by calling FieldOf")
		}
	}
	args := make([]*pb.Value, len(fields))
	for i, f := range fields {
		if f == nil {
			return nil, fmt.Errorf("firestore: internal error: removeFieldsStage contains nil expression")
		}
		pb, err := f.toProto()
		if err != nil {
			return nil, err
		}
		args[i] = pb
	}
	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    args,
		Options: optionsPb,
	}, nil
}

type replaceWithStage struct {
	fieldpathOrExpr any
	options         map[string]any
}

func newReplaceWithStage(fieldpathOrExpr any, options map[string]any) (*replaceWithStage, error) {
	return &replaceWithStage{fieldpathOrExpr: fieldpathOrExpr, options: options}, nil
}
func (s *replaceWithStage) name() string { return stageNameReplaceWith }
func (s *replaceWithStage) toProto() (*pb.Pipeline_Stage, error) {
	var expr Expression
	switch v := s.fieldpathOrExpr.(type) {
	case string:
		expr = FieldOf(v)
	case FieldPath:
		expr = FieldOf(v)
	case Expression:
		expr = v
	default:
		return nil, errInvalidArg("ReplaceWith", s.fieldpathOrExpr, "string", "FieldPath", "Expression")
	}
	if expr == nil {
		return nil, fmt.Errorf("firestore: internal error: replaceWithStage fieldpathOrExpr resolved to nil expression")
	}
	exprPb, err := expr.toProto()
	if err != nil {
		return nil, err
	}
	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    []*pb.Value{exprPb, {ValueType: &pb.Value_StringValue{StringValue: "full_replace"}}},
		Options: optionsPb,
	}, nil
}

type sampleStage struct {
	sampler *Sampler
	options map[string]any
}

func newSampleStage(sampler *Sampler, options map[string]any) (*sampleStage, error) {
	return &sampleStage{sampler: sampler, options: options}, nil
}
func (s *sampleStage) name() string { return stageNameSample }
func (s *sampleStage) toProto() (*pb.Pipeline_Stage, error) {
	var sizePb *pb.Value
	switch v := s.sampler.Size.(type) {
	case int:
		sizePb = &pb.Value{ValueType: &pb.Value_IntegerValue{IntegerValue: int64(v)}}
	case int64:
		sizePb = &pb.Value{ValueType: &pb.Value_IntegerValue{IntegerValue: v}}
	case float64:
		sizePb = &pb.Value{ValueType: &pb.Value_DoubleValue{DoubleValue: v}}
	default:
		return nil, fmt.Errorf("firestore: invalid type for sample size: %T", s.sampler.Size)
	}
	modePb := &pb.Value{ValueType: &pb.Value_StringValue{StringValue: string(s.sampler.Mode)}}
	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    []*pb.Value{sizePb, modePb},
		Options: optionsPb,
	}, nil
}

type searchStage struct {
	options map[string]any
}

func newSearchStage(options map[string]any) (*searchStage, error) {
	return &searchStage{options: options}, nil
}
func (s *searchStage) name() string { return stageNameSearch }
func (s *searchStage) toProto() (*pb.Pipeline_Stage, error) {
	if len(s.options) == 0 {
		return &pb.Pipeline_Stage{
			Name: s.name(),
			Args: []*pb.Value{},
		}, nil
	}

	optionsPb := make(map[string]*pb.Value)

	for k, v := range s.options {
		switch k {
		case "query":
			var queryExpr Expression
			switch qv := v.(type) {
			case string:
				queryExpr = DocumentMatches(qv)
			case BooleanExpression:
				queryExpr = qv
			default:
				return nil, errInvalidArg("Search", v, "string", "BooleanExpression")
			}
			if queryExpr == nil {
				return nil, fmt.Errorf("firestore: internal error: searchStage query resolved to nil expression")
			}
			queryPb, err := queryExpr.toProto()
			if err != nil {
				return nil, err
			}
			optionsPb[k] = queryPb

		case "add_fields":
			fields, ok := v.([]Selectable)
			if !ok {
				return nil, fmt.Errorf("firestore: invalid type for Search add_fields: %T", v)
			}
			mapVal, err := projectionsToMapValue(fields)
			if err != nil {
				return nil, err
			}
			optionsPb[k] = mapVal

		case "sort":
			orders, ok := v.([]Ordering)
			if !ok {
				return nil, fmt.Errorf("firestore: invalid type for Search sort: %T", v)
			}
			sortVals := make([]*pb.Value, len(orders))
			for i, so := range orders {
				fieldPb, err := so.Expr.toProto()
				if err != nil {
					return nil, err
				}
				sortVals[i] = &pb.Value{
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
			optionsPb[k] = &pb.Value{
				ValueType: &pb.Value_ArrayValue{
					ArrayValue: &pb.ArrayValue{
						Values: sortVals,
					},
				},
			}

		case "retrieval_depth":
			switch depth := v.(type) {
			case int64:
				optionsPb[k] = &pb.Value{ValueType: &pb.Value_IntegerValue{IntegerValue: depth}}
			case int:
				optionsPb[k] = &pb.Value{ValueType: &pb.Value_IntegerValue{IntegerValue: int64(depth)}}
			default:
				return nil, fmt.Errorf("firestore: invalid type for Search retrieval_depth: %T", v)
			}

		case "limit":
			switch limit := v.(type) {
			case int64:
				optionsPb[k] = &pb.Value{ValueType: &pb.Value_IntegerValue{IntegerValue: limit}}
			case int:
				optionsPb[k] = &pb.Value{ValueType: &pb.Value_IntegerValue{IntegerValue: int64(limit)}}
			default:
				return nil, fmt.Errorf("firestore: invalid type for Search limit: %T", v)
			}

		case "offset":
			switch offset := v.(type) {
			case int64:
				optionsPb[k] = &pb.Value{ValueType: &pb.Value_IntegerValue{IntegerValue: offset}}
			case int:
				optionsPb[k] = &pb.Value{ValueType: &pb.Value_IntegerValue{IntegerValue: int64(offset)}}
			default:
				return nil, fmt.Errorf("firestore: invalid type for Search offset: %T", v)
			}

		case "language_code":
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("firestore: invalid type for Search language_code: %T", v)
			}
			optionsPb[k] = &pb.Value{ValueType: &pb.Value_StringValue{StringValue: s}}

		default:
			valPb, _, err := toProtoValue(reflect.ValueOf(v))
			if err != nil {
				return nil, fmt.Errorf("firestore: error converting stage option %q: %w", k, err)
			}
			optionsPb[k] = valPb
		}
	}

	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    []*pb.Value{},
		Options: optionsPb,
	}, nil
}

type selectStage struct {
	fields  []any
	options map[string]any
}

func newSelectStage(fields []any, options map[string]any) (*selectStage, error) {
	return &selectStage{fields: fields, options: options}, nil
}
func (s *selectStage) name() string { return stageNameSelect }
func (s *selectStage) toProto() (*pb.Pipeline_Stage, error) {
	selectables, err := fieldsOrSelectablesToSelectables(s.fields...)
	if err != nil {
		return nil, err
	}
	mapVal, err := projectionsToMapValue(selectables)
	if err != nil {
		return nil, err
	}
	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    []*pb.Value{mapVal},
		Options: optionsPb,
	}, nil
}

type sortStage struct {
	orders  []Ordering
	options map[string]any
}

func newSortStage(orders []Ordering, options map[string]any) *sortStage {
	return &sortStage{orders: orders, options: options}
}
func (s *sortStage) name() string { return stageNameSort }
func (s *sortStage) toProto() (*pb.Pipeline_Stage, error) {
	sortOrders := make([]*pb.Value, len(s.orders))
	for i, so := range s.orders {
		if so.Expr == nil {
			return nil, fmt.Errorf("firestore: internal error: sortStage contains ordering with nil expression")
		}
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
	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    sortOrders,
		Options: optionsPb,
	}, nil
}

type unionStage struct {
	other   *Pipeline
	options map[string]any
}

func newUnionStage(other *Pipeline, options map[string]any) (*unionStage, error) {
	return &unionStage{other: other, options: options}, nil
}
func (s *unionStage) name() string { return stageNameUnion }
func (s *unionStage) toProto() (*pb.Pipeline_Stage, error) {
	if s.other == nil {
		return nil, fmt.Errorf("firestore: internal error: unionStage contains nil pipeline")
	}
	otherPb, err := s.other.toProto()
	if err != nil {
		return nil, err
	}
	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name: s.name(),
		Args: []*pb.Value{
			{ValueType: &pb.Value_PipelineValue{PipelineValue: otherPb}},
		},
		Options: optionsPb,
	}, nil
}

type unnestStage struct {
	callerName string
	field      Selectable
	options    map[string]any
}

func newUnnestStage(callerName string, field Selectable, options map[string]any) (*unnestStage, error) {
	return &unnestStage{callerName: callerName, field: field, options: options}, nil
}
func (s *unnestStage) name() string { return stageNameUnnest }
func (s *unnestStage) toProto() (*pb.Pipeline_Stage, error) {
	if s.field == nil {
		return nil, fmt.Errorf("firestore: internal error: unnestStage contains nil field")
	}
	alias, expr := s.field.getSelectionDetails()
	exprPb, err := expr.toProto()
	if err != nil {
		return nil, err
	}
	aliasPb, err := FieldOf(alias).toProto()
	if err != nil {
		return nil, err
	}

	optsCopy := make(map[string]any)
	for k, v := range s.options {
		optsCopy[k] = v
	}

	var indexPb *pb.Value
	if idx, ok := optsCopy["index_field"]; ok {
		delete(optsCopy, "index_field")
		var indexFieldExpr Expression
		switch v := idx.(type) {
		case FieldPath:
			indexFieldExpr = FieldOf(v)
		case string:
			indexFieldExpr = FieldOf(v)
		default:
			return nil, errInvalidArg(s.callerName, idx, "string", "FieldPath")
		}
		if indexFieldExpr != nil {
			var err error
			indexPb, err = indexFieldExpr.toProto()
			if err != nil {
				return nil, err
			}
		}
	}

	optionsPb, err := stageOptionsToProto(optsCopy)
	if err != nil {
		return nil, err
	}
	if indexPb != nil {
		if optionsPb == nil {
			optionsPb = make(map[string]*pb.Value)
		}
		optionsPb["index_field"] = indexPb
	}

	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    []*pb.Value{exprPb, aliasPb},
		Options: optionsPb,
	}, nil
}

type whereStage struct {
	condition BooleanExpression
	options   map[string]any
}

func newWhereStage(condition BooleanExpression, options map[string]any) (*whereStage, error) {
	return &whereStage{condition: condition, options: options}, nil
}
func (s *whereStage) name() string { return stageNameWhere }
func (s *whereStage) toProto() (*pb.Pipeline_Stage, error) {
	if s.condition == nil {
		return nil, fmt.Errorf("firestore: internal error: whereStage condition is nil")
	}
	argsPb, err := s.condition.toProto()
	if err != nil {
		return nil, err
	}
	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    []*pb.Value{argsPb},
		Options: optionsPb,
	}, nil
}

type rawStage struct {
	stageName string
	args      []any
	options   map[string]any
}

func (s *rawStage) name() string { return s.stageName }

func (s *rawStage) toProto() (*pb.Pipeline_Stage, error) {
	argsPb := make([]*pb.Value, len(s.args))
	for i, arg := range s.args {
		val, _, err := toProtoValue(reflect.ValueOf(arg))
		if err != nil {
			return nil, fmt.Errorf("firestore: error converting raw stage argument %d: %w", i, err)
		}
		argsPb[i] = val
	}

	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}

	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    argsPb,
		Options: optionsPb,
	}, nil
}

type updateStage struct {
	options map[string]any
}

func newUpdateStage(options map[string]any) (*updateStage, error) {
	return &updateStage{options: options}, nil
}

func (s *updateStage) name() string { return stageNameUpdate }

func (s *updateStage) toProto() (*pb.Pipeline_Stage, error) {
	var mapVal *pb.Value
	var fields []Selectable

	optsCopy := make(map[string]any)
	for k, v := range s.options {
		optsCopy[k] = v
	}

	if t, ok := optsCopy["transformations"].([]Selectable); ok {
		fields = t
		delete(optsCopy, "transformations")
	}

	if len(fields) > 0 {
		var err error
		mapVal, err = projectionsToMapValue(fields)
		if err != nil {
			return nil, err
		}
	} else {
		mapVal = &pb.Value{ValueType: &pb.Value_MapValue{MapValue: &pb.MapValue{}}}
	}

	optionsPb, err := stageOptionsToProto(optsCopy)
	if err != nil {
		return nil, err
	}

	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    []*pb.Value{mapVal},
		Options: optionsPb,
	}, nil
}

type deleteStage struct {
	options map[string]any
}

func newDeleteStage(options map[string]any) *deleteStage {
	return &deleteStage{options: options}
}

func (s *deleteStage) name() string { return stageNameDelete }

func (s *deleteStage) toProto() (*pb.Pipeline_Stage, error) {
	optionsPb, err := stageOptionsToProto(s.options)
	if err != nil {
		return nil, err
	}
	return &pb.Pipeline_Stage{
		Name:    s.name(),
		Args:    []*pb.Value{},
		Options: optionsPb,
	}, nil
}
