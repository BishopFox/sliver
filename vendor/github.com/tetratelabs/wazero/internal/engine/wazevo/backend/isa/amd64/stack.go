package amd64

import (
	"encoding/binary"
	"reflect"
	"unsafe"

	"github.com/tetratelabs/wazero/internal/wasmdebug"
)

func stackView(rbp, top uintptr) []byte {
	var stackBuf []byte
	{
		// TODO: use unsafe.Slice after floor version is set to Go 1.20.
		hdr := (*reflect.SliceHeader)(unsafe.Pointer(&stackBuf))
		hdr.Data = rbp
		setSliceLimits(hdr, top-rbp)
	}
	return stackBuf
}

// UnwindStack implements wazevo.unwindStack.
func UnwindStack(_, rbp, top uintptr, returnAddresses []uintptr) []uintptr {
	stackBuf := stackView(rbp, top)

	for i := uint64(0); i < uint64(len(stackBuf)); {
		//       (high address)
		//    +-----------------+
		//    |     .......     |
		//    |      ret Y      |
		//    |     .......     |
		//    |      ret 0      |
		//    |      arg X      |
		//    |     .......     |
		//    |      arg 1      |
		//    |      arg 0      |
		//    |  ReturnAddress  |
		//    |   Caller_RBP    |
		//    +-----------------+ <---- Caller_RBP
		//    |   ...........   |
		//    |   clobbered  M  |
		//    |   ............  |
		//    |   clobbered  0  |
		//    |   spill slot N  |
		//    |   ............  |
		//    |   spill slot 0  |
		//    |  ReturnAddress  |
		//    |   Caller_RBP    |
		//    +-----------------+ <---- RBP
		//       (low address)

		callerRBP := binary.LittleEndian.Uint64(stackBuf[i:])
		retAddr := binary.LittleEndian.Uint64(stackBuf[i+8:])
		returnAddresses = append(returnAddresses, uintptr(retAddr))
		i = callerRBP - uint64(rbp)
		if len(returnAddresses) == wasmdebug.MaxFrames {
			break
		}
	}
	return returnAddresses
}

// GoCallStackView implements wazevo.goCallStackView.
func GoCallStackView(stackPointerBeforeGoCall *uint64) []uint64 {
	//                  (high address)
	//              +-----------------+ <----+
	//              |   xxxxxxxxxxx   |      | ;; optional unused space to make it 16-byte aligned.
	//           ^  |  arg[N]/ret[M]  |      |
	// sliceSize |  |  ............   |      | SizeInBytes/8
	//           |  |  arg[1]/ret[1]  |      |
	//           v  |  arg[0]/ret[0]  | <----+
	//              |   SizeInBytes   |
	//              +-----------------+ <---- stackPointerBeforeGoCall
	//                 (low address)
	data := unsafe.Pointer(uintptr(unsafe.Pointer(stackPointerBeforeGoCall)) + 8)
	size := *stackPointerBeforeGoCall / 8
	return unsafe.Slice((*uint64)(data), int(size))
}

func AdjustClonedStack(oldRsp, oldTop, rsp, rbp, top uintptr) {
	diff := uint64(rsp - oldRsp)

	newBuf := stackView(rbp, top)
	for i := uint64(0); i < uint64(len(newBuf)); {
		//       (high address)
		//    +-----------------+
		//    |     .......     |
		//    |      ret Y      |
		//    |     .......     |
		//    |      ret 0      |
		//    |      arg X      |
		//    |     .......     |
		//    |      arg 1      |
		//    |      arg 0      |
		//    |  ReturnAddress  |
		//    |   Caller_RBP    |
		//    +-----------------+ <---- Caller_RBP
		//    |   ...........   |
		//    |   clobbered  M  |
		//    |   ............  |
		//    |   clobbered  0  |
		//    |   spill slot N  |
		//    |   ............  |
		//    |   spill slot 0  |
		//    |  ReturnAddress  |
		//    |   Caller_RBP    |
		//    +-----------------+ <---- RBP
		//       (low address)

		callerRBP := binary.LittleEndian.Uint64(newBuf[i:])
		if callerRBP == 0 {
			// End of stack.
			break
		}
		if i64 := int64(callerRBP); i64 < int64(oldRsp) || i64 >= int64(oldTop) {
			panic("BUG: callerRBP is out of range")
		}
		if int(callerRBP) < 0 {
			panic("BUG: callerRBP is negative")
		}
		adjustedCallerRBP := callerRBP + diff
		if int(adjustedCallerRBP) < 0 {
			panic("BUG: adjustedCallerRBP is negative")
		}
		binary.LittleEndian.PutUint64(newBuf[i:], adjustedCallerRBP)
		i = adjustedCallerRBP - uint64(rbp)
	}
}
