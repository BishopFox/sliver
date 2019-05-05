package pass

import (
	"reflect"
	"sort"
	"testing"

	"github.com/mmcloughlin/avo/ir"
	"github.com/mmcloughlin/avo/operand"
)

func TestLabelTarget(t *testing.T) {
	expect := map[ir.Label]*ir.Instruction{
		"lblA": {Opcode: "A"},
		"lblB": {Opcode: "B"},
	}

	f := ir.NewFunction("happypath")
	for lbl, i := range expect {
		f.AddLabel(lbl)
		f.AddComment("comments should be ignored")
		f.AddInstruction(i)
		f.AddInstruction(&ir.Instruction{Opcode: "IDK"})
	}

	if err := LabelTarget(f); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expect, f.LabelTarget) {
		t.Fatalf("incorrect LabelTarget value\ngot=%#v\nexpext=%#v\n", f.LabelTarget, expect)
	}
}

func TestLabelTargetDuplicate(t *testing.T) {
	f := ir.NewFunction("dupelabel")
	f.AddLabel(ir.Label("lblA"))
	f.AddInstruction(&ir.Instruction{Opcode: "A"})
	f.AddLabel(ir.Label("lblA"))
	f.AddInstruction(&ir.Instruction{Opcode: "A"})

	err := LabelTarget(f)

	if err == nil || err.Error() != "duplicate label \"lblA\"" {
		t.Fatalf("expected error on duplcate label; got %v", err)
	}
}

func TestLabelTargetEndsWithLabel(t *testing.T) {
	f := ir.NewFunction("endswithlabel")
	f.AddInstruction(&ir.Instruction{Opcode: "A"})
	f.AddLabel(ir.Label("theend"))

	err := LabelTarget(f)

	if err == nil || err.Error() != "function ends with label" {
		t.Fatalf("expected error when function ends with label; got %v", err)
	}
}

func TestLabelTargetInstructionFollowLabel(t *testing.T) {
	f := ir.NewFunction("expectinstafterlabel")
	f.AddLabel(ir.Label("lblA"))
	f.AddLabel(ir.Label("lblB"))
	f.AddInstruction(&ir.Instruction{Opcode: "A"})

	err := LabelTarget(f)

	if err == nil || err.Error() != "instruction should follow a label" {
		t.Fatalf("expected error when label is not followed by instruction; got %v", err)
	}
}

func TestCFGSingleBasicBlock(t *testing.T) {
	f := ir.NewFunction("simple")
	f.AddInstruction(&ir.Instruction{Opcode: "A"})
	f.AddInstruction(&ir.Instruction{Opcode: "B"})
	f.AddInstruction(Terminal("RET"))

	if err := ComputeCFG(t, f); err != nil {
		t.Fatal(err)
	}

	AssertSuccessors(t, f, map[string][]string{
		"A":   {"B"},
		"B":   {"RET"},
		"RET": {},
	})

	AssertPredecessors(t, f, map[string][]string{
		"A":   {},
		"B":   {"A"},
		"RET": {"B"},
	})
}

func TestCFGCondBranch(t *testing.T) {
	f := ir.NewFunction("condbranch")
	f.AddInstruction(&ir.Instruction{Opcode: "A"})
	f.AddLabel(ir.Label("lblB"))
	f.AddInstruction(&ir.Instruction{Opcode: "B"})
	f.AddInstruction(&ir.Instruction{Opcode: "C"})
	f.AddInstruction(CondBranch("J", "lblB"))
	f.AddInstruction(Terminal("RET"))

	if err := ComputeCFG(t, f); err != nil {
		t.Fatal(err)
	}

	AssertSuccessors(t, f, map[string][]string{
		"A":   {"B"},
		"B":   {"C"},
		"C":   {"J"},
		"J":   {"B", "RET"},
		"RET": {},
	})
}

func TestCFGUncondBranch(t *testing.T) {
	f := ir.NewFunction("uncondbranch")
	f.AddInstruction(&ir.Instruction{Opcode: "A"})
	f.AddLabel(ir.Label("lblB"))
	f.AddInstruction(&ir.Instruction{Opcode: "B"})
	f.AddInstruction(UncondBranch("JMP", "lblB"))

	if err := ComputeCFG(t, f); err != nil {
		t.Fatal(err)
	}

	AssertSuccessors(t, f, map[string][]string{
		"A":   {"B"},
		"B":   {"JMP"},
		"JMP": {"B"},
	})
}

func TestCFGJumpForward(t *testing.T) {
	f := ir.NewFunction("forward")
	f.AddInstruction(&ir.Instruction{Opcode: "A"})
	f.AddInstruction(CondBranch("J", "done"))
	f.AddInstruction(&ir.Instruction{Opcode: "B"})
	f.AddLabel(ir.Label("done"))
	f.AddInstruction(Terminal("RET"))

	if err := ComputeCFG(t, f); err != nil {
		t.Fatal(err)
	}

	AssertSuccessors(t, f, map[string][]string{
		"A":   {"J"},
		"J":   {"B", "RET"},
		"B":   {"RET"},
		"RET": {},
	})
}

func TestCFGMultiReturn(t *testing.T) {
	f := ir.NewFunction("multireturn")
	f.AddInstruction(&ir.Instruction{Opcode: "A"})
	f.AddInstruction(CondBranch("J", "fork"))
	f.AddInstruction(&ir.Instruction{Opcode: "B"})
	f.AddInstruction(Terminal("RET1"))
	f.AddLabel(ir.Label("fork"))
	f.AddInstruction(&ir.Instruction{Opcode: "C"})
	f.AddInstruction(Terminal("RET2"))

	if err := ComputeCFG(t, f); err != nil {
		t.Fatal(err)
	}

	AssertSuccessors(t, f, map[string][]string{
		"A":    {"J"},
		"J":    {"B", "C"},
		"B":    {"RET1"},
		"RET1": {},
		"C":    {"RET2"},
		"RET2": {},
	})
}

func TestCFGShortLoop(t *testing.T) {
	f := ir.NewFunction("shortloop")
	f.AddLabel(ir.Label("cycle"))
	f.AddInstruction(UncondBranch("JMP", "cycle"))

	if err := ComputeCFG(t, f); err != nil {
		t.Fatal(err)
	}

	AssertSuccessors(t, f, map[string][]string{
		"JMP": {"JMP"},
	})
}

func TestCFGUndefinedLabel(t *testing.T) {
	f := ir.NewFunction("undeflabel")
	f.AddInstruction(&ir.Instruction{Opcode: "A"})
	f.AddInstruction(CondBranch("J", "undef"))

	err := ComputeCFG(t, f)

	if err == nil {
		t.Fatal("expect error on undefined label")
	}
}

func TestCFGMissingLabel(t *testing.T) {
	f := ir.NewFunction("missinglabel")
	f.AddInstruction(&ir.Instruction{Opcode: "A"})
	f.AddInstruction(&ir.Instruction{Opcode: "J", IsBranch: true}) // no label operand
	err := ComputeCFG(t, f)
	if err == nil {
		t.Fatal("expect error on missing label")
	}
}

// Terminal builds a terminal instruction.
func Terminal(opcode string) *ir.Instruction {
	return &ir.Instruction{Opcode: opcode, IsTerminal: true}
}

// CondBranch builds a conditional branch instruction to the given label.
func CondBranch(opcode, lbl string) *ir.Instruction {
	return &ir.Instruction{
		Opcode:        opcode,
		Operands:      []operand.Op{operand.LabelRef(lbl)},
		IsBranch:      true,
		IsConditional: true,
	}
}

// UncondBranch builds an unconditional branch instruction to the given label.
func UncondBranch(opcode, lbl string) *ir.Instruction {
	return &ir.Instruction{
		Opcode:        opcode,
		Operands:      []operand.Op{operand.LabelRef(lbl)},
		IsBranch:      true,
		IsConditional: false,
	}
}

func ComputeCFG(t *testing.T, f *ir.Function) error {
	t.Helper()
	if err := LabelTarget(f); err != nil {
		t.Fatal(err)
	}
	return CFG(f)
}

func AssertSuccessors(t *testing.T, f *ir.Function, expect map[string][]string) {
	AssertEqual(t, "successors", OpcodeSuccessorGraph(f), expect)
}

func AssertPredecessors(t *testing.T, f *ir.Function, expect map[string][]string) {
	AssertEqual(t, "predecessors", OpcodePredecessorGraph(f), expect)
}

func AssertEqual(t *testing.T, what string, got, expect interface{}) {
	t.Logf("%s=%#v\n", what, got)
	if reflect.DeepEqual(expect, got) {
		return
	}
	t.Fatalf("bad %s; expected=%#v", what, expect)
}

// OpcodeSuccessorGraph builds a map from opcode name to successor opcode names.
func OpcodeSuccessorGraph(f *ir.Function) map[string][]string {
	return OpcodeGraph(f, func(i *ir.Instruction) []*ir.Instruction { return i.Succ })
}

// OpcodePredecessorGraph builds a map from opcode name to predecessor opcode names.
func OpcodePredecessorGraph(f *ir.Function) map[string][]string {
	return OpcodeGraph(f, func(i *ir.Instruction) []*ir.Instruction { return i.Pred })
}

// OpcodeGraph builds a map from opcode name to neighboring opcode names. Each list of neighbors is sorted.
func OpcodeGraph(f *ir.Function, neighbors func(*ir.Instruction) []*ir.Instruction) map[string][]string {
	g := map[string][]string{}
	for _, i := range f.Instructions() {
		opcodes := []string{}
		for _, n := range neighbors(i) {
			opcode := "<nil>"
			if n != nil {
				opcode = n.Opcode
			}
			opcodes = append(opcodes, opcode)
		}
		sort.Strings(opcodes)
		g[i.Opcode] = opcodes
	}
	return g
}
