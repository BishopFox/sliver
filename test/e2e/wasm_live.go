package e2e

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// wasmExitSeven is a minimal WASI module. Its _start calls
// wasi_snapshot_preview1.proc_exit(7). A distinctive exit code proves that
// the implant decoded, compiled, and ran the registered module.
var wasmExitSeven = [...]byte{
	0, 97, 115, 109, 1, 0, 0, 0,
	1, 8, 2, 96, 1, 127, 0, 96, 0, 0,
	2, 36, 1, 22, 119, 97, 115, 105, 95, 115, 110, 97, 112, 115,
	104, 111, 116, 95, 112, 114, 101, 118, 105, 101, 119, 49,
	9, 112, 114, 111, 99, 95, 101, 120, 105, 116, 0, 0,
	3, 2, 1, 1,
	7, 10, 1, 6, 95, 115, 116, 97, 114, 116, 0, 1,
	10, 8, 1, 6, 0, 65, 7, 16, 0, 11,
}

func gzipWasmExitSeven() ([]byte, error) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(wasmExitSeven[:]); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return compressed.Bytes(), nil
}

func (s *suite) liveWasmStep(target implantTarget, transport, method string, fn func() error) error {
	started := time.Now()
	if err := fn(); err != nil {
		return fmt.Errorf("%s (%s/%s %s/%s, %s): %w", method,
			s.opts.targetOS, s.opts.targetArch, transport, target.mode(),
			time.Since(started).Round(time.Millisecond), err)
	}
	s.t.Logf("PASS live %s/%s %s %s %s (%s)", s.opts.targetOS,
		s.opts.targetArch, transport, target.mode(), method,
		time.Since(started).Round(time.Millisecond))
	return nil
}

func (s *suite) exerciseWasmExtensionLifecycle(target implantTarget, transport string) error {
	const name = "sliver-e2e-exit-seven.wasm"
	compressed, err := gzipWasmExitSeven()
	if err != nil {
		return fmt.Errorf("compress deterministic WASM fixture: %w", err)
	}
	if err := s.liveWasmStep(target, transport, "RegisterWasmExtension", func() error {
		_, err := invokeRPC(s, target, "RegisterWasmExtension", func(ctx context.Context, request *commonpb.Request) (*sliverpb.RegisterWasmExtension, error) {
			return s.rpc.RegisterWasmExtension(ctx, &sliverpb.RegisterWasmExtensionReq{
				Name: name, WasmGz: compressed, Request: request,
			})
		}, func(response *sliverpb.RegisterWasmExtension) *commonpb.Response {
			return response.GetResponse()
		})
		return err
	}); err != nil {
		return err
	}
	if err := s.liveWasmStep(target, transport, "ListWasmExtensions", func() error {
		response, err := invokeRPC(s, target, "ListWasmExtensions", func(ctx context.Context, request *commonpb.Request) (*sliverpb.ListWasmExtensions, error) {
			return s.rpc.ListWasmExtensions(ctx, &sliverpb.ListWasmExtensionsReq{Request: request})
		}, func(response *sliverpb.ListWasmExtensions) *commonpb.Response {
			return response.GetResponse()
		})
		if err != nil {
			return err
		}
		if len(response.Names) != 1 || response.Names[0] != name {
			return fmt.Errorf("registered WASM inventory = %v, want [%s]", response.Names, name)
		}
		return nil
	}); err != nil {
		return err
	}
	// Beacon tasks use the standard handler table. WASM execution is a tunnel
	// handler, so only a session can exercise this execution path today.
	if target.beacon != nil {
		s.t.Logf("SKIP live %s/%s %s beacon ExecWasmExtension: beacon tasks do not dispatch tunnel handlers",
			s.opts.targetOS, s.opts.targetArch, transport)
	} else if err := s.liveWasmStep(target, transport, "ExecWasmExtension", func() error {
		response, err := invokeRPC(s, target, "ExecWasmExtension", func(ctx context.Context, request *commonpb.Request) (*sliverpb.ExecWasmExtension, error) {
			return s.rpc.ExecWasmExtension(ctx, &sliverpb.ExecWasmExtensionReq{
				Name: name, Request: request,
			})
		}, func(response *sliverpb.ExecWasmExtension) *commonpb.Response {
			return response.GetResponse()
		})
		if err != nil {
			return err
		}
		if response.ExitCode != 7 {
			return fmt.Errorf("WASM exit code = %d, want 7", response.ExitCode)
		}
		return nil
	}); err != nil {
		return err
	}
	return s.liveWasmStep(target, transport, "Ping after WASM lifecycle", func() error {
		const nonce = int32(0x5a17c0de)
		response, err := invokeRPC(s, target, "Ping", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Ping, error) {
			return s.rpc.Ping(ctx, &sliverpb.Ping{Nonce: nonce, Request: request})
		}, func(response *sliverpb.Ping) *commonpb.Response {
			return response.GetResponse()
		})
		if err != nil {
			return err
		}
		if response.Nonce != nonce {
			return fmt.Errorf("Ping nonce = %d, want %d", response.Nonce, nonce)
		}
		return nil
	})
}
