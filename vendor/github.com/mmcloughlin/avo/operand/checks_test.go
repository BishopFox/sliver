package operand

import (
	"math"
	"reflect"
	"runtime"
	"testing"

	"github.com/mmcloughlin/avo/reg"
)

func TestChecks(t *testing.T) {
	cases := []struct {
		Predicate func(Op) bool
		Operand   Op
		Expect    bool
	}{
		// Immediates
		{Is1, Imm(1), true},
		{Is1, Imm(23), false},

		{Is3, Imm(3), true},
		{Is3, Imm(23), false},

		{IsIMM2U, Imm(3), true},
		{IsIMM2U, Imm(4), false},

		{IsIMM8, Imm(255), true},
		{IsIMM8, Imm(256), false},

		{IsIMM16, Imm((1 << 16) - 1), true},
		{IsIMM16, Imm(1 << 16), false},

		{IsIMM32, Imm((1 << 32) - 1), true},
		{IsIMM32, Imm(1 << 32), false},

		{IsIMM64, Imm((1 << 64) - 1), true},

		// Specific registers
		{IsAL, reg.AL, true},
		{IsAL, reg.CL, false},

		{IsCL, reg.CL, true},
		{IsCL, reg.DH, false},

		{IsAX, reg.AX, true},
		{IsAX, reg.DX, false},

		{IsEAX, reg.EAX, true},
		{IsEAX, reg.ECX, false},

		{IsRAX, reg.RAX, true},
		{IsRAX, reg.R13, false},

		// General-purpose registers
		{IsR8, reg.AL, true},
		{IsR8, reg.CH, true},
		{IsR8, reg.EAX, false},

		{IsR16, reg.DX, true},
		{IsR16, reg.R10W, true},
		{IsR16, reg.R10B, false},

		{IsR32, reg.EBP, true},
		{IsR32, reg.R14L, true},
		{IsR32, reg.R8, false},

		{IsR64, reg.RDX, true},
		{IsR64, reg.R10, true},
		{IsR64, reg.EBX, false},

		// Vector registers
		{IsXMM0, reg.X0, true},
		{IsXMM0, reg.X13, false},
		{IsXMM0, reg.Y3, false},

		{IsXMM, reg.X0, true},
		{IsXMM, reg.X13, true},
		{IsXMM, reg.Y3, false},
		{IsXMM, reg.Z23, false},

		{IsYMM, reg.Y0, true},
		{IsYMM, reg.Y13, true},
		{IsYMM, reg.Y31, true},
		{IsYMM, reg.X3, false},
		{IsYMM, reg.Z3, false},

		// Pseudo registers.
		{IsPseudo, reg.FramePointer, true},
		{IsPseudo, reg.ProgramCounter, true},
		{IsPseudo, reg.StaticBase, true},
		{IsPseudo, reg.StackPointer, true},
		{IsPseudo, reg.ECX, false},
		{IsPseudo, reg.X9, false},

		// Memory operands
		{IsM, Mem{Base: reg.CX}, true},
		{IsM, Mem{Base: reg.ECX}, true},
		{IsM, Mem{Base: reg.RCX}, true},
		{IsM, Mem{Base: reg.X0}, false},

		{IsM8, Mem{Disp: 8, Base: reg.CL}, true},
		{IsM8, Mem{Disp: 8, Base: reg.CL, Index: reg.AH, Scale: 2}, true},
		{IsM8, Mem{Disp: 8, Base: reg.X0, Index: reg.AH, Scale: 2}, false},
		{IsM8, Mem{Disp: 8, Base: reg.CL, Index: reg.X0, Scale: 2}, false},

		{IsM16, Mem{Disp: 4, Base: reg.DX}, true},
		{IsM16, Mem{Disp: 4, Base: reg.R13W, Index: reg.R8W, Scale: 2}, true},
		{IsM16, Mem{Disp: 4, Base: reg.X0, Index: reg.R8W, Scale: 2}, false},
		{IsM16, Mem{Disp: 4, Base: reg.R13W, Index: reg.X0, Scale: 2}, false},

		{IsM32, Mem{Base: reg.R13L, Index: reg.EBX, Scale: 2}, true},
		{IsM32, Mem{Base: reg.X0}, false},

		{IsM64, Mem{Base: reg.RBX, Index: reg.R12, Scale: 2}, true},
		{IsM64, Mem{Base: reg.X0}, false},

		{IsM128, Mem{Base: reg.RBX, Index: reg.R12, Scale: 2}, true},
		{IsM128, Mem{Base: reg.X0}, false},

		{IsM256, Mem{Base: reg.RBX, Index: reg.R12, Scale: 2}, true},
		{IsM256, Mem{Base: reg.X0}, false},

		// Argument references (special cases of memory operands)
		{IsM, NewParamAddr("foo", 4), true},
		{IsM8, NewParamAddr("foo", 4), true},
		{IsM16, NewParamAddr("foo", 4), true},
		{IsM32, NewParamAddr("foo", 4), true},
		{IsM64, NewParamAddr("foo", 4), true},

		// Vector memory operands
		{IsVM32X, Mem{Base: reg.R14, Index: reg.X11}, true},
		{IsVM32X, Mem{Base: reg.R14L, Index: reg.X11}, false},
		{IsVM32X, Mem{Base: reg.R14, Index: reg.Y11}, false},

		{IsVM64X, Mem{Base: reg.R14, Index: reg.X11}, true},
		{IsVM64X, Mem{Base: reg.R14L, Index: reg.X11}, false},
		{IsVM64X, Mem{Base: reg.R14, Index: reg.Y11}, false},

		{IsVM32Y, Mem{Base: reg.R9, Index: reg.Y11}, true},
		{IsVM32Y, Mem{Base: reg.R11L, Index: reg.Y11}, false},
		{IsVM32Y, Mem{Base: reg.R8, Index: reg.Z11}, false},

		{IsVM64Y, Mem{Base: reg.R9, Index: reg.Y11}, true},
		{IsVM64Y, Mem{Base: reg.R11L, Index: reg.Y11}, false},
		{IsVM64Y, Mem{Base: reg.R8, Index: reg.Z11}, false},

		// Relative operands
		{IsREL8, Rel(math.MinInt8), true},
		{IsREL8, Rel(math.MaxInt8), true},
		{IsREL8, Rel(math.MinInt8 - 1), false},
		{IsREL8, Rel(math.MaxInt8 + 1), false},
		{IsREL8, reg.R9B, false},

		{IsREL32, Rel(math.MinInt32), true},
		{IsREL32, Rel(math.MaxInt32), true},
		{IsREL32, LabelRef("label"), true},
		{IsREL32, reg.R9L, false},
	}

	for _, c := range cases {
		if c.Predicate(c.Operand) != c.Expect {
			t.Errorf("%s( %#v ) != %v", funcname(c.Predicate), c.Operand, c.Expect)
		}
	}
}

func funcname(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}
