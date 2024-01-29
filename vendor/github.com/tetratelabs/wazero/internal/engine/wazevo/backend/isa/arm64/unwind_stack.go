package arm64

import (
	"encoding/binary"
	"reflect"
	"unsafe"
)

// UnwindStack is a function to unwind the stack, and appends return addresses to `returnAddresses` slice.
// The implementation must be aligned with the ABI/Calling convention as in machine_pro_epi_logue.go/abi.go.
func UnwindStack(sp, top uintptr, returnAddresses []uintptr) []uintptr {
	l := int(top - sp)

	var stackBuf []byte
	{
		// TODO: use unsafe.Slice after floor version is set to Go 1.20.
		hdr := (*reflect.SliceHeader)(unsafe.Pointer(&stackBuf))
		hdr.Data = sp
		hdr.Len = l
		hdr.Cap = l
	}

	for i := uint64(0); i < uint64(l); {
		//       (high address)
		//    +-----------------+
		//    |     .......     |
		//    |      ret Y      |  <----+
		//    |     .......     |       |
		//    |      ret 0      |       |
		//    |      arg X      |       |  size_of_arg_ret
		//    |     .......     |       |
		//    |      arg 1      |       |
		//    |      arg 0      |  <----+
		//    | size_of_arg_ret |
		//    |  ReturnAddress  |
		//    +-----------------+ <----+
		//    |   ...........   |      |
		//    |   spill slot M  |      |
		//    |   ............  |      |
		//    |   spill slot 2  |      |
		//    |   spill slot 1  |      | frame size
		//    |   spill slot 1  |      |
		//    |   clobbered N   |      |
		//    |   ............  |      |
		//    |   clobbered 0   | <----+
		//    |     xxxxxx      |  ;; unused space to make it 16-byte aligned.
		//    |   frame_size    |
		//    +-----------------+ <---- SP
		//       (low address)

		frameSize := binary.LittleEndian.Uint64(stackBuf[i:])
		i += frameSize +
			16 // frame size + aligned space.
		retAddr := binary.LittleEndian.Uint64(stackBuf[i:])
		i += 8 // ret addr.
		sizeOfArgRet := binary.LittleEndian.Uint64(stackBuf[i:])
		i += 8 + sizeOfArgRet
		returnAddresses = append(returnAddresses, uintptr(retAddr))
	}
	return returnAddresses
}

// GoCallStackView is a function to get a view of the stack before a Go call, which
// is the view of the stack allocated in CompileGoFunctionTrampoline.
func GoCallStackView(stackPointerBeforeGoCall *uint64) []uint64 {
	//                  (high address)
	//              +-----------------+ <----+
	//              |   xxxxxxxxxxx   |      | ;; optional unused space to make it 16-byte aligned.
	//           ^  |  arg[N]/ret[M]  |      |
	// sliceSize |  |  ............   |      | sliceSize
	//           |  |  arg[1]/ret[1]  |      |
	//           v  |  arg[0]/ret[0]  | <----+
	//              |    sliceSize    |
	//              |   frame_size    |
	//              +-----------------+ <---- stackPointerBeforeGoCall
	//                 (low address)
	size := *(*uint64)(unsafe.Pointer(uintptr(unsafe.Pointer(stackPointerBeforeGoCall)) + 8))
	var view []uint64
	{
		sh := (*reflect.SliceHeader)(unsafe.Pointer(&view))
		sh.Data = uintptr(unsafe.Pointer(stackPointerBeforeGoCall)) + 16 // skips the (frame_size, sliceSize).
		sh.Len = int(size)
		sh.Cap = int(size)
	}
	return view
}
