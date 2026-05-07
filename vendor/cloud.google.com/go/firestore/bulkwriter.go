// Copyright 2022 Google LLC
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
	"sync"
	"time"

	vkit "cloud.google.com/go/firestore/apiv1"
	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
	"golang.org/x/time/rate"
	"google.golang.org/api/support/bundler"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// maxBatchSize is the max number of writes to send in a request
	maxBatchSize = 20
	// maxRetryAttempts is the max number of times to retry a write
	maxRetryAttempts = 10
	// defaultStartingMaximumOpsPerSecond is the starting max number of requests to the service per second
	defaultStartingMaximumOpsPerSecond = 500
	// maxWritesPerSecond is the starting limit of writes allowed to callers per second
	maxWritesPerSecond = maxBatchSize * defaultStartingMaximumOpsPerSecond
)

var (
	batchWriteRetryCodes = map[codes.Code]bool{
		codes.ResourceExhausted: true,
		codes.Unavailable:       true,
		codes.Aborted:           true,
	}
)

// bulkWriterResult contains the WriteResult or error results from an individual
// write to the database.
type bulkWriterResult struct {
	result *pb.WriteResult // (cached) result from the operation
	err    error           // (cached) any errors that occurred
}

// BulkWriterJob provides read-only access to the results of a BulkWriter write attempt.
type BulkWriterJob struct {
	resultChan  chan bulkWriterResult // send errors and results to this channel
	write       *pb.Write             // the writes to apply to the database
	attempts    int                   // number of times this write has been attempted
	resultsLock sync.Mutex            // guards the cached wr and e values for the job
	result      *WriteResult          // (cached) result from the operation
	err         error                 // (cached) any errors that occurred
	ctx         context.Context       // context for canceling/timing out results
}

// Results gets the results of the BulkWriter write attempt.
// This method blocks if the results for this BulkWriterJob haven't been
// received.
func (j *BulkWriterJob) Results() (*WriteResult, error) {
	j.resultsLock.Lock()
	defer j.resultsLock.Unlock()
	if j.result == nil && j.err == nil {
		j.result, j.err = j.processResults() // cache the results for additional calls
	}
	return j.result, j.err
}

// processResults checks for errors returned from send() and packages up the
// results as WriteResult objects
func (j *BulkWriterJob) processResults() (*WriteResult, error) {
	select {
	case <-j.ctx.Done():
		return nil, j.ctx.Err()
	case bwr := <-j.resultChan:
		if bwr.err != nil {
			return nil, bwr.err
		}
		return writeResultFromProto(bwr.result)
	}
}

// setError ensures that an error is returned on the error channel of BulkWriterJob.
func (j *BulkWriterJob) setError(e error) {
	bwr := bulkWriterResult{
		err:    e,
		result: nil,
	}
	j.resultChan <- bwr
	close(j.resultChan)
}

// A BulkWriter supports concurrent writes to multiple documents. The BulkWriter
// submits document writes in maximum batches of 20 writes per request. Each
// request can contain many different document writes: create, delete, update,
// and set are all supported.
//
// Only one operation (create, set, update, delete) per document is allowed.
// BulkWriter cannot promise atomicity: individual writes can fail or succeed
// independent of each other. Bulkwriter does not apply writes in any set order;
// thus a document can't have set on it immediately after creation.
type BulkWriter struct {
	database           string           // the database as resource name: projects/[PROJECT]/databases/[DATABASE]
	start              time.Time        // when this BulkWriter was started; used to calculate qps and rate increases
	vc                 *vkit.Client     // internal client
	maxOpsPerSecond    int              // number of requests that can be sent per second
	docUpdatePaths     map[string]bool  // document paths with corresponding writes in the queue
	docUpdatePathsLock sync.Mutex       // guards docUpdatePaths
	limiter            rate.Limiter     // limit requests to server to <= 500 qps
	bundler            *bundler.Bundler // handle bundling up writes to Firestore
	ctx                context.Context  // context for canceling all BulkWriter operations
	isOpenLock         sync.RWMutex     // guards against setting isOpen concurrently
	isOpen             bool             // flag that the BulkWriter is closed
}

// newBulkWriter creates a new instance of the BulkWriter.
func newBulkWriter(ctx context.Context, c *Client, database string) *BulkWriter {
	// Although typically we shouldn't store Context objects, in this case we
	// need to pass this Context through to the Bundler handler.
	ctx = withResourceHeader(ctx, c.path())

	bw := &BulkWriter{
		database:        database,
		start:           time.Now(),
		vc:              c.c,
		isOpen:          true,
		maxOpsPerSecond: defaultStartingMaximumOpsPerSecond,
		docUpdatePaths:  make(map[string]bool),
		ctx:             ctx,
		limiter:         *rate.NewLimiter(rate.Limit(maxWritesPerSecond), 1),
	}

	// can't initialize within struct above; need instance reference to BulkWriter.send()
	bw.bundler = bundler.NewBundler(&BulkWriterJob{}, bw.send)
	bw.bundler.HandlerLimit = bw.maxOpsPerSecond
	bw.bundler.BundleCountThreshold = maxBatchSize

	return bw
}

// End sends all enqueued writes in parallel and closes the BulkWriter to new requests.
// After calling End(), calling any additional method automatically returns
// with an error. This method completes when there are no more pending writes
// in the queue.
func (bw *BulkWriter) End() {
	bw.isOpenLock.Lock()
	bw.isOpen = false
	bw.isOpenLock.Unlock()
	bw.Flush()
}

// Flush commits all writes that have been enqueued up to this point in parallel.
// This method blocks execution.
func (bw *BulkWriter) Flush() {
	bw.bundler.Flush()
}

// Create adds a document creation write to the queue of writes to send.
// Note: You cannot write to (Create, Update, Set, or Delete) the same document more than once.
func (bw *BulkWriter) Create(doc *DocumentRef, datum interface{}) (*BulkWriterJob, error) {
	bw.isOpenLock.RLock()
	defer bw.isOpenLock.RUnlock()
	err := bw.checkWriteConditions(doc)
	if err != nil {
		return nil, err
	}

	w, err := doc.newCreateWrites(datum)
	if err != nil {
		return nil, fmt.Errorf("firestore: cannot create %v with %v. %w", doc.ID, datum, err)
	}

	if len(w) > 1 {
		return nil, fmt.Errorf("firestore: too many document writes sent to bulkwriter")
	}

	j := bw.write(w[0])
	return j, nil
}

// Delete adds a document deletion write to the queue of writes to send.
// Note: You cannot write to (Create, Update, Set, or Delete) the same document more than once.
func (bw *BulkWriter) Delete(doc *DocumentRef, preconds ...Precondition) (*BulkWriterJob, error) {
	bw.isOpenLock.RLock()
	defer bw.isOpenLock.RUnlock()
	err := bw.checkWriteConditions(doc)
	if err != nil {
		return nil, err
	}

	w, err := doc.newDeleteWrites(preconds)
	if err != nil {
		return nil, fmt.Errorf("firestore: cannot delete doc %v. %w", doc.ID, err)
	}

	if len(w) > 1 {
		return nil, fmt.Errorf("firestore: too many document writes sent to bulkwriter")
	}

	j := bw.write(w[0])
	return j, nil
}

// Set adds a document set write to the queue of writes to send.
// Note: You cannot write to (Create, Update, Set, or Delete) the same document more than once.
func (bw *BulkWriter) Set(doc *DocumentRef, datum interface{}, opts ...SetOption) (*BulkWriterJob, error) {
	bw.isOpenLock.RLock()
	defer bw.isOpenLock.RUnlock()
	err := bw.checkWriteConditions(doc)
	if err != nil {
		return nil, err
	}

	w, err := doc.newSetWrites(datum, opts)
	if err != nil {
		return nil, fmt.Errorf("firestore: cannot set %v on doc %v. %w", datum, doc.ID, err)
	}

	if len(w) > 1 {
		return nil, fmt.Errorf("firestore: too many writes sent to bulkwriter")
	}

	j := bw.write(w[0])
	return j, nil
}

// Update adds a document update write to the queue of writes to send.
// Note: You cannot write to (Create, Update, Set, or Delete) the same document more than once.
func (bw *BulkWriter) Update(doc *DocumentRef, updates []Update, preconds ...Precondition) (*BulkWriterJob, error) {
	bw.isOpenLock.RLock()
	defer bw.isOpenLock.RUnlock()
	err := bw.checkWriteConditions(doc)
	if err != nil {
		return nil, err
	}

	w, err := doc.newUpdatePathWrites(updates, preconds)
	if err != nil {
		return nil, fmt.Errorf("firestore: cannot update doc %v. %w", doc.ID, err)
	}

	if len(w) > 1 {
		return nil, fmt.Errorf("firestore: too many writes sent to bulkwriter")
	}

	j := bw.write(w[0])
	return j, nil
}

// checkConditions determines whether this write attempt is valid. It returns
// an error if either the BulkWriter has already been closed or if it
// receives a nil document reference.
func (bw *BulkWriter) checkWriteConditions(doc *DocumentRef) error {
	if !bw.isOpen {
		return errors.New("firestore: BulkWriter has been closed")
	}

	if doc == nil {
		return errors.New("firestore: nil document contents")
	}

	bw.docUpdatePathsLock.Lock()
	defer bw.docUpdatePathsLock.Unlock()
	_, havePath := bw.docUpdatePaths[doc.shortPath]
	if havePath {
		return fmt.Errorf("firestore: BulkWriter received duplicate write for path: %v", doc.shortPath)
	}

	bw.docUpdatePaths[doc.shortPath] = true

	return nil
}

// write packages up write requests into bulkWriterJob objects.
func (bw *BulkWriter) write(w *pb.Write) *BulkWriterJob {

	j := &BulkWriterJob{
		resultChan: make(chan bulkWriterResult, 1),
		write:      w,
		ctx:        bw.ctx,
	}

	bw.limiter.Wait(bw.ctx)
	// ignore operation size constraints and related errors; can't be inferred at compile time
	// Bundler is set to accept an unlimited amount of bytes
	_ = bw.bundler.Add(j, 0)

	return j
}

// send transmits writes to the service and matches response results to job channels.
func (bw *BulkWriter) send(i interface{}) {
	bwj := i.([]*BulkWriterJob)

	if len(bwj) == 0 {
		return
	}

	var ws []*pb.Write
	for _, w := range bwj {
		ws = append(ws, w.write)
	}

	bwr := &pb.BatchWriteRequest{
		Database: bw.database,
		Writes:   ws,
		Labels:   map[string]string{},
	}

	select {
	case <-bw.ctx.Done():
		return
	default:
		resp, err := bw.vc.BatchWrite(bw.ctx, bwr)
		if err != nil {
			// Do we need to be selective about what kind of errors we send?
			for _, j := range bwj {
				j.setError(err)
			}
			return
		}
		// Match write results with BulkWriterJob objects
		for i, res := range resp.WriteResults {
			s := resp.Status[i]
			c := s.GetCode()
			if c != 0 { // Should we do an explicit check against rpc.Code enum?
				j := bwj[i]
				j.attempts++

				// Do we need separate retry bundler?
				_, isRetryable := batchWriteRetryCodes[codes.Code(s.Code)]
				if j.attempts < maxRetryAttempts && isRetryable {
					// ignore operation size constraints and related errors; job size can't be inferred at compile time
					// Bundler is set to accept an unlimited amount of bytes
					_ = bw.bundler.Add(j, 0)
				} else {
					j.setError(status.Error(codes.Code(s.Code), s.Message))
				}
				continue
			}

			bwj[i].resultChan <- bulkWriterResult{err: nil, result: res}
			close(bwj[i].resultChan)
		}
	}
}
