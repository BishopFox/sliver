package crack

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/klauspost/compress/zstd"
	"google.golang.org/grpc"
)

func TestLocalCrackFileInfoHandlesStatErrors(t *testing.T) {
	wantErr := errors.New("permission denied")
	info, err := localCrackFileInfoWith("/restricted/file", func(string) (os.FileInfo, error) {
		return nil, wantErr
	})
	if info != nil {
		t.Fatalf("localCrackFileInfoWith info = %#v, want nil", info)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("localCrackFileInfoWith error = %v, want wrapped %v", err, wantErr)
	}
	if !strings.Contains(err.Error(), "/restricted/file") {
		t.Fatalf("localCrackFileInfoWith error = %q, want path context", err)
	}
}

func TestLocalCrackFileInfoRejectsDirectories(t *testing.T) {
	_, err := localCrackFileInfo(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "is a directory") {
		t.Fatalf("localCrackFileInfo directory error = %v, want directory error", err)
	}
}

func TestLocalCrackFileInfoAcceptsRegularFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "crack-file-")
	if err != nil {
		t.Fatalf("create local crack file: %v", err)
	}
	const payload = "hashcat input"
	if _, err := file.WriteString(payload); err != nil {
		t.Fatalf("write local crack file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close local crack file: %v", err)
	}

	info, err := localCrackFileInfo(file.Name())
	if err != nil {
		t.Fatalf("localCrackFileInfo: %v", err)
	}
	if info.Size() != int64(len(payload)) {
		t.Fatalf("localCrackFileInfo size = %d, want %d", info.Size(), len(payload))
	}
}

func TestUploadCrackFileCleansUpPostCreateFailures(t *testing.T) {
	readErr := errors.New("read failed")
	chunkErr := errors.New("chunk upload failed")
	completeErr := errors.New("complete failed")

	tests := []struct {
		name              string
		reader            io.Reader
		configureRPC      func(*fakeCrackFileUploadRPC)
		wantErr           error
		wantChunkCalls    int
		wantCompleteCalls int
	}{
		{
			name:    "read failure",
			reader:  errorReader{err: readErr},
			wantErr: readErr,
		},
		{
			name:   "chunk upload failure",
			reader: bytes.NewReader([]byte("payload")),
			configureRPC: func(rpc *fakeCrackFileUploadRPC) {
				rpc.chunkErr = chunkErr
			},
			wantErr:        chunkErr,
			wantChunkCalls: 1,
		},
		{
			name:   "completion failure",
			reader: bytes.NewReader([]byte("payload")),
			configureRPC: func(rpc *fakeCrackFileUploadRPC) {
				rpc.completeErr = completeErr
			},
			wantErr:           completeErr,
			wantChunkCalls:    1,
			wantCompleteCalls: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rpc := &fakeCrackFileUploadRPC{}
			if test.configureRPC != nil {
				test.configureRPC(rpc)
			}
			reservation := &clientpb.CrackFile{ID: "reserved-file", ChunkSize: 1024}

			_, err := uploadCrackFile(test.reader, reservation, rpc, nil)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("uploadCrackFile error = %v, want wrapped %v", err, test.wantErr)
			}
			if rpc.chunkCalls != test.wantChunkCalls {
				t.Fatalf("CrackFileChunkUpload calls = %d, want %d", rpc.chunkCalls, test.wantChunkCalls)
			}
			if rpc.completeCalls != test.wantCompleteCalls {
				t.Fatalf("CrackFileComplete calls = %d, want %d", rpc.completeCalls, test.wantCompleteCalls)
			}
			if rpc.deleteCalls != 1 {
				t.Fatalf("CrackFileDelete calls = %d, want 1", rpc.deleteCalls)
			}
			if rpc.deleted == nil || rpc.deleted.ID != reservation.ID {
				t.Fatalf("CrackFileDelete request = %#v, want ID %q", rpc.deleted, reservation.ID)
			}
		})
	}
}

func TestUploadCrackFileReportsCleanupFailure(t *testing.T) {
	uploadErr := errors.New("chunk upload failed")
	cleanupErr := errors.New("delete failed")
	rpc := &fakeCrackFileUploadRPC{chunkErr: uploadErr, deleteErr: cleanupErr}
	reservation := &clientpb.CrackFile{ID: "reserved-file", ChunkSize: 1024}

	_, err := uploadCrackFile(bytes.NewReader([]byte("payload")), reservation, rpc, nil)
	if !errors.Is(err, uploadErr) {
		t.Fatalf("uploadCrackFile error = %v, want wrapped upload error %v", err, uploadErr)
	}
	if !errors.Is(err, cleanupErr) {
		t.Fatalf("uploadCrackFile error = %v, want wrapped cleanup error %v", err, cleanupErr)
	}
	if !strings.Contains(err.Error(), reservation.ID) {
		t.Fatalf("uploadCrackFile error = %q, want reservation ID", err)
	}
	if rpc.deleteCalls != 1 {
		t.Fatalf("CrackFileDelete calls = %d, want 1", rpc.deleteCalls)
	}
}

func TestUploadCrackFileSuccessCompletesWithoutCleanup(t *testing.T) {
	payload := []byte("correct horse battery staple\n")
	rpc := &fakeCrackFileUploadRPC{}
	reservation := &clientpb.CrackFile{ID: "reserved-file", ChunkSize: 8}
	progressCalls := 0

	total, err := uploadCrackFile(bytes.NewReader(payload), reservation, rpc, func(int64, uint32, int) {
		progressCalls++
	})
	if err != nil {
		t.Fatalf("uploadCrackFile: %v", err)
	}
	if total == 0 {
		t.Fatal("uploadCrackFile compressed byte count = 0, want non-zero")
	}
	if rpc.chunkCalls == 0 {
		t.Fatal("CrackFileChunkUpload calls = 0, want at least 1")
	}
	if progressCalls != rpc.chunkCalls {
		t.Fatalf("progress calls = %d, want %d", progressCalls, rpc.chunkCalls)
	}
	if rpc.completeCalls != 1 {
		t.Fatalf("CrackFileComplete calls = %d, want 1", rpc.completeCalls)
	}
	if rpc.deleteCalls != 0 {
		t.Fatalf("CrackFileDelete calls = %d, want 0", rpc.deleteCalls)
	}
	wantDigest := sha256.Sum256(payload)
	if rpc.completed == nil || rpc.completed.ID != reservation.ID {
		t.Fatalf("CrackFileComplete request = %#v, want ID %q", rpc.completed, reservation.ID)
	}
	if rpc.completed.Sha2_256 != hex.EncodeToString(wantDigest[:]) {
		t.Fatalf("CrackFileComplete digest = %q, want %q", rpc.completed.Sha2_256, hex.EncodeToString(wantDigest[:]))
	}
}

func TestChunkReaderZstdRoundTrip(t *testing.T) {
	largePayload := make([]byte, 256*1024)
	if _, err := rand.New(rand.NewSource(42)).Read(largePayload); err != nil {
		t.Fatalf("create test payload: %v", err)
	}

	tests := []struct {
		name          string
		payload       []byte
		chunkSize     int64
		wantMinChunks int
	}{
		{
			name:          "single chunk",
			payload:       []byte("correct horse battery staple\n"),
			chunkSize:     1024,
			wantMinChunks: 1,
		},
		{
			name:          "many chunks from one read",
			payload:       largePayload,
			chunkSize:     1024,
			wantMinChunks: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			compressed, chunkCount, err := collectCompressedCrackFile(bytes.NewReader(test.payload), test.chunkSize)
			if err != nil {
				t.Fatalf("chunkReader: %v", err)
			}
			if chunkCount < test.wantMinChunks {
				t.Fatalf("chunk count = %d, want at least %d", chunkCount, test.wantMinChunks)
			}

			decoder, err := zstd.NewReader(bytes.NewReader(compressed))
			if err != nil {
				t.Fatalf("open zstd stream: %v", err)
			}
			decoded, err := io.ReadAll(decoder)
			decoder.Close()
			if err != nil {
				t.Fatalf("decode zstd stream: %v", err)
			}
			if !bytes.Equal(decoded, test.payload) {
				t.Fatalf("decoded payload does not match input: got %d bytes, want %d", len(decoded), len(test.payload))
			}
		})
	}
}

func TestChunkReaderHandlesDataWithEOF(t *testing.T) {
	payload := []byte("reader returns data and EOF together")
	compressed, _, err := collectCompressedCrackFile(&dataEOFReader{data: payload}, 8)
	if err != nil {
		t.Fatalf("chunkReader: %v", err)
	}
	decoder, err := zstd.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("open zstd stream: %v", err)
	}
	decoded, err := io.ReadAll(decoder)
	decoder.Close()
	if err != nil {
		t.Fatalf("decode zstd stream: %v", err)
	}
	if !bytes.Equal(decoded, payload) {
		t.Fatalf("decoded payload = %q, want %q", decoded, payload)
	}
}

func TestChunkReaderRejectsInvalidChunkSize(t *testing.T) {
	for _, chunkSize := range []int64{0, -1} {
		compressed, chunkCount, err := collectCompressedCrackFile(bytes.NewReader([]byte("payload")), chunkSize)
		if err == nil {
			t.Fatalf("chunkReader accepted chunk size %d", chunkSize)
		}
		if chunkCount != 0 || len(compressed) != 0 {
			t.Fatalf("invalid chunk size %d emitted %d chunks (%d bytes)", chunkSize, chunkCount, len(compressed))
		}
	}
}

func TestChunkReaderPropagatesReadError(t *testing.T) {
	wantErr := errors.New("read failed")
	compressed, chunkCount, err := collectCompressedCrackFile(errorReader{err: wantErr}, 1024)
	if !errors.Is(err, wantErr) {
		t.Fatalf("chunkReader error = %v, want %v", err, wantErr)
	}
	if chunkCount != 0 || len(compressed) != 0 {
		t.Fatalf("failed read emitted %d chunks (%d bytes)", chunkCount, len(compressed))
	}
}

func collectCompressedCrackFile(reader io.Reader, chunkSize int64) ([]byte, int, error) {
	chunks, errCh := readCrackFileChunks(reader, chunkSize)

	compressed := bytes.Buffer{}
	chunkCount := 0
	invalidChunk := false
	for chunk := range chunks {
		if len(chunk) == 0 || int64(len(chunk)) > chunkSize {
			invalidChunk = true
		}
		_, _ = compressed.Write(chunk)
		chunkCount++
	}
	readerErr := <-errCh
	if invalidChunk {
		return nil, chunkCount, errors.New("chunkReader emitted an invalid chunk size")
	}
	return compressed.Bytes(), chunkCount, readerErr
}

type dataEOFReader struct {
	data []byte
	done bool
}

func (r *dataEOFReader) Read(buffer []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	return copy(buffer, r.data), io.EOF
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

type fakeCrackFileUploadRPC struct {
	rpcpb.SliverRPCClient

	chunkErr    error
	completeErr error
	deleteErr   error

	chunkCalls    int
	completeCalls int
	deleteCalls   int

	uploaded  []*clientpb.CrackFileChunk
	completed *clientpb.CrackFile
	deleted   *clientpb.CrackFile
}

func (f *fakeCrackFileUploadRPC) CrackFileChunkUpload(_ context.Context, request *clientpb.CrackFileChunk, _ ...grpc.CallOption) (*commonpb.Empty, error) {
	f.chunkCalls++
	f.uploaded = append(f.uploaded, &clientpb.CrackFileChunk{
		CrackFileID: request.CrackFileID,
		N:           request.N,
		Data:        append([]byte(nil), request.Data...),
	})
	if f.chunkErr != nil {
		return nil, f.chunkErr
	}
	return &commonpb.Empty{}, nil
}

func (f *fakeCrackFileUploadRPC) CrackFileComplete(_ context.Context, request *clientpb.CrackFile, _ ...grpc.CallOption) (*commonpb.Empty, error) {
	f.completeCalls++
	f.completed = &clientpb.CrackFile{ID: request.ID, Sha2_256: request.Sha2_256}
	if f.completeErr != nil {
		return nil, f.completeErr
	}
	return &commonpb.Empty{}, nil
}

func (f *fakeCrackFileUploadRPC) CrackFileDelete(_ context.Context, request *clientpb.CrackFile, _ ...grpc.CallOption) (*commonpb.Empty, error) {
	f.deleteCalls++
	f.deleted = &clientpb.CrackFile{ID: request.ID}
	if f.deleteErr != nil {
		return nil, f.deleteErr
	}
	return &commonpb.Empty{}, nil
}
