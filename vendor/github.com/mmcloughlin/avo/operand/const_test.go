package operand

import "testing"

func TestConstants(t *testing.T) {
	cases := []struct {
		Const Constant
		Asm   string
		Bytes int
	}{
		{F32(3.1415), "$(3.1415)", 4},
		{F64(3.1415), "$(3.1415)", 8},
		{U8(42), "$0x2a", 1},
		{U16(42), "$0x002a", 2},
		{U32(42), "$0x0000002a", 4},
		{U64(42), "$0x000000000000002a", 8},
		{I8(-42), "$-42", 1},
		{I16(-42), "$-42", 2},
		{I32(-42), "$-42", 4},
		{I64(-42), "$-42", 8},
		{String("hello"), "$\"hello\"", 5},
		{String("quot:\"q\""), "$\"quot:\\\"q\\\"\"", 8},
	}
	for _, c := range cases {
		if c.Const.Asm() != c.Asm {
			t.Errorf("%v.Asm() = %v; expect %v", c.Const, c.Const.Asm(), c.Asm)
		}
		if c.Const.Bytes() != c.Bytes {
			t.Errorf("%v.Bytes() = %v; expect %v", c.Const, c.Const.Bytes(), c.Bytes)
		}
	}
}
