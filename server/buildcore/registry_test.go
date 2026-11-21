package buildcore

import (
	"context"
	"testing"

	"github.com/bishopfox/sliver/server/buildplan"
)

type testAdapter struct{ id string }

func (adapter testAdapter) ExecutorID() string { return adapter.id }
func (adapter testAdapter) Build(context.Context, buildplan.Plan) (Result, error) {
	return Result{FileName: "example.exe", Data: []byte("example")}, nil
}

func TestRegistryIsolatesExecutorIDs(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(testAdapter{id: "example-v1"}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(testAdapter{id: "other-v1"}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(testAdapter{id: "example-v1"}); err == nil {
		t.Fatal("duplicate executor ID was accepted")
	}
	if _, ok := registry.Get("missing"); ok {
		t.Fatal("unknown executor ID resolved")
	}
	adapter, ok := registry.Get("example-v1")
	if !ok {
		t.Fatal("registered executor ID did not resolve")
	}
	result, err := adapter.Build(context.Background(), buildplan.Plan{})
	if err != nil || result.FileName != "example.exe" {
		t.Fatalf("wrong adapter result: %+v, %v", result, err)
	}
	ids := registry.InstalledExecutorIDs()
	if len(ids) != 2 || ids[0] != "example-v1" || ids[1] != "other-v1" {
		t.Fatalf("executor IDs are not deterministic: %v", ids)
	}
}
