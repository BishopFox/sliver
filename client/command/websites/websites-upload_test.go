package websites

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestUploadedFileName(t *testing.T) {
	tests := []struct {
		name string
		id   string
		path string
		want string
	}{
		{
			name: "regular path",
			id:   "abc",
			path: "/drop/report.bin",
			want: "abc_report.bin",
		},
		{
			name: "root path",
			id:   "abc",
			path: "/",
			want: "abc_upload",
		},
		{
			name: "empty path",
			id:   "abc",
			path: "",
			want: "abc_upload",
		},
		{
			name: "windows separators",
			id:   "abc",
			path: `C:\temp\loot.txt`,
			want: "abc_loot.txt",
		},
		{
			name: "unsafe characters are replaced",
			id:   "abc",
			path: "/drop/re port;rm -rf.bin",
			want: "abc_re_port_rm_-rf.bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uploadedFileName(&clientpb.WebUploadedContent{ID: tt.id, Path: tt.path})
			if got != tt.want {
				t.Fatalf("unexpected file name: got %q want %q", got, tt.want)
			}
		})
	}
}

func TestUploadedMetadataNestsJSONHeaders(t *testing.T) {
	metadata := uploadedMetadataOf(&clientpb.WebUploadedContent{
		Headers: `{"User-Agent":["curl/8.0"]}`,
	}, "abc_report.bin")

	headers, ok := metadata.Headers.(map[string][]string)
	if !ok {
		t.Fatalf("headers were not parsed as json, got %T", metadata.Headers)
	}
	if got := headers["User-Agent"]; len(got) != 1 || got[0] != "curl/8.0" {
		t.Fatalf("unexpected headers: %v", headers)
	}
	if metadata.ContentFile != "abc_report.bin" {
		t.Fatalf("unexpected content file: %q", metadata.ContentFile)
	}
}

func TestUploadedMetadataKeepsUnparsableHeaders(t *testing.T) {
	metadata := uploadedMetadataOf(&clientpb.WebUploadedContent{
		Headers: "not json",
	}, "abc_report.bin")

	headers, ok := metadata.Headers.(string)
	if !ok {
		t.Fatalf("headers should have been kept as a string, got %T", metadata.Headers)
	}
	if headers != "not json" {
		t.Fatalf("unexpected headers: %q", headers)
	}
}

func TestUploadedRemoveConfirmationPrompt(t *testing.T) {
	tests := []struct {
		name         string
		websiteName  string
		contentCount int
		want         string
	}{
		{
			name:         "single upload",
			websiteName:  "test",
			contentCount: 1,
			want:         "Delete 1 upload received by 'test'?",
		},
		{
			name:         "multiple uploads",
			websiteName:  "test",
			contentCount: 2,
			want:         "Delete 2 uploads received by 'test'?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uploadedRemoveConfirmationPrompt(tt.websiteName, tt.contentCount)
			if got != tt.want {
				t.Fatalf("unexpected prompt: got %q want %q", got, tt.want)
			}
		})
	}
}

func TestUploadedRemoveSuccessMessage(t *testing.T) {
	got := uploadedRemoveSuccessMessage("test", 2)
	want := "Removed 2 uploads from test"
	if got != want {
		t.Fatalf("unexpected message: got %q want %q", got, want)
	}
}
