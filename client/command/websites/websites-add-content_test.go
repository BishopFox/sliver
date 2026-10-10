package websites

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"google.golang.org/protobuf/proto"
)

func TestWebAddFileMarksContentReplacement(t *testing.T) {
	for _, data := range [][]byte{nil, []byte("replacement")} {
		name := "nonempty"
		if len(data) == 0 {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "upload.txt")
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			request := &clientpb.WebsiteAddContent{Contents: map[string]*clientpb.WebContent{}}
			if err := webAddFile(request, "/upload.txt", "text/plain", path); err != nil {
				t.Fatal(err)
			}
			encoded, err := proto.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			decoded := &clientpb.WebsiteAddContent{}
			if err := proto.Unmarshal(encoded, decoded); err != nil {
				t.Fatal(err)
			}
			content := decoded.Contents["/upload.txt"]
			if content == nil || !content.ReplaceContent || !bytes.Equal(content.Content, data) {
				t.Fatalf("decoded upload = %v, want explicit replacement with %q", content, data)
			}
		})
	}
}
