package wazevo

import (
	"runtime"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend/isa/amd64"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend/isa/arm64"
)

func newMachine() backend.Machine {
	switch runtime.GOARCH {
	case "arm64":
		return arm64.NewBackend()
	case "amd64":
		return amd64.NewBackend()
	default:
		panic("unsupported architecture")
	}
}

// unwindStack is a function to unwind the stack, and appends return addresses to `returnAddresses` slice.
// The implementation must be aligned with the ABI/Calling convention.
func unwindStack(sp, fp, top uintptr, returnAddresses []uintptr) []uintptr {
	switch runtime.GOARCH {
	case "arm64":
		return arm64.UnwindStack(sp, fp, top, returnAddresses)
	case "amd64":
		return amd64.UnwindStack(sp, fp, top, returnAddresses)
	default:
		panic("unsupported architecture")
	}
}

// goCallStackView is a function to get a view of the stack before a Go call, which
// is the view of the stack allocated in CompileGoFunctionTrampoline.
func goCallStackView(stackPointerBeforeGoCall *uint64) []uint64 {
	switch runtime.GOARCH {
	case "arm64":
		return arm64.GoCallStackView(stackPointerBeforeGoCall)
	case "amd64":
		return amd64.GoCallStackView(stackPointerBeforeGoCall)
	default:
		panic("unsupported architecture")
	}
}

// adjustClonedStack is a function to adjust the stack after it is grown.
// More precisely, absolute addresses (frame pointers) in the stack must be adjusted.
func adjustClonedStack(oldsp, oldTop, sp, fp, top uintptr) {
	switch runtime.GOARCH {
	case "arm64":
		// TODO: currently, the frame pointers are not used, and saved old sps are relative to the current stack pointer,
		//  so no need to adjustment on arm64. However, when we make it absolute, which in my opinion is better perf-wise
		//  at the expense of slightly costly stack growth, we need to adjust the pushed frame pointers.
	case "amd64":
		amd64.AdjustClonedStack(oldsp, oldTop, sp, fp, top)
	default:
		panic("unsupported architecture")
	}
}
