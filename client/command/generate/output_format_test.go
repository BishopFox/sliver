package generate

import (
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestNameOfOutputFormatGoArchive(t *testing.T) {
	if got := nameOfOutputFormat(clientpb.OutputFormat_GO_ARCHIVE); got != "Go Archive" {
		t.Fatalf("nameOfOutputFormat(GO_ARCHIVE) = %q, want %q", got, "Go Archive")
	}
}

func TestSelectOutputFormat(t *testing.T) {
	tests := []struct {
		name string
		want outputFormatSelection
	}{
		{"exe", outputFormatSelection{format: clientpb.OutputFormat_EXECUTABLE}},
		{"shared", outputFormatSelection{format: clientpb.OutputFormat_SHARED_LIB, shared: true}},
		{"archive", outputFormatSelection{format: clientpb.OutputFormat_GO_ARCHIVE, shared: true}},
		{"service", outputFormatSelection{format: clientpb.OutputFormat_SERVICE, service: true}},
		{"shellcode", outputFormatSelection{format: clientpb.OutputFormat_SHELLCODE, shellcode: true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := selectOutputFormat(test.name)
			if err != nil || got != test.want {
				t.Fatalf("selectOutputFormat(%q) = %+v, %v; want %+v", test.name, got, err, test.want)
			}
		})
	}
	_, err := selectOutputFormat("typo")
	if err == nil || !strings.Contains(err.Error(), "no core capability") {
		t.Fatalf("unknown output format should return a capability error, got %v", err)
	}
}
