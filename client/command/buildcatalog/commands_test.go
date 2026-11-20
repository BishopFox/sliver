package buildcatalog

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestCatalogCommandFilterFlags(t *testing.T) {
	root := catalogCommand("cores", clientpb.BuildSpecKind_BUILD_SPEC_CORE, nil)
	if err := root.PersistentFlags().Set("target", "linux/amd64"); err != nil {
		t.Fatal(err)
	}
	if err := root.PersistentFlags().Set("availability", "unavailable"); err != nil {
		t.Fatal(err)
	}
	list, _, err := root.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	query, err := queryFromFlags(list, clientpb.BuildSpecKind_BUILD_SPEC_CORE)
	if err != nil {
		t.Fatal(err)
	}
	if query.TargetGOOS != "linux" || query.TargetGOARCH != "amd64" || query.Availability != clientpb.BuildSpecAvailability_BUILD_SPEC_UNAVAILABLE {
		t.Fatalf("inherited filters were not applied: %+v", query)
	}
	if err := root.PersistentFlags().Set("target", "linux"); err != nil {
		t.Fatal(err)
	}
	if _, err := queryFromFlags(list, clientpb.BuildSpecKind_BUILD_SPEC_CORE); err == nil {
		t.Fatal("expected a malformed target filter to fail")
	}
}
