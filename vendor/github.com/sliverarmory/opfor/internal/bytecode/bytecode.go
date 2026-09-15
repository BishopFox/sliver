// Package bytecode defines OPFOR's resumable control-flow instruction stream.
// Expressions remain typed AST operands in the first compiler generation;
// fibers still own an explicit program counter, loop state, and exception
// stack, which makes yield and asynchronous callback resumption deterministic.
package bytecode

import (
	"github.com/sliverarmory/opfor/internal/ast"
	"github.com/sliverarmory/opfor/internal/lexer"
)

// Op identifies a VM instruction.
type Op uint8

const (
	OpEval Op = iota
	OpJump
	OpJumpFalse
	OpAssignWhile
	OpIterInit
	OpIterNext
	OpIterDestroy
	OpEnterTry
	OpLeaveTry
	OpCatch
	OpReturn
	OpBreak
	OpContinue
	OpYield
	OpThrow
	OpAssert
	OpBind
	OpImport
	OpHalt
	OpDone
	OpCallCC
	OpEnd
)

func (op Op) String() string {
	switch op {
	case OpEval:
		return "eval"
	case OpJump:
		return "jump"
	case OpJumpFalse:
		return "jump-false"
	case OpAssignWhile:
		return "assign-while"
	case OpIterInit:
		return "iter-init"
	case OpIterNext:
		return "iter-next"
	case OpIterDestroy:
		return "iter-destroy"
	case OpEnterTry:
		return "enter-try"
	case OpLeaveTry:
		return "leave-try"
	case OpCatch:
		return "catch"
	case OpReturn:
		return "return"
	case OpBreak:
		return "break"
	case OpContinue:
		return "continue"
	case OpYield:
		return "yield"
	case OpThrow:
		return "throw"
	case OpAssert:
		return "assert"
	case OpBind:
		return "bind"
	case OpImport:
		return "import"
	case OpHalt:
		return "halt"
	case OpDone:
		return "done"
	case OpCallCC:
		return "callcc"
	case OpEnd:
		return "end"
	default:
		return "unknown"
	}
}

// Instruction is one control-flow operation. Fields are interpreted according
// to Op and intentionally remain explicit for readable compiler golden tests.
type Instruction struct {
	Op   Op
	Span lexer.Span

	Expr    ast.Expr
	Message ast.Expr
	Target  int
	// ClearResult is valid for OpJump and clears the fiber's implicit last
	// expression before transferring. Lexical break/continue use it because
	// Sleep's Return/Goto path carries a null result; ordinary jumps leave the
	// result untouched.
	ClearResult bool

	Name        string
	Name2       string
	Keyword     string
	Environment ast.EnvironmentForm
	Selectors   []ast.Selector
	Predicate   ast.Expr
	Body        *Function

	ImportTarget string
	ImportFrom   ast.Expr
}

// BlockRecovery preserves the exception boundary imposed by Sleep's
// engine.Block. Java bridge exceptions are reported as warnings by the
// innermost Block and execution resumes in the Step which owns that Block.
// The compiler flattens those Steps into one instruction stream, so the VM
// needs this small amount of structural metadata to recover at the same
// boundary.
//
// Start is inclusive and End is exclusive. Target is the next instruction to
// execute after the warning. TryDepth and IteratorDepth are the execution
// stack depths which remain live at Target.
type BlockRecovery struct {
	Start         int
	End           int
	Target        int
	TryDepth      int
	IteratorDepth int
}

// LoopRecovery records the dynamic boundary owned by one Sleep Goto Step.
// Break applies throughout the condition/body region. Continue applies only
// to the true-body Block; after a continue, the condition/increment runs
// outside that continue-catching boundary, exactly as Goto.evaluate does.
//
// All ranges are half-open. TryDepth and IteratorDepth are the execution stack
// depths retained while transferring to either target.
type LoopRecovery struct {
	Start          int
	End            int
	BodyStart      int
	BodyEnd        int
	BreakTarget    int
	ContinueTarget int
	TryDepth       int
	IteratorDepth  int
}

// Function is an immutable instruction stream for a script body or closure.
type Function struct {
	Name            string
	Span            lexer.Span
	Instructions    []Instruction
	BlockRecoveries []BlockRecovery
	LoopRecoveries  []LoopRecovery
	// ClosureTemplates holds the immutable executable body compiled for each
	// anonymous closure literal referenced by this function. Evaluating the
	// literal still creates a fresh SleepClosure and state; only compilation is
	// shared.
	ClosureTemplates map[*ast.ClosureExpr]*Function
}
