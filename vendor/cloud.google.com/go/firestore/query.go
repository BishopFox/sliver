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
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"time"

	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
	"cloud.google.com/go/internal/btree"
	"cloud.google.com/go/internal/protostruct"
	"cloud.google.com/go/internal/trace"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var (
	errMetricsBeforeEnd     = errors.New("firestore: ExplainMetrics are available only after the iterator reaches the end")
	errInvalidVector        = errors.New("firestore: queryVector must be Vector32 or Vector64")
	errMalformedVectorQuery = errors.New("firestore: Malformed VectorQuery. Use FindNearest or FindNearestPath to create VectorQuery")
)

func errInvalidRunesField(field string) error {
	return fmt.Errorf("firestore: %q contains an invalid rune (one of %s)", field, invalidRunes)
}

// Query represents a Firestore query.
//
// Query values are immutable. Each Query method creates
// a new Query; it does not modify the old.
type Query struct {
	c                  *Client
	path               string // path to query (collection)
	parentPath         string // path of the collection's parent (document)
	collectionID       string
	selection          []*pb.StructuredQuery_FieldReference
	filters            []*pb.StructuredQuery_Filter
	orders             []order
	offset             int32
	limit              *wrapperspb.Int32Value
	limitToLast        bool
	startVals, endVals []interface{}
	startDoc, endDoc   *DocumentSnapshot

	// Set startBefore to true when doc in startVals needs to be included in result
	// Set endBefore to false when doc in endVals needs to be included in result
	startBefore, endBefore bool
	err                    error

	// allDescendants indicates whether this query is for all collections
	// that match the ID under the specified parentPath.
	allDescendants bool

	// readOptions specifies constraints for reading results from the query
	// e.g. read time
	readSettings *readSettings

	// readOptions specifies constraints for running the query
	// e.g. explainOptions
	runQuerySettings *runQuerySettings

	findNearest *pb.StructuredQuery_FindNearest
}

// ExplainMetrics represents explain metrics for the query.
type ExplainMetrics struct {

	// Planning phase information for the query.
	PlanSummary *PlanSummary
	// Aggregated stats from the execution of the query. Only present when
	// ExplainOptions.analyze is set to true
	ExecutionStats *ExecutionStats
}

// PlanSummary represents planning phase information for the query.
type PlanSummary struct {
	// The indexes selected for the query. For example:
	//
	//	[
	//	  {"query_scope": "Collection", "properties": "(foo ASC, __name__ ASC)"},
	//	  {"query_scope": "Collection", "properties": "(bar ASC, __name__ ASC)"}
	//	]
	IndexesUsed []*map[string]any
}

// ExecutionStats represents execution statistics for the query.
type ExecutionStats struct {
	// Total number of results returned, including documents, projections,
	// aggregation results, keys.
	ResultsReturned int64
	// Total time to execute the query in the backend.
	ExecutionDuration *time.Duration
	// Total billable read operations.
	ReadOperations int64
	// Debugging statistics from the execution of the query. Note that the
	// debugging stats are subject to change as Firestore evolves. It could
	// include:
	//
	//	{
	//	  "indexes_entries_scanned": "1000",
	//	  "documents_scanned": "20",
	//	  "billing_details" : {
	//	     "documents_billable": "20",
	//	     "index_entries_billable": "1000",
	//	     "min_query_cost": "0"
	//	  }
	//	}
	DebugStats *map[string]any
}

func fromExplainMetricsProto(pbExplainMetrics *pb.ExplainMetrics) *ExplainMetrics {
	if pbExplainMetrics == nil {
		return nil
	}
	return &ExplainMetrics{
		PlanSummary:    fromPlanSummaryProto(pbExplainMetrics.PlanSummary),
		ExecutionStats: fromExecutionStatsProto(pbExplainMetrics.ExecutionStats),
	}
}

func fromPlanSummaryProto(pbPlanSummary *pb.PlanSummary) *PlanSummary {
	if pbPlanSummary == nil {
		return nil
	}

	planSummary := &PlanSummary{}
	indexesUsed := []*map[string]any{}
	for _, pbIndexUsed := range pbPlanSummary.GetIndexesUsed() {
		indexUsed := protostruct.DecodeToMap(pbIndexUsed)
		indexesUsed = append(indexesUsed, &indexUsed)
	}

	planSummary.IndexesUsed = indexesUsed
	return planSummary
}

func fromExecutionStatsProto(pbstats *pb.ExecutionStats) *ExecutionStats {
	if pbstats == nil {
		return nil
	}

	executionStats := &ExecutionStats{
		ResultsReturned: pbstats.GetResultsReturned(),
		ReadOperations:  pbstats.GetReadOperations(),
	}

	executionDuration := pbstats.GetExecutionDuration().AsDuration()
	executionStats.ExecutionDuration = &executionDuration

	debugStats := protostruct.DecodeToMap(pbstats.GetDebugStats())
	executionStats.DebugStats = &debugStats

	return executionStats
}

// DocumentID is the special field name representing the ID of a document
// in queries.
const DocumentID = "__name__"

// Select returns a new Query that specifies the paths
// to return from the result documents.
// Each path argument can be a single field or a dot-separated sequence of
// fields, and must not contain any of the runes "˜*/[]".
//
// An empty Select call will produce a query that returns only document IDs.
func (q Query) Select(paths ...string) Query {
	var fps []FieldPath
	for _, s := range paths {
		fp, err := parseDotSeparatedString(s)
		if err != nil {
			q.err = err
			return q
		}
		fps = append(fps, fp)
	}
	return q.SelectPaths(fps...)
}

// SelectPaths returns a new Query that specifies the field paths
// to return from the result documents.
//
// An empty SelectPaths call will produce a query that returns only document IDs.
func (q Query) SelectPaths(fieldPaths ...FieldPath) Query {

	if len(fieldPaths) == 0 {
		ref, err := fref(FieldPath{DocumentID})
		if err != nil {
			q.err = err
			return q
		}
		q.selection = []*pb.StructuredQuery_FieldReference{
			ref,
		}
	} else {
		q.selection = make([]*pb.StructuredQuery_FieldReference, len(fieldPaths))
		for i, fieldPath := range fieldPaths {
			ref, err := fref(fieldPath)
			if err != nil {
				q.err = err
				return q
			}
			q.selection[i] = ref
		}
	}
	return q
}

// Where returns a new Query that filters the set of results.
// A Query can have multiple filters.
// The path argument can be a single field or a dot-separated sequence of
// fields, and must not contain any of the runes "˜*/[]".
// The op argument must be one of "==", "!=", "<", "<=", ">", ">=",
// "array-contains", "array-contains-any", "in" or "not-in".
// WARNING: Using WhereEntity with Simple and Composite filters is recommended.
func (q Query) Where(path, op string, value interface{}) Query {
	fp, err := parseDotSeparatedString(path)
	if err != nil {
		q.err = err
		return q
	}
	return q.WherePath(fp, op, value)
}

// WherePath returns a new Query that filters the set of results.
// A Query can have multiple filters.
// The op argument must be one of "==", "!=", "<", "<=", ">", ">=",
// "array-contains", "array-contains-any", "in" or "not-in".
// WARNING: Using WhereEntity with Simple and Composite filters is recommended.
func (q Query) WherePath(fp FieldPath, op string, value interface{}) Query {
	return q.WhereEntity(PropertyPathFilter{
		Path:     fp,
		Operator: op,
		Value:    value,
	})
}

// WhereEntity returns a query with provided filter.
//
// EntityFilter can be a simple filter or a composite filter
// PropertyFilter and PropertyPathFilter are supported simple filters
// AndFilter and OrFilter are supported composite filters
// Entity filters in multiple calls are joined together by AND
func (q Query) WhereEntity(ef EntityFilter) Query {
	proto, err := ef.toProto()
	if err != nil {
		q.err = err
		return q
	}
	q.filters = append(append([]*pb.StructuredQuery_Filter(nil), q.filters...), proto)
	return q
}

// Direction is the sort direction for result ordering.
type Direction int32

const (
	// Asc sorts results from smallest to largest.
	Asc Direction = Direction(pb.StructuredQuery_ASCENDING)

	// Desc sorts results from largest to smallest.
	Desc Direction = Direction(pb.StructuredQuery_DESCENDING)
)

// OrderBy returns a new Query that specifies the order in which results are
// returned. A Query can have multiple OrderBy/OrderByPath specifications.
// OrderBy appends the specification to the list of existing ones.
//
// The path argument can be a single field or a dot-separated sequence of
// fields, and must not contain any of the runes "˜*/[]".
//
// To order by document name, use the special field path DocumentID.
func (q Query) OrderBy(path string, dir Direction) Query {
	fp, err := parseDotSeparatedString(path)
	if err != nil {
		q.err = err
		return q
	}
	q.orders = append(q.copyOrders(), order{fieldPath: fp, dir: dir})
	return q
}

// OrderByPath returns a new Query that specifies the order in which results are
// returned. A Query can have multiple OrderBy/OrderByPath specifications.
// OrderByPath appends the specification to the list of existing ones.
func (q Query) OrderByPath(fp FieldPath, dir Direction) Query {
	q.orders = append(q.copyOrders(), order{fieldPath: fp, dir: dir})
	return q
}

func (q *Query) copyOrders() []order {
	return append([]order(nil), q.orders...)
}

// Offset returns a new Query that specifies the number of initial results to skip.
// It must not be negative.
func (q Query) Offset(n int) Query {
	q.offset = trunc32(n)
	return q
}

// Limit returns a new Query that specifies the maximum number of first results
// to return. It must not be negative.
func (q Query) Limit(n int) Query {
	q.limit = &wrapperspb.Int32Value{Value: trunc32(n)}
	q.limitToLast = false
	return q
}

// LimitToLast returns a new Query that specifies the maximum number of last
// results to return. It must not be negative.
func (q Query) LimitToLast(n int) Query {
	q.limit = &wrapperspb.Int32Value{Value: trunc32(n)}
	q.limitToLast = true
	return q
}

// StartAt returns a new Query that specifies that results should start at
// the document with the given field values.
//
// StartAt may be called with a single DocumentSnapshot, representing an
// existing document within the query. The document must be a direct child of
// the location being queried (not a parent document, or document in a
// different collection, or a grandchild document, for example).
//
// Otherwise, StartAt should be called with one field value for each OrderBy clause,
// in the order that they appear. For example, in
//
//	q.OrderBy("X", Asc).OrderBy("Y", Desc).StartAt(1, 2)
//
// results will begin at the first document where X = 1 and Y = 2.
//
// If an OrderBy call uses the special DocumentID field path, the corresponding value
// should be the document ID relative to the query's collection. For example, to
// start at the document "NewYork" in the "States" collection, write
//
//	client.Collection("States").OrderBy(DocumentID, firestore.Asc).StartAt("NewYork")
//
// Calling StartAt overrides a previous call to StartAt or StartAfter.
func (q Query) StartAt(docSnapshotOrFieldValues ...interface{}) Query {
	q.startBefore = true
	q.startVals, q.startDoc, q.err = q.processCursorArg("StartAt", docSnapshotOrFieldValues)
	return q
}

// StartAfter returns a new Query that specifies that results should start just after
// the document with the given field values. See Query.StartAt for more information.
//
// Calling StartAfter overrides a previous call to StartAt or StartAfter.
func (q Query) StartAfter(docSnapshotOrFieldValues ...interface{}) Query {
	q.startBefore = false
	q.startVals, q.startDoc, q.err = q.processCursorArg("StartAfter", docSnapshotOrFieldValues)
	return q
}

// EndAt returns a new Query that specifies that results should end at the
// document with the given field values. See Query.StartAt for more information.
//
// Calling EndAt overrides a previous call to EndAt or EndBefore.
func (q Query) EndAt(docSnapshotOrFieldValues ...interface{}) Query {
	q.endBefore = false
	q.endVals, q.endDoc, q.err = q.processCursorArg("EndAt", docSnapshotOrFieldValues)
	return q
}

// EndBefore returns a new Query that specifies that results should end just before
// the document with the given field values. See Query.StartAt for more information.
//
// Calling EndBefore overrides a previous call to EndAt or EndBefore.
func (q Query) EndBefore(docSnapshotOrFieldValues ...interface{}) Query {
	q.endBefore = true
	q.endVals, q.endDoc, q.err = q.processCursorArg("EndBefore", docSnapshotOrFieldValues)
	return q
}

// WithRunOptions allows passing options to the query
// Calling WithRunOptions overrides a previous call to WithRunOptions.
func (q Query) WithRunOptions(opts ...RunOption) Query {
	settings, err := newRunQuerySettings(opts)
	if err != nil {
		q.err = err
		return q
	}

	q.runQuerySettings = settings
	return q
}

func (q *Query) processCursorArg(name string, docSnapshotOrFieldValues []interface{}) ([]interface{}, *DocumentSnapshot, error) {
	for _, e := range docSnapshotOrFieldValues {
		if ds, ok := e.(*DocumentSnapshot); ok {
			if len(docSnapshotOrFieldValues) == 1 {
				return nil, ds, nil
			}
			return nil, nil, fmt.Errorf("firestore: a document snapshot must be the only argument to %s", name)
		}
	}
	return docSnapshotOrFieldValues, nil, nil
}

func (q *Query) processLimitToLast() {
	if q.limitToLast {
		// Firestore service does not provide limit to last behaviour out of the box. This is a client-side concept
		// So, flip order statements and cursors before posting a request. The response is flipped by other methods before returning to user
		// E.g.
		// If id of documents is 1, 2, 3, 4, 5, 6, 7 and query is (OrderBy(id, ASC), StartAt(2), EndAt(6), LimitToLast(3))
		// request sent to server is  (OrderBy(id, DESC), StartAt(6), EndAt(2), Limit(3))
		for i := range q.orders {
			if q.orders[i].dir == Asc {
				q.orders[i].dir = Desc
			} else {
				q.orders[i].dir = Asc
			}
		}

		if q.startBefore == q.endBefore && q.startCursorSpecified() && q.endCursorSpecified() {
			// E.g. query.StartAt(2).EndBefore(6).LimitToLast(3).OrderBy(Asc) i.e. cursors are [2, 6)
			// E.g. query.StartAfter(2).EndAt(6).LimitToLast(3).OrderBy(Asc)  i.e. cursors are (2, 6]
			q.startBefore, q.endBefore = !q.startBefore, !q.endBefore
		} else if !q.startCursorSpecified() && q.endCursorSpecified() {
			// E.g. query.EndAt(6).LimitToLast(3).OrderBy(Asc) i.e. cursors are (-inf, 6]
			q.startBefore = !q.endBefore
			q.endBefore = false
		} else if q.startCursorSpecified() && !q.endCursorSpecified() {
			// E.g. query.StartAt(2).LimitToLast(3).OrderBy(Asc) i.e. cursors are [2, inf)
			q.endBefore = !q.startBefore
			q.startBefore = false
		}

		// Swap cursors.
		q.startVals, q.endVals = q.endVals, q.startVals
		q.startDoc, q.endDoc = q.endDoc, q.startDoc

		q.limitToLast = false
	}
}

func (q Query) query() *Query { return &q }

// Serialize creates a RunQueryRequest wire-format byte slice from a Query object.
// This can be used in combination with Deserialize to marshal Query objects.
// This could be useful, for instance, if executing a query formed in one
// process in another.
func (q Query) Serialize() ([]byte, error) {
	req, err := q.toRunQueryRequestProto()
	if err != nil {
		return nil, err
	}
	return proto.Marshal(req)
}

// Deserialize takes a slice of bytes holding the wire-format message of RunQueryRequest,
// the underlying proto message used by Queries. It then populates and returns a
// Query object that can be used to execute that Query.
func (q Query) Deserialize(bytes []byte) (Query, error) {
	runQueryRequest := pb.RunQueryRequest{}
	err := proto.Unmarshal(bytes, &runQueryRequest)
	if err != nil {
		q.err = err
		return q, err
	}
	return q.fromProto(&runQueryRequest)
}

func (q Query) toRunQueryRequestProto() (*pb.RunQueryRequest, error) {
	structuredQuery, err := q.toProto()
	if err != nil {
		return nil, err
	}

	var explainOptions *pb.ExplainOptions
	if q.runQuerySettings != nil && q.runQuerySettings.explainOptions != nil {
		explainOptions = q.runQuerySettings.explainOptions
	}
	p := &pb.RunQueryRequest{
		Parent:         q.parentPath,
		ExplainOptions: explainOptions,
		QueryType:      &pb.RunQueryRequest_StructuredQuery{StructuredQuery: structuredQuery},
	}
	return p, nil
}

// DistanceMeasure is the distance measure to use when comparing vectors with [Query.FindNearest] or [Query.FindNearestPath].
type DistanceMeasure int32

const (
	// DistanceMeasureEuclidean is used to measures the Euclidean distance between the vectors. See
	// [Euclidean] to learn more.
	//
	// [Euclidean]: https://en.wikipedia.org/wiki/Euclidean_distance
	DistanceMeasureEuclidean DistanceMeasure = DistanceMeasure(pb.StructuredQuery_FindNearest_EUCLIDEAN)

	// DistanceMeasureCosine compares vectors based on the angle between them, which allows you to
	// measure similarity that isn't based on the vectors magnitude.
	// We recommend using dot product with unit normalized vectors instead of
	// cosine distance, which is mathematically equivalent with better
	// performance. See [Cosine Similarity] to learn more.
	//
	// [Cosine Similarity]: https://en.wikipedia.org/wiki/Cosine_similarity
	DistanceMeasureCosine DistanceMeasure = DistanceMeasure(pb.StructuredQuery_FindNearest_COSINE)

	// DistanceMeasureDotProduct is similar to cosine but is affected by the magnitude of the vectors. See
	// [Dot Product] to learn more.
	//
	// [Dot Product]: https://en.wikipedia.org/wiki/Dot_product
	DistanceMeasureDotProduct DistanceMeasure = DistanceMeasure(pb.StructuredQuery_FindNearest_DOT_PRODUCT)
)

// Ptr returns a pointer to its argument.
// It can be used to initialize pointer fields:
//
//	findNearestOptions.DistanceThreshold = firestore.Ptr[float64](0.1)
func Ptr[T any](t T) *T { return &t }

// FindNearestOptions are options for a FindNearest vector query.
type FindNearestOptions struct {
	// DistanceThreshold specifies a threshold for which no less similar documents
	// will be returned. The behavior of the specified [DistanceMeasure] will
	// affect the meaning of the distance threshold. Since [DistanceMeasureDotProduct]
	// distances increase when the vectors are more similar, the comparison is inverted.
	// For [DistanceMeasureEuclidean], [DistanceMeasureCosine]: WHERE distance <= distanceThreshold
	// For [DistanceMeasureDotProduct]:                         WHERE distance >= distance_threshold
	DistanceThreshold *float64

	// DistanceResultField specifies name of the document field to output the result of
	// the vector distance calculation.
	// If the field already exists in the document, its value get overwritten with the distance calculation.
	// Otherwise, a new field gets added to the document.
	DistanceResultField string
}

// VectorQuery represents a query that uses [Query.FindNearest] or [Query.FindNearestPath].
type VectorQuery struct {
	q Query
}

// FindNearest returns a query that can perform vector distance (similarity) search.
//
// The returned query, when executed, performs a distance search on the specified
// vectorField against the given queryVector and returns the top documents that are closest
// to the queryVector according to measure. At most limit documents are returned.
//
// Only documents whose vectorField field is a Vector32 or Vector64 of the same dimension
// as queryVector participate in the query; all other documents are ignored.
// In particular, fields of type []float32 or []float64 are ignored.
//
// The vectorField argument can be a single field or a dot-separated sequence of
// fields, and must not contain any of the runes "˜*/[]".
//
// The queryVector argument can be any of the following types:
//   - []float32
//   - []float64
//   - Vector32
//   - Vector64
func (q Query) FindNearest(vectorField string, queryVector any, limit int, measure DistanceMeasure, options *FindNearestOptions) VectorQuery {
	// Validate field path
	fieldPath, err := parseDotSeparatedString(vectorField)
	if err != nil {
		q.err = err
		return VectorQuery{q: q}
	}
	return q.FindNearestPath(fieldPath, queryVector, limit, measure, options)
}

// Documents returns an iterator over the vector query's resulting documents.
func (vq VectorQuery) Documents(ctx context.Context) *DocumentIterator {
	return vq.q.Documents(ctx)
}

// FindNearestPath is like [Query.FindNearest] but it accepts a [FieldPath].
func (q Query) FindNearestPath(vectorFieldPath FieldPath, queryVector any, limit int, measure DistanceMeasure, options *FindNearestOptions) VectorQuery {
	vq := VectorQuery{q: q}

	// Convert field path to field reference
	vectorFieldRef, err := fref(vectorFieldPath)
	if err != nil {
		vq.q.err = err
		return vq
	}

	var fnvq *pb.Value
	switch v := queryVector.(type) {
	case Vector32:
		fnvq = vectorToProtoValue([]float32(v))
	case []float32:
		fnvq = vectorToProtoValue(v)
	case Vector64:
		fnvq = vectorToProtoValue([]float64(v))
	case []float64:
		fnvq = vectorToProtoValue(v)
	default:
		vq.q.err = errInvalidVector
		return vq
	}

	vq.q.findNearest = &pb.StructuredQuery_FindNearest{
		VectorField:     vectorFieldRef,
		QueryVector:     fnvq,
		Limit:           &wrapperspb.Int32Value{Value: trunc32(limit)},
		DistanceMeasure: pb.StructuredQuery_FindNearest_DistanceMeasure(measure),
	}

	if options != nil {
		if options.DistanceThreshold != nil {
			vq.q.findNearest.DistanceThreshold = &wrapperspb.DoubleValue{Value: *options.DistanceThreshold}
		}
		vq.q.findNearest.DistanceResultField = *&options.DistanceResultField
	}
	return vq
}

// NewAggregationQuery returns an AggregationQuery with this query as its
// base query.
func (q *Query) NewAggregationQuery() *AggregationQuery {
	return &AggregationQuery{
		query: q,
	}
}

// fromProto creates a new Query object from a RunQueryRequest. This can be used
// in combination with ToProto to serialize Query objects. This could be useful,
// for instance, if executing a query formed in one process in another.
func (q Query) fromProto(pbQuery *pb.RunQueryRequest) (Query, error) {
	// Ensure we are starting from an empty query, but with this client.
	q = Query{c: q.c}

	pbq := pbQuery.GetStructuredQuery()
	if from := pbq.GetFrom(); len(from) > 0 {
		if len(from) > 1 {
			err := errors.New("can only deserialize query with exactly one collection selector")
			q.err = err
			return q, err
		}

		// collectionID           string
		q.collectionID = from[0].CollectionId
		// allDescendants indicates whether this query is for all collections
		// that match the ID under the specified parentPath.
		q.allDescendants = from[0].AllDescendants
	}

	// 	path                   string // path to query (collection)
	// 	parentPath             string // path of the collection's parent (document)
	parent := pbQuery.GetParent()
	q.parentPath = parent
	q.path = parent + "/" + q.collectionID

	// 	startVals, endVals     []interface{}
	// 	startDoc, endDoc       *DocumentSnapshot
	// 	startBefore, endBefore bool
	if startAt := pbq.GetStartAt(); startAt != nil {
		if startAt.GetBefore() {
			q.startBefore = true
		}
		for _, v := range startAt.GetValues() {
			c, err := createFromProtoValue(v, q.c)
			if err != nil {
				q.err = err
				return q, err
			}

			var newQ Query
			if startAt.GetBefore() {
				newQ = q.StartAt(c)
			} else {
				newQ = q.StartAfter(c)
			}

			q.startVals = append(q.startVals, newQ.startVals...)
		}
	}
	if endAt := pbq.GetEndAt(); endAt != nil {
		for _, v := range endAt.GetValues() {
			c, err := createFromProtoValue(v, q.c)

			if err != nil {
				q.err = err
				return q, err
			}

			var newQ Query
			if endAt.GetBefore() {
				newQ = q.EndBefore(c)
				q.endBefore = true
			} else {
				newQ = q.EndAt(c)
			}
			q.endVals = append(q.endVals, newQ.endVals...)

		}
	}

	// 	selection              []*pb.StructuredQuery_FieldReference
	if s := pbq.GetSelect(); s != nil {
		q.selection = s.GetFields()
	}

	// 	filters                []*pb.StructuredQuery_Filter
	if w := pbq.GetWhere(); w != nil {
		if cf := w.GetCompositeFilter(); cf != nil && cf.Op == pb.StructuredQuery_CompositeFilter_AND {
			q.filters = cf.GetFilters()
		} else {
			q.filters = []*pb.StructuredQuery_Filter{w}
		}
	}

	// 	orders                 []order
	if orderBy := pbq.GetOrderBy(); orderBy != nil {
		for _, v := range orderBy {
			fp := v.GetField()
			q.orders = append(q.orders, order{fieldReference: fp, dir: Direction(v.GetDirection())})
		}
	}

	// 	offset                 int32
	q.offset = pbq.GetOffset()

	// 	limit                  *wrapperspb.Int32Value
	if limit := pbq.GetLimit(); limit != nil {
		q.limit = limit
	}

	var err error
	q.runQuerySettings, err = newRunQuerySettings(nil)
	if err != nil {
		q.err = err
		return q, q.err
	}
	q.runQuerySettings.explainOptions = pbQuery.GetExplainOptions()
	q.findNearest = pbq.GetFindNearest()

	// NOTE: limit to last isn't part of the proto, this is a client-side concept
	// 	limitToLast            bool
	return q, q.err
}

func (q Query) startCursorSpecified() bool {
	return len(q.startVals) != 0 || q.startDoc != nil
}

func (q Query) endCursorSpecified() bool {
	return len(q.endVals) != 0 || q.endDoc != nil
}

func (q Query) toProto() (*pb.StructuredQuery, error) {
	if q.err != nil {
		return nil, q.err
	}
	if q.collectionID == "" {
		return nil, errors.New("firestore: query created without CollectionRef")
	}
	if q.startBefore {
		if !q.startCursorSpecified() {
			return nil, errors.New("firestore: StartAt/StartAfter must be called with at least one value")
		}
	}
	if q.endBefore {
		if !q.endCursorSpecified() {
			return nil, errors.New("firestore: EndAt/EndBefore must be called with at least one value")
		}
	}
	p := &pb.StructuredQuery{
		From: []*pb.StructuredQuery_CollectionSelector{{
			CollectionId:   q.collectionID,
			AllDescendants: q.allDescendants,
		}},
		Offset: q.offset,
		Limit:  q.limit,
	}
	if len(q.selection) > 0 {
		p.Select = &pb.StructuredQuery_Projection{}
		p.Select.Fields = q.selection
	}
	// If there is only filter, use it directly. Otherwise, construct
	// a CompositeFilter.
	if len(q.filters) == 1 {
		pf := q.filters[0]

		p.Where = pf
	} else if len(q.filters) > 1 {
		cf := &pb.StructuredQuery_CompositeFilter{
			Op: pb.StructuredQuery_CompositeFilter_AND,
		}
		p.Where = &pb.StructuredQuery_Filter{
			FilterType: &pb.StructuredQuery_Filter_CompositeFilter{
				CompositeFilter: cf,
			},
		}
		cf.Filters = append(cf.Filters, q.filters...)
	}
	orders := q.orders
	if q.startDoc != nil || q.endDoc != nil {
		orders = q.adjustOrders()
	}
	for _, ord := range orders {
		po, err := ord.toProto()
		if err != nil {
			return nil, err
		}
		p.OrderBy = append(p.OrderBy, po)
	}

	cursor, err := q.toCursor(q.startVals, q.startDoc, q.startBefore, orders)
	if err != nil {
		return nil, err
	}
	p.StartAt = cursor
	cursor, err = q.toCursor(q.endVals, q.endDoc, q.endBefore, orders)
	if err != nil {
		return nil, err
	}
	p.EndAt = cursor
	p.FindNearest = q.findNearest
	return p, nil
}

// If there is a start/end that uses a Document Snapshot, we may need to adjust the OrderBy
// clauses that the user provided: we add OrderBy(__name__) if it isn't already present, and
// we make sure we don't invalidate the original query by adding an OrderBy for inequality filters.
func (q *Query) adjustOrders() []order {
	// If the user is already ordering by document ID, don't change anything.
	for _, ord := range q.orders {
		if ord.isDocumentID() {
			return q.orders
		}
	}
	// If there are OrderBy clauses, append an OrderBy(DocumentID), using the direction of the last OrderBy clause.
	if len(q.orders) > 0 {
		return append(q.copyOrders(), order{
			fieldPath: FieldPath{DocumentID},
			dir:       q.orders[len(q.orders)-1].dir,
		})
	}
	// If there are no OrderBy clauses but there is an inequality, add an OrderBy clause
	// for the field of the first inequality.
	var orders []order
	for _, f := range q.filters {
		if fieldFilter := f.GetFieldFilter(); fieldFilter != nil {
			if fieldFilter.Op != pb.StructuredQuery_FieldFilter_EQUAL {
				fp := f.GetFieldFilter().Field
				orders = []order{{fieldReference: fp, dir: Asc}}
				break
			}
		}
	}
	// Add an ascending OrderBy(DocumentID).
	return append(orders, order{fieldPath: FieldPath{DocumentID}, dir: Asc})
}

func (q *Query) toCursor(fieldValues []interface{}, ds *DocumentSnapshot, before bool, orders []order) (*pb.Cursor, error) {
	var vals []*pb.Value
	var err error
	if ds != nil {
		vals, err = q.docSnapshotToCursorValues(ds, orders)
	} else if len(fieldValues) != 0 {
		vals, err = q.fieldValuesToCursorValues(fieldValues)
	} else {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pb.Cursor{Values: vals, Before: before}, nil
}

// toPositionValues converts the field values to protos.
func (q *Query) fieldValuesToCursorValues(fieldValues []interface{}) ([]*pb.Value, error) {
	if len(fieldValues) != len(q.orders) {
		return nil, errors.New("firestore: number of field values in StartAt/StartAfter/EndAt/EndBefore does not match number of OrderBy fields")
	}
	vals := make([]*pb.Value, len(fieldValues))
	var err error
	for i, ord := range q.orders {
		fval := fieldValues[i]
		if ord.isDocumentID() {
			// TODO(jba): error if document ref does not belong to the right collection.

			switch docID := fval.(type) {
			case string:
				vals[i] = &pb.Value{ValueType: &pb.Value_ReferenceValue{ReferenceValue: q.path + "/" + docID}}
				continue
			case *DocumentRef:
				// DocumentRef can be transformed in usual way.
			default:
				return nil, fmt.Errorf("firestore: expected doc ID for DocumentID field, got %T", fval)
			}
		}

		var sawTransform bool
		vals[i], sawTransform, err = toProtoValue(reflect.ValueOf(fval))
		if err != nil {
			return nil, err
		}
		if sawTransform {
			return nil, errors.New("firestore: transforms disallowed in query value")
		}
	}
	return vals, nil
}

func (q *Query) docSnapshotToCursorValues(ds *DocumentSnapshot, orders []order) ([]*pb.Value, error) {
	vals := make([]*pb.Value, len(orders))
	for i, ord := range orders {
		if ord.isDocumentID() {
			dp, qp := ds.Ref.Parent.Path, q.path
			if !q.allDescendants && dp != qp {
				return nil, fmt.Errorf("firestore: document snapshot for %s passed to query on %s", dp, qp)
			}
			vals[i] = &pb.Value{ValueType: &pb.Value_ReferenceValue{ReferenceValue: ds.Ref.Path}}
		} else {
			var val *pb.Value
			if len(ord.fieldPath) > 0 {
				var err error
				val, err = valueAtPath(ord.fieldPath, ds.proto.Fields)
				if err != nil {
					return nil, err
				}
			} else {
				// parse the field reference field path so we can use it to look up
				fp, err := parseDotSeparatedString(ord.fieldReference.FieldPath)
				if err != nil {
					return nil, err
				}
				val, err = valueAtPath(fp, ds.proto.Fields)
				if err != nil {
					return nil, err
				}
			}
			vals[i] = val
		}
	}
	return vals, nil
}

// Returns a function that compares DocumentSnapshots according to q's ordering.
func (q Query) compareFunc() func(d1, d2 *DocumentSnapshot) (int, error) {
	// Add implicit sorting by name, using the last specified direction.
	lastDir := Asc
	if len(q.orders) > 0 {
		lastDir = q.orders[len(q.orders)-1].dir
	}
	orders := append(q.copyOrders(), order{fieldPath: []string{DocumentID}, dir: lastDir})
	return func(d1, d2 *DocumentSnapshot) (int, error) {
		for _, ord := range orders {
			var cmp int
			if ord.isDocumentID() {
				cmp = compareReferences(d1.Ref.Path, d2.Ref.Path)
			} else {
				v1, err := valueAtPath(ord.fieldPath, d1.proto.Fields)
				if err != nil {
					return 0, err
				}
				v2, err := valueAtPath(ord.fieldPath, d2.proto.Fields)
				if err != nil {
					return 0, err
				}
				cmp = compareValues(v1, v2)
			}
			if cmp != 0 {
				if ord.dir == Desc {
					cmp = -cmp
				}
				return cmp, nil
			}
		}
		return 0, nil
	}
}

// EntityFilter represents a Firestore filter.
type EntityFilter interface {
	toProto() (*pb.StructuredQuery_Filter, error)
}

// CompositeFilter represents a composite Firestore filter.
type CompositeFilter interface {
	EntityFilter
	isCompositeFilter()
}

// OrFilter represents a union of two or more filters.
type OrFilter struct {
	Filters []EntityFilter
}

func (OrFilter) isCompositeFilter() {}

func (f OrFilter) toProto() (*pb.StructuredQuery_Filter, error) {
	var pbFilters []*pb.StructuredQuery_Filter

	for _, filter := range f.Filters {
		pbFilter, err := filter.toProto()
		if err != nil {
			return nil, err
		}
		pbFilters = append(pbFilters, pbFilter)
	}

	cf := &pb.StructuredQuery_CompositeFilter{
		Op: pb.StructuredQuery_CompositeFilter_OR,
	}
	cf.Filters = append(cf.Filters, pbFilters...)

	return &pb.StructuredQuery_Filter{
		FilterType: &pb.StructuredQuery_Filter_CompositeFilter{
			CompositeFilter: cf,
		},
	}, nil

}

// AndFilter represents the intersection of two or more filters.
type AndFilter struct {
	Filters []EntityFilter
}

func (AndFilter) isCompositeFilter() {}

func (f AndFilter) toProto() (*pb.StructuredQuery_Filter, error) {
	var pbFilters []*pb.StructuredQuery_Filter

	for _, filter := range f.Filters {
		pbFilter, err := filter.toProto()
		if err != nil {
			return nil, err
		}
		pbFilters = append(pbFilters, pbFilter)
	}

	cf := &pb.StructuredQuery_CompositeFilter{
		Op: pb.StructuredQuery_CompositeFilter_AND,
	}
	cf.Filters = append(cf.Filters, pbFilters...)

	return &pb.StructuredQuery_Filter{
		FilterType: &pb.StructuredQuery_Filter_CompositeFilter{
			CompositeFilter: cf,
		},
	}, nil

}

// SimpleFilter represents a simple Firestore filter.
type SimpleFilter interface {
	EntityFilter
	isSimpleFilter()
}

// PropertyFilter represents a filter on single property.
//
// Path can be a single field or a dot-separated sequence of fields
// denoting property path, and must not contain any of the runes "˜*/[]".
// Operator must be one of "==", "!=", "<", "<=", ">", ">=",
// "array-contains", "array-contains-any", "in" or "not-in".
type PropertyFilter struct {
	Path     string
	Operator string
	Value    interface{}
}

func (PropertyFilter) isSimpleFilter() {}

func (f PropertyFilter) toPropertyPathFilter() (PropertyPathFilter, error) {
	fp, err := parseDotSeparatedString(f.Path)
	if err != nil {
		return PropertyPathFilter{}, err
	}

	ppf := PropertyPathFilter{
		Path:     fp,
		Operator: f.Operator,
		Value:    f.Value,
	}
	return ppf, nil
}

func (f PropertyFilter) toProto() (*pb.StructuredQuery_Filter, error) {
	ppf, err := f.toPropertyPathFilter()
	if err != nil {
		return nil, err
	}
	return ppf.toProto()
}

// PropertyPathFilter represents a filter on single property.
//
// Path can be an array of fields denoting property path.
// Operator must be one of "==", "!=", "<", "<=", ">", ">=",
// "array-contains", "array-contains-any", "in" or "not-in".
type PropertyPathFilter struct {
	Path     FieldPath
	Operator string
	Value    interface{}
}

func (PropertyPathFilter) isSimpleFilter() {}

func (f PropertyPathFilter) toProto() (*pb.StructuredQuery_Filter, error) {
	if err := f.Path.validate(); err != nil {
		return nil, err
	}
	if uop, ok := unaryOpFor(f.Value); ok {
		if f.Operator != "==" && !(f.Operator == "!=" && (f.Value == nil || isNaN(f.Value))) {
			return nil, fmt.Errorf("firestore: must use '==' or '!=' when comparing %v", f.Value)
		}
		ref, err := fref(f.Path)
		if err != nil {
			return nil, err
		}
		if f.Operator == "!=" {
			if uop == pb.StructuredQuery_UnaryFilter_IS_NULL {
				uop = pb.StructuredQuery_UnaryFilter_IS_NOT_NULL
			} else if uop == pb.StructuredQuery_UnaryFilter_IS_NAN {
				uop = pb.StructuredQuery_UnaryFilter_IS_NOT_NAN
			}
		}
		return &pb.StructuredQuery_Filter{
			FilterType: &pb.StructuredQuery_Filter_UnaryFilter{
				UnaryFilter: &pb.StructuredQuery_UnaryFilter{
					OperandType: &pb.StructuredQuery_UnaryFilter_Field{
						Field: ref,
					},
					Op: uop,
				},
			},
		}, nil
	}
	var op pb.StructuredQuery_FieldFilter_Operator
	switch f.Operator {
	case "<":
		op = pb.StructuredQuery_FieldFilter_LESS_THAN
	case "<=":
		op = pb.StructuredQuery_FieldFilter_LESS_THAN_OR_EQUAL
	case ">":
		op = pb.StructuredQuery_FieldFilter_GREATER_THAN
	case ">=":
		op = pb.StructuredQuery_FieldFilter_GREATER_THAN_OR_EQUAL
	case "==":
		op = pb.StructuredQuery_FieldFilter_EQUAL
	case "!=":
		op = pb.StructuredQuery_FieldFilter_NOT_EQUAL
	case "in":
		op = pb.StructuredQuery_FieldFilter_IN
	case "not-in":
		op = pb.StructuredQuery_FieldFilter_NOT_IN
	case "array-contains":
		op = pb.StructuredQuery_FieldFilter_ARRAY_CONTAINS
	case "array-contains-any":
		op = pb.StructuredQuery_FieldFilter_ARRAY_CONTAINS_ANY
	default:
		return nil, fmt.Errorf("firestore: invalid operator %q", f.Operator)
	}
	val, sawTransform, err := toProtoValue(reflect.ValueOf(f.Value))
	if err != nil {
		return nil, err
	}
	if sawTransform {
		return nil, errors.New("firestore: transforms disallowed in query value")
	}
	ref, err := fref(f.Path)
	if err != nil {
		return nil, err
	}
	return &pb.StructuredQuery_Filter{
		FilterType: &pb.StructuredQuery_Filter_FieldFilter{
			FieldFilter: &pb.StructuredQuery_FieldFilter{
				Field: ref,
				Op:    op,
				Value: val,
			},
		},
	}, nil
}

func unaryOpFor(value interface{}) (pb.StructuredQuery_UnaryFilter_Operator, bool) {
	switch {
	case value == nil:
		return pb.StructuredQuery_UnaryFilter_IS_NULL, true
	case isNaN(value):
		return pb.StructuredQuery_UnaryFilter_IS_NAN, true
	default:
		return pb.StructuredQuery_UnaryFilter_OPERATOR_UNSPECIFIED, false
	}
}

func isNaN(x interface{}) bool {
	switch x := x.(type) {
	case float32:
		return math.IsNaN(float64(x))
	case float64:
		return math.IsNaN(x)
	default:
		return false
	}
}

type order struct {
	fieldPath      FieldPath
	fieldReference *pb.StructuredQuery_FieldReference
	dir            Direction
}

func (r order) isDocumentID() bool {
	if r.fieldReference != nil {
		return r.fieldReference.GetFieldPath() == DocumentID
	}
	return len(r.fieldPath) == 1 && r.fieldPath[0] == DocumentID
}

func (r order) toProto() (*pb.StructuredQuery_Order, error) {
	if r.fieldReference != nil {
		return &pb.StructuredQuery_Order{
			Field:     r.fieldReference,
			Direction: pb.StructuredQuery_Direction(r.dir),
		}, nil
	}

	field, err := fref(r.fieldPath)
	if err != nil {
		return nil, err
	}

	return &pb.StructuredQuery_Order{
		Field:     field,
		Direction: pb.StructuredQuery_Direction(r.dir),
	}, nil
}

func fref(fp FieldPath) (*pb.StructuredQuery_FieldReference, error) {
	err := fp.validate()
	if err != nil {
		return &pb.StructuredQuery_FieldReference{}, err
	}
	return &pb.StructuredQuery_FieldReference{FieldPath: fp.toServiceFieldPath()}, nil
}

func trunc32(i int) int32 {
	if i > math.MaxInt32 {
		i = math.MaxInt32
	}
	return int32(i)
}

// Documents returns an iterator over the query's resulting documents.
func (q Query) Documents(ctx context.Context) *DocumentIterator {
	return &DocumentIterator{
		iter: newQueryDocumentIterator(withResourceHeader(ctx, q.c.path()), &q, nil, q.readSettings), q: &q,
	}
}

// DocumentIterator is an iterator over documents returned by a query.
type DocumentIterator struct {
	iter docIterator
	err  error
	q    *Query
}

// Unexported interface so we can have two different kinds of DocumentIterator: one
// for straight queries, and one for query snapshots. We do it this way instead of
// making DocumentIterator an interface because in the client libraries, iterators are
// always concrete types, and the fact that this one has two different implementations
// is an internal detail.
type docIterator interface {
	next() (*DocumentSnapshot, error)
	getExplainMetrics() (*ExplainMetrics, error)
	stop()
}

// ExplainMetrics returns query explain metrics.
// This is only present when [ExplainOptions] is added to the query
// (see [Query.WithRunOptions]), and after the iterator reaches the end.
// An error is returned if either of those conditions does not hold.
func (it *DocumentIterator) ExplainMetrics() (*ExplainMetrics, error) {
	if it == nil {
		return nil, errors.New("firestore: iterator is nil")
	}
	if it.err == nil || it.err != iterator.Done {
		return nil, errMetricsBeforeEnd
	}
	return it.iter.getExplainMetrics()
}

// Next returns the next result. Its second return value is iterator.Done if there
// are no more results. Once Next returns Done, all subsequent calls will return
// Done.
func (it *DocumentIterator) Next() (*DocumentSnapshot, error) {
	if it.err != nil {
		return nil, it.err
	}
	if it.q.limitToLast {
		return nil, errors.New("firestore: queries that include limitToLast constraints cannot be streamed. Use DocumentIterator.GetAll() instead")
	}

	ds, err := it.iter.next()
	if err != nil {
		it.err = err
	}

	return ds, err
}

// Stop stops the iterator, freeing its resources.
// Always call Stop when you are done with a DocumentIterator.
// It is not safe to call Stop concurrently with Next.
func (it *DocumentIterator) Stop() {
	if it.iter != nil { // possible in error cases
		it.iter.stop()
	}
	if it.err == nil {
		it.err = iterator.Done
	}
}

// GetAll returns all the documents remaining from the iterator.
// It is not necessary to call Stop on the iterator after calling GetAll.
func (it *DocumentIterator) GetAll() ([]*DocumentSnapshot, error) {
	if it.err != nil {
		return nil, it.err
	}

	defer it.Stop()

	q := it.q
	limitedToLast := q.limitToLast
	q.processLimitToLast()
	var docs []*DocumentSnapshot
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	if limitedToLast {
		// Flip docs order before return.
		for i, j := 0, len(docs)-1; i < j; {
			docs[i], docs[j] = docs[j], docs[i]
			i++
			j--
		}
	}
	return docs, nil
}

type queryDocumentIterator struct {
	ctx          context.Context
	cancel       func()
	q            *Query
	tid          []byte // transaction ID, if any
	streamClient pb.Firestore_RunQueryClient
	readSettings *readSettings // readOptions, if any

	// Query explain metrics. This is only present when ExplainOptions is used.
	explainMetrics *ExplainMetrics
}

func newQueryDocumentIterator(ctx context.Context, q *Query, tid []byte, rs *readSettings) *queryDocumentIterator {
	ctx, cancel := context.WithCancel(ctx)
	return &queryDocumentIterator{
		ctx:          ctx,
		cancel:       cancel,
		q:            q,
		tid:          tid,
		readSettings: rs,
	}
}

// opts override the options stored in it.q.runQuerySettings
func (it *queryDocumentIterator) next() (_ *DocumentSnapshot, err error) {
	client := it.q.c
	if it.streamClient == nil {
		it.ctx = trace.StartSpan(it.ctx, "cloud.google.com/go/firestore.Query.RunQuery")
		defer func() {
			if errors.Is(err, iterator.Done) {
				trace.EndSpan(it.ctx, nil)
			} else {
				trace.EndSpan(it.ctx, err)
			}
		}()

		req, err := it.q.toRunQueryRequestProto()
		if err != nil {
			return nil, err
		}

		// Respect transactions first and read options (read time) second
		if rt, hasOpts := parseReadTime(client, it.readSettings); hasOpts {
			req.ConsistencySelector = &pb.RunQueryRequest_ReadTime{ReadTime: rt}
		}
		if it.tid != nil {
			req.ConsistencySelector = &pb.RunQueryRequest_Transaction{Transaction: it.tid}
		}
		it.streamClient, err = client.c.RunQuery(it.ctx, req)
		if err != nil {
			return nil, err
		}
	}
	var res *pb.RunQueryResponse
	for {
		res, err = it.streamClient.Recv()
		if err == io.EOF {
			return nil, iterator.Done
		}
		if err != nil {
			return nil, err
		}
		if res.Document != nil {
			break
		}
		// No document => partial progress; keep receiving.
		it.explainMetrics = fromExplainMetricsProto(res.GetExplainMetrics())
	}

	it.explainMetrics = fromExplainMetricsProto(res.GetExplainMetrics())

	docRef, err := pathToDoc(res.Document.Name, client)
	if err != nil {
		return nil, err
	}
	doc, err := newDocumentSnapshot(docRef, res.Document, client, res.ReadTime)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func (it *queryDocumentIterator) getExplainMetrics() (*ExplainMetrics, error) {
	if it == nil {
		return nil, fmt.Errorf("firestore: iterator is nil")
	}
	return it.explainMetrics, nil
}

func (it *queryDocumentIterator) stop() {
	it.cancel()
}

// Snapshots returns an iterator over snapshots of the query. Each time the query
// results change, a new snapshot will be generated.
func (q Query) Snapshots(ctx context.Context) *QuerySnapshotIterator {
	ws, err := newWatchStreamForQuery(ctx, q)
	if err != nil {
		return &QuerySnapshotIterator{err: err}
	}
	return &QuerySnapshotIterator{
		Query: q,
		ws:    ws,
	}
}

// QuerySnapshotIterator is an iterator over snapshots of a query.
// Call Next on the iterator to get a snapshot of the query's results each time they change.
// Call Stop on the iterator when done.
//
// For an example, see Query.Snapshots.
type QuerySnapshotIterator struct {
	// The Query used to construct this iterator.
	Query Query

	ws  *watchStream
	err error
}

// Next blocks until the query's results change, then returns a QuerySnapshot for
// the current results.
//
// Next is not expected to return iterator.Done unless it is called after Stop.
// Rarely, networking issues may also cause iterator.Done to be returned.
func (it *QuerySnapshotIterator) Next() (*QuerySnapshot, error) {
	if it.err != nil {
		return nil, it.err
	}
	btree, changes, readTime, err := it.ws.nextSnapshot()
	if err != nil {
		if err == io.EOF {
			err = iterator.Done
		}
		it.err = err
		return nil, it.err
	}
	return &QuerySnapshot{
		Documents: &DocumentIterator{
			iter: (*btreeDocumentIterator)(btree.BeforeIndex(0)), q: &it.Query,
		},
		Size:     btree.Len(),
		Changes:  changes,
		ReadTime: readTime,
	}, nil
}

// Stop stops receiving snapshots. You should always call Stop when you are done with
// a QuerySnapshotIterator, to free up resources. It is not safe to call Stop
// concurrently with Next.
func (it *QuerySnapshotIterator) Stop() {
	if it.ws != nil {
		it.ws.stop()
	}
}

// A QuerySnapshot is a snapshot of query results. It is returned by
// QuerySnapshotIterator.Next whenever the results of a query change.
type QuerySnapshot struct {
	// An iterator over the query results.
	// It is not necessary to call Stop on this iterator.
	Documents *DocumentIterator

	// The number of results in this snapshot.
	Size int

	// The changes since the previous snapshot.
	Changes []DocumentChange

	// The time at which this snapshot was obtained from Firestore.
	ReadTime time.Time
}

type btreeDocumentIterator btree.Iterator

func (it *btreeDocumentIterator) next() (*DocumentSnapshot, error) {
	if !(*btree.Iterator)(it).Next() {
		return nil, iterator.Done
	}
	return it.Key.(*DocumentSnapshot), nil
}

func (*btreeDocumentIterator) stop() {}
func (*btreeDocumentIterator) getExplainMetrics() (*ExplainMetrics, error) {
	return nil, nil
}

// WithReadOptions specifies constraints for accessing documents from the database,
// e.g. at what time snapshot to read the documents.
func (q *Query) WithReadOptions(opts ...ReadOption) *Query {
	if q.readSettings == nil {
		q.readSettings = &readSettings{}
	}
	for _, ro := range opts {
		ro.apply(q.readSettings)
	}
	return q
}

// AggregationQuery allows for generating aggregation results of an underlying
// basic query. A single AggregationQuery can contain multiple aggregations.
type AggregationQuery struct {
	// aggregateQueries contains all of the queries for this request.
	aggregateQueries []*pb.StructuredAggregationQuery_Aggregation
	// query contains a reference pointer to the underlying structured query.
	query *Query
	//  tx points to an already active transaction within which the AggregationQuery runs
	tx *Transaction
}

// Transaction specifies that aggregation query should run within provided transaction
func (a *AggregationQuery) Transaction(tx *Transaction) *AggregationQuery {
	a = a.clone()
	a.tx = tx
	return a
}

func (a *AggregationQuery) clone() *AggregationQuery {
	x := *a
	// Copy the contents of the slice-typed fields to a new backing store.
	if len(a.aggregateQueries) > 0 {
		x.aggregateQueries = make([]*pb.StructuredAggregationQuery_Aggregation, len(a.aggregateQueries))
		copy(x.aggregateQueries, a.aggregateQueries)
	}
	return &x
}

// WithCount specifies that the aggregation query provide a count of results
// returned by the underlying Query.
func (a *AggregationQuery) WithCount(alias string) *AggregationQuery {
	aq := &pb.StructuredAggregationQuery_Aggregation{
		Alias:    alias,
		Operator: &pb.StructuredAggregationQuery_Aggregation_Count_{},
	}

	a.aggregateQueries = append(a.aggregateQueries, aq)

	return a
}

// WithSumPath specifies that the aggregation query should provide a sum of the values
// of the provided field in the results returned by the underlying Query.
// The path argument can be a single field or a dot-separated sequence of
// fields, and must not contain any of the runes "˜*/[]".
// The alias argument can be empty or a valid Firestore document field name. It can be used
// as key in the AggregationResult to get the sum value. If alias is empty, Firestore
// will autogenerate a key.
func (a *AggregationQuery) WithSumPath(fp FieldPath, alias string) *AggregationQuery {
	ref, err := fref(fp)
	if err != nil {
		a.query.err = err
		return a
	}

	aq := &pb.StructuredAggregationQuery_Aggregation{
		Alias: alias,
		Operator: &pb.StructuredAggregationQuery_Aggregation_Sum_{
			Sum: &pb.StructuredAggregationQuery_Aggregation_Sum{
				Field: ref,
			},
		},
	}

	a.aggregateQueries = append(a.aggregateQueries, aq)
	return a
}

// WithSum specifies that the aggregation query should provide a sum of the values
// of the provided field in the results returned by the underlying Query.
// The alias argument can be empty or a valid Firestore document field name. It can be used
// as key in the AggregationResult to get the sum value. If alias is empty, Firestore
// will autogenerate a key.
func (a *AggregationQuery) WithSum(path string, alias string) *AggregationQuery {
	fp, err := parseDotSeparatedString(path)
	if err != nil {
		a.query.err = err
		return a
	}
	return a.WithSumPath(fp, alias)
}

// WithAvgPath specifies that the aggregation query should provide an average of the values
// of the provided field in the results returned by the underlying Query.
// The path argument can be a single field or a dot-separated sequence of
// fields, and must not contain any of the runes "˜*/[]".
// The alias argument can be empty or a valid Firestore document field name. It can be used
// as key in the AggregationResult to get the average value. If alias is empty, Firestore
// will autogenerate a key.
func (a *AggregationQuery) WithAvgPath(fp FieldPath, alias string) *AggregationQuery {
	ref, err := fref(fp)
	if err != nil {
		a.query.err = err
		return a
	}

	aq := &pb.StructuredAggregationQuery_Aggregation{
		Alias: alias,
		Operator: &pb.StructuredAggregationQuery_Aggregation_Avg_{
			Avg: &pb.StructuredAggregationQuery_Aggregation_Avg{
				Field: ref,
			},
		},
	}

	a.aggregateQueries = append(a.aggregateQueries, aq)
	return a
}

// WithAvg specifies that the aggregation query should provide an average of the values
// of the provided field in the results returned by the underlying Query.
// The alias argument can be empty or a valid Firestore document field name. It can be used
// as key in the AggregationResult to get the average value. If alias is empty, Firestore
// will autogenerate a key.
func (a *AggregationQuery) WithAvg(path string, alias string) *AggregationQuery {
	fp, err := parseDotSeparatedString(path)
	if err != nil {
		a.query.err = err
		return a
	}
	return a.WithAvgPath(fp, alias)
}

// Get retrieves the aggregation query results from the service.
func (a *AggregationQuery) Get(ctx context.Context) (AggregationResult, error) {
	aro, err := a.GetResponse(ctx)
	if aro != nil {
		return aro.Result, err
	}
	return nil, err
}

// GetResponse runs the aggregation with the options provided in the query
func (a *AggregationQuery) GetResponse(ctx context.Context) (aro *AggregationResponse, err error) {

	a.query.processLimitToLast()
	client := a.query.c.c
	q, err := a.query.toProto()
	if err != nil {
		return aro, err
	}

	req := &pb.RunAggregationQueryRequest{
		Parent: a.query.parentPath,
		QueryType: &pb.RunAggregationQueryRequest_StructuredAggregationQuery{
			StructuredAggregationQuery: &pb.StructuredAggregationQuery{
				QueryType: &pb.StructuredAggregationQuery_StructuredQuery{
					StructuredQuery: q,
				},
				Aggregations: a.aggregateQueries,
			},
		},
	}

	if a.query.runQuerySettings != nil {
		req.ExplainOptions = a.query.runQuerySettings.explainOptions
	}

	if a.tx != nil {
		req.ConsistencySelector = &pb.RunAggregationQueryRequest_Transaction{
			Transaction: a.tx.id,
		}
	}

	ctx = withResourceHeader(ctx, a.query.c.path())
	stream, err := client.RunAggregationQuery(ctx, req)
	if err != nil {
		return nil, err
	}

	aro = &AggregationResponse{}
	var resp AggregationResult

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if res.Result != nil {
			if resp == nil {
				resp = make(AggregationResult)
			}
			f := res.Result.AggregateFields

			for k, v := range f {
				resp[k] = v
			}
		}
		aro.ExplainMetrics = fromExplainMetricsProto(res.GetExplainMetrics())
	}
	aro.Result = resp
	return aro, nil
}

// AggregationResult contains the results of an aggregation query.
type AggregationResult map[string]interface{}

// AggregationResponse contains AggregationResult and response from the run options in the query
type AggregationResponse struct {
	Result AggregationResult

	// Query explain metrics. This is only present when ExplainOptions is provided.
	ExplainMetrics *ExplainMetrics
}
