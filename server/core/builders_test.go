package core

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestRemoveBuilderIfCurrentPreservesReplacement(t *testing.T) {
	first := &clientpb.Builder{Name: "builder-current-test"}
	replacement := &clientpb.Builder{Name: first.Name}
	if err := AddBuilder(first); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { RemoveBuilder(first.Name) })
	if RemoveBuilderIfCurrent(replacement) || GetBuilder(first.Name) != first {
		t.Fatal("unregistered replacement removed active builder")
	}
	if !RemoveBuilderIfCurrent(first) {
		t.Fatal("active builder was not removed")
	}
	if err := AddBuilder(replacement); err != nil {
		t.Fatal(err)
	}
	if RemoveBuilderIfCurrent(first) || GetBuilder(first.Name) != replacement {
		t.Fatal("stale registration removed replacement")
	}
}
