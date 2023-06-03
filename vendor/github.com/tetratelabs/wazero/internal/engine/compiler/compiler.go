package compiler

import (
	"github.com/tetratelabs/wazero/internal/asm"
	"github.com/tetratelabs/wazero/internal/wasm"
	"github.com/tetratelabs/wazero/internal/wazeroir"
)

// compiler is the interface of architecture-specific native code compiler,
// and this is responsible for compiling native code for all wazeroir operations.
type compiler interface {
	Init(functionType *wasm.FunctionType, ir *wazeroir.CompilationResult, withListener bool)

	// String is for debugging purpose.
	String() string
	// compilePreamble is called before compiling any wazeroir operation.
	// This is used, for example, to initialize the reserved registers, etc.
	compilePreamble() error
	// compile generates the native code into buf.
	// stackPointerCeil is the max stack pointer that the target function would reach.
	compile(buf asm.Buffer) (stackPointerCeil uint64, err error)
	// compileGoHostFunction adds the trampoline code from which native code can jump into the Go-defined host function.
	// TODO: maybe we wouldn't need to have trampoline for host functions.
	compileGoDefinedHostFunction() error
	// compileLabel notify compilers of the beginning of a label.
	// Return true if the compiler decided to skip the entire label.
	// See wazeroir.NewOperationLabel
	compileLabel(o *wazeroir.UnionOperation) (skipThisLabel bool)
	// compileUnreachable adds instruction to perform wazeroir.OperationUnreachable.
	compileUnreachable() error
	// compileSet adds instruction to perform wazeroir.OperationSet.
	compileSet(o *wazeroir.UnionOperation) error
	// compileGlobalGet adds instructions to perform wazeroir.OperationGlobalGet.
	compileGlobalGet(o *wazeroir.UnionOperation) error
	// compileGlobalSet adds instructions to perform wazeroir.OperationGlobalSet.
	compileGlobalSet(o *wazeroir.UnionOperation) error
	// compileBr adds instructions to perform wazeroir.NewOperationBr.
	compileBr(o *wazeroir.UnionOperation) error
	// compileBrIf adds instructions to perform wazeroir.NewOperationBrIf.
	compileBrIf(o *wazeroir.UnionOperation) error
	// compileBrTable adds instructions to perform wazeroir.NewOperationBrTable.
	compileBrTable(o *wazeroir.UnionOperation) error
	// compileCall adds instructions to perform wazeroir.OperationCall.
	compileCall(o *wazeroir.UnionOperation) error
	// compileCallIndirect adds instructions to perform wazeroir.OperationCallIndirect.
	compileCallIndirect(o *wazeroir.UnionOperation) error
	// compileDrop adds instructions to perform wazeroir.NewOperationDrop.
	compileDrop(o *wazeroir.UnionOperation) error
	// compileSelect adds instructions to perform wazeroir.OperationSelect.
	compileSelect(o *wazeroir.UnionOperation) error
	// compilePick adds instructions to perform wazeroir.OperationPick.
	compilePick(o *wazeroir.UnionOperation) error
	// compileAdd adds instructions to perform wazeroir.OperationAdd.
	compileAdd(o *wazeroir.UnionOperation) error
	// compileSub adds instructions to perform wazeroir.OperationSub.
	compileSub(o *wazeroir.UnionOperation) error
	// compileMul adds instructions to perform wazeroir.OperationMul.
	compileMul(o *wazeroir.UnionOperation) error
	// compileClz adds instructions to perform wazeroir.OperationClz.
	compileClz(o *wazeroir.UnionOperation) error
	// compileCtz adds instructions to perform wazeroir.OperationCtz.
	compileCtz(o *wazeroir.UnionOperation) error
	// compilePopcnt adds instructions to perform wazeroir.OperationPopcnt.
	compilePopcnt(o *wazeroir.UnionOperation) error
	// compileDiv adds instructions to perform wazeroir.OperationDiv.
	compileDiv(o *wazeroir.UnionOperation) error
	// compileRem adds instructions to perform wazeroir.OperationRem.
	compileRem(o *wazeroir.UnionOperation) error
	// compileAnd adds instructions to perform wazeroir.OperationAnd.
	compileAnd(o *wazeroir.UnionOperation) error
	// compileOr adds instructions to perform wazeroir.OperationOr.
	compileOr(o *wazeroir.UnionOperation) error
	// compileXor adds instructions to perform wazeroir.OperationXor.
	compileXor(o *wazeroir.UnionOperation) error
	// compileShl adds instructions to perform wazeroir.OperationShl.
	compileShl(o *wazeroir.UnionOperation) error
	// compileShr adds instructions to perform wazeroir.OperationShr.
	compileShr(o *wazeroir.UnionOperation) error
	// compileRotl adds instructions to perform wazeroir.OperationRotl.
	compileRotl(o *wazeroir.UnionOperation) error
	// compileRotr adds instructions to perform wazeroir.OperationRotr.
	compileRotr(o *wazeroir.UnionOperation) error
	// compileNeg adds instructions to perform wazeroir.OperationAbs.
	compileAbs(o *wazeroir.UnionOperation) error
	// compileNeg adds instructions to perform wazeroir.OperationNeg.
	compileNeg(o *wazeroir.UnionOperation) error
	// compileCeil adds instructions to perform wazeroir.OperationCeil.
	compileCeil(o *wazeroir.UnionOperation) error
	// compileFloor adds instructions to perform wazeroir.OperationFloor.
	compileFloor(o *wazeroir.UnionOperation) error
	// compileTrunc adds instructions to perform wazeroir.OperationTrunc.
	compileTrunc(o *wazeroir.UnionOperation) error
	// compileNearest adds instructions to perform wazeroir.OperationNearest.
	compileNearest(o *wazeroir.UnionOperation) error
	// compileSqrt adds instructions perform wazeroir.OperationSqrt.
	compileSqrt(o *wazeroir.UnionOperation) error
	// compileMin adds instructions perform wazeroir.OperationMin.
	compileMin(o *wazeroir.UnionOperation) error
	// compileMax adds instructions perform wazeroir.OperationMax.
	compileMax(o *wazeroir.UnionOperation) error
	// compileCopysign adds instructions to perform wazeroir.OperationCopysign.
	compileCopysign(o *wazeroir.UnionOperation) error
	// compileI32WrapFromI64 adds instructions to perform wazeroir.OperationI32WrapFromI64.
	compileI32WrapFromI64() error
	// compileITruncFromF adds instructions to perform wazeroir.NewOperationITruncFromF.
	compileITruncFromF(o *wazeroir.UnionOperation) error
	// compileFConvertFromI adds instructions to perform wazeroir.NewOperationFConvertFromI.
	compileFConvertFromI(o *wazeroir.UnionOperation) error
	// compileF32DemoteFromF64 adds instructions to perform wazeroir.OperationF32DemoteFromF64.
	compileF32DemoteFromF64() error
	// compileF64PromoteFromF32 adds instructions to perform wazeroir.OperationF64PromoteFromF32.
	compileF64PromoteFromF32() error
	// compileI32ReinterpretFromF32 adds instructions to perform wazeroir.OperationI32ReinterpretFromF32.
	compileI32ReinterpretFromF32() error
	// compileI64ReinterpretFromF64 adds instructions to perform wazeroir.OperationI64ReinterpretFromF64.
	compileI64ReinterpretFromF64() error
	// compileF32ReinterpretFromI32 adds instructions to perform wazeroir.OperationF32ReinterpretFromI32.
	compileF32ReinterpretFromI32() error
	// compileF64ReinterpretFromI64 adds instructions to perform wazeroir.OperationF64ReinterpretFromI64.
	compileF64ReinterpretFromI64() error
	// compileExtend adds instructions to perform wazeroir.NewOperationExtend.
	compileExtend(o *wazeroir.UnionOperation) error
	// compileEq adds instructions to perform wazeroir.OperationEq.
	compileEq(o *wazeroir.UnionOperation) error
	// compileEq adds instructions to perform wazeroir.OperationNe.
	compileNe(o *wazeroir.UnionOperation) error
	// compileEq adds instructions to perform wazeroir.OperationEqz.
	compileEqz(o *wazeroir.UnionOperation) error
	// compileLt adds instructions to perform wazeroir.OperationLt.
	compileLt(o *wazeroir.UnionOperation) error
	// compileGt adds instructions to perform wazeroir.OperationGt.
	compileGt(o *wazeroir.UnionOperation) error
	// compileLe adds instructions to perform wazeroir.OperationLe.
	compileLe(o *wazeroir.UnionOperation) error
	// compileLe adds instructions to perform wazeroir.OperationGe.
	compileGe(o *wazeroir.UnionOperation) error
	// compileLoad adds instructions to perform wazeroir.OperationLoad.
	compileLoad(o *wazeroir.UnionOperation) error
	// compileLoad8 adds instructions to perform wazeroir.OperationLoad8.
	compileLoad8(o *wazeroir.UnionOperation) error
	// compileLoad16 adds instructions to perform wazeroir.OperationLoad16.
	compileLoad16(o *wazeroir.UnionOperation) error
	// compileLoad32 adds instructions to perform wazeroir.OperationLoad32.
	compileLoad32(o *wazeroir.UnionOperation) error
	// compileStore adds instructions to perform wazeroir.OperationStore.
	compileStore(o *wazeroir.UnionOperation) error
	// compileStore8 adds instructions to perform wazeroir.OperationStore8.
	compileStore8(o *wazeroir.UnionOperation) error
	// compileStore16 adds instructions to perform wazeroir.OperationStore16.
	compileStore16(o *wazeroir.UnionOperation) error
	// compileStore32 adds instructions to perform wazeroir.OperationStore32.
	compileStore32(o *wazeroir.UnionOperation) error
	// compileMemorySize adds instruction to perform wazeroir.OperationMemoryGrow.
	compileMemoryGrow() error
	// compileMemorySize adds instruction to perform wazeroir.OperationMemorySize.
	compileMemorySize() error
	// compileConstI32 adds instruction to perform wazeroir.NewOperationConstI32.
	compileConstI32(o *wazeroir.UnionOperation) error
	// compileConstI64 adds instruction to perform wazeroir.NewOperationConstI64.
	compileConstI64(o *wazeroir.UnionOperation) error
	// compileConstF32 adds instruction to perform wazeroir.NewOperationConstF32.
	compileConstF32(o *wazeroir.UnionOperation) error
	// compileConstF64 adds instruction to perform wazeroir.NewOperationConstF64.
	compileConstF64(o *wazeroir.UnionOperation) error
	// compileSignExtend32From8 adds instructions to perform wazeroir.OperationSignExtend32From8.
	compileSignExtend32From8() error
	// compileSignExtend32From16 adds instructions to perform wazeroir.OperationSignExtend32From16.
	compileSignExtend32From16() error
	// compileSignExtend64From8 adds instructions to perform wazeroir.OperationSignExtend64From8.
	compileSignExtend64From8() error
	// compileSignExtend64From16 adds instructions to perform wazeroir.OperationSignExtend64From16.
	compileSignExtend64From16() error
	// compileSignExtend64From32 adds instructions to perform wazeroir.OperationSignExtend64From32.
	compileSignExtend64From32() error
	// compileMemoryInit adds instructions to perform wazeroir.NewOperationMemoryInit.
	compileMemoryInit(*wazeroir.UnionOperation) error
	// compileDataDrop adds instructions to perform wazeroir.NewOperationDataDrop.
	compileDataDrop(*wazeroir.UnionOperation) error
	// compileMemoryCopy adds instructions to perform wazeroir.OperationMemoryCopy.
	compileMemoryCopy() error
	// compileMemoryFill adds instructions to perform wazeroir.OperationMemoryFill.
	compileMemoryFill() error
	// compileTableInit adds instructions to perform wazeroir.NewOperationTableInit.
	compileTableInit(*wazeroir.UnionOperation) error
	// compileTableCopy adds instructions to perform wazeroir.NewOperationTableCopy.
	compileTableCopy(*wazeroir.UnionOperation) error
	// compileElemDrop adds instructions to perform wazeroir.NewOperationElemDrop.
	compileElemDrop(*wazeroir.UnionOperation) error
	// compileRefFunc adds instructions to perform wazeroir.NewOperationRefFunc.
	compileRefFunc(*wazeroir.UnionOperation) error
	// compileTableGet adds instructions to perform wazeroir.NewOperationTableGet.
	compileTableGet(*wazeroir.UnionOperation) error
	// compileTableSet adds instructions to perform wazeroir.NewOperationTableSet.
	compileTableSet(*wazeroir.UnionOperation) error
	// compileTableGrow adds instructions to perform wazeroir.NewOperationTableGrow.
	compileTableGrow(*wazeroir.UnionOperation) error
	// compileTableSize adds instructions to perform wazeroir.NewOperationTableSize.
	compileTableSize(*wazeroir.UnionOperation) error
	// compileTableFill adds instructions to perform wazeroir.NewOperationTableFill.
	compileTableFill(*wazeroir.UnionOperation) error
	// compileV128Const adds instructions to perform wazeroir.NewOperationV128Const.
	compileV128Const(*wazeroir.UnionOperation) error
	// compileV128Add adds instructions to perform wazeroir.OperationV128Add.
	compileV128Add(o *wazeroir.UnionOperation) error
	// compileV128Sub adds instructions to perform wazeroir.OperationV128Sub.
	compileV128Sub(o *wazeroir.UnionOperation) error
	// compileV128Load adds instructions to perform wazeroir.OperationV128Load.
	compileV128Load(o *wazeroir.UnionOperation) error
	// compileV128LoadLane adds instructions to perform wazeroir.OperationV128LoadLane.
	compileV128LoadLane(o *wazeroir.UnionOperation) error
	// compileV128Store adds instructions to perform wazeroir.NewOperationV128Store.
	compileV128Store(o *wazeroir.UnionOperation) error
	// compileV128StoreLane adds instructions to perform wazeroir.NewOperationV128StoreLane.
	compileV128StoreLane(o *wazeroir.UnionOperation) error
	// compileV128ExtractLane adds instructions to perform wazeroir.NewOperationV128ExtractLane.
	compileV128ExtractLane(o *wazeroir.UnionOperation) error
	// compileV128ReplaceLane adds instructions to perform wazeroir.NewOperationV128ReplaceLane.
	compileV128ReplaceLane(o *wazeroir.UnionOperation) error
	// compileV128Splat adds instructions to perform wazeroir.NewOperationV128Splat.
	compileV128Splat(o *wazeroir.UnionOperation) error
	// compileV128Shuffle adds instructions to perform wazeroir.NewOperationV128Shuffle.
	compileV128Shuffle(o *wazeroir.UnionOperation) error
	// compileV128Swizzle adds instructions to perform wazeroir.OperationV128Swizzle.
	compileV128Swizzle(o *wazeroir.UnionOperation) error
	// compileV128AnyTrue adds instructions to perform wazeroir.OperationV128AnyTrue.
	compileV128AnyTrue(o *wazeroir.UnionOperation) error
	// compileV128AllTrue adds instructions to perform wazeroir.NewOperationV128AllTrue.
	compileV128AllTrue(o *wazeroir.UnionOperation) error
	// compileV128BitMask adds instructions to perform wazeroir.NewOperationV128BitMask.
	compileV128BitMask(*wazeroir.UnionOperation) error
	// compileV128And adds instructions to perform wazeroir.OperationV128And.
	compileV128And(*wazeroir.UnionOperation) error
	// compileV128Not adds instructions to perform wazeroir.OperationV128Not.
	compileV128Not(*wazeroir.UnionOperation) error
	// compileV128Or adds instructions to perform wazeroir.OperationV128Or.
	compileV128Or(*wazeroir.UnionOperation) error
	// compileV128Xor adds instructions to perform wazeroir.OperationV128Xor.
	compileV128Xor(*wazeroir.UnionOperation) error
	// compileV128Bitselect adds instructions to perform wazeroir.OperationV128Bitselect.
	compileV128Bitselect(*wazeroir.UnionOperation) error
	// compileV128AndNot adds instructions to perform wazeroir.OperationV128AndNot.
	compileV128AndNot(*wazeroir.UnionOperation) error
	// compileV128Shr adds instructions to perform wazeroir.NewOperationV128Shr.
	compileV128Shr(*wazeroir.UnionOperation) error
	// compileV128Shl adds instructions to perform wazeroir.NewOperationV128Shl.
	compileV128Shl(*wazeroir.UnionOperation) error
	// compileV128Cmp adds instructions to perform wazeroir.NewOperationV128Cmp.
	compileV128Cmp(*wazeroir.UnionOperation) error
	// compileV128AddSat adds instructions to perform wazeroir.NewOperationV128AddSat.
	compileV128AddSat(*wazeroir.UnionOperation) error
	// compileV128SubSat adds instructions to perform wazeroir.NewOperationV128SubSat.
	compileV128SubSat(*wazeroir.UnionOperation) error
	// compileV128Mul adds instructions to perform wazeroir.NewOperationV128Mul.
	compileV128Mul(*wazeroir.UnionOperation) error
	// compileV128Div adds instructions to perform wazeroir.NewOperationV128Div.
	compileV128Div(*wazeroir.UnionOperation) error
	// compileV128Neg adds instructions to perform wazeroir.NewOperationV128Neg.
	compileV128Neg(*wazeroir.UnionOperation) error
	// compileV128Sqrt adds instructions to perform wazeroir.NewOperationV128Sqrt.
	compileV128Sqrt(*wazeroir.UnionOperation) error
	// compileV128Abs adds instructions to perform wazeroir.NewOperationV128Abs.
	compileV128Abs(*wazeroir.UnionOperation) error
	// compileV128Popcnt adds instructions to perform wazeroir.NewOperationV128Popcnt.
	compileV128Popcnt(*wazeroir.UnionOperation) error
	// compileV128Min adds instructions to perform wazeroir.NewOperationV128Min.
	compileV128Min(*wazeroir.UnionOperation) error
	// compileV128Max adds instructions to perform wazeroir.NewOperationV128Max.
	compileV128Max(*wazeroir.UnionOperation) error
	// compileV128AvgrU adds instructions to perform wazeroir.NewOperationV128AvgrU.
	compileV128AvgrU(*wazeroir.UnionOperation) error
	// compileV128Pmin adds instructions to perform wazeroir.NewOperationV128Pmin.
	compileV128Pmin(*wazeroir.UnionOperation) error
	// compileV128Pmax adds instructions to perform wazeroir.NewOperationV128Pmax.
	compileV128Pmax(*wazeroir.UnionOperation) error
	// compileV128Ceil adds instructions to perform wazeroir.NewOperationV128Ceil.
	compileV128Ceil(*wazeroir.UnionOperation) error
	// compileV128Floor adds instructions to perform wazeroir.NewOperationV128Floor.
	compileV128Floor(*wazeroir.UnionOperation) error
	// compileV128Trunc adds instructions to perform wazeroir.NewOperationV128Trunc.
	compileV128Trunc(*wazeroir.UnionOperation) error
	// compileV128Nearest adds instructions to perform wazeroir.NewOperationV128Nearest.
	compileV128Nearest(*wazeroir.UnionOperation) error
	// compileV128Extend adds instructions to perform wazeroir.NewOperationV128Extend.
	compileV128Extend(*wazeroir.UnionOperation) error
	// compileV128ExtMul adds instructions to perform wazeroir.NewOperationV128ExtMul.
	compileV128ExtMul(*wazeroir.UnionOperation) error
	// compileV128Q15mulrSatS adds instructions to perform wazeroir.OperationV128Q15mulrSatS.
	compileV128Q15mulrSatS(*wazeroir.UnionOperation) error
	// compileV128ExtAddPairwise adds instructions to perform wazeroir.NewOperationV128ExtAddPairwise.
	compileV128ExtAddPairwise(o *wazeroir.UnionOperation) error
	// compileV128FloatPromote adds instructions to perform wazeroir.OperationV128FloatPromote.
	compileV128FloatPromote(o *wazeroir.UnionOperation) error
	// compileV128FloatDemote adds instructions to perform wazeroir.OperationV128FloatDemote.
	compileV128FloatDemote(o *wazeroir.UnionOperation) error
	// compileV128FConvertFromI adds instructions to perform wazeroir.NewOperationV128FConvertFromI.
	compileV128FConvertFromI(o *wazeroir.UnionOperation) error
	// compileV128Dot adds instructions to perform wazeroir.OperationV128Dot.
	compileV128Dot(o *wazeroir.UnionOperation) error
	// compileV128Narrow adds instructions to perform wazeroir.NewOperationV128Narrow.
	compileV128Narrow(o *wazeroir.UnionOperation) error
	// compileV128ITruncSatFromF adds instructions to perform wazeroir.NewOperationV128ITruncSatFromF.
	compileV128ITruncSatFromF(o *wazeroir.UnionOperation) error

	// compileBuiltinFunctionCheckExitCode adds instructions to perform wazeroir.OperationBuiltinFunctionCheckExitCode.
	compileBuiltinFunctionCheckExitCode() error

	// compileReleaseRegisterToStack adds instructions to write the value on a register back to memory stack region.
	compileReleaseRegisterToStack(loc *runtimeValueLocation)
	// compileLoadValueOnStackToRegister adds instructions to load the value located on the stack to the assigned register.
	compileLoadValueOnStackToRegister(loc *runtimeValueLocation)

	// maybeCompileMoveTopConditionalToGeneralPurposeRegister moves the top value on the stack
	// if the value is located on a conditional register.
	//
	// This is usually called at the beginning of methods on compiler interface where we possibly
	// compile instructions without saving the conditional register value.
	// The compileXXX functions without calling this function is saving the conditional
	// value to the stack or register by invoking compileEnsureOnRegister for the top.
	maybeCompileMoveTopConditionalToGeneralPurposeRegister() error
	// allocateRegister returns an unused register of the given type. The register will be taken
	// either from the free register pool or by stealing a used register.
	//
	// Note: resulting registers will not be marked as used so the call site should
	// mark it used if necessary.
	allocateRegister(t registerType) (reg asm.Register, err error)
	// runtimeValueLocationStack returns the current runtimeValueLocationStack of the compiler implementation.
	runtimeValueLocationStack() *runtimeValueLocationStack
	// pushRuntimeValueLocationOnRegister pushes a new runtimeValueLocation on a register `reg` and of the type `vt`.
	pushRuntimeValueLocationOnRegister(reg asm.Register, vt runtimeValueType) (ret *runtimeValueLocation)
	// pushRuntimeValueLocationOnRegister pushes a new vector value's runtimeValueLocation on a register `reg`.
	pushVectorRuntimeValueLocationOnRegister(reg asm.Register) (lowerBitsLocation *runtimeValueLocation)
	// compileNOP compiles NOP instruction and returns the corresponding asm.Node in the assembled native code.
	// This is used to emit DWARF based stack traces.
	compileNOP() asm.Node
}
