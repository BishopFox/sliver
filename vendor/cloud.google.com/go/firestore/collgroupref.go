// Copyright 2019 Google LLC
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
	"sort"

	"cloud.google.com/go/firestore/apiv1/firestorepb"
	"google.golang.org/api/iterator"
)

// A CollectionGroupRef is a reference to a group of collections sharing the
// same ID.
type CollectionGroupRef struct {
	c *Client

	// Use the methods of Query on a CollectionGroupRef to create and run queries.
	Query
}

func newCollectionGroupRef(c *Client, dbPath, collectionID string) *CollectionGroupRef {
	return &CollectionGroupRef{
		c: c,

		Query: Query{
			c:              c,
			collectionID:   collectionID,
			path:           dbPath,
			parentPath:     dbPath + "/documents",
			allDescendants: true,
			readSettings:   &readSettings{},
		},
	}
}

// GetPartitionedQueries returns a slice of Query objects, each containing a
// partition of a collection group. partitionCount must be a positive value and
// the number of returned partitions may be less than the requested number if
// providing the desired number would result in partitions with very few documents.
//
// If a Collection Group Query would return a large number of documents, this
// can help to subdivide the query to smaller working units that can be distributed.
//
// If the goal is to run the queries across processes or workers, it may be useful to use
// `Query.Serialize` and `Query.Deserialize` to serialize the query.
func (cgr CollectionGroupRef) GetPartitionedQueries(ctx context.Context, partitionCount int) ([]Query, error) {
	qp, err := cgr.getPartitions(ctx, partitionCount)
	if err != nil {
		return nil, err
	}
	queries := make([]Query, len(qp))
	for i, part := range qp {
		queries[i] = part.toQuery()
	}
	return queries, nil
}

// getPartitions returns a slice of queryPartition objects, describing a start
// and end range to query a subsection of the collection group. partitionCount
// must be a positive value and the number of returned partitions may be less
// than the requested number if providing the desired number would result in
// partitions with very few documents.
func (cgr CollectionGroupRef) getPartitions(ctx context.Context, partitionCount int) ([]queryPartition, error) {
	orderedQuery := cgr.query().OrderBy(DocumentID, Asc)

	if partitionCount <= 0 {
		return nil, errors.New("a positive partitionCount must be provided")
	} else if partitionCount == 1 {
		return []queryPartition{{CollectionGroupQuery: orderedQuery}}, nil
	}

	db := cgr.c.path()
	ctx = withResourceHeader(ctx, db)

	// CollectionGroup Queries need to be ordered by __name__ ASC.
	query, err := orderedQuery.toProto()
	if err != nil {
		return nil, err
	}
	structuredQuery := &firestorepb.PartitionQueryRequest_StructuredQuery{
		StructuredQuery: query,
	}

	// Uses default PageSize
	pbr := &firestorepb.PartitionQueryRequest{
		Parent:         db + "/documents",
		PartitionCount: int64(partitionCount),
		QueryType:      structuredQuery,
	}
	cursorReferences := make([]*firestorepb.Value, 0, partitionCount)
	iter := cgr.c.c.PartitionQuery(ctx, pbr)
	for {
		cursor, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("GetPartitions: %w", err)
		}
		cursorReferences = append(cursorReferences, cursor.GetValues()...)
	}

	// From Proto documentation:
	// To obtain a complete result set ordered with respect to the results of the
	// query supplied to PartitionQuery, the results sets should be merged:
	// cursor A, cursor B, cursor M, cursor Q, cursor U, cursor W
	// Once we have exhausted the pages, the cursor values need to be sorted in
	// lexicographical order by segment (areas between '/').
	sort.Sort(byFirestoreValue(cursorReferences))

	queryPartitions := make([]queryPartition, 0, len(cursorReferences))
	previousCursor := ""

	for _, cursor := range cursorReferences {
		cursorRef := cursor.GetReferenceValue()

		// remove the root path from the reference, as queries take cursors
		// relative to a collection
		cursorRef = cursorRef[len(orderedQuery.path)+1:]

		qp := queryPartition{
			CollectionGroupQuery: orderedQuery,
			StartAt:              previousCursor,
			EndBefore:            cursorRef,
		}
		queryPartitions = append(queryPartitions, qp)
		previousCursor = cursorRef
	}

	// In the case there were no partitions, we still add a single partition to
	// the result, that covers the complete range.
	lastPart := queryPartition{CollectionGroupQuery: orderedQuery}
	if len(cursorReferences) > 0 {
		cursorRef := cursorReferences[len(cursorReferences)-1].GetReferenceValue()
		lastPart.StartAt = cursorRef[len(orderedQuery.path)+1:]
	}
	queryPartitions = append(queryPartitions, lastPart)

	return queryPartitions, nil
}

// queryPartition provides a Collection Group Reference and start and end split
// points allowing for a section of a collection group to be queried. This is
// used by GetPartitions which, given a CollectionGroupReference returns smaller
// sub-queries or partitions
type queryPartition struct {
	// CollectionGroupQuery is an ordered query on a CollectionGroupReference.
	// This query must be ordered Asc on __name__.
	// Example: client.CollectionGroup("collectionID").query().OrderBy(DocumentID, Asc)
	CollectionGroupQuery Query

	// StartAt is a document reference value, relative to the collection, not
	// a complete parent path.
	// Example: "documents/collectionName/documentName"
	StartAt string

	// EndBefore is a document reference value, relative to the collection, not
	// a complete parent path.
	// Example: "documents/collectionName/documentName"
	EndBefore string
}

// toQuery converts a queryPartition object to a Query object
func (qp queryPartition) toQuery() Query {
	q := *qp.CollectionGroupQuery.query()

	// Remove the leading path before calling StartAt, EndBefore
	if qp.StartAt != "" {
		q = q.StartAt(qp.StartAt)
	}
	if qp.EndBefore != "" {
		q = q.EndBefore(qp.EndBefore)
	}
	return q
}
