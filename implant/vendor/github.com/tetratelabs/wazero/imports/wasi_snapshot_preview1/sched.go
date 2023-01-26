package wasi_snapshot_preview1

import (
	. "github.com/tetratelabs/wazero/internal/wasi_snapshot_preview1"
)

// schedYield is the WASI function named SchedYieldName which temporarily
// yields execution of the calling thread.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-sched_yield---errno
var schedYield = stubFunction(SchedYieldName, nil)
