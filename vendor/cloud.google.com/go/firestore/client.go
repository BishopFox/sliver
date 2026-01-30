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
	"net/url"
	"os"
	"strings"
	"time"

	vkit "cloud.google.com/go/firestore/apiv1"
	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
	"cloud.google.com/go/firestore/internal"
	"cloud.google.com/go/internal/detect"
	"cloud.google.com/go/internal/trace"
	gax "github.com/googleapis/gax-go/v2"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// resourcePrefixHeader is the name of the metadata header used to indicate
// the resource being operated on.
const resourcePrefixHeader = "google-cloud-resource-prefix"

// requestParamsHeader is routing header required to access named databases
const reqParamsHeader = "x-goog-request-params"

// reqParamsHeaderVal constructs header from dbPath
// dbPath is of the form projects/{project_id}/databases/{database_id}
func reqParamsHeaderVal(dbPath string) string {
	splitPath := strings.Split(dbPath, "/")
	projectID := splitPath[1]
	databaseID := splitPath[3]
	return fmt.Sprintf("project_id=%s&database_id=%s", url.QueryEscape(projectID), url.QueryEscape(databaseID))
}

// DetectProjectID is a sentinel value that instructs NewClient to detect the
// project ID. It is given in place of the projectID argument. NewClient will
// use the project ID from the given credentials or the default credentials
// (https://developers.google.com/accounts/docs/application-default-credentials)
// if no credentials were provided. When providing credentials, not all
// options will allow NewClient to extract the project ID. Specifically a JWT
// does not have the project ID encoded.
const DetectProjectID = detect.ProjectIDSentinel

// DefaultDatabaseID is name of the default database
const DefaultDatabaseID = "(default)"

// A Client provides access to the Firestore service.
type Client struct {
	c            *vkit.Client
	projectID    string
	databaseID   string        // A client is tied to a single database.
	readSettings *readSettings // readSettings allows setting a snapshot time to read the database
	UsesEmulator bool          // a boolean that indicates if the client is using the emulator
}

// newClient creates a new Firestore client, using the given createClient function to create the underlying client.
func newClient(ctx context.Context, projectID string, createClient func(ctx context.Context, opts ...option.ClientOption) (*vkit.Client, error), supportsEmulator bool, opts ...option.ClientOption) (*Client, error) {
	if projectID == "" {
		return nil, errors.New("firestore: projectID was empty")
	}
	var o []option.ClientOption
	var usesEmulator bool
	// If this environment variable is defined, configure the client to talk to the emulator.
	if addr := os.Getenv("FIRESTORE_EMULATOR_HOST"); addr != "" {
		if !supportsEmulator {
			return nil, fmt.Errorf("firestore: emulator is not supported for this client type")
		}

		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithPerRPCCredentials(emulatorCreds{}))
		if err != nil {
			return nil, fmt.Errorf("firestore: dialing address from env var FIRESTORE_EMULATOR_HOST: %s", err)
		}
		usesEmulator = true
		o = []option.ClientOption{option.WithGRPCConn(conn)}
		projectID, _ = detect.ProjectID(ctx, projectID, "", opts...)
		if projectID == "" {
			projectID = "dummy-emulator-firestore-project"
		}
	}
	o = append(o, opts...)

	// Detect project ID.
	projectID, err := detect.ProjectID(ctx, projectID, "", o...)
	if err != nil {
		return nil, err
	}

	vc, err := createClient(ctx, o...)
	if err != nil {
		return nil, err
	}
	vc.SetGoogleClientInfo("gccl", internal.Version)
	c := &Client{
		c:            vc,
		projectID:    projectID,
		databaseID:   DefaultDatabaseID,
		readSettings: &readSettings{},
		UsesEmulator: usesEmulator,
	}
	return c, nil
}

// NewClient creates a new Firestore client that uses the given project.
func NewClient(ctx context.Context, projectID string, opts ...option.ClientOption) (*Client, error) {
	return newClient(ctx, projectID, vkit.NewClient, true, opts...)
}

// NewRESTClient creates a new Firestore client that uses the REST API.
func NewRESTClient(ctx context.Context, projectID string, opts ...option.ClientOption) (*Client, error) {
	return newClient(ctx, projectID, vkit.NewRESTClient, false, opts...)
}

// NewClientWithDatabase creates a new Firestore client that accesses the
// specified database.
func NewClientWithDatabase(ctx context.Context, projectID string, databaseID string, opts ...option.ClientOption) (*Client, error) {
	if databaseID == "" {
		return nil, fmt.Errorf("firestore: To create a client using the %s database, please use NewClient", DefaultDatabaseID)
	}

	client, err := NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, err
	}

	client.databaseID = databaseID
	return client, nil
}

// Close closes any resources held by the client.
//
// Close need not be called at program exit.
func (c *Client) Close() error {
	return c.c.Close()
}

func (c *Client) path() string {
	return fmt.Sprintf("projects/%s/databases/%s", c.projectID, c.databaseID)
}

func withResourceHeader(ctx context.Context, resource string) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	md = md.Copy()
	md[resourcePrefixHeader] = []string{resource}
	return metadata.NewOutgoingContext(ctx, md)
}

func withRequestParamsHeader(ctx context.Context, requestParams string) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	md = md.Copy()
	md[reqParamsHeader] = []string{requestParams}
	return metadata.NewOutgoingContext(ctx, md)
}

// Collection creates a reference to a collection with the given path.
// A path is a sequence of IDs separated by slashes.
//
// Collection returns nil if path contains an even number of IDs or any ID is empty.
func (c *Client) Collection(path string) *CollectionRef {
	coll, _ := c.idsToRef(strings.Split(path, "/"), c.path())
	return coll
}

// Doc creates a reference to a document with the given path.
// A path is a sequence of IDs separated by slashes.
//
// Doc returns nil if path contains an odd number of IDs or any ID is empty.
func (c *Client) Doc(path string) *DocumentRef {
	_, doc := c.idsToRef(strings.Split(path, "/"), c.path())
	return doc
}

// DocFromFullPath creates a reference to a document from its full, absolute path,
// also known as its Google Cloud resource name.
// The path must be in the format:
// "projects/{projectID}/databases/{databaseID}/documents/{collectionID}/{documentID}/..."
// This method returns nil if:
//   - The fullPath is empty.
//   - The fullPath does not match the expected resource name format (e.g., missing "projects/" or "/documents/").
//   - The projectID or databaseID in the fullPath do not match the client's configuration.
//   - The fullPath refers to a collection instead of a document (i.e., has an odd number of segments after "/documents/").
//   - The fullPath contains any empty path segments.
func (c *Client) DocFromFullPath(fullPath string) *DocumentRef {
	if fullPath == "" {
		return nil
	}

	const documentsPrefix = "/documents/"
	if !strings.HasPrefix(fullPath, "projects/") || !strings.Contains(fullPath, documentsPrefix) {
		return nil
	}
	parts := strings.SplitN(fullPath, documentsPrefix, 2)
	if len(parts) != 2 {
		return nil
	}

	actualDBPathFromFullPath := parts[0]
	expectedDBPath := c.path()
	if actualDBPathFromFullPath != expectedDBPath {
		return nil
	}

	_, docRef := c.idsToRef(strings.Split(parts[1], "/"), actualDBPathFromFullPath)
	return docRef
}

// CollectionGroup creates a reference to a group of collections that include
// the given ID, regardless of parent document.
//
// For example, consider:
// France/Cities/Paris = {population: 100}
// Canada/Cities/Montreal = {population: 90}
//
// CollectionGroup can be used to query across all "Cities" regardless of
// its parent "Countries". See ExampleCollectionGroup for a complete example.
func (c *Client) CollectionGroup(collectionID string) *CollectionGroupRef {
	return newCollectionGroupRef(c, c.path(), collectionID)
}

func (c *Client) idsToRef(IDs []string, dbPath string) (*CollectionRef, *DocumentRef) {
	if len(IDs) == 0 {
		return nil, nil
	}
	for _, id := range IDs {
		if id == "" {
			return nil, nil
		}
	}
	coll := newTopLevelCollRef(c, dbPath, IDs[0])
	i := 1
	for i < len(IDs) {
		doc := newDocRef(coll, IDs[i])
		i++
		if i == len(IDs) {
			return nil, doc
		}
		coll = newCollRefWithParent(c, doc, IDs[i])
		i++
	}
	return coll, nil
}

// GetAll retrieves multiple documents with a single call. The
// DocumentSnapshots are returned in the order of the given DocumentRefs.
// The return value will always contain the same number of DocumentSnapshots
// as the number of DocumentRefs in the input.
//
// If the same DocumentRef is specified multiple times in the input, the return
// value will contain the same number of DocumentSnapshots referencing the same
// document.
//
// If a document is not present, the corresponding DocumentSnapshot's Exists
// method will return false.
func (c *Client) GetAll(ctx context.Context, docRefs []*DocumentRef) (_ []*DocumentSnapshot, err error) {
	ctx = trace.StartSpan(ctx, "cloud.google.com/go/firestore.GetAll")
	defer func() { trace.EndSpan(ctx, err) }()

	return c.getAll(ctx, docRefs, nil, nil)
}

func (c *Client) getAll(ctx context.Context, docRefs []*DocumentRef, tid []byte, rs *readSettings) (_ []*DocumentSnapshot, err error) {
	ctx = trace.StartSpan(ctx, "cloud.google.com/go/firestore.Client.BatchGetDocuments")
	defer func() { trace.EndSpan(ctx, err) }()

	var docNames []string
	docIndices := map[string][]int{} // doc name to positions in docRefs
	for i, dr := range docRefs {
		if err := dr.isValid(); err != nil {
			return nil, err
		}
		docNames = append(docNames, dr.Path)
		docIndices[dr.Path] = append(docIndices[dr.Path], i)
	}
	req := &pb.BatchGetDocumentsRequest{
		Database:  c.path(),
		Documents: docNames,
	}

	// Note that transaction ID and other consistency selectors are mutually exclusive.
	// We respect the transaction first, any read options passed by the caller second,
	// and any read options stored in the client third.
	if rt, hasOpts := parseReadTime(c, rs); hasOpts {
		req.ConsistencySelector = &pb.BatchGetDocumentsRequest_ReadTime{ReadTime: rt}
	}

	if tid != nil {
		req.ConsistencySelector = &pb.BatchGetDocumentsRequest_Transaction{Transaction: tid}
	}

	batchGetDocsCtx := withResourceHeader(ctx, req.Database)
	batchGetDocsCtx = withRequestParamsHeader(batchGetDocsCtx, reqParamsHeaderVal(c.path()))
	streamClient, err := c.c.BatchGetDocuments(batchGetDocsCtx, req)
	if err != nil {
		return nil, err
	}

	// Read and remember all results from the stream.
	var resps []*pb.BatchGetDocumentsResponse
	for {
		resp, err := streamClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		resps = append(resps, resp)
	}

	// Results may arrive out of order. Put each at the right indices.
	docs := make([]*DocumentSnapshot, len(docNames))
	for _, resp := range resps {
		var (
			indices []int
			doc     *pb.Document
			err     error
		)
		switch r := resp.Result.(type) {
		case *pb.BatchGetDocumentsResponse_Found:
			indices = docIndices[r.Found.Name]
			doc = r.Found
		case *pb.BatchGetDocumentsResponse_Missing:
			indices = docIndices[r.Missing]
			doc = nil
		default:
			return nil, errors.New("firestore: unknown BatchGetDocumentsResponse result type")
		}
		for _, index := range indices {
			if docs[index] != nil {
				return nil, fmt.Errorf("firestore: %q seen twice", docRefs[index].Path)
			}
			docs[index], err = newDocumentSnapshot(docRefs[index], doc, c, resp.ReadTime)
			if err != nil {
				return nil, err
			}
		}
	}
	return docs, nil
}

// Collections returns an iterator over the top-level collections.
func (c *Client) Collections(ctx context.Context) *CollectionIterator {
	ctx = trace.StartSpan(ctx, "cloud.google.com/go/firestore.Client.ListCollectionIds")
	defer func() { trace.EndSpan(ctx, nil) }()

	it := &CollectionIterator{
		client: c,
		it: c.c.ListCollectionIds(
			withResourceHeader(ctx, c.path()),
			&pb.ListCollectionIdsRequest{Parent: c.path() + "/documents"}),
	}
	it.pageInfo, it.nextFunc = iterator.NewPageInfo(
		it.fetch,
		func() int { return len(it.items) },
		func() interface{} { b := it.items; it.items = nil; return b })
	return it
}

// Batch returns a WriteBatch.
//
// Deprecated: The WriteBatch API has been replaced with the transaction and
// the bulk writer API. For atomic transaction operations, use `Transaction`.
// For bulk read and write operations, use `BulkWriter`.
func (c *Client) Batch() *WriteBatch {
	return &WriteBatch{c: c}
}

// BulkWriter returns a BulkWriter instance.
// The context passed to the BulkWriter remains stored through the lifecycle
// of the object. This context allows callers to cancel BulkWriter operations.
func (c *Client) BulkWriter(ctx context.Context) *BulkWriter {
	bw := newBulkWriter(ctx, c, c.path())
	return bw
}

// WithReadOptions specifies constraints for accessing documents from the database,
// e.g. at what time snapshot to read the documents.
func (c *Client) WithReadOptions(opts ...ReadOption) *Client {
	for _, ro := range opts {
		ro.apply(c.readSettings)
	}
	return c
}

// commit calls the Commit RPC outside of a transaction.
func (c *Client) commit(ctx context.Context, ws []*pb.Write) (_ []*WriteResult, err error) {
	ctx = trace.StartSpan(ctx, "cloud.google.com/go/firestore.Client.commit")
	defer func() { trace.EndSpan(ctx, err) }()

	req := &pb.CommitRequest{
		Database: c.path(),
		Writes:   ws,
	}
	res, err := c.c.Commit(withResourceHeader(ctx, req.Database), req)
	if err != nil {
		return nil, err
	}
	if len(res.WriteResults) == 0 {
		return nil, errors.New("firestore: missing WriteResult")
	}
	var wrs []*WriteResult
	for _, pwr := range res.WriteResults {
		wr, err := writeResultFromProto(pwr)
		if err != nil {
			return nil, err
		}
		wrs = append(wrs, wr)
	}
	return wrs, nil
}

func (c *Client) commit1(ctx context.Context, ws []*pb.Write) (*WriteResult, error) {
	wrs, err := c.commit(ctx, ws)
	if err != nil {
		return nil, err
	}
	return wrs[0], nil
}

// A WriteResult is returned by methods that write documents.
type WriteResult struct {
	// The time at which the document was updated, or created if it did not
	// previously exist. Writes that do not actually change the document do
	// not change the update time.
	UpdateTime time.Time
}

func writeResultFromProto(wr *pb.WriteResult) (*WriteResult, error) {
	// TODO(jba): Follow up if Delete is supposed to return a nil timestamp.
	var t time.Time
	if err := wr.GetUpdateTime().CheckValid(); err == nil {
		t = wr.GetUpdateTime().AsTime()
	}
	return &WriteResult{UpdateTime: t}, nil
}

func sleep(ctx context.Context, dur time.Duration) error {
	switch err := gax.Sleep(ctx, dur); err {
	case context.Canceled:
		return status.Error(codes.Canceled, "context canceled")
	case context.DeadlineExceeded:
		return status.Error(codes.DeadlineExceeded, "context deadline exceeded")
	default:
		return err
	}
}

// emulatorCreds is an instance of grpc.PerRPCCredentials that will configure a
// client to act as an admin for the Firestore emulator. It always hardcodes
// the "authorization" metadata field to contain "Bearer owner", which the
// Firestore emulator accepts as valid admin credentials.
type emulatorCreds struct{}

func (ec emulatorCreds) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer owner"}, nil
}
func (ec emulatorCreds) RequireTransportSecurity() bool {
	return false
}

// ReadTime specifies a time-specific snapshot of the database to read.
func ReadTime(t time.Time) ReadOption {
	return readTime(t)
}

type readTime time.Time

func (rt readTime) apply(rs *readSettings) {
	rs.readTime = time.Time(rt)
}

// ReadOption interface allows for abstraction of computing read time settings.
type ReadOption interface {
	apply(*readSettings)
}

// readSettings contains the ReadOptions for a read operation
type readSettings struct {
	readTime time.Time
}

// parseReadTime ensures that fallback order of read options is respected.
func parseReadTime(c *Client, rs *readSettings) (*timestamppb.Timestamp, bool) {
	if rs != nil && !rs.readTime.IsZero() {
		return &timestamppb.Timestamp{Seconds: int64(rs.readTime.Unix())}, true
	}
	if c.readSettings != nil && !c.readSettings.readTime.IsZero() {
		return &timestamppb.Timestamp{Seconds: int64(c.readSettings.readTime.Unix())}, true
	}
	return nil, false
}
