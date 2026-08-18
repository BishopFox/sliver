//go:build !((windows && (amd64 || arm64 || 386)) || (darwin && (arm64 || amd64)) || (linux && (arm64 || amd64 || 386)))

package wasmnet

import (
	"context"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// Host is a no-op network host on platforms where Sliver does not expose the
// networking ABI. Modules that do not import ModuleName continue to run;
// networking-aware modules fail instantiation because the host module is not
// registered.
type Host struct{}

func New(context.Context) *Host {
	return &Host{}
}

func (*Host) Instantiate(context.Context, wazero.Runtime) (api.Module, error) {
	return nil, nil
}

func (*Host) Close() error {
	return nil
}
