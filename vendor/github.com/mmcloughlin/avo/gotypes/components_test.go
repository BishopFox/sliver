package gotypes

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/mmcloughlin/avo/reg"

	"github.com/mmcloughlin/avo/operand"
)

func TestBasicKindsArePrimitive(t *testing.T) {
	kinds := []types.BasicKind{
		types.Bool,
		types.Int,
		types.Int8,
		types.Int16,
		types.Int32,
		types.Int64,
		types.Uint,
		types.Uint8,
		types.Uint16,
		types.Uint32,
		types.Uint64,
		types.Uintptr,
		types.Float32,
		types.Float64,
	}
	for _, k := range kinds {
		AssertPrimitive(t, types.Typ[k])
	}
}

func TestPointersArePrimitive(t *testing.T) {
	typ := types.NewPointer(types.Typ[types.Uint32])
	AssertPrimitive(t, typ)
}

func AssertPrimitive(t *testing.T, typ types.Type) {
	c := NewComponent(typ, operand.NewParamAddr("primitive", 0))
	if _, err := c.Resolve(); err != nil {
		t.Errorf("expected type %s to be primitive: got error '%s'", typ, err)
	}
}

func TestComponentErrors(t *testing.T) {
	comp := NewComponent(types.Typ[types.Uint32], operand.Mem{})
	cases := []struct {
		Component      Component
		ErrorSubstring string
	}{
		{comp.Base(), "only slices and strings"},
		{comp.Len(), "only slices and strings"},
		{comp.Cap(), "only slices have"},
		{comp.Real(), "only complex"},
		{comp.Imag(), "only complex"},
		{comp.Index(42), "not array type"},
		{comp.Field("a"), "not struct type"},
		{comp.Dereference(reg.R12), "not pointer type"},
	}
	for _, c := range cases {
		_, err := c.Component.Resolve()
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), c.ErrorSubstring) {
			t.Fatalf("error message %q; expected substring %q", err.Error(), c.ErrorSubstring)
		}
	}
}

func TestComponentErrorChaining(t *testing.T) {
	// Build a component with an error.
	comp := NewComponent(types.Typ[types.Uint32], operand.Mem{}).Index(3)
	_, expect := comp.Resolve()
	if expect == nil {
		t.Fatal("expected error")
	}

	// Confirm that the error is preserved through chaining operations.
	cases := []Component{
		comp.Dereference(reg.R13),
		comp.Base(),
		comp.Len(),
		comp.Cap(),
		comp.Real(),
		comp.Imag(),
		comp.Index(42),
		comp.Field("field"),
	}
	for _, c := range cases {
		_, err := c.Resolve()
		if err != expect {
			t.Fatal("chaining should preserve error")
		}
	}
}

func TestComponentDeconstruction(t *testing.T) {
	cases := []struct {
		Name   string
		Type   types.Type
		Chain  func(Component) Component
		Param  string
		Offset int
	}{
		{
			Name:   "slice_base",
			Type:   types.NewSlice(types.Typ[types.Uint64]),
			Chain:  func(c Component) Component { return c.Base() },
			Param:  "base",
			Offset: 0,
		},
		{
			Name:   "slice_len",
			Type:   types.NewSlice(types.Typ[types.Uint64]),
			Chain:  func(c Component) Component { return c.Len() },
			Param:  "len",
			Offset: 8,
		},
		{
			Name:   "slice_cap",
			Type:   types.NewSlice(types.Typ[types.Uint64]),
			Chain:  func(c Component) Component { return c.Cap() },
			Param:  "cap",
			Offset: 16,
		},
		{
			Name:   "string_base",
			Type:   types.Typ[types.String],
			Chain:  func(c Component) Component { return c.Base() },
			Param:  "base",
			Offset: 0,
		},
		{
			Name:   "slice_len",
			Type:   types.Typ[types.String],
			Chain:  func(c Component) Component { return c.Len() },
			Param:  "len",
			Offset: 8,
		},
		{
			Name:   "complex64_real",
			Type:   types.Typ[types.Complex64],
			Chain:  func(c Component) Component { return c.Real() },
			Param:  "real",
			Offset: 0,
		},
		{
			Name:   "complex64_imag",
			Type:   types.Typ[types.Complex64],
			Chain:  func(c Component) Component { return c.Imag() },
			Param:  "imag",
			Offset: 4,
		},
		{
			Name:   "complex128_real",
			Type:   types.Typ[types.Complex128],
			Chain:  func(c Component) Component { return c.Real() },
			Param:  "real",
			Offset: 0,
		},
		{
			Name:   "complex128_imag",
			Type:   types.Typ[types.Complex128],
			Chain:  func(c Component) Component { return c.Imag() },
			Param:  "imag",
			Offset: 8,
		},
		{
			Name:   "array",
			Type:   types.NewArray(types.Typ[types.Uint32], 7),
			Chain:  func(c Component) Component { return c.Index(3) },
			Param:  "3",
			Offset: 12,
		},
		{
			Name: "struct",
			Type: types.NewStruct([]*types.Var{
				types.NewField(token.NoPos, nil, "Byte", types.Typ[types.Byte], false),
				types.NewField(token.NoPos, nil, "Uint64", types.Typ[types.Uint64], false),
			}, nil),
			Chain:  func(c Component) Component { return c.Field("Uint64") },
			Param:  "Uint64",
			Offset: 8,
		},
	}

	// For every test case, generate the same case but when the type is wrapped in
	// a named type.
	n := len(cases)
	for i := 0; i < n; i++ {
		wrapped := cases[i]
		wrapped.Name += "_wrapped"
		wrapped.Type = types.NewNamed(
			types.NewTypeName(token.NoPos, nil, "wrapped", nil),
			wrapped.Type,
			nil,
		)
		cases = append(cases, wrapped)
	}

	for _, c := range cases {
		c := c // avoid scopelint error
		t.Run(c.Name, func(t *testing.T) {
			t.Log(c.Type)
			base := operand.NewParamAddr("test", 0)
			comp := NewComponent(c.Type, base)
			comp = c.Chain(comp)

			b, err := comp.Resolve()
			if err != nil {
				t.Fatal(err)
			}

			expectname := "test_" + c.Param
			if b.Addr.Symbol.Name != expectname {
				t.Errorf("parameter name %q; expected %q", b.Addr.Symbol.Name, expectname)
			}

			if b.Addr.Disp != c.Offset {
				t.Errorf("offset %d; expected %d", b.Addr.Disp, c.Offset)
			}
		})
	}
}
