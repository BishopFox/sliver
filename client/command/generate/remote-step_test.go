package generate

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestSelectPinnedExternalBuilder(t *testing.T) {
	builders := []*clientpb.Builder{{Name: "alpha"}, {Name: "bravo"}}
	selected, err := selectPinnedExternalBuilder(builders, "bravo")
	if err != nil || selected.Name != "bravo" {
		t.Fatalf("pinned builder selection: %+v, %v", selected, err)
	}
	if _, err := selectPinnedExternalBuilder(builders, "offline"); err == nil {
		t.Fatal("unknown pinned builder was accepted")
	}
}
