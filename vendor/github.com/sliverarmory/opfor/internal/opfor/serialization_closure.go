package opfor

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/sliverarmory/opfor/internal/ast"
	"github.com/sliverarmory/opfor/internal/bytecode"
	"github.com/sliverarmory/opfor/internal/compiler"
	"github.com/sliverarmory/opfor/internal/javaser"
)

// Sleep 2.1 serializes executable closures as its legacy Java evaluator graph.
// These descriptors are pinned to the official 2.1 JAR, not to a recompilation
// of the open-source tree. ObjectStreamClass orders primitive fields before
// reference fields and sorts names within those groups.
var (
	sleepBlockType        = javaser.NewString("Lsleep/engine/Block;")
	sleepStepType         = javaser.NewString("Lsleep/engine/Step;")
	sleepCheckType        = javaser.NewString("Lsleep/engine/atoms/Check;")
	sleepScalarObjectType = javaser.NewString("Lsleep/runtime/Scalar;")
	sleepVariableType     = javaser.NewString("Lsleep/interfaces/Variable;")
	sleepScriptType       = javaser.NewString("Lsleep/runtime/ScriptInstance;")
	javaClassType         = javaser.NewString("Ljava/lang/Class;")
	javaHashMapType       = javaser.NewString("Ljava/util/HashMap;")
	javaHashtableType     = javaser.NewString("Ljava/util/Hashtable;")
	javaStackType         = javaser.NewString("Ljava/util/Stack;")
	javaObjectArrayType   = javaser.NewString("[Ljava/lang/Object;")

	sleepClosureDescriptor = &javaser.ClassDesc{
		Name:             "sleep.bridges.SleepClosure",
		SerialVersionUID: 5328795954519346800,
		Flags:            javaser.SCSerializable | javaser.SCWriteMethod,
		Fields: []javaser.FieldDesc{
			{TypeCode: javaser.TypeInt, Name: "id"},
			{TypeCode: javaser.TypeObject, Name: "code", ClassName: sleepBlockType},
			{TypeCode: javaser.TypeObject, Name: "context", ClassName: javaStackType},
			{TypeCode: javaser.TypeObject, Name: "metadata", ClassName: javaHashMapType},
			{TypeCode: javaser.TypeObject, Name: "owner", ClassName: sleepScriptType},
			{TypeCode: javaser.TypeObject, Name: "variables", ClassName: sleepVariableType},
		},
	}
	sleepContextDescriptor = &javaser.ClassDesc{
		Name:             "sleep.runtime.ScriptEnvironment$Context",
		SerialVersionUID: -243798196192373463,
		Flags:            javaser.SCSerializable,
		Fields: []javaser.FieldDesc{
			{TypeCode: javaser.TypeBoolean, Name: "moreHandlers"},
			{TypeCode: javaser.TypeObject, Name: "block", ClassName: sleepBlockType},
			{TypeCode: javaser.TypeObject, Name: "handler", ClassName: javaser.NewString("Lsleep/runtime/ScriptEnvironment$ExceptionContext;")},
			{TypeCode: javaser.TypeObject, Name: "last", ClassName: sleepStepType},
		},
	}
	sleepExceptionContextDescriptor = &javaser.ClassDesc{
		Name:             "sleep.runtime.ScriptEnvironment$ExceptionContext",
		SerialVersionUID: 6006916407518094063,
		Flags:            javaser.SCSerializable,
		Fields: []javaser.FieldDesc{
			{TypeCode: javaser.TypeObject, Name: "handler", ClassName: sleepBlockType},
			{TypeCode: javaser.TypeObject, Name: "owner", ClassName: sleepBlockType},
			{TypeCode: javaser.TypeObject, Name: "varname", ClassName: javaStringType},
		},
	}
	sleepDefaultVariableDescriptor = &javaser.ClassDesc{
		Name:             "sleep.bridges.DefaultVariable",
		SerialVersionUID: -2706370801224485626,
		Flags:            javaser.SCSerializable,
		Fields: []javaser.FieldDesc{
			{TypeCode: javaser.TypeObject, Name: "values", ClassName: javaHashtableType},
		},
	}
	sleepBlockDescriptor = &javaser.ClassDesc{
		Name:             "sleep.engine.Block",
		SerialVersionUID: -6469549406898888342,
		Flags:            javaser.SCSerializable,
		Fields: []javaser.FieldDesc{
			{TypeCode: javaser.TypeObject, Name: "first", ClassName: sleepStepType},
			{TypeCode: javaser.TypeObject, Name: "last", ClassName: sleepStepType},
			{TypeCode: javaser.TypeObject, Name: "source", ClassName: javaStringType},
		},
	}
	sleepStepDescriptor = &javaser.ClassDesc{
		Name:             "sleep.engine.Step",
		SerialVersionUID: 4687083741384946712,
		Flags:            javaser.SCSerializable,
		Fields: []javaser.FieldDesc{
			{TypeCode: javaser.TypeInt, Name: "line"},
			{TypeCode: javaser.TypeObject, Name: "next", ClassName: sleepStepType},
		},
	}
	javaHashtableDescriptor = &javaser.ClassDesc{
		Name:             "java.util.Hashtable",
		SerialVersionUID: 1421746759512286392,
		Flags:            javaser.SCSerializable | javaser.SCWriteMethod,
		Fields: []javaser.FieldDesc{
			{TypeCode: javaser.TypeFloat, Name: "loadFactor"},
			{TypeCode: javaser.TypeInt, Name: "threshold"},
		},
	}
	javaObjectArrayDescriptor = &javaser.ClassDesc{
		Name:             "[Ljava.lang.Object;",
		SerialVersionUID: -8012369246846506644,
		Flags:            javaser.SCSerializable,
	}
	javaVectorDescriptor = &javaser.ClassDesc{
		Name:             "java.util.Vector",
		SerialVersionUID: -2767605614048989439,
		Flags:            javaser.SCSerializable | javaser.SCWriteMethod,
		Fields: []javaser.FieldDesc{
			{TypeCode: javaser.TypeInt, Name: "capacityIncrement"},
			{TypeCode: javaser.TypeInt, Name: "elementCount"},
			{TypeCode: javaser.TypeArray, Name: "elementData", ClassName: javaObjectArrayType},
		},
	}
	javaStackDescriptor = &javaser.ClassDesc{
		Name:             "java.util.Stack",
		SerialVersionUID: 1224463164541339165,
		Flags:            javaser.SCSerializable,
		Super:            javaVectorDescriptor,
	}
	javaLinkedListDescriptor = &javaser.ClassDesc{
		Name:             "java.util.LinkedList",
		SerialVersionUID: 876323262645176354,
		Flags:            javaser.SCSerializable | javaser.SCWriteMethod,
	}
	sleepJavaSystemClassDescriptor = &javaser.ClassDesc{
		Name: "java.lang.System",
	}
	sleepSleepUtilsClassDescriptor = &javaser.ClassDesc{
		Name: "sleep.runtime.SleepUtils",
	}
	sleepJavaSystemClass = &javaser.Class{Descriptor: sleepJavaSystemClassDescriptor}
	sleepSleepUtilsClass = &javaser.Class{Descriptor: sleepSleepUtilsClassDescriptor}
)

func sleepAtomDescriptor(name string, uid int64, fields ...javaser.FieldDesc) *javaser.ClassDesc {
	return &javaser.ClassDesc{
		Name:             "sleep.engine.atoms." + name,
		SerialVersionUID: uid,
		Flags:            javaser.SCSerializable,
		Fields:           fields,
		Super:            sleepStepDescriptor,
	}
}

func sleepCheckDescriptor(name string, uid int64, fields ...javaser.FieldDesc) *javaser.ClassDesc {
	return &javaser.ClassDesc{
		Name:             "sleep.engine.atoms." + name,
		SerialVersionUID: uid,
		Flags:            javaser.SCSerializable,
		Fields:           fields,
	}
}

var (
	sleepPopTryDescriptor        = sleepAtomDescriptor("PopTry", 1793037301910736647)
	sleepObjectNewDescriptor     = sleepAtomDescriptor("ObjectNew", -3546671562778485616, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "name", ClassName: javaClassType})
	sleepAssignDescriptor        = sleepAtomDescriptor("Assign", 4431316378192485615, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "operator", ClassName: sleepStepType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "variable", ClassName: sleepBlockType})
	sleepSValueDescriptor        = sleepAtomDescriptor("SValue", -52027826365450316, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "value", ClassName: sleepScalarObjectType})
	sleepIndexDescriptor         = sleepAtomDescriptor("Index", 988798933407250440, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "index", ClassName: sleepBlockType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "value", ClassName: javaStringType})
	sleepAssignTDescriptor       = sleepAtomDescriptor("AssignT", 8155187448575691943, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "operator", ClassName: sleepStepType})
	sleepReturnDescriptor        = sleepAtomDescriptor("Return", 7137976826892793952, javaser.FieldDesc{TypeCode: javaser.TypeInt, Name: "return_type"})
	sleepBindDescriptor          = sleepAtomDescriptor("Bind", 7326839684366194925, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "code", ClassName: sleepBlockType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "funcenv", ClassName: javaStringType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "name", ClassName: sleepBlockType})
	sleepDecideDescriptor        = sleepAtomDescriptor("Decide", -4161386451313906454, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "iffalse", ClassName: sleepBlockType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "iftrue", ClassName: sleepBlockType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "start", ClassName: sleepCheckType})
	sleepOperateDescriptor       = sleepAtomDescriptor("Operate", -7440583617175514999, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "oper", ClassName: javaStringType})
	sleepCheckOrDescriptor       = sleepCheckDescriptor("CheckOr", 234794920572704341, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "left", ClassName: sleepCheckType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "right", ClassName: sleepCheckType})
	sleepBindFilterDescriptor    = sleepAtomDescriptor("BindFilter", 8233057403547197939, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "code", ClassName: sleepBlockType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "filter", ClassName: javaStringType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "funcenv", ClassName: javaStringType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "name", ClassName: javaStringType})
	sleepCreateClosureDescriptor = sleepAtomDescriptor("CreateClosure", -4885505274906556436, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "block", ClassName: sleepBlockType})
	sleepTryDescriptor           = sleepAtomDescriptor("Try", -1993853311765774487, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "handler", ClassName: sleepBlockType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "owner", ClassName: sleepBlockType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "var", ClassName: javaStringType})
	sleepCheckAndDescriptor      = sleepCheckDescriptor("CheckAnd", -8755907560721465205, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "left", ClassName: sleepCheckType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "right", ClassName: sleepCheckType})
	sleepCreateFrameDescriptor   = sleepAtomDescriptor("CreateFrame", -2345463092852894191)
	sleepPLiteralDescriptor      = sleepAtomDescriptor("PLiteral", -287560155198654390, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "fragments", ClassName: javaListType})
	sleepIterateDescriptor       = sleepAtomDescriptor("Iterate", -3981563856783479004, javaser.FieldDesc{TypeCode: javaser.TypeInt, Name: "type"}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "key", ClassName: javaStringType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "value", ClassName: javaStringType})
	sleepGetDescriptor           = sleepAtomDescriptor("Get", -6481651174984159681, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "value", ClassName: javaStringType})
	sleepObjectAccessDescriptor  = sleepAtomDescriptor("ObjectAccess", -7729782646560321434, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "classRef", ClassName: javaClassType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "name", ClassName: javaStringType})
	sleepBindPredicateDescriptor = sleepAtomDescriptor("BindPredicate", -4019087267673106002, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "code", ClassName: sleepBlockType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "funcenv", ClassName: javaStringType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "pred", ClassName: sleepCheckType})
	sleepCallDescriptor          = sleepAtomDescriptor("Call", -1318377443254558623, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "function", ClassName: javaStringType})
	sleepCheckEvalDescriptor     = sleepCheckDescriptor("CheckEval", -277595782952626545, javaser.FieldDesc{TypeCode: javaser.TypeInt, Name: "hint"}, javaser.FieldDesc{TypeCode: javaser.TypeBoolean, Name: "negate"}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "iffalse", ClassName: sleepCheckType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "iftrue", ClassName: sleepCheckType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "name", ClassName: javaStringType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "setup", ClassName: sleepBlockType})
	sleepGotoDescriptor          = sleepAtomDescriptor("Goto", -5718734842451501858, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "iftrue", ClassName: sleepBlockType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "increment", ClassName: sleepBlockType}, javaser.FieldDesc{TypeCode: javaser.TypeObject, Name: "start", ClassName: sleepCheckType})

	sleepPLiteralFragmentDescriptor = &javaser.ClassDesc{
		Name:             "sleep.engine.atoms.PLiteral$Fragment",
		SerialVersionUID: -4452432280959099189,
		Flags:            javaser.SCSerializable,
		Fields: []javaser.FieldDesc{
			{TypeCode: javaser.TypeInt, Name: "type"},
			{TypeCode: javaser.TypeObject, Name: "element", ClassName: javaObjectType},
		},
	}
)

const (
	sleepFlowReturn   = 1
	sleepFlowBreak    = 2
	sleepFlowContinue = 4
	sleepFlowYield    = 8
	sleepFlowThrow    = 16
	sleepFlowCallCC   = 72

	sleepIteratorCreate  = 1
	sleepIteratorDestroy = 2
	sleepIteratorNext    = 3
)

func sleepClosureClassData(descriptor *javaser.ClassDesc) (javaser.ClassDataLayout, bool, error) {
	if descriptor == nil {
		return javaser.ClassDataAuto, false, errors.New("sleep serialization: nil class descriptor")
	}
	var want *javaser.ClassDesc
	var layout javaser.ClassDataLayout
	switch descriptor.Name {
	case sleepClosureDescriptor.Name:
		want, layout = sleepClosureDescriptor, javaser.ClassDataAnnotationOnly
	case javaHashtableDescriptor.Name:
		want, layout = javaHashtableDescriptor, javaser.ClassDataDefaultFieldsAndAnnotation
	case javaLinkedListDescriptor.Name:
		want, layout = javaLinkedListDescriptor, javaser.ClassDataDefaultFieldsAndAnnotation
	case javaVectorDescriptor.Name:
		want, layout = javaVectorDescriptor, javaser.ClassDataDefaultFieldsAndAnnotation
	default:
		return javaser.ClassDataAuto, false, nil
	}
	if err := validateSleepDescriptor(descriptor, want); err != nil {
		return javaser.ClassDataAuto, true, err
	}
	return layout, true, nil
}

func (state *sleepSerializationDecoder) closure(object *javaser.Object) (Value, error) {
	if object == nil {
		return Null(), errors.New("sleep serialization: nil SleepClosure")
	}
	if state.script == nil {
		return Null(), &UnsupportedError{Operation: "serialization", Name: "Sleep closure without a receiving script"}
	}
	if state.script.globals != nil && state.script.globals.container != nil {
		return Null(), &UnsupportedError{Operation: "serialization", Name: "Sleep closure with importer variable container"}
	}
	if existing := state.closures[object]; existing != nil {
		return FunctionValue(existing), nil
	}
	if err := validateSleepDescriptor(object.Descriptor, sleepClosureDescriptor); err != nil {
		return Null(), err
	}
	data, ok := object.DataFor(sleepClosureDescriptor.Name)
	if !ok || len(data.Fields) != 0 || len(data.Annotation) != 4 {
		return Null(), errors.New("sleep serialization: invalid SleepClosure custom data")
	}
	idBlock, ok := data.Annotation[0].(*javaser.BlockData)
	if !ok || len(idBlock.Data) != 4 {
		return Null(), errors.New("sleep serialization: invalid SleepClosure id")
	}
	id := int32(binary.BigEndian.Uint32(idBlock.Data))
	code, ok := data.Annotation[1].(*javaser.Object)
	if !ok {
		return Null(), errors.New("sleep serialization: SleepClosure code is not a Block")
	}
	context, ok := data.Annotation[2].(*javaser.Object)
	if !ok {
		return Null(), errors.New("sleep serialization: SleepClosure context is not a Stack")
	}
	variables, ok := data.Annotation[3].(*javaser.Object)
	if !ok {
		return Null(), errors.New("sleep serialization: SleepClosure variables are not a DefaultVariable")
	}

	// Install the shell before decoding DefaultVariable. Its $this Scalar points
	// back to this exact Java object handle and must become the same Go closure.
	closure := state.script.newClosure(nil, state.script.globals)
	closure.id = uint64(uint32(id))
	closure.state = state.script.globals.child()
	closure.thisHash = NewHash()
	state.closures[object] = closure
	if err := preflightSleepClosureContexts(context); err != nil {
		delete(state.closures, object)
		return Null(), err
	}

	graph, err := state.legacyFunctionGraph(code)
	if err != nil {
		delete(state.closures, object)
		return Null(), err
	}
	cells, err := state.defaultVariables(variables)
	if err != nil {
		delete(state.closures, object)
		return Null(), err
	}
	closure.function = graph.function
	closure.state.mu.Lock()
	for name, cell := range cells {
		closure.state.cells[normalizeVariableName(name)] = cell
	}
	closure.state.mu.Unlock()
	if self, ok := cells["$this"]; !ok {
		closure.state.local("$this").Set(FunctionValue(closure))
	} else if callable, callableOK := self.Get().Function(); !callableOK || callable != closure {
		delete(state.closures, object)
		return Null(), errors.New("sleep serialization: SleepClosure $this does not reference its owner")
	}
	suspended, err := state.closureContexts(context, closure, graph)
	if err != nil {
		delete(state.closures, object)
		return Null(), err
	}
	closure.suspended = suspended
	return FunctionValue(closure), nil
}

func preflightSleepClosureContexts(stack *javaser.Object) error {
	toplevels, err := sleepStackElements(stack)
	if err != nil {
		return err
	}
	for _, value := range toplevels {
		toplevel, ok := value.(*javaser.Object)
		if !ok {
			return errors.New("sleep serialization: closure context entry is not a Stack")
		}
		entries, err := sleepStackElements(toplevel)
		if err != nil {
			return err
		}
		if len(entries) < 2 {
			return errors.New("sleep serialization: suspended closure context has no saved Block and local levels")
		}
		for _, entry := range entries[:len(entries)-1] {
			object, ok := entry.(*javaser.Object)
			if !ok {
				return errors.New("sleep serialization: saved closure Context is not an object")
			}
			_, err := decodeSleepContext(object)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func sleepClosureContextSize(stack *javaser.Object) (int, error) {
	elements, err := sleepStackElements(stack)
	return len(elements), err
}

func sleepStackElements(stack *javaser.Object) ([]javaser.Element, error) {
	if stack == nil || stack.Descriptor == nil {
		return nil, errors.New("sleep serialization: invalid closure context Stack")
	}
	if err := validateSleepDescriptor(stack.Descriptor, javaStackDescriptor); err != nil {
		return nil, err
	}
	vector, ok := stack.DataFor(javaVectorDescriptor.Name)
	if !ok {
		return nil, errors.New("sleep serialization: closure Stack has no Vector data")
	}
	countValue, ok := vector.Field("elementCount")
	if !ok {
		return nil, errors.New("sleep serialization: closure Stack has no elementCount")
	}
	count, ok := countValue.(javaser.Int)
	if !ok || count < 0 {
		return nil, errors.New("sleep serialization: closure Stack has invalid elementCount")
	}
	arrayValue, ok := vector.Field("elementData")
	if !ok {
		return nil, errors.New("sleep serialization: closure Stack has no elementData")
	}
	array, ok := arrayValue.(*javaser.Array)
	if !ok || array.Descriptor == nil {
		return nil, errors.New("sleep serialization: closure Stack elementData is not an array")
	}
	if err := validateSleepDescriptor(array.Descriptor, javaObjectArrayDescriptor); err != nil {
		return nil, err
	}
	if int(count) > len(array.Values) {
		return nil, errors.New("sleep serialization: closure Stack elementCount exceeds its backing array")
	}
	if _, ok := stack.DataFor(javaStackDescriptor.Name); !ok {
		return nil, errors.New("sleep serialization: closure Stack has no Stack data")
	}
	return array.Values[:int(count)], nil
}

type sleepDecodedContext struct {
	block        *javaser.Object
	last         *javaser.Object
	handler      *sleepDecodedExceptionContext
	moreHandlers bool
}

type sleepDecodedExceptionContext struct {
	owner   *javaser.Object
	handler *javaser.Object
	varname string
}

func (state *sleepSerializationDecoder) closureContexts(stack *javaser.Object, closure *scriptClosure, code *sleepLegacyDecodedBlock) ([]*fiber, error) {
	toplevels, err := sleepStackElements(stack)
	if err != nil {
		return nil, err
	}
	result := make([]*fiber, 0, len(toplevels))
	for _, value := range toplevels {
		toplevel, ok := value.(*javaser.Object)
		if !ok {
			return nil, errors.New("sleep serialization: closure context entry is not a Stack")
		}
		fiber, err := state.closureToplevelContext(toplevel, closure, code)
		if err != nil {
			return nil, err
		}
		result = append(result, fiber)
	}
	return result, nil
}

func (state *sleepSerializationDecoder) closureToplevelContext(stack *javaser.Object, closure *scriptClosure, code *sleepLegacyDecodedBlock) (*fiber, error) {
	values, err := sleepStackElements(stack)
	if err != nil {
		return nil, err
	}
	if len(values) < 2 {
		return nil, errors.New("sleep serialization: suspended closure context has no saved Block and local levels")
	}
	localList, ok := values[len(values)-1].(*javaser.Object)
	if !ok || localList.Descriptor == nil || localList.Descriptor.Name != javaLinkedListDescriptor.Name {
		return nil, errors.New("sleep serialization: suspended closure context does not end with local levels")
	}
	active, locals, err := state.closureLocalLevels(localList, closure)
	if err != nil {
		return nil, err
	}

	contexts := make([]sleepDecodedContext, len(values)-1)
	for index, value := range values[:len(values)-1] {
		object, ok := value.(*javaser.Object)
		if !ok {
			return nil, errors.New("sleep serialization: saved closure Context is not an object")
		}
		context, err := decodeSleepContext(object)
		if err != nil {
			return nil, err
		}
		contexts[index] = context
	}

	// One serialized toplevel Stack is the FIFO ScriptEnvironment context group
	// accumulated by one SleepClosure invocation. Most ordinary suspensions end
	// in SleepClosure.code, but eval/expr/include may leave only dynamically
	// compiled Blocks or may precede an independently suspended outer Block.
	// Inline calls are represented in the same flat list, inner Block first.
	// Decode every entry before partitioning that list into independent roots.
	resumes := make([]sleepDecodedResume, len(contexts))
	for index, saved := range contexts {
		graph := code
		if saved.block != code.block {
			graph, err = state.legacyFunctionGraph(saved.block)
			if err != nil {
				return nil, err
			}
		}
		resumePC, err := graph.contextPC(saved.last)
		if err != nil {
			return nil, err
		}
		resumes[index] = sleepDecodedResume{
			context: saved,
			graph:   graph,
			fiber: &fiber{
				closure:                closure,
				function:               graph.function,
				scope:                  active,
				locals:                 append([]*scope(nil), locals...),
				pc:                     resumePC,
				serializedMoreHandlers: saved.moreHandlers,
			},
		}
		if saved.handler != nil {
			if err := state.installDecodedExceptionContext(resumes[index].fiber, graph, saved.handler); err != nil {
				return nil, err
			}
		}
		if decodedSleepSerializedReturn(resumes[index]) {
			resumes[index].fiber.serializedReturn = true
		}
	}

	roots := make([]*fiber, 0, len(resumes))
	for index := 0; index < len(resumes); {
		if foreachFiber, consumed, matched, foreachErr := decodedSleepForeachResume(resumes[index:], closure, code, active, locals); matched || foreachErr != nil {
			if foreachErr != nil {
				return nil, foreachErr
			}
			roots = append(roots, foreachFiber)
			index += consumed
			continue
		}

		root := resumes[index].fiber
		rootBlock := resumes[index].context.block
		index++
		for index < len(resumes) && decodedSleepInlineParent(&resumes[index], root) {
			parent := resumes[index].fiber
			callPC := parent.pc - 1
			call, _ := directLegacyInlineCall(parent.function, callPC)
			parent.pc = callPC
			parent.inlineAt = map[*ast.CallExpr]*fiber{call: root}
			root.inline = true
			root = parent
			rootBlock = resumes[index].context.block
			index++
		}
		if rootBlock != code.block {
			// Java serializes no marker distinguishing eval, expr, and include,
			// and it also omits include-chain metadata. The independent Block
			// identity is the complete durable signal. An empty dynamic marker
			// restores continuation behavior without inventing lost metadata.
			root.dynamicSource = &dynamicSourceExecution{}
		}
		roots = append(roots, root)
	}
	if len(roots) == 0 {
		return nil, errors.New("sleep serialization: suspended closure context has no resumable Block")
	}
	head := roots[0]
	head.continuationTail = append(head.continuationTail[:0], roots[1:]...)
	return head, nil
}

func (state *sleepSerializationDecoder) installDecodedExceptionContext(fiber *fiber, owner *sleepLegacyDecodedBlock, saved *sleepDecodedExceptionContext) error {
	if fiber == nil || owner == nil || owner.function == nil || saved == nil {
		return errors.New("sleep serialization: invalid saved exception-handler context")
	}
	if saved.owner != owner.block {
		return errors.New("sleep serialization: saved exception-handler owner does not share Context.block handle")
	}
	handler, err := state.legacyFunctionGraph(saved.handler)
	if err != nil {
		return err
	}
	instructions := append([]bytecode.Instruction(nil), owner.function.Instructions...)
	handlerPC := len(instructions)
	span := owner.function.Span
	instructions = append(instructions, bytecode.Instruction{Op: bytecode.OpCatch, Span: span, Name: saved.varname})
	instructions = appendRebasedLegacyInstructions(instructions, handler.function.Instructions, false)
	fiber.function = &bytecode.Function{
		Name:         owner.function.Name,
		Span:         owner.function.Span,
		Instructions: instructions,
	}
	fiber.tries = append(fiber.tries, tryFrame{handler: handlerPC, depth: len(fiber.iterators)})
	return nil
}

type sleepDecodedResume struct {
	context sleepDecodedContext
	graph   *sleepLegacyDecodedBlock
	fiber   *fiber
}

func decodedSleepInlineParent(parent *sleepDecodedResume, _ *fiber) bool {
	if parent == nil || parent.fiber == nil || parent.fiber.pc <= 0 {
		return false
	}
	_, ok := directLegacyInlineCall(parent.fiber.function, parent.fiber.pc-1)
	return ok
}

func decodedSleepSerializedReturn(resume sleepDecodedResume) bool {
	if resume.fiber == nil || resume.graph == nil || resume.graph.function == nil || resume.fiber.pc != 0 {
		return false
	}
	last := resume.context.last
	if last == nil || last.Descriptor == nil || last.Descriptor.Name != sleepReturnDescriptor.Name {
		return false
	}
	instructions := resume.graph.function.Instructions
	return len(instructions) == 2 && instructions[0].Op == bytecode.OpReturn && instructions[1].Op == bytecode.OpEnd
}

func decodedSleepForeachResume(
	resumes []sleepDecodedResume,
	closure *scriptClosure,
	code *sleepLegacyDecodedBlock,
	active *scope,
	locals []*scope,
) (*fiber, int, bool, error) {
	if len(resumes) == 0 {
		return nil, 0, false, nil
	}
	// A Goto context without its immediately preceding body context is the
	// outer half of a nested foreach, an inline foreach suspension, or a body
	// tail. Java restores these entries independently in FIFO order. Retain the
	// omitted-iterator marker so execution reproduces the official warning pair.
	if last := resumes[0].context.last; last != nil && last.Descriptor != nil && last.Descriptor.Name == sleepGotoDescriptor.Name {
		loop := resumes[0].graph.foreachByGoto[last]
		if loop == nil {
			return nil, 0, true, &UnsupportedError{Operation: "serialization", Name: "unrecognized foreach Goto context"}
		}
		result := resumes[0].fiber
		result.pc = loop.jumpPC
		result.serializedForeach = &sleepSerializedForeachResume{
			iterNextPC: loop.iterNextPC,
			span:       loop.span,
		}
		if resumes[0].context.block != code.block {
			result.dynamicSource = &dynamicSourceExecution{}
		}
		return result, 1, true, nil
	}
	if len(resumes) < 2 {
		return nil, 0, false, nil
	}
	inner := resumes[0]
	outer := resumes[1]
	if outer.context.last == nil || outer.context.last.Descriptor == nil || outer.context.last.Descriptor.Name != sleepGotoDescriptor.Name {
		return nil, 0, false, nil
	}
	loop := outer.graph.foreachByGoto[outer.context.last]
	if loop == nil {
		return nil, 0, true, &UnsupportedError{Operation: "serialization", Name: "unrecognized foreach Goto context"}
	}
	if inner.context.block != loop.bodyBlock {
		return nil, 0, true, errors.New("sleep serialization: foreach inner Context.block does not share Goto.iftrue handle")
	}
	if inner.context.last != nil {
		// The body must resume before the outer Goto is re-entered. Let the
		// ordinary context decoder build that body/inline root; the next loop
		// iteration will recognize the lone Goto context above.
		return nil, 0, false, nil
	}
	result := &fiber{
		closure:  closure,
		function: outer.graph.function,
		scope:    active,
		locals:   append([]*scope(nil), locals...),
		pc:       loop.jumpPC,
		serializedForeach: &sleepSerializedForeachResume{
			iterNextPC:  loop.iterNextPC,
			span:        loop.span,
			includeBody: true,
		},
	}
	if outer.context.block != code.block {
		result.dynamicSource = &dynamicSourceExecution{}
	}
	return result, 2, true, nil
}

func decodeSleepContext(object *javaser.Object) (sleepDecodedContext, error) {
	if object == nil {
		return sleepDecodedContext{}, errors.New("sleep serialization: nil saved Context")
	}
	if err := validateSleepDescriptor(object.Descriptor, sleepContextDescriptor); err != nil {
		return sleepDecodedContext{}, err
	}
	moreValue, err := sleepObjectField(object, sleepContextDescriptor.Name, "moreHandlers")
	if err != nil {
		return sleepDecodedContext{}, err
	}
	more, ok := moreValue.(javaser.Boolean)
	if !ok {
		return sleepDecodedContext{}, errors.New("sleep serialization: Context.moreHandlers is not boolean")
	}
	handlerValue, err := sleepObjectField(object, sleepContextDescriptor.Name, "handler")
	if err != nil {
		return sleepDecodedContext{}, err
	}
	handlerValueRef, ok := handlerValue.(javaser.Value)
	if !ok {
		return sleepDecodedContext{}, errors.New("sleep serialization: Context.handler is not an object reference")
	}
	var handler *sleepDecodedExceptionContext
	if !javaSerializationNull(handlerValueRef) {
		object, ok := handlerValueRef.(*javaser.Object)
		if !ok {
			return sleepDecodedContext{}, errors.New("sleep serialization: Context.handler is not an ExceptionContext object")
		}
		if err := validateSleepDescriptor(object.Descriptor, sleepExceptionContextDescriptor); err != nil {
			return sleepDecodedContext{}, err
		}
		ownerValue, err := sleepObjectField(object, sleepExceptionContextDescriptor.Name, "owner")
		if err != nil {
			return sleepDecodedContext{}, err
		}
		owner, ok := ownerValue.(*javaser.Object)
		if !ok {
			return sleepDecodedContext{}, errors.New("sleep serialization: ExceptionContext.owner is not a Block")
		}
		handlerBlockValue, err := sleepObjectField(object, sleepExceptionContextDescriptor.Name, "handler")
		if err != nil {
			return sleepDecodedContext{}, err
		}
		handlerBlock, ok := handlerBlockValue.(*javaser.Object)
		if !ok {
			return sleepDecodedContext{}, errors.New("sleep serialization: ExceptionContext.handler is not a Block")
		}
		varnameValue, err := sleepObjectField(object, sleepExceptionContextDescriptor.Name, "varname")
		if err != nil {
			return sleepDecodedContext{}, err
		}
		varname, ok := varnameValue.(*javaser.String)
		if !ok {
			return sleepDecodedContext{}, errors.New("sleep serialization: ExceptionContext.varname is not a string")
		}
		handler = &sleepDecodedExceptionContext{owner: owner, handler: handlerBlock, varname: sleepStringFromJava(varname)}
	}
	blockValue, err := sleepObjectField(object, sleepContextDescriptor.Name, "block")
	if err != nil {
		return sleepDecodedContext{}, err
	}
	block, ok := blockValue.(*javaser.Object)
	if !ok {
		return sleepDecodedContext{}, errors.New("sleep serialization: Context.block is not a Block")
	}
	if err := validateSleepDescriptor(block.Descriptor, sleepBlockDescriptor); err != nil {
		return sleepDecodedContext{}, err
	}
	lastValue, err := sleepObjectField(object, sleepContextDescriptor.Name, "last")
	if err != nil {
		return sleepDecodedContext{}, err
	}
	last, _, err := sleepStepReference(lastValue)
	if err != nil {
		return sleepDecodedContext{}, err
	}
	if handler != nil && handler.owner != block {
		return sleepDecodedContext{}, errors.New("sleep serialization: ExceptionContext.owner does not share Context.block handle")
	}
	return sleepDecodedContext{block: block, last: last, handler: handler, moreHandlers: bool(more)}, nil
}

func (state *sleepSerializationDecoder) closureLocalLevels(list *javaser.Object, closure *scriptClosure) (*scope, []*scope, error) {
	if err := validateSleepDescriptor(list.Descriptor, javaLinkedListDescriptor); err != nil {
		return nil, nil, err
	}
	data, ok := list.DataFor(javaLinkedListDescriptor.Name)
	if !ok {
		return nil, nil, errors.New("sleep serialization: local-level LinkedList has no class data")
	}
	count, values, err := sleepAnnotationCount(data.Annotation)
	if err != nil {
		return nil, nil, err
	}
	if count != len(values) || count == 0 {
		return nil, nil, errors.New("sleep serialization: suspended closure has no active local level")
	}
	levels := make([]*scope, count)
	for index, value := range values {
		variables, ok := value.(*javaser.Object)
		if !ok {
			return nil, nil, errors.New("sleep serialization: local level is not a DefaultVariable")
		}
		cells, err := state.defaultVariables(variables)
		if err != nil {
			return nil, nil, err
		}
		level, err := closure.state.localChildAt(context.Background(), Span{})
		if err != nil {
			return nil, nil, err
		}
		level.mu.Lock()
		for name, cell := range cells {
			level.cells[normalizeVariableName(name)] = cell
		}
		level.mu.Unlock()
		levels[index] = level
	}
	locals := make([]*scope, 0, len(levels)-1)
	for index := len(levels) - 1; index >= 1; index-- {
		locals = append(locals, levels[index])
	}
	return levels[0], locals, nil
}

func (graph *sleepLegacyDecodedBlock) contextPC(last *javaser.Object) (int, error) {
	if graph == nil || graph.function == nil {
		return 0, errors.New("sleep serialization: missing translated Context block")
	}
	if last == nil {
		return len(graph.function.Instructions) - 1, nil
	}
	pc, ok := graph.stepPC[last]
	if !ok {
		return 0, errors.New("sleep serialization: Context.last does not share a Step handle with Context.block")
	}
	return pc, nil
}

func directLegacyInlineCall(function *bytecode.Function, pc int) (*ast.CallExpr, bool) {
	if function == nil || pc < 0 || pc >= len(function.Instructions) {
		return nil, false
	}
	instruction := function.Instructions[pc]
	if instruction.Op != bytecode.OpEval {
		return nil, false
	}
	call, ok := instruction.Expr.(*ast.CallExpr)
	return call, ok
}

func directLegacyReturnInlineCall(function *bytecode.Function, pc int) (*ast.CallExpr, bool) {
	if function == nil || pc < 0 || pc >= len(function.Instructions) {
		return nil, false
	}
	instruction := function.Instructions[pc]
	if instruction.Op != bytecode.OpReturn {
		return nil, false
	}
	expression := instruction.Expr
	for {
		group, ok := expression.(*ast.GroupExpr)
		if !ok {
			break
		}
		expression = group.Expr
	}
	call, ok := expression.(*ast.CallExpr)
	return call, ok
}

func (state *sleepSerializationDecoder) defaultVariables(object *javaser.Object) (map[string]*Cell, error) {
	if object == nil {
		return nil, errors.New("sleep serialization: nil DefaultVariable")
	}
	if err := validateSleepDescriptor(object.Descriptor, sleepDefaultVariableDescriptor); err != nil {
		return nil, err
	}
	field, err := sleepObjectField(object, sleepDefaultVariableDescriptor.Name, "values")
	if err != nil {
		return nil, err
	}
	table, ok := field.(*javaser.Object)
	if !ok {
		return nil, errors.New("sleep serialization: DefaultVariable.values is not a Hashtable")
	}
	if err := validateSleepDescriptor(table.Descriptor, javaHashtableDescriptor); err != nil {
		return nil, err
	}
	data, ok := table.DataFor(javaHashtableDescriptor.Name)
	if !ok || len(data.Annotation) == 0 {
		return nil, errors.New("sleep serialization: invalid Hashtable custom data")
	}
	block, ok := data.Annotation[0].(*javaser.BlockData)
	if !ok || len(block.Data) != 8 {
		return nil, errors.New("sleep serialization: invalid Hashtable capacity/count data")
	}
	count := int(int32(binary.BigEndian.Uint32(block.Data[4:])))
	if count < 0 || len(data.Annotation) != 1+count*2 {
		return nil, fmt.Errorf("sleep serialization: invalid Hashtable entry count %d", count)
	}
	cells := make(map[string]*Cell, count)
	for index := 0; index < count; index++ {
		key, ok := data.Annotation[1+index*2].(*javaser.String)
		if !ok {
			return nil, errors.New("sleep serialization: DefaultVariable key is not a string")
		}
		scalar, ok := data.Annotation[2+index*2].(*javaser.Object)
		if !ok {
			return nil, errors.New("sleep serialization: DefaultVariable value is not a Scalar")
		}
		cell, err := state.scalarCell(scalar)
		if err != nil {
			return nil, err
		}
		cells[sleepStringFromJava(key)] = cell
	}
	return cells, nil
}

func (state *sleepSerializationDecoder) scalarCell(object *javaser.Object) (*Cell, error) {
	if object == nil {
		return nil, errors.New("sleep serialization: nil Scalar cell")
	}
	if state.scalarCells == nil {
		state.scalarCells = make(map[*javaser.Object]*Cell)
	}
	if cell := state.scalarCells[object]; cell != nil {
		return cell, nil
	}
	cell := NewCell(Null())
	state.scalarCells[object] = cell
	value, err := state.scalar(object)
	if err != nil {
		delete(state.scalarCells, object)
		return nil, err
	}
	cell.Set(value)
	return cell, nil
}

type sleepLegacyTranslator struct {
	decoder      *sleepSerializationDecoder
	source       string
	functionSpan Span
	frames       [][]ast.Expr
	instructions []bytecode.Instruction
	stepPC       map[*javaser.Object]int
	foreach      []*sleepLegacyDecodedForeach
	openForeach  []*sleepLegacyDecodedForeach
}

type sleepLegacyDecodedBlock struct {
	block         *javaser.Object
	function      *bytecode.Function
	stepPC        map[*javaser.Object]int
	foreachByGoto map[*javaser.Object]*sleepLegacyDecodedForeach
}

type sleepLegacyDecodedForeach struct {
	bodyBlock  *javaser.Object
	gotoStep   *javaser.Object
	key        string
	value      string
	iterNextPC int
	jumpPC     int
	destroyPC  int
	span       Span
}

// sleepSerializedForeachResume marks a Java-deserialized yielded foreach.
// SleepClosure intentionally omits ScriptEnvironment metadata, including the
// iterator cursor, from its stream. The official runtime therefore cannot
// resume this context and emits its null/EmptyStack warning pair instead.
type sleepSerializedForeachResume struct {
	iterNextPC  int
	span        Span
	includeBody bool
}

func (state *sleepSerializationDecoder) legacyFunctionGraph(block *javaser.Object) (*sleepLegacyDecodedBlock, error) {
	if block == nil {
		return nil, errors.New("sleep serialization: nil closure Block")
	}
	if state.legacyFunctions == nil {
		state.legacyFunctions = make(map[*javaser.Object]*sleepLegacyDecodedBlock)
	}
	if existing := state.legacyFunctions[block]; existing != nil {
		return existing, nil
	}
	if err := validateSleepDescriptor(block.Descriptor, sleepBlockDescriptor); err != nil {
		return nil, err
	}
	sourceValue, err := sleepObjectField(block, sleepBlockDescriptor.Name, "source")
	if err != nil {
		return nil, err
	}
	source, ok := sourceValue.(*javaser.String)
	if !ok {
		return nil, errors.New("sleep serialization: Block.source is not a string")
	}
	translator := &sleepLegacyTranslator{
		decoder: state,
		source:  sleepStringFromJava(source),
		frames:  [][]ast.Expr{nil}, // ScriptEnvironment supplies the implicit parent frame.
		stepPC:  make(map[*javaser.Object]int),
	}
	firstValue, err := sleepObjectField(block, sleepBlockDescriptor.Name, "first")
	if err != nil {
		return nil, err
	}
	lastValue, err := sleepObjectField(block, sleepBlockDescriptor.Name, "last")
	if err != nil {
		return nil, err
	}
	first, firstNull, err := sleepStepReference(firstValue)
	if err != nil {
		return nil, err
	}
	last, lastNull, err := sleepStepReference(lastValue)
	if err != nil {
		return nil, err
	}
	if firstNull != lastNull {
		return nil, errors.New("sleep serialization: Block first/last nullity differs")
	}
	seen := make(map[*javaser.Object]struct{})
	current := first
	var reached *javaser.Object
	for current != nil {
		if _, duplicate := seen[current]; duplicate {
			return nil, errors.New("sleep serialization: cyclic Step.next chain")
		}
		if len(seen) >= 100_000 {
			return nil, errors.New("sleep serialization: closure Step count exceeds 100000")
		}
		seen[current] = struct{}{}
		reached = current
		translator.stepPC[current] = len(translator.instructions)
		if err := translator.step(current); err != nil {
			return nil, err
		}
		nextValue, err := sleepObjectField(current, sleepStepDescriptor.Name, "next")
		if err != nil {
			return nil, err
		}
		next, _, err := sleepStepReference(nextValue)
		if err != nil {
			return nil, err
		}
		current = next
	}
	if len(translator.openForeach) != 0 {
		return nil, errors.New("sleep serialization: closure Block ends with an unterminated foreach graph")
	}
	if reached != last {
		return nil, errors.New("sleep serialization: Block.last is not the tail of Step.next")
	}
	if len(translator.frames) != 1 {
		return nil, errors.New("sleep serialization: closure Block ends with an unterminated evaluator frame")
	}
	translator.flushRoot()
	span := translator.functionSpan
	if span.Source == "" {
		span = legacySpan(translator.source, 1)
	}
	translator.instructions = append(translator.instructions, bytecode.Instruction{Op: bytecode.OpEnd, Span: span})
	function := &bytecode.Function{
		Name:         "<deserialized closure>",
		Span:         span,
		Instructions: translator.instructions,
	}
	foreachByGoto := make(map[*javaser.Object]*sleepLegacyDecodedForeach, len(translator.foreach))
	for _, loop := range translator.foreach {
		foreachByGoto[loop.gotoStep] = loop
	}
	graph := &sleepLegacyDecodedBlock{
		block:         block,
		function:      function,
		stepPC:        translator.stepPC,
		foreachByGoto: foreachByGoto,
	}
	state.legacyFunctions[block] = graph
	return graph, nil
}

func sleepStepReference(value javaser.Element) (*javaser.Object, bool, error) {
	reference, ok := value.(javaser.Value)
	if !ok {
		return nil, false, errors.New("sleep serialization: Step reference is not an object reference")
	}
	if javaSerializationNull(reference) {
		return nil, true, nil
	}
	object, ok := reference.(*javaser.Object)
	if !ok || object.Descriptor == nil {
		return nil, false, errors.New("sleep serialization: Step reference is not an object")
	}
	return object, false, nil
}

func legacySpan(source string, line int) Span {
	if line <= 0 {
		line = 1
	}
	return Span{
		Source: source,
		Start:  Position{Line: line, Column: 1},
		End:    Position{Line: line, Column: 1},
	}
}

func appendRebasedLegacyInstructions(destination, source []bytecode.Instruction, omitEnd bool) []bytecode.Instruction {
	limit := len(source)
	if omitEnd && limit != 0 && source[limit-1].Op == bytecode.OpEnd {
		limit--
	}
	offset := len(destination)
	for index := 0; index < limit; index++ {
		instruction := source[index]
		switch instruction.Op {
		case bytecode.OpJump, bytecode.OpJumpFalse, bytecode.OpIterNext, bytecode.OpEnterTry:
			if instruction.Target >= 0 {
				instruction.Target += offset
			}
		}
		destination = append(destination, instruction)
	}
	return destination
}

func slicedLegacyInstructions(source []bytecode.Instruction, start, end int) []bytecode.Instruction {
	if start < 0 || end < start || end > len(source) {
		return nil
	}
	result := append([]bytecode.Instruction(nil), source[start:end]...)
	for index := range result {
		switch result[index].Op {
		case bytecode.OpJump, bytecode.OpJumpFalse, bytecode.OpIterNext, bytecode.OpEnterTry:
			if result[index].Target >= start && result[index].Target <= end {
				result[index].Target -= start
			}
		}
	}
	return result
}

func (translator *sleepLegacyTranslator) step(object *javaser.Object) error {
	if object == nil || object.Descriptor == nil {
		return errors.New("sleep serialization: invalid closure Step")
	}
	lineValue, err := sleepObjectField(object, sleepStepDescriptor.Name, "line")
	if err != nil {
		return err
	}
	line, ok := lineValue.(javaser.Int)
	if !ok {
		return errors.New("sleep serialization: Step.line is not an int")
	}
	span := legacySpan(translator.source, int(line))
	if translator.functionSpan.Source == "" {
		translator.functionSpan = span
	} else {
		if span.Start.Line < translator.functionSpan.Start.Line {
			translator.functionSpan.Start = span.Start
		}
		if span.End.Line > translator.functionSpan.End.Line {
			translator.functionSpan.End = span.End
		}
	}

	switch object.Descriptor.Name {
	case sleepCreateFrameDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepCreateFrameDescriptor); err != nil {
			return err
		}
		translator.frames = append(translator.frames, nil)
		return nil
	case sleepGetDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepGetDescriptor); err != nil {
			return err
		}
		name, err := sleepStepStringField(object, sleepGetDescriptor.Name, "value")
		if err != nil {
			return err
		}
		translator.push(legacyGetExpression(name, span))
		return nil
	case sleepSValueDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepSValueDescriptor); err != nil {
			return err
		}
		field, err := sleepObjectField(object, sleepSValueDescriptor.Name, "value")
		if err != nil {
			return err
		}
		scalar, ok := field.(*javaser.Object)
		if !ok {
			return errors.New("sleep serialization: SValue.value is not a Scalar")
		}
		value, err := translator.decoder.scalar(scalar)
		if err != nil {
			return err
		}
		expression, err := legacyValueExpression(value, span)
		if err != nil {
			return err
		}
		translator.push(expression)
		return nil
	case sleepPLiteralDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepPLiteralDescriptor); err != nil {
			return err
		}
		expression, err := translator.parsedLiteral(object, span)
		if err != nil {
			return err
		}
		return translator.collapse(expression, span)
	case sleepCreateClosureDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepCreateClosureDescriptor); err != nil {
			return err
		}
		blockValue, err := sleepObjectField(object, sleepCreateClosureDescriptor.Name, "block")
		if err != nil {
			return err
		}
		block, ok := blockValue.(*javaser.Object)
		if !ok {
			return errors.New("sleep serialization: CreateClosure.block is not a Block")
		}
		graph, err := translator.decoder.legacyFunctionGraph(block)
		if err != nil {
			return err
		}
		body, err := legacyFunctionBlock(graph.function)
		if err != nil {
			return err
		}
		translator.push(&ast.ClosureExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
			Body:     body,
		})
		return nil
	case sleepTryDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepTryDescriptor); err != nil {
			return err
		}
		ownerValue, err := sleepObjectField(object, sleepTryDescriptor.Name, "owner")
		if err != nil {
			return err
		}
		owner, ok := ownerValue.(*javaser.Object)
		if !ok {
			return errors.New("sleep serialization: Try.owner is not a Block")
		}
		handlerValue, err := sleepObjectField(object, sleepTryDescriptor.Name, "handler")
		if err != nil {
			return err
		}
		handler, ok := handlerValue.(*javaser.Object)
		if !ok {
			return errors.New("sleep serialization: Try.handler is not a Block")
		}
		variable, err := sleepStepStringField(object, sleepTryDescriptor.Name, "var")
		if err != nil {
			return err
		}
		ownerGraph, err := translator.decoder.legacyFunctionGraph(owner)
		if err != nil {
			return err
		}
		handlerGraph, err := translator.decoder.legacyFunctionGraph(handler)
		if err != nil {
			return err
		}
		translator.flushRootAt(span)
		enterPC := len(translator.instructions)
		translator.instructions = append(translator.instructions, bytecode.Instruction{Op: bytecode.OpEnterTry, Span: span, Target: -1})
		translator.instructions = appendRebasedLegacyInstructions(translator.instructions, ownerGraph.function.Instructions, true)
		jumpPC := len(translator.instructions)
		translator.instructions = append(translator.instructions, bytecode.Instruction{Op: bytecode.OpJump, Span: span, Target: -1})
		catchPC := len(translator.instructions)
		translator.instructions[enterPC].Target = catchPC
		translator.instructions = append(translator.instructions, bytecode.Instruction{Op: bytecode.OpCatch, Span: span, Name: variable})
		translator.instructions = appendRebasedLegacyInstructions(translator.instructions, handlerGraph.function.Instructions, true)
		translator.instructions[jumpPC].Target = len(translator.instructions)
		return nil
	case sleepPopTryDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepPopTryDescriptor); err != nil {
			return err
		}
		translator.flushRootAt(span)
		translator.instructions = append(translator.instructions, bytecode.Instruction{Op: bytecode.OpLeaveTry, Span: span})
		return nil
	case sleepIterateDescriptor.Name:
		return translator.iterate(object, span)
	case sleepGotoDescriptor.Name:
		return translator.foreachGoto(object, span)
	case sleepCallDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepCallDescriptor); err != nil {
			return err
		}
		name, err := sleepStepStringField(object, sleepCallDescriptor.Name, "function")
		if err != nil {
			return err
		}
		frame, err := translator.popFrame()
		if err != nil {
			return err
		}
		arguments := make([]ast.Expr, len(frame))
		for index := range frame {
			arguments[len(frame)-1-index] = frame[index]
		}
		name = strings.TrimPrefix(name, "&")
		expression := &ast.CallExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
			Callee:   &ast.IdentifierExpr{ExprBase: ast.ExprBase{Base: ast.Base{Range: span}}, Name: name},
			Args:     arguments,
		}
		return translator.pushCollapsed(expression, span)
	case sleepObjectAccessDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepObjectAccessDescriptor); err != nil {
			return err
		}
		frame, err := translator.popFrame()
		if err != nil {
			return err
		}
		classValue, err := sleepObjectField(object, sleepObjectAccessDescriptor.Name, "classRef")
		if err != nil {
			return err
		}
		nameValue, err := sleepObjectField(object, sleepObjectAccessDescriptor.Name, "name")
		if err != nil {
			return err
		}
		var message *ast.ObjectMessage
		switch value := nameValue.(type) {
		case *javaser.Null:
		case *javaser.String:
			message = &ast.ObjectMessage{Range: span, Name: sleepStringFromJava(value)}
		default:
			return errors.New("sleep serialization: ObjectAccess.name is neither string nor null")
		}
		if class, ok := classValue.(*javaser.Class); ok {
			className, err := sleepPortableStaticClass(class)
			if err != nil {
				return err
			}
			if message == nil {
				return errors.New("sleep serialization: static ObjectAccess.name is null")
			}
			arguments := reverseLegacyFrame(frame)
			name := message.Name
			if className == "SleepUtils" && name == "getIOHandle" {
				if expression, ok := legacyConsoleHandleExpression(arguments, span); ok {
					return translator.pushCollapsed(expression, span)
				}
			}
			expression := &ast.ObjectExpr{
				ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
				Target: &ast.IdentifierExpr{
					ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
					Name:     className,
				},
				Message: message,
				Args:    arguments,
			}
			return translator.pushCollapsed(expression, span)
		}
		class, ok := classValue.(javaser.Value)
		if !ok || !javaSerializationNull(class) {
			return errors.New("sleep serialization: ObjectAccess.classRef is neither Class nor null")
		}
		if len(frame) == 0 {
			return errors.New("sleep serialization: instance ObjectAccess frame has no target")
		}
		expression := &ast.ObjectExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
			Target:   frame[len(frame)-1],
			Message:  message,
			Args:     reverseLegacyFrame(frame[:len(frame)-1]),
		}
		return translator.pushCollapsed(expression, span)
	case sleepDecideDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepDecideDescriptor); err != nil {
			return err
		}
		conditionValue, err := sleepObjectField(object, sleepDecideDescriptor.Name, "start")
		if err != nil {
			return err
		}
		condition, ok := conditionValue.(*javaser.Object)
		if !ok {
			return errors.New("sleep serialization: Decide.start is not a Check")
		}
		predicate, err := translator.checkExpression(condition, span)
		if err != nil {
			return err
		}
		trueBranch, err := translator.decideBranchExpression(object, "iftrue")
		if err != nil {
			return err
		}
		falseBranch, err := translator.decideBranchExpression(object, "iffalse")
		if err != nil {
			return err
		}
		expression := &ast.CallExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
			Callee: &ast.IdentifierExpr{
				ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
				Name:     "iff",
			},
			Args: []ast.Expr{predicate, trueBranch, falseBranch},
		}
		return translator.pushCollapsed(expression, span)
	case sleepOperateDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepOperateDescriptor); err != nil {
			return err
		}
		operator, err := sleepStepStringField(object, sleepOperateDescriptor.Name, "oper")
		if err != nil {
			return err
		}
		frame, err := translator.popFrame()
		if err != nil {
			return err
		}
		if len(frame) < 2 {
			return fmt.Errorf("sleep serialization: Operate %q has %d operands", operator, len(frame))
		}
		if len(frame) > 2 {
			expression := &ast.ParameterOperatorExpr{
				ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
				Left:     frame[len(frame)-1],
				Op:       operator,
				Right:    append([]ast.Expr(nil), frame[:len(frame)-1]...),
			}
			return translator.pushCollapsed(expression, span)
		}
		expression := &ast.BinaryExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
			Left:     frame[1],
			Op:       operator,
			Right:    frame[0],
		}
		return translator.pushCollapsed(expression, span)
	case sleepAssignDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepAssignDescriptor); err != nil {
			return err
		}
		frame, err := translator.popFrame()
		if err != nil {
			return err
		}
		if len(frame) != 1 {
			return fmt.Errorf("sleep serialization: Assign evaluator frame has %d values", len(frame))
		}
		variableValue, err := sleepObjectField(object, sleepAssignDescriptor.Name, "variable")
		if err != nil {
			return err
		}
		variable, ok := variableValue.(*javaser.Object)
		if !ok {
			return errors.New("sleep serialization: Assign.variable is not a Block")
		}
		target, err := translator.blockExpression(variable)
		if err != nil {
			return err
		}
		operatorValue, err := sleepObjectField(object, sleepAssignDescriptor.Name, "operator")
		if err != nil {
			return err
		}
		operator := "="
		if reference, ok := operatorValue.(javaser.Value); !ok {
			return errors.New("sleep serialization: Assign.operator is not an object reference")
		} else if !javaSerializationNull(reference) {
			operation, ok := reference.(*javaser.Object)
			if !ok {
				return errors.New("sleep serialization: Assign.operator is not a Step")
			}
			if err := validateSleepDescriptor(operation.Descriptor, sleepOperateDescriptor); err != nil {
				return &UnsupportedError{Operation: "serialization", Name: "compound Assign Step " + operation.Descriptor.Name}
			}
			base, err := sleepStepStringField(operation, sleepOperateDescriptor.Name, "oper")
			if err != nil {
				return err
			}
			operator = base + "="
		}
		expression := &ast.AssignExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
			Target:   target,
			Op:       operator,
			Value:    frame[0],
		}
		return translator.pushCollapsed(expression, span)
	case sleepReturnDescriptor.Name:
		if err := validateSleepDescriptor(object.Descriptor, sleepReturnDescriptor); err != nil {
			return err
		}
		kindValue, err := sleepObjectField(object, sleepReturnDescriptor.Name, "return_type")
		if err != nil {
			return err
		}
		kind, ok := kindValue.(javaser.Int)
		if !ok {
			return errors.New("sleep serialization: Return.return_type is not an int")
		}
		frame, err := translator.popFrame()
		if err != nil {
			return err
		}
		var expression ast.Expr
		if len(frame) > 1 {
			return errors.New("sleep serialization: Return evaluator frame has multiple values")
		}
		if len(frame) == 1 {
			expression = frame[0]
		}
		var operation bytecode.Op
		switch int(kind) {
		case sleepFlowReturn:
			operation = bytecode.OpReturn
		case sleepFlowYield:
			operation = bytecode.OpYield
		case sleepFlowThrow:
			operation = bytecode.OpThrow
		case sleepFlowBreak, sleepFlowContinue:
			return &UnsupportedError{Operation: "serialization", Name: "detached break/continue closure Step"}
		case sleepFlowCallCC:
			operation = bytecode.OpCallCC
		default:
			return fmt.Errorf("sleep serialization: unknown Return.return_type %d", kind)
		}
		translator.flushRoot()
		translator.instructions = append(translator.instructions, bytecode.Instruction{Op: operation, Span: span, Expr: expression})
		return nil
	default:
		return &UnsupportedError{Operation: "serialization", Name: "Sleep closure Step " + object.Descriptor.Name}
	}
}

func (translator *sleepLegacyTranslator) iterate(object *javaser.Object, span Span) error {
	if err := validateSleepDescriptor(object.Descriptor, sleepIterateDescriptor); err != nil {
		return err
	}
	typeValue, err := sleepObjectField(object, sleepIterateDescriptor.Name, "type")
	if err != nil {
		return err
	}
	kind, ok := typeValue.(javaser.Int)
	if !ok {
		return errors.New("sleep serialization: Iterate.type is not an int")
	}
	key, keyNull, err := sleepNullableStepStringField(object, sleepIterateDescriptor.Name, "key")
	if err != nil {
		return err
	}
	value, valueNull, err := sleepNullableStepStringField(object, sleepIterateDescriptor.Name, "value")
	if err != nil {
		return err
	}

	switch int(kind) {
	case sleepIteratorCreate:
		if valueNull || value == "" {
			return errors.New("sleep serialization: foreach Iterate CREATE has no value variable")
		}
		if len(translator.openForeach) != 0 {
			return &UnsupportedError{Operation: "serialization", Name: "nested foreach Sleep closure graph"}
		}
		frame, err := translator.popFrame()
		if err != nil {
			return fmt.Errorf("sleep serialization: foreach Iterate CREATE: %w", err)
		}
		if len(frame) != 1 {
			return fmt.Errorf("sleep serialization: foreach Iterate CREATE frame has %d values", len(frame))
		}
		if !keyNull && key == "" {
			return errors.New("sleep serialization: foreach Iterate CREATE has an empty key variable")
		}
		translator.flushRoot()
		translator.instructions = append(translator.instructions, bytecode.Instruction{
			Op:    bytecode.OpIterInit,
			Span:  span,
			Expr:  frame[0],
			Name:  key,
			Name2: value,
		})
		loop := &sleepLegacyDecodedForeach{key: key, value: value, span: span, iterNextPC: -1, jumpPC: -1, destroyPC: -1}
		translator.openForeach = append(translator.openForeach, loop)
		return nil
	case sleepIteratorDestroy:
		if !keyNull || !valueNull {
			return errors.New("sleep serialization: foreach Iterate DESTROY carries variable names")
		}
		if len(translator.openForeach) == 0 {
			return errors.New("sleep serialization: foreach Iterate DESTROY has no owning CREATE")
		}
		loop := translator.openForeach[len(translator.openForeach)-1]
		if loop.gotoStep == nil || loop.iterNextPC < 0 {
			return errors.New("sleep serialization: foreach Iterate DESTROY precedes its Goto")
		}
		loop.destroyPC = len(translator.instructions)
		translator.instructions[loop.iterNextPC].Target = loop.destroyPC
		translator.instructions = append(translator.instructions, bytecode.Instruction{Op: bytecode.OpIterDestroy, Span: span})
		translator.openForeach = translator.openForeach[:len(translator.openForeach)-1]
		translator.foreach = append(translator.foreach, loop)
		return nil
	case sleepIteratorNext:
		return &UnsupportedError{Operation: "serialization", Name: "detached foreach Iterate NEXT Step"}
	default:
		return fmt.Errorf("sleep serialization: unknown Iterate.type %d", kind)
	}
}

func (translator *sleepLegacyTranslator) foreachGoto(object *javaser.Object, span Span) error {
	if err := validateSleepDescriptor(object.Descriptor, sleepGotoDescriptor); err != nil {
		return err
	}
	if len(translator.openForeach) == 0 {
		return &UnsupportedError{Operation: "serialization", Name: "Goto without foreach Iterate CREATE"}
	}
	loop := translator.openForeach[len(translator.openForeach)-1]
	if loop.gotoStep != nil {
		return &UnsupportedError{Operation: "serialization", Name: "multiple Goto Steps in one foreach graph"}
	}
	if len(translator.frames) != 1 || len(translator.frames[0]) != 0 {
		return errors.New("sleep serialization: foreach Goto has a pending evaluator frame")
	}
	bodyValue, err := sleepObjectField(object, sleepGotoDescriptor.Name, "iftrue")
	if err != nil {
		return err
	}
	body, ok := bodyValue.(*javaser.Object)
	if !ok {
		return errors.New("sleep serialization: foreach Goto.iftrue is not a Block")
	}
	if err := validateSleepDescriptor(body.Descriptor, sleepBlockDescriptor); err != nil {
		return err
	}
	incrementValue, err := sleepObjectField(object, sleepGotoDescriptor.Name, "increment")
	if err != nil {
		return err
	}
	increment, ok := incrementValue.(javaser.Value)
	if !ok || !javaSerializationNull(increment) {
		return &UnsupportedError{Operation: "serialization", Name: "foreach Goto with increment Block"}
	}
	startValue, err := sleepObjectField(object, sleepGotoDescriptor.Name, "start")
	if err != nil {
		return err
	}
	start, ok := startValue.(*javaser.Object)
	if !ok {
		return errors.New("sleep serialization: foreach Goto.start is not a CheckEval")
	}
	if err := validateSleepDescriptor(start.Descriptor, sleepCheckEvalDescriptor); err != nil {
		return err
	}
	if err := validateSleepForeachCheck(start); err != nil {
		return err
	}

	bodyGraph, err := translator.decoder.legacyFunctionGraph(body)
	if err != nil {
		return err
	}
	loop.bodyBlock = body
	loop.gotoStep = object
	loop.span = span
	loop.iterNextPC = len(translator.instructions)
	translator.instructions = append(translator.instructions, bytecode.Instruction{
		Op:     bytecode.OpIterNext,
		Span:   span,
		Name:   loop.key,
		Name2:  loop.value,
		Target: -1,
	})
	translator.instructions = appendRebasedLegacyInstructions(translator.instructions, bodyGraph.function.Instructions, true)
	loop.jumpPC = len(translator.instructions)
	translator.instructions = append(translator.instructions, bytecode.Instruction{
		Op:     bytecode.OpJump,
		Span:   span,
		Target: loop.iterNextPC,
	})
	return nil
}

func validateSleepForeachCheck(check *javaser.Object) error {
	name, err := sleepStepStringField(check, sleepCheckEvalDescriptor.Name, "name")
	if err != nil {
		return err
	}
	if name != "-istrue" {
		return &UnsupportedError{Operation: "serialization", Name: "foreach CheckEval predicate " + name}
	}
	negateValue, err := sleepObjectField(check, sleepCheckEvalDescriptor.Name, "negate")
	if err != nil {
		return err
	}
	negate, ok := negateValue.(javaser.Boolean)
	if !ok || bool(negate) {
		return errors.New("sleep serialization: foreach CheckEval.negate is not false")
	}
	for _, fieldName := range []string{"iftrue", "iffalse"} {
		value, err := sleepObjectField(check, sleepCheckEvalDescriptor.Name, fieldName)
		if err != nil {
			return err
		}
		reference, ok := value.(javaser.Value)
		if !ok || !javaSerializationNull(reference) {
			return &UnsupportedError{Operation: "serialization", Name: "foreach CheckEval with linked " + fieldName}
		}
	}
	setupValue, err := sleepObjectField(check, sleepCheckEvalDescriptor.Name, "setup")
	if err != nil {
		return err
	}
	setup, ok := setupValue.(*javaser.Object)
	if !ok {
		return errors.New("sleep serialization: foreach CheckEval.setup is not a Block")
	}
	firstValue, err := sleepObjectField(setup, sleepBlockDescriptor.Name, "first")
	if err != nil {
		return err
	}
	lastValue, err := sleepObjectField(setup, sleepBlockDescriptor.Name, "last")
	if err != nil {
		return err
	}
	first, firstNull, err := sleepStepReference(firstValue)
	if err != nil {
		return err
	}
	last, lastNull, err := sleepStepReference(lastValue)
	if err != nil {
		return err
	}
	if firstNull || lastNull || first != last || first.Descriptor == nil || first.Descriptor.Name != sleepIterateDescriptor.Name {
		return &UnsupportedError{Operation: "serialization", Name: "foreach CheckEval setup is not one Iterate NEXT Step"}
	}
	if err := validateSleepDescriptor(first.Descriptor, sleepIterateDescriptor); err != nil {
		return err
	}
	typeValue, err := sleepObjectField(first, sleepIterateDescriptor.Name, "type")
	if err != nil {
		return err
	}
	kind, ok := typeValue.(javaser.Int)
	if !ok || int(kind) != sleepIteratorNext {
		return &UnsupportedError{Operation: "serialization", Name: "foreach CheckEval setup Iterate is not NEXT"}
	}
	for _, fieldName := range []string{"key", "value"} {
		_, null, err := sleepNullableStepStringField(first, sleepIterateDescriptor.Name, fieldName)
		if err != nil {
			return err
		}
		if !null {
			return errors.New("sleep serialization: foreach Iterate NEXT carries variable names")
		}
	}
	return nil
}

func (translator *sleepLegacyTranslator) decideBranchExpression(decide *javaser.Object, fieldName string) (ast.Expr, error) {
	value, err := sleepObjectField(decide, sleepDecideDescriptor.Name, fieldName)
	if err != nil {
		return nil, err
	}
	block, ok := value.(*javaser.Object)
	if !ok {
		return nil, fmt.Errorf("sleep serialization: Decide.%s is not a Block", fieldName)
	}
	expressions, err := translator.blockResultExpressions(block)
	if err != nil {
		return nil, err
	}
	if len(expressions) != 1 {
		return nil, &UnsupportedError{Operation: "serialization", Name: fmt.Sprintf("Decide.%s Block with %d results", fieldName, len(expressions))}
	}
	return expressions[0], nil
}

func (translator *sleepLegacyTranslator) checkExpression(check *javaser.Object, span Span) (ast.Expr, error) {
	if check == nil || check.Descriptor == nil {
		return nil, errors.New("sleep serialization: invalid Decide Check")
	}
	switch check.Descriptor.Name {
	case sleepCheckAndDescriptor.Name, sleepCheckOrDescriptor.Name:
		descriptor := sleepCheckAndDescriptor
		operator := "&&"
		if check.Descriptor.Name == sleepCheckOrDescriptor.Name {
			descriptor = sleepCheckOrDescriptor
			operator = "||"
		}
		if err := validateSleepDescriptor(check.Descriptor, descriptor); err != nil {
			return nil, err
		}
		leftValue, err := sleepObjectField(check, descriptor.Name, "left")
		if err != nil {
			return nil, err
		}
		rightValue, err := sleepObjectField(check, descriptor.Name, "right")
		if err != nil {
			return nil, err
		}
		leftObject, leftOK := leftValue.(*javaser.Object)
		rightObject, rightOK := rightValue.(*javaser.Object)
		if !leftOK || !rightOK {
			return nil, errors.New("sleep serialization: composite Check child is not a Check")
		}
		left, err := translator.checkExpression(leftObject, span)
		if err != nil {
			return nil, err
		}
		right, err := translator.checkExpression(rightObject, span)
		if err != nil {
			return nil, err
		}
		return &ast.BinaryExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
			Left:     left,
			Op:       operator,
			Right:    right,
		}, nil
	case sleepCheckEvalDescriptor.Name:
		if err := validateSleepDescriptor(check.Descriptor, sleepCheckEvalDescriptor); err != nil {
			return nil, err
		}
		name, err := sleepStepStringField(check, sleepCheckEvalDescriptor.Name, "name")
		if err != nil {
			return nil, err
		}
		negateValue, err := sleepObjectField(check, sleepCheckEvalDescriptor.Name, "negate")
		if err != nil {
			return nil, err
		}
		negate, ok := negateValue.(javaser.Boolean)
		if !ok {
			return nil, errors.New("sleep serialization: CheckEval.negate is not boolean")
		}
		for _, field := range []string{"iftrue", "iffalse"} {
			value, err := sleepObjectField(check, sleepCheckEvalDescriptor.Name, field)
			if err != nil {
				return nil, err
			}
			reference, ok := value.(javaser.Value)
			if !ok || !javaSerializationNull(reference) {
				return nil, &UnsupportedError{Operation: "serialization", Name: "CheckEval with linked " + field}
			}
		}
		setupValue, err := sleepObjectField(check, sleepCheckEvalDescriptor.Name, "setup")
		if err != nil {
			return nil, err
		}
		setup, ok := setupValue.(*javaser.Object)
		if !ok {
			return nil, errors.New("sleep serialization: CheckEval.setup is not a Block")
		}
		expressions, err := translator.blockResultExpressions(setup)
		if err != nil {
			return nil, err
		}
		var result ast.Expr
		switch len(expressions) {
		case 1:
			result = &ast.UnaryExpr{
				ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
				Op:       name,
				Operand:  expressions[0],
			}
		case 2:
			result = &ast.BinaryExpr{
				ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
				Left:     expressions[0],
				Op:       name,
				Right:    expressions[1],
			}
		default:
			return nil, &UnsupportedError{Operation: "serialization", Name: fmt.Sprintf("CheckEval setup with %d operands", len(expressions))}
		}
		if bool(negate) {
			result = &ast.UnaryExpr{
				ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
				Op:       "!",
				Operand:  result,
			}
		}
		return result, nil
	default:
		return nil, &UnsupportedError{Operation: "serialization", Name: "Decide Check " + check.Descriptor.Name}
	}
}

func (translator *sleepLegacyTranslator) blockResultExpressions(block *javaser.Object) ([]ast.Expr, error) {
	graph, err := translator.decoder.legacyFunctionGraph(block)
	if err != nil {
		return nil, err
	}
	result := make([]ast.Expr, 0, len(graph.function.Instructions))
	for _, instruction := range graph.function.Instructions {
		switch instruction.Op {
		case bytecode.OpEval:
			result = append(result, instruction.Expr)
		case bytecode.OpEnd:
		default:
			return nil, &UnsupportedError{Operation: "serialization", Name: "result Block bytecode " + instruction.Op.String()}
		}
	}
	return result, nil
}

func reverseLegacyFrame(frame []ast.Expr) []ast.Expr {
	result := make([]ast.Expr, len(frame))
	for index := range frame {
		result[len(frame)-1-index] = frame[index]
	}
	return result
}

func legacyConsoleHandleExpression(arguments []ast.Expr, span Span) (ast.Expr, bool) {
	if len(arguments) != 2 {
		return nil, false
	}
	if _, ok := arguments[0].(*ast.NullExpr); !ok {
		return nil, false
	}
	stream, ok := arguments[1].(*ast.ObjectExpr)
	if !ok || stream.Message == nil || stream.Message.Name != "out" || len(stream.Args) != 0 {
		return nil, false
	}
	target, ok := stream.Target.(*ast.IdentifierExpr)
	if !ok || target.Name != "System" {
		return nil, false
	}
	return &ast.CallExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
		Callee: &ast.IdentifierExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
			Name:     "getConsole",
		},
	}, true
}

func sleepStepStringField(object *javaser.Object, className, fieldName string) (string, error) {
	field, err := sleepObjectField(object, className, fieldName)
	if err != nil {
		return "", err
	}
	value, ok := field.(*javaser.String)
	if !ok {
		return "", fmt.Errorf("sleep serialization: %s.%s is not a string", className, fieldName)
	}
	return sleepStringFromJava(value), nil
}

func sleepNullableStepStringField(object *javaser.Object, className, fieldName string) (string, bool, error) {
	field, err := sleepObjectField(object, className, fieldName)
	if err != nil {
		return "", false, err
	}
	if value, ok := field.(*javaser.String); ok {
		return sleepStringFromJava(value), false, nil
	}
	reference, ok := field.(javaser.Value)
	if ok && javaSerializationNull(reference) {
		return "", true, nil
	}
	return "", false, fmt.Errorf("sleep serialization: %s.%s is neither string nor null", className, fieldName)
}

func sleepPortableStaticClass(class *javaser.Class) (string, error) {
	if class == nil || class.Descriptor == nil {
		return "", errors.New("sleep serialization: ObjectAccess.classRef is not a Class")
	}
	var descriptor *javaser.ClassDesc
	var short string
	switch class.Descriptor.Name {
	case sleepJavaSystemClassDescriptor.Name:
		descriptor, short = sleepJavaSystemClassDescriptor, "System"
	case sleepSleepUtilsClassDescriptor.Name:
		descriptor, short = sleepSleepUtilsClassDescriptor, "SleepUtils"
	default:
		return "", &UnsupportedError{Operation: "serialization", Name: "static ObjectAccess class " + class.Descriptor.Name}
	}
	if err := validateSleepDescriptor(class.Descriptor, descriptor); err != nil {
		return "", err
	}
	return short, nil
}

func (translator *sleepLegacyTranslator) parsedLiteral(object *javaser.Object, span Span) (ast.Expr, error) {
	field, err := sleepObjectField(object, sleepPLiteralDescriptor.Name, "fragments")
	if err != nil {
		return nil, err
	}
	list, ok := field.(*javaser.Object)
	if !ok || list.Descriptor == nil {
		return nil, errors.New("sleep serialization: PLiteral.fragments is not a list")
	}
	if err := validateSleepDescriptor(list.Descriptor, javaLinkedListDescriptor); err != nil {
		return nil, err
	}
	data, ok := list.DataFor(javaLinkedListDescriptor.Name)
	if !ok {
		return nil, errors.New("sleep serialization: PLiteral LinkedList has no class data")
	}
	count, values, err := sleepAnnotationCount(data.Annotation)
	if err != nil {
		return nil, err
	}
	if count != len(values) {
		return nil, errors.New("sleep serialization: PLiteral fragment count mismatch")
	}
	frame := translator.currentFrame()
	valueIndex := 0
	parts := make([]ast.Expr, 0, count)
	for _, value := range values {
		fragment, ok := value.(*javaser.Object)
		if !ok {
			return nil, errors.New("sleep serialization: PLiteral fragment is not an object")
		}
		if err := validateSleepDescriptor(fragment.Descriptor, sleepPLiteralFragmentDescriptor); err != nil {
			return nil, err
		}
		typeValue, err := sleepObjectField(fragment, sleepPLiteralFragmentDescriptor.Name, "type")
		if err != nil {
			return nil, err
		}
		fragmentType, ok := typeValue.(javaser.Int)
		if !ok {
			return nil, errors.New("sleep serialization: PLiteral Fragment.type is not an int")
		}
		switch int(fragmentType) {
		case 1:
			element, err := sleepObjectField(fragment, sleepPLiteralFragmentDescriptor.Name, "element")
			if err != nil {
				return nil, err
			}
			text, ok := element.(*javaser.String)
			if !ok {
				return nil, errors.New("sleep serialization: string Fragment.element is not a string")
			}
			parts = append(parts, legacyStringExpression(sleepStringFromJava(text), span))
		case 2:
			return nil, &UnsupportedError{Operation: "serialization", Name: "aligned parsed-literal closure Step"}
		case 3:
			if valueIndex >= len(frame) {
				return nil, errors.New("sleep serialization: PLiteral has too few evaluated values")
			}
			parts = append(parts, frame[valueIndex])
			valueIndex++
		default:
			return nil, fmt.Errorf("sleep serialization: unknown PLiteral fragment type %d", fragmentType)
		}
	}
	if valueIndex != len(frame) {
		return nil, errors.New("sleep serialization: PLiteral has unused evaluated values")
	}
	return legacyConcatenation(parts, span), nil
}

func (translator *sleepLegacyTranslator) blockExpression(block *javaser.Object) (ast.Expr, error) {
	if block == nil {
		return nil, errors.New("sleep serialization: nil expression Block")
	}
	if err := validateSleepDescriptor(block.Descriptor, sleepBlockDescriptor); err != nil {
		return nil, err
	}
	firstValue, err := sleepObjectField(block, sleepBlockDescriptor.Name, "first")
	if err != nil {
		return nil, err
	}
	lastValue, err := sleepObjectField(block, sleepBlockDescriptor.Name, "last")
	if err != nil {
		return nil, err
	}
	first, firstNull, err := sleepStepReference(firstValue)
	if err != nil {
		return nil, err
	}
	last, lastNull, err := sleepStepReference(lastValue)
	if err != nil {
		return nil, err
	}
	if firstNull != lastNull || firstNull {
		return nil, errors.New("sleep serialization: assignment target Block is empty or malformed")
	}
	nested := &sleepLegacyTranslator{
		decoder: translator.decoder,
		source:  translator.source,
		frames:  [][]ast.Expr{nil},
	}
	seen := make(map[*javaser.Object]struct{})
	current := first
	var reached *javaser.Object
	for current != nil {
		if _, duplicate := seen[current]; duplicate {
			return nil, errors.New("sleep serialization: cyclic assignment target Step.next chain")
		}
		seen[current] = struct{}{}
		reached = current
		if err := nested.step(current); err != nil {
			return nil, err
		}
		nextValue, err := sleepObjectField(current, sleepStepDescriptor.Name, "next")
		if err != nil {
			return nil, err
		}
		current, _, err = sleepStepReference(nextValue)
		if err != nil {
			return nil, err
		}
	}
	if reached != last {
		return nil, errors.New("sleep serialization: assignment target Block.last is not the tail")
	}
	if len(nested.frames) != 1 || len(nested.instructions) != 0 || len(nested.frames[0]) != 1 {
		return nil, &UnsupportedError{Operation: "serialization", Name: "complex assignment target Block"}
	}
	return nested.frames[0][0], nil
}

func legacyConcatenation(parts []ast.Expr, span Span) ast.Expr {
	if len(parts) == 0 {
		return legacyStringExpression("", span)
	}
	expression := parts[0]
	for _, part := range parts[1:] {
		expression = &ast.BinaryExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
			Left:     expression,
			Op:       ".",
			Right:    part,
		}
	}
	return expression
}

func legacyStringExpression(value string, span Span) ast.Expr {
	return &ast.StringExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
		Kind:     ast.SingleQuotedString,
		Text:     value,
		Raw:      value,
	}
}

func legacyGetExpression(name string, span Span) ast.Expr {
	if strings.HasPrefix(name, "&") {
		return &ast.FunctionRefExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
			Name:     strings.TrimPrefix(name, "&"),
			Raw:      name,
		}
	}
	kind := ast.ScalarVariable
	if strings.HasPrefix(name, "@") {
		kind = ast.ArrayVariable
	} else if strings.HasPrefix(name, "%") {
		kind = ast.HashVariable
	}
	return &ast.VariableExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
		Kind:     kind,
		Name:     strings.TrimLeft(name, "$@%"),
		Raw:      name,
	}
}

func legacyValueExpression(value Value, span Span) (ast.Expr, error) {
	base := ast.ExprBase{Base: ast.Base{Range: span}}
	switch value.Kind() {
	case KindNull:
		return &ast.NullExpr{ExprBase: base, Raw: "$null"}, nil
	case KindInt:
		text := strconv.FormatInt(int64(value.Int32()), 10)
		return &ast.NumberExpr{ExprBase: base, Kind: ast.IntegerNumber, Raw: text, Text: text}, nil
	case KindLong:
		text := strconv.FormatInt(value.Int64(), 10)
		return &ast.NumberExpr{ExprBase: base, Kind: ast.LongNumber, Raw: text + "L", Text: text}, nil
	case KindDouble:
		text := strconv.FormatFloat(value.Float64(), 'g', -1, 64)
		return &ast.NumberExpr{ExprBase: base, Kind: ast.DoubleNumber, Raw: text, Text: text}, nil
	case KindString:
		return legacyStringExpression(value.String(), span), nil
	default:
		return nil, &UnsupportedError{Operation: "serialization", Name: "SValue kind " + value.Kind().String()}
	}
}

func legacyFunctionBlock(function *bytecode.Function) (*ast.BlockStmt, error) {
	if function == nil {
		return nil, errors.New("sleep serialization: nil CreateClosure function")
	}
	statements := make([]ast.Stmt, 0, len(function.Instructions))
	for _, instruction := range function.Instructions {
		base := ast.StmtBase{Base: ast.Base{Range: instruction.Span}}
		switch instruction.Op {
		case bytecode.OpEval:
			statements = append(statements, &ast.ExprStmt{StmtBase: base, Expr: instruction.Expr})
		case bytecode.OpReturn:
			statements = append(statements, &ast.ReturnStmt{StmtBase: base, Value: instruction.Expr})
		case bytecode.OpYield:
			statements = append(statements, &ast.YieldStmt{StmtBase: base, Value: instruction.Expr})
		case bytecode.OpThrow:
			statements = append(statements, &ast.ThrowStmt{StmtBase: base, Value: instruction.Expr})
		case bytecode.OpCallCC:
			statements = append(statements, &ast.CallCCStmt{StmtBase: base, Closure: instruction.Expr})
		case bytecode.OpHalt:
			statements = append(statements, &ast.HaltStmt{StmtBase: base})
		case bytecode.OpDone:
			statements = append(statements, &ast.DoneStmt{StmtBase: base})
		case bytecode.OpEnd:
		default:
			return nil, &UnsupportedError{Operation: "serialization", Name: "CreateClosure bytecode " + instruction.Op.String()}
		}
	}
	return &ast.BlockStmt{
		StmtBase:   ast.StmtBase{Base: ast.Base{Range: function.Span}},
		Statements: statements,
	}, nil
}

func (translator *sleepLegacyTranslator) currentFrame() []ast.Expr {
	return translator.frames[len(translator.frames)-1]
}

func (translator *sleepLegacyTranslator) push(expression ast.Expr) {
	last := len(translator.frames) - 1
	translator.frames[last] = append(translator.frames[last], expression)
}

func (translator *sleepLegacyTranslator) popFrame() ([]ast.Expr, error) {
	if len(translator.frames) <= 1 {
		return nil, errors.New("sleep serialization: closure Step dissolved the implicit evaluator frame")
	}
	last := len(translator.frames) - 1
	frame := translator.frames[last]
	translator.frames[last] = nil
	translator.frames = translator.frames[:last]
	return frame, nil
}

func (translator *sleepLegacyTranslator) collapse(expression ast.Expr, span Span) error {
	if _, err := translator.popFrame(); err != nil {
		return err
	}
	return translator.pushCollapsed(expression, span)
}

func (translator *sleepLegacyTranslator) pushCollapsed(expression ast.Expr, span Span) error {
	translator.push(expression)
	if len(translator.frames) == 1 {
		translator.flushRootAt(span)
	}
	return nil
}

func (translator *sleepLegacyTranslator) flushRoot() {
	translator.flushRootAt(translator.functionSpan)
}

func (translator *sleepLegacyTranslator) flushRootAt(span Span) {
	if len(translator.frames) != 1 {
		return
	}
	for _, expression := range translator.frames[0] {
		translator.instructions = append(translator.instructions, bytecode.Instruction{Op: bytecode.OpEval, Span: span, Expr: expression})
	}
	translator.frames[0] = nil
}

type sleepLegacyStep struct {
	object *javaser.Object
	line   int
}

type sleepLegacyBlockBuilder struct {
	encoder *sleepSerializationEncoder
	source  string
	steps   []*sleepLegacyStep
}

type sleepLegacyEncodedBlock struct {
	block    *javaser.Object
	function *bytecode.Function
	pcStep   map[int]*javaser.Object
	foreach  []*sleepLegacyEncodedForeach
	tries    []*sleepLegacyEncodedTry
}

type sleepLegacyEncodedForeach struct {
	bodyGraph  *sleepLegacyEncodedBlock
	gotoStep   *javaser.Object
	iterInitPC int
	iterNextPC int
	bodyStart  int
	jumpPC     int
	destroyPC  int
}

type sleepLegacyEncodedTry struct {
	ownerGraph   *sleepLegacyEncodedBlock
	handlerGraph *sleepLegacyEncodedBlock
	tryStep      *javaser.Object
	enterPC      int
	bodyStart    int
	leavePC      int
	catchPC      int
	handlerStart int
	endPC        int
	varname      string
}

type sleepCompiledTry struct {
	bodyStart    int
	leavePC      int
	catchPC      int
	handlerStart int
	endPC        int
	varname      string
}

func sleepCompiledTryRegion(function *bytecode.Function, enterPC int) (*sleepCompiledTry, error) {
	if function == nil || enterPC < 0 || enterPC >= len(function.Instructions) || function.Instructions[enterPC].Op != bytecode.OpEnterTry {
		return nil, errors.New("sleep serialization: invalid try EnterTry program counter")
	}
	catchPC := function.Instructions[enterPC].Target
	if catchPC <= enterPC+1 || catchPC >= len(function.Instructions) || function.Instructions[catchPC].Op != bytecode.OpCatch {
		return nil, &UnsupportedError{Operation: "serialization", Name: "try Sleep closure with invalid catch target"}
	}
	jumpPC := catchPC - 1
	if function.Instructions[jumpPC].Op != bytecode.OpJump {
		return nil, &UnsupportedError{Operation: "serialization", Name: "try Sleep closure without handler jump"}
	}
	endPC := function.Instructions[jumpPC].Target
	if endPC < catchPC+1 || endPC > len(function.Instructions) {
		return nil, &UnsupportedError{Operation: "serialization", Name: "try Sleep closure with invalid end target"}
	}
	leavePC := jumpPC - 1
	if leavePC <= enterPC || function.Instructions[leavePC].Op != bytecode.OpLeaveTry {
		return nil, &UnsupportedError{Operation: "serialization", Name: "try Sleep closure without LeaveTry"}
	}
	return &sleepCompiledTry{
		bodyStart:    enterPC + 1,
		leavePC:      leavePC,
		catchPC:      catchPC,
		handlerStart: catchPC + 1,
		endPC:        endPC,
		varname:      function.Instructions[catchPC].Name,
	}, nil
}

type sleepCompiledForeach struct {
	key        string
	value      string
	iterNextPC int
	bodyStart  int
	jumpPC     int
	destroyPC  int
}

func sleepCompiledForeachRegion(function *bytecode.Function, iterInitPC int) (*sleepCompiledForeach, error) {
	if function == nil || iterInitPC < 0 || iterInitPC >= len(function.Instructions) || function.Instructions[iterInitPC].Op != bytecode.OpIterInit {
		return nil, errors.New("sleep serialization: invalid foreach IterInit program counter")
	}
	iterNextPC := iterInitPC + 1
	if iterNextPC >= len(function.Instructions) || function.Instructions[iterNextPC].Op != bytecode.OpIterNext {
		return nil, &UnsupportedError{Operation: "serialization", Name: "foreach Sleep closure without adjacent IterNext"}
	}
	next := function.Instructions[iterNextPC]
	if next.Name2 == "" {
		return nil, errors.New("sleep serialization: foreach IterNext has no value variable")
	}
	destroyPC := next.Target
	if destroyPC <= iterNextPC+1 || destroyPC >= len(function.Instructions) || function.Instructions[destroyPC].Op != bytecode.OpIterDestroy {
		return nil, &UnsupportedError{Operation: "serialization", Name: "foreach Sleep closure with invalid destroy target"}
	}
	jumpPC := destroyPC - 1
	jump := function.Instructions[jumpPC]
	if jump.Op != bytecode.OpJump || jump.Target != iterNextPC {
		return nil, &UnsupportedError{Operation: "serialization", Name: "foreach Sleep closure without terminal back-edge"}
	}
	bodyStart := iterNextPC + 1
	return &sleepCompiledForeach{
		key:        next.Name,
		value:      next.Name2,
		iterNextPC: iterNextPC,
		bodyStart:  bodyStart,
		jumpPC:     jumpPC,
		destroyPC:  destroyPC,
	}, nil
}

func (state *sleepSerializationEncoder) closure(closure *scriptClosure) (*javaser.Object, error) {
	if closure == nil {
		return nil, errors.New("sleep serialization: nil script closure")
	}
	if state.closures == nil {
		state.closures = make(map[*scriptClosure]*javaser.Object)
	}
	if existing := state.closures[closure]; existing != nil {
		return existing, nil
	}
	if closure.script != nil && closure.script.globals != nil && closure.script.globals.container != nil {
		return nil, &UnsupportedError{Operation: "serialization", Name: "Sleep closure with importer variable container"}
	}
	closure.mu.Lock()
	if closure.inline {
		closure.mu.Unlock()
		return nil, &UnsupportedError{Operation: "serialization", Name: "inline Sleep closure"}
	}
	if closure.function == nil {
		closure.mu.Unlock()
		return nil, errors.New("sleep serialization: closure has no function")
	}
	closure.ensureStateLocked()
	function := closure.function
	variables := closure.state
	id := closure.id
	suspended := append([]*fiber(nil), closure.suspended...)
	closure.mu.Unlock()

	// Publish the shell first because DefaultVariable always contains a $this
	// Scalar referring to this exact object handle.
	root := &javaser.Object{Descriptor: sleepClosureDescriptor}
	state.closures[closure] = root
	code, err := state.legacyBlockGraph(function)
	if err != nil {
		delete(state.closures, closure)
		return nil, err
	}
	context, err := state.closureContextStack(closure, code, suspended)
	if err != nil {
		delete(state.closures, closure)
		return nil, err
	}
	captured, err := state.closureVariables(closure, variables)
	if err != nil {
		delete(state.closures, closure)
		return nil, err
	}
	var idData [4]byte
	binary.BigEndian.PutUint32(idData[:], uint32(id))
	root.Data = []javaser.ClassData{{
		Descriptor: sleepClosureDescriptor,
		Annotation: []javaser.Content{
			&javaser.BlockData{Data: idData[:]},
			code.block,
			context,
			captured,
		},
	}}
	return root, nil
}

func sleepEmptyClosureStack() *javaser.Object {
	return sleepJavaStack(nil)
}

func sleepJavaStack(values []javaser.Value) *javaser.Object {
	capacity := 10
	for capacity < len(values) {
		capacity *= 2
	}
	elements := make([]javaser.Element, capacity)
	for index := range elements {
		elements[index] = javaser.NullValue
	}
	for index, value := range values {
		elements[index] = value
	}
	array := &javaser.Array{Descriptor: javaObjectArrayDescriptor, Values: elements}
	return &javaser.Object{
		Descriptor: javaStackDescriptor,
		Data: []javaser.ClassData{
			{
				Descriptor: javaVectorDescriptor,
				Fields: []javaser.FieldValue{
					{Field: javaVectorDescriptor.Fields[0], Value: javaser.Int(0)},
					{Field: javaVectorDescriptor.Fields[1], Value: javaser.Int(len(values))},
					{Field: javaVectorDescriptor.Fields[2], Value: array},
				},
			},
			{Descriptor: javaStackDescriptor},
		},
	}
}

func (state *sleepSerializationEncoder) closureContextStack(closure *scriptClosure, code *sleepLegacyEncodedBlock, suspended []*fiber) (*javaser.Object, error) {
	values := make([]javaser.Value, 0, len(suspended))
	for _, outer := range suspended {
		toplevel, err := state.closureToplevelStack(closure, code, outer)
		if err != nil {
			return nil, err
		}
		values = append(values, toplevel)
	}
	return sleepJavaStack(values), nil
}

func (state *sleepSerializationEncoder) closureToplevelStack(closure *scriptClosure, code *sleepLegacyEncodedBlock, head *fiber) (*javaser.Object, error) {
	if head == nil {
		return nil, errors.New("sleep serialization: nil suspended fiber")
	}
	roots := make([]*fiber, 0, 1+len(head.continuationTail))
	roots = append(roots, head)
	roots = append(roots, head.continuationTail...)
	seen := make(map[*fiber]struct{}, len(roots))
	contexts := make([]javaser.Value, 0, len(roots)+1)
	for _, root := range roots {
		if root == nil {
			return nil, errors.New("sleep serialization: nil continuation-tail fiber")
		}
		if _, duplicate := seen[root]; duplicate {
			return nil, errors.New("sleep serialization: cyclic continuation-tail fiber context")
		}
		seen[root] = struct{}{}
		if root != head && len(root.continuationTail) != 0 {
			return nil, errors.New("sleep serialization: nested continuation-tail fiber context")
		}
		chain, err := sleepSuspendedFiberChain(closure, root)
		if err != nil {
			return nil, err
		}
		if len(chain) == 0 {
			return nil, errors.New("sleep serialization: suspended context has no resumable fiber")
		}
		if len(chain[0].tries) != 0 {
			if len(chain) != 1 {
				return nil, &UnsupportedError{Operation: "serialization", Name: "saved exception-handler context nested with inline Context"}
			}
			tryContexts, err := state.closureTryContexts(code, chain[0])
			if err != nil {
				return nil, err
			}
			contexts = append(contexts, tryContexts...)
			continue
		}
		if chain[0].function != closure.function && chain[0].dynamicSource == nil && !chain[0].inline {
			return nil, &UnsupportedError{Operation: "serialization", Name: "suspended Context block outside SleepClosure.code without dynamic-source identity"}
		}
		for _, current := range chain {
			if current.scope != head.scope || !sameSleepLocalStack(current.locals, head.locals) {
				return nil, &UnsupportedError{Operation: "serialization", Name: "continuation Context with divergent local levels"}
			}
		}

		graph := code
		if chain[0].function != closure.function {
			graph, err = state.legacyBlockGraph(chain[0].function)
			if err != nil {
				return nil, err
			}
		}
		if current := chain[0]; len(current.iterators) != 0 || current.serializedForeach != nil {
			for index := len(chain) - 1; index >= 1; index-- {
				child := chain[index]
				childGraph := code
				if child.function != closure.function {
					childGraph, err = state.legacyBlockGraph(child.function)
					if err != nil {
						return nil, err
					}
				}
				resumePC := child.pc
				serializedInlineReturn := false
				if index+1 < len(chain) {
					if _, direct := directLegacyInlineCall(child.function, child.pc); direct {
						resumePC++
					} else if _, direct := directLegacyReturnInlineCall(child.function, child.pc); direct {
						serializedInlineReturn = true
					} else {
						return nil, &UnsupportedError{Operation: "serialization", Name: "inline Context with non-direct owning call"}
					}
				}
				var last javaser.Value = javaser.NullValue
				if child.serializedReturn || serializedInlineReturn {
					step, stepErr := sleepLegacyStepWithDescriptor(childGraph.block, sleepReturnDescriptor.Name)
					if stepErr != nil {
						return nil, stepErr
					}
					last = step
				} else if step := childGraph.pcStep[resumePC]; step != nil {
					last = step
				}
				contexts = append(contexts, sleepJavaContextWithHandler(childGraph.block, last, nil, child.serializedMoreHandlers))
			}
			foreachContexts, err := state.closureForeachContexts(graph, current)
			if err != nil {
				return nil, err
			}
			contexts = append(contexts, foreachContexts...)
			continue
		}
		for index := len(chain) - 1; index >= 0; index-- {
			current := chain[index]
			currentGraph := code
			if current.function != closure.function {
				currentGraph, err = state.legacyBlockGraph(current.function)
				if err != nil {
					return nil, err
				}
			}
			resumePC := current.pc
			serializedInlineReturn := false
			if index+1 < len(chain) {
				if _, direct := directLegacyInlineCall(current.function, current.pc); direct {
					resumePC++ // Java resumes after the inline Call; OPFOR re-enters that Call.
				} else if _, direct := directLegacyReturnInlineCall(current.function, current.pc); direct {
					serializedInlineReturn = true
				} else {
					return nil, &UnsupportedError{Operation: "serialization", Name: "inline Context with non-direct owning call"}
				}
			}
			if resumePC < 0 || resumePC >= len(current.function.Instructions) {
				return nil, errors.New("sleep serialization: suspended Context program counter is outside its Block")
			}
			var last javaser.Value = javaser.NullValue
			if current.serializedReturn || serializedInlineReturn {
				step, stepErr := sleepLegacyStepWithDescriptor(currentGraph.block, sleepReturnDescriptor.Name)
				if stepErr != nil {
					return nil, stepErr
				}
				last = step
			} else if step := currentGraph.pcStep[resumePC]; step != nil {
				last = step
			}
			contexts = append(contexts, sleepJavaContextWithHandler(currentGraph.block, last, nil, current.serializedMoreHandlers))
		}
	}
	levels, err := state.closureLocalLevelList(head)
	if err != nil {
		return nil, err
	}
	contexts = append(contexts, levels)
	return sleepJavaStack(contexts), nil
}

func (state *sleepSerializationEncoder) closureForeachContexts(code *sleepLegacyEncodedBlock, current *fiber) ([]javaser.Value, error) {
	if code == nil || current == nil || current.function == nil {
		return nil, errors.New("sleep serialization: invalid suspended foreach context")
	}
	var loop *sleepLegacyEncodedForeach
	includeBody := true
	if missing := current.serializedForeach; missing != nil {
		includeBody = missing.includeBody
		for _, candidate := range code.foreach {
			if candidate.iterNextPC == missing.iterNextPC {
				loop = candidate
				break
			}
		}
	} else {
		if len(current.inlineAt) == 0 && (current.pc <= 0 || current.pc >= len(current.function.Instructions) || current.function.Instructions[current.pc-1].Op != bytecode.OpYield) {
			return nil, &UnsupportedError{Operation: "serialization", Name: "foreach Sleep closure context not suspended by yield"}
		}
		for _, candidate := range code.foreach {
			if current.pc >= candidate.bodyStart && current.pc <= candidate.jumpPC {
				loop = candidate
				break
			}
		}
	}
	if loop == nil {
		return nil, &UnsupportedError{Operation: "serialization", Name: "foreach Sleep closure context does not match a compiled loop"}
	}
	if len(current.iterators) > 1 {
		bodyPC := current.pc - loop.bodyStart
		child := *current
		child.function = loop.bodyGraph.function
		child.pc = bodyPC
		child.iterators = append([]valueIterator(nil), current.iterators[1:]...)
		child.serializedForeach = nil
		contexts, err := state.closureForeachContexts(loop.bodyGraph, &child)
		if err != nil {
			return nil, err
		}
		return append(contexts, sleepJavaContext(code.block, loop.gotoStep)), nil
	}
	contexts := make([]javaser.Value, 0, 2)
	if includeBody {
		var last javaser.Value = javaser.NullValue
		if current.serializedForeach == nil && current.pc != loop.jumpPC {
			bodyPC := current.pc - loop.bodyStart
			if bodyPC < 0 || bodyPC >= len(loop.bodyGraph.function.Instructions) {
				return nil, errors.New("sleep serialization: foreach body resume program counter is outside its Block")
			}
			if step := loop.bodyGraph.pcStep[bodyPC]; step != nil {
				last = step
			}
		}
		contexts = append(contexts, sleepJavaContext(loop.bodyGraph.block, last))
	}
	contexts = append(contexts, sleepJavaContext(code.block, loop.gotoStep))
	return contexts, nil
}

func (state *sleepSerializationEncoder) closureTryContexts(code *sleepLegacyEncodedBlock, current *fiber) ([]javaser.Value, error) {
	if code == nil || current == nil || current.function == nil || len(current.tries) == 0 {
		return nil, errors.New("sleep serialization: invalid saved exception-handler context")
	}
	if len(current.tries) > 1 {
		return state.closureNestedTryContexts(code, current)
	}
	frame := current.tries[0]
	for _, region := range code.tries {
		if region.catchPC != frame.handler || current.pc < region.bodyStart || current.pc > region.leavePC {
			continue
		}
		bodyPC := current.pc - region.bodyStart
		if len(current.iterators) != 0 {
			child := *current
			child.function = region.ownerGraph.function
			child.pc = bodyPC
			child.tries = nil
			foreachContexts, err := state.closureForeachContexts(region.ownerGraph, &child)
			if err != nil {
				return nil, err
			}
			handler := sleepJavaExceptionContext(region.ownerGraph.block, region.handlerGraph.block, region.varname)
			for index, value := range foreachContexts {
				object, ok := value.(*javaser.Object)
				if !ok {
					return nil, errors.New("sleep serialization: foreach try Context is not an object")
				}
				if index+1 < len(foreachContexts) {
					setSleepObjectField(object, sleepContextDescriptor.Name, "moreHandlers", javaser.Boolean(true))
				} else {
					setSleepObjectField(object, sleepContextDescriptor.Name, "handler", handler)
				}
			}
			var outerLast javaser.Value = javaser.NullValue
			if step := code.pcStep[region.endPC]; step != nil {
				outerLast = step
			}
			return append(foreachContexts, sleepJavaContext(code.block, outerLast)), nil
		}
		var bodyLast javaser.Value = javaser.NullValue
		if step := region.ownerGraph.pcStep[bodyPC]; step != nil {
			bodyLast = step
		}
		handler := sleepJavaExceptionContext(region.ownerGraph.block, region.handlerGraph.block, region.varname)
		contexts := []javaser.Value{sleepJavaContextWithHandler(region.ownerGraph.block, bodyLast, handler, false)}
		var outerLast javaser.Value = javaser.NullValue
		if step := code.pcStep[region.endPC]; step != nil {
			outerLast = step
		}
		return append(contexts, sleepJavaContext(code.block, outerLast)), nil
	}

	// Java-produced saved handlers are reconstructed as an owner function
	// followed by an unreachable Catch entry. Split that durable representation
	// back into the two official Blocks without flattening the handler.
	if frame.handler <= 0 || frame.handler >= len(current.function.Instructions) || current.function.Instructions[frame.handler].Op != bytecode.OpCatch {
		return nil, &UnsupportedError{Operation: "serialization", Name: "saved exception-handler Sleep closure context with invalid catch target"}
	}
	ownerFunction := &bytecode.Function{
		Name:         current.function.Name + "<saved-try-owner>",
		Span:         current.function.Span,
		Instructions: append([]bytecode.Instruction(nil), current.function.Instructions[:frame.handler]...),
	}
	if len(ownerFunction.Instructions) == 0 || ownerFunction.Instructions[len(ownerFunction.Instructions)-1].Op != bytecode.OpEnd {
		ownerFunction.Instructions = append(ownerFunction.Instructions, bytecode.Instruction{Op: bytecode.OpEnd, Span: current.function.Span})
	}
	handlerFunction := &bytecode.Function{
		Name:         current.function.Name + "<saved-try-handler>",
		Span:         current.function.Span,
		Instructions: slicedLegacyInstructions(current.function.Instructions, frame.handler+1, len(current.function.Instructions)),
	}
	if len(handlerFunction.Instructions) == 0 || handlerFunction.Instructions[len(handlerFunction.Instructions)-1].Op != bytecode.OpEnd {
		handlerFunction.Instructions = append(handlerFunction.Instructions, bytecode.Instruction{Op: bytecode.OpEnd, Span: current.function.Span})
	}
	ownerGraph, err := state.legacyBlockGraph(ownerFunction)
	if err != nil {
		return nil, err
	}
	handlerGraph, err := state.legacyBlockGraph(handlerFunction)
	if err != nil {
		return nil, err
	}
	var last javaser.Value = javaser.NullValue
	if step := ownerGraph.pcStep[current.pc]; step != nil {
		last = step
	}
	variable := current.function.Instructions[frame.handler].Name
	handler := sleepJavaExceptionContext(ownerGraph.block, handlerGraph.block, variable)
	return []javaser.Value{sleepJavaContextWithHandler(ownerGraph.block, last, handler, current.serializedMoreHandlers)}, nil
}

func (state *sleepSerializationEncoder) closureNestedTryContexts(code *sleepLegacyEncodedBlock, current *fiber) ([]javaser.Value, error) {
	type activeTry struct {
		graph    *sleepLegacyEncodedBlock
		region   *sleepLegacyEncodedTry
		resumePC int
	}
	path := make([]activeTry, 0, len(current.tries))
	graph := code
	resumePC := current.pc
	baseOffset := 0
	for _, frame := range current.tries {
		localHandler := frame.handler - baseOffset
		var region *sleepLegacyEncodedTry
		for _, candidate := range graph.tries {
			if candidate.catchPC == localHandler && resumePC >= candidate.bodyStart && resumePC <= candidate.leavePC {
				region = candidate
				break
			}
		}
		if region == nil {
			return nil, &UnsupportedError{Operation: "serialization", Name: "nested saved exception-handler context does not match compiled try regions"}
		}
		path = append(path, activeTry{graph: graph, region: region, resumePC: resumePC})
		resumePC -= region.bodyStart
		baseOffset += region.bodyStart
		graph = region.ownerGraph
	}
	contexts := make([]javaser.Value, 0, len(path)+1)
	for index := len(path) - 1; index >= 0; index-- {
		active := path[index]
		bodyPC := active.resumePC - active.region.bodyStart
		if index+1 < len(path) {
			bodyPC = path[index+1].region.endPC
		}
		var last javaser.Value = javaser.NullValue
		if step := active.region.ownerGraph.pcStep[bodyPC]; step != nil {
			last = step
		}
		handler := sleepJavaExceptionContext(active.region.ownerGraph.block, active.region.handlerGraph.block, active.region.varname)
		contexts = append(contexts, sleepJavaContextWithHandler(active.region.ownerGraph.block, last, handler, index > 0))
	}
	outer := path[0]
	var last javaser.Value = javaser.NullValue
	if step := code.pcStep[outer.region.endPC]; step != nil {
		last = step
	}
	return append(contexts, sleepJavaContext(code.block, last)), nil
}

func sleepSuspendedFiberChain(closure *scriptClosure, outer *fiber) ([]*fiber, error) {
	if outer == nil {
		return nil, errors.New("sleep serialization: nil suspended fiber")
	}
	chain := make([]*fiber, 0, 2)
	seen := make(map[*fiber]struct{})
	hasForeach := false
	for current := outer; current != nil; {
		if _, duplicate := seen[current]; duplicate {
			return nil, errors.New("sleep serialization: cyclic inline fiber context")
		}
		seen[current] = struct{}{}
		if current.closure != closure || current.function == nil || current.scope == nil {
			return nil, errors.New("sleep serialization: suspended fiber has invalid closure ownership")
		}
		if current.serializedForeach != nil && len(current.iterators) != 0 {
			return nil, errors.New("sleep serialization: foreach context has both live and omitted iterator state")
		}
		if len(current.iterators) == 1 || current.serializedForeach != nil {
			hasForeach = true
		}
		if current.yieldedInBinary {
			return nil, &UnsupportedError{Operation: "serialization", Name: "inline binary-expression Sleep closure context"}
		}
		chain = append(chain, current)
		if len(current.inlineAt) == 0 {
			break
		}
		if len(current.inlineAt) != 1 {
			return nil, &UnsupportedError{Operation: "serialization", Name: "multiple inline Sleep closure contexts at one program counter"}
		}
		call, ok := directLegacyInlineCall(current.function, current.pc)
		if !ok {
			call, ok = directLegacyReturnInlineCall(current.function, current.pc)
		}
		if !ok {
			return nil, &UnsupportedError{Operation: "serialization", Name: "inline Context with non-direct owning call"}
		}
		child := current.inlineAt[call]
		if child == nil {
			return nil, errors.New("sleep serialization: inline Context call handle does not match its program counter")
		}
		current = child
	}
	_ = hasForeach
	return chain, nil
}

func sameSleepLocalStack(left, right []*scope) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sleepJavaContext(block *javaser.Object, last javaser.Value) *javaser.Object {
	return sleepJavaContextWithHandler(block, last, nil, false)
}

func sleepJavaContextWithHandler(block *javaser.Object, last javaser.Value, handler *javaser.Object, moreHandlers bool) *javaser.Object {
	handlerValue := javaser.Element(javaser.NullValue)
	if handler != nil {
		handlerValue = handler
	}
	return &javaser.Object{
		Descriptor: sleepContextDescriptor,
		Data: []javaser.ClassData{{
			Descriptor: sleepContextDescriptor,
			Fields: []javaser.FieldValue{
				{Field: sleepContextDescriptor.Fields[0], Value: javaser.Boolean(moreHandlers)},
				{Field: sleepContextDescriptor.Fields[1], Value: block},
				{Field: sleepContextDescriptor.Fields[2], Value: handlerValue},
				{Field: sleepContextDescriptor.Fields[3], Value: last},
			},
		}},
	}
}

func sleepJavaExceptionContext(owner, handler *javaser.Object, varname string) *javaser.Object {
	return &javaser.Object{
		Descriptor: sleepExceptionContextDescriptor,
		Data: []javaser.ClassData{{
			Descriptor: sleepExceptionContextDescriptor,
			Fields: []javaser.FieldValue{
				{Field: sleepExceptionContextDescriptor.Fields[0], Value: handler},
				{Field: sleepExceptionContextDescriptor.Fields[1], Value: owner},
				{Field: sleepExceptionContextDescriptor.Fields[2], Value: sleepJavaString(varname)},
			},
		}},
	}
}

func sleepLegacyStepWithDescriptor(block *javaser.Object, descriptorName string) (*javaser.Object, error) {
	if block == nil {
		return nil, errors.New("sleep serialization: nil Block while locating resume Step")
	}
	first, err := sleepObjectField(block, sleepBlockDescriptor.Name, "first")
	if err != nil {
		return nil, err
	}
	current, _, err := sleepStepReference(first)
	if err != nil {
		return nil, err
	}
	seen := make(map[*javaser.Object]struct{})
	for current != nil {
		if _, duplicate := seen[current]; duplicate {
			return nil, errors.New("sleep serialization: cyclic Step.next chain while locating resume Step")
		}
		seen[current] = struct{}{}
		if current.Descriptor != nil && current.Descriptor.Name == descriptorName {
			return current, nil
		}
		next, fieldErr := sleepObjectField(current, sleepStepDescriptor.Name, "next")
		if fieldErr != nil {
			return nil, fieldErr
		}
		current, _, err = sleepStepReference(next)
		if err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("sleep serialization: Block has no %s resume Step", descriptorName)
}

func (state *sleepSerializationEncoder) closureLocalLevelList(outer *fiber) (*javaser.Object, error) {
	if outer == nil || outer.scope == nil {
		return nil, errors.New("sleep serialization: suspended closure has no active local level")
	}
	levels := make([]*scope, 0, len(outer.locals)+1)
	levels = append(levels, outer.scope)
	for index := len(outer.locals) - 1; index >= 0; index-- {
		levels = append(levels, outer.locals[index])
	}
	annotation := make([]javaser.Content, 0, len(levels)+1)
	annotation = append(annotation, javaIntBlock(len(levels)))
	for _, level := range levels {
		variables, err := state.scopeVariables(level)
		if err != nil {
			return nil, err
		}
		annotation = append(annotation, variables)
	}
	return &javaser.Object{
		Descriptor: javaLinkedListDescriptor,
		Data: []javaser.ClassData{{
			Descriptor: javaLinkedListDescriptor,
			Annotation: annotation,
		}},
	}, nil
}

func (state *sleepSerializationEncoder) scopeVariables(level *scope) (*javaser.Object, error) {
	cells, err := level.defaultCellsSnapshot()
	if err != nil {
		return nil, err
	}
	return state.defaultVariable(cells)
}

func (state *sleepSerializationEncoder) closureVariables(closure *scriptClosure, variables *scope) (*javaser.Object, error) {
	cells, err := variables.defaultCellsSnapshot()
	if err != nil {
		return nil, err
	}
	if _, ok := cells["$this"]; !ok {
		cells["$this"] = NewCell(FunctionValue(closure))
	}
	return state.defaultVariable(cells)
}

func (state *sleepSerializationEncoder) defaultVariable(cells map[string]*Cell) (*javaser.Object, error) {
	names := make([]string, 0, len(cells))
	for name := range cells {
		names = append(names, name)
	}
	sort.Strings(names)
	capacity := sleepHashtableCapacity(len(names))
	annotation := []javaser.Content{javaTwoIntBlock(capacity, len(names))}
	for _, name := range names {
		scalar, err := state.scalarCell(cells[name])
		if err != nil {
			return nil, err
		}
		annotation = append(annotation, sleepJavaString(name), scalar)
	}
	table := &javaser.Object{
		Descriptor: javaHashtableDescriptor,
		Data: []javaser.ClassData{{
			Descriptor: javaHashtableDescriptor,
			Fields: []javaser.FieldValue{
				{Field: javaHashtableDescriptor.Fields[0], Value: javaser.Float(0.75)},
				{Field: javaHashtableDescriptor.Fields[1], Value: javaser.Int(int32(float64(capacity) * 0.75))},
			},
			Annotation: annotation,
		}},
	}
	return &javaser.Object{
		Descriptor: sleepDefaultVariableDescriptor,
		Data: []javaser.ClassData{{
			Descriptor: sleepDefaultVariableDescriptor,
			Fields: []javaser.FieldValue{{
				Field: sleepDefaultVariableDescriptor.Fields[0], Value: table,
			}},
		}},
	}, nil
}

func sleepHashtableCapacity(size int) int32 {
	capacity := int32(11)
	threshold := int(float64(capacity) * 0.75)
	for size > threshold && capacity < (1<<30) {
		capacity = capacity*2 + 1
		threshold = int(float64(capacity) * 0.75)
	}
	return capacity
}

func (state *sleepSerializationEncoder) scalarCell(cell *Cell) (*javaser.Object, error) {
	if cell == nil {
		return state.scalar(Null())
	}
	if state.scalarCells == nil {
		state.scalarCells = make(map[*Cell]*javaser.Object)
	}
	if existing := state.scalarCells[cell]; existing != nil {
		return existing, nil
	}
	root := &javaser.Object{Descriptor: sleepScalarDescriptor}
	state.scalarCells[cell] = root
	if err := state.fillScalar(root, cell.Get()); err != nil {
		delete(state.scalarCells, cell)
		return nil, err
	}
	return root, nil
}

func (state *sleepSerializationEncoder) fillScalar(root *javaser.Object, value Value) error {
	if root == nil {
		return errors.New("sleep serialization: nil Scalar destination")
	}
	data := javaser.ClassData{Descriptor: sleepScalarDescriptor}
	actual := javaser.Value(javaser.NullValue)
	array := javaser.Value(javaser.NullValue)
	hash := javaser.Value(javaser.NullValue)
	var err error
	switch value.Kind() {
	case KindNull:
	case KindInt, KindLong, KindDouble, KindString, KindObject, KindFunction:
		actual, err = state.scalarType(value)
	case KindArray:
		arrayValue, _ := value.Array()
		array, err = state.array(arrayValue)
	case KindHash:
		hashValue, _ := value.Hash()
		hash, err = state.hash(hashValue)
	default:
		err = fmt.Errorf("sleep serialization: unsupported scalar kind %s", value.Kind())
	}
	if err != nil {
		return err
	}
	data.Annotation = []javaser.Content{actual, array, hash}
	root.Descriptor = sleepScalarDescriptor
	root.Data = []javaser.ClassData{data}
	return nil
}

func (state *sleepSerializationEncoder) legacyBlock(function *bytecode.Function) (*javaser.Object, error) {
	graph, err := state.legacyBlockGraph(function)
	if err != nil {
		return nil, err
	}
	return graph.block, nil
}

func (state *sleepSerializationEncoder) legacyBlockGraph(function *bytecode.Function) (*sleepLegacyEncodedBlock, error) {
	if function == nil {
		return nil, errors.New("sleep serialization: nil closure function")
	}
	if state.legacyFunctions == nil {
		state.legacyFunctions = make(map[*bytecode.Function]*sleepLegacyEncodedBlock)
	}
	if existing := state.legacyFunctions[function]; existing != nil {
		return existing, nil
	}
	source := function.Span.Source
	if source == "" {
		source = "unknown"
	}
	builder := &sleepLegacyBlockBuilder{encoder: state, source: source}
	pcStep := make(map[int]*javaser.Object, len(function.Instructions))
	foreach := make([]*sleepLegacyEncodedForeach, 0, 1)
	tries := make([]*sleepLegacyEncodedTry, 0, 1)
	for pc := 0; pc < len(function.Instructions); pc++ {
		instructionPC := pc
		instruction := function.Instructions[pc]
		start := len(builder.steps)
		switch instruction.Op {
		case bytecode.OpEval:
			if err := builder.expression(instruction.Expr); err != nil {
				return nil, err
			}
		case bytecode.OpJumpFalse:
			if instruction.Target != pc+1 {
				return nil, &UnsupportedError{Operation: "serialization", Name: "closure bytecode jump-false"}
			}
			condition, err := builder.checkObject(instruction.Expr)
			if err != nil {
				return nil, err
			}
			falseBuilder := &sleepLegacyBlockBuilder{encoder: state, source: source}
			trueBuilder := &sleepLegacyBlockBuilder{encoder: state, source: source}
			placeholder := instruction.Span
			placeholder.Start.Line++
			placeholder.End.Line++
			trueBuilder.append(sleepStepDescriptor, placeholder)
			builder.appendWithFields(
				sleepDecideDescriptor,
				instruction.Span,
				falseBuilder.object(),
				trueBuilder.object(),
				condition,
			)
		case bytecode.OpIterInit:
			region, err := sleepCompiledForeachRegion(function, pc)
			if err != nil {
				return nil, err
			}
			builder.append(sleepCreateFrameDescriptor, instruction.Span)
			if err := builder.expression(instruction.Expr); err != nil {
				return nil, err
			}
			key := javaser.Element(javaser.NullValue)
			if region.key != "" {
				key = sleepJavaString(region.key)
			}
			builder.appendWithFields(
				sleepIterateDescriptor,
				instruction.Span,
				javaser.Int(sleepIteratorCreate),
				key,
				sleepJavaString(region.value),
			)

			bodyFunction := &bytecode.Function{
				Name:         function.Name + "<foreach>",
				Span:         function.Span,
				Instructions: slicedLegacyInstructions(function.Instructions, region.bodyStart, region.jumpPC),
			}
			bodyFunction.Instructions = append(bodyFunction.Instructions, bytecode.Instruction{Op: bytecode.OpEnd, Span: instruction.Span})
			bodyGraph, err := state.legacyBlockGraph(bodyFunction)
			if err != nil {
				var unsupported *UnsupportedError
				if errors.As(err, &unsupported) {
					return nil, &UnsupportedError{Operation: "serialization", Name: "foreach Sleep closure body: " + unsupported.Name}
				}
				return nil, err
			}
			setup := &sleepLegacyBlockBuilder{encoder: state, source: source}
			setup.appendWithFields(
				sleepIterateDescriptor,
				function.Instructions[region.iterNextPC].Span,
				javaser.Int(sleepIteratorNext),
				javaser.NullValue,
				javaser.NullValue,
			)
			check := builder.checkEval("-istrue", false, setup.object(), function.Instructions[region.iterNextPC].Span)
			gotoStep := builder.appendWithFields(
				sleepGotoDescriptor,
				function.Instructions[region.iterNextPC].Span,
				bodyGraph.block,
				javaser.NullValue,
				check,
			)
			destroyStep := builder.appendWithFields(
				sleepIterateDescriptor,
				function.Instructions[region.destroyPC].Span,
				javaser.Int(sleepIteratorDestroy),
				javaser.NullValue,
				javaser.NullValue,
			)
			pcStep[region.iterNextPC] = gotoStep
			pcStep[region.jumpPC] = gotoStep
			pcStep[region.destroyPC] = destroyStep
			foreach = append(foreach, &sleepLegacyEncodedForeach{
				bodyGraph:  bodyGraph,
				gotoStep:   gotoStep,
				iterInitPC: pc,
				iterNextPC: region.iterNextPC,
				bodyStart:  region.bodyStart,
				jumpPC:     region.jumpPC,
				destroyPC:  region.destroyPC,
			})
			pc = region.destroyPC
		case bytecode.OpEnterTry:
			region, err := sleepCompiledTryRegion(function, pc)
			if err != nil {
				return nil, err
			}
			ownerFunction := &bytecode.Function{
				Name:         function.Name + "<try>",
				Span:         function.Span,
				Instructions: slicedLegacyInstructions(function.Instructions, region.bodyStart, region.leavePC+1),
			}
			ownerFunction.Instructions = append(ownerFunction.Instructions, bytecode.Instruction{Op: bytecode.OpEnd, Span: instruction.Span})
			ownerGraph, err := state.legacyBlockGraph(ownerFunction)
			if err != nil {
				return nil, err
			}
			handlerFunction := &bytecode.Function{
				Name:         function.Name + "<catch>",
				Span:         function.Span,
				Instructions: []bytecode.Instruction{{Op: bytecode.OpLeaveTry, Span: function.Instructions[region.catchPC].Span}},
			}
			handlerFunction.Instructions = append(handlerFunction.Instructions, slicedLegacyInstructions(function.Instructions, region.handlerStart, region.endPC)...)
			handlerFunction.Instructions = append(handlerFunction.Instructions, bytecode.Instruction{Op: bytecode.OpEnd, Span: instruction.Span})
			handlerGraph, err := state.legacyBlockGraph(handlerFunction)
			if err != nil {
				return nil, err
			}
			tryStep := builder.appendWithFields(
				sleepTryDescriptor,
				instruction.Span,
				handlerGraph.block,
				ownerGraph.block,
				sleepJavaString(region.varname),
			)
			pcStep[pc] = tryStep
			tries = append(tries, &sleepLegacyEncodedTry{
				ownerGraph: ownerGraph, handlerGraph: handlerGraph, tryStep: tryStep,
				enterPC: pc, bodyStart: region.bodyStart, leavePC: region.leavePC,
				catchPC: region.catchPC, handlerStart: region.handlerStart, endPC: region.endPC,
				varname: region.varname,
			})
			pc = region.endPC - 1
		case bytecode.OpLeaveTry:
			builder.append(sleepPopTryDescriptor, instruction.Span)
		case bytecode.OpReturn:
			builder.append(sleepCreateFrameDescriptor, instruction.Span)
			if instruction.Expr == nil {
				builder.appendScalar(Null(), instruction.Span)
			} else if err := builder.expression(instruction.Expr); err != nil {
				return nil, err
			}
			builder.appendWithFields(sleepReturnDescriptor, instruction.Span,
				javaser.Int(sleepFlowReturn))
		case bytecode.OpYield:
			builder.append(sleepCreateFrameDescriptor, instruction.Span)
			if instruction.Expr == nil {
				builder.appendScalar(Null(), instruction.Span)
			} else if err := builder.expression(instruction.Expr); err != nil {
				return nil, err
			}
			builder.appendWithFields(sleepReturnDescriptor, instruction.Span,
				javaser.Int(sleepFlowYield))
		case bytecode.OpThrow:
			builder.append(sleepCreateFrameDescriptor, instruction.Span)
			if instruction.Expr == nil {
				builder.appendScalar(Null(), instruction.Span)
			} else if err := builder.expression(instruction.Expr); err != nil {
				return nil, err
			}
			builder.appendWithFields(sleepReturnDescriptor, instruction.Span,
				javaser.Int(sleepFlowThrow))
		case bytecode.OpCallCC:
			builder.append(sleepCreateFrameDescriptor, instruction.Span)
			if instruction.Expr == nil {
				builder.appendScalar(Null(), instruction.Span)
			} else if err := builder.expression(instruction.Expr); err != nil {
				return nil, err
			}
			builder.appendWithFields(sleepReturnDescriptor, instruction.Span,
				javaser.Int(sleepFlowCallCC))
		case bytecode.OpHalt:
			builder.append(sleepCreateFrameDescriptor, instruction.Span)
			builder.appendScalar(Int(2), instruction.Span)
			builder.appendWithFields(sleepReturnDescriptor, instruction.Span,
				javaser.Int(sleepFlowReturn))
		case bytecode.OpDone:
			builder.append(sleepCreateFrameDescriptor, instruction.Span)
			builder.appendScalar(Int(1), instruction.Span)
			builder.appendWithFields(sleepReturnDescriptor, instruction.Span,
				javaser.Int(sleepFlowReturn))
		case bytecode.OpEnd:
			// Block termination is represented by a null Step.next reference.
		default:
			return nil, &UnsupportedError{Operation: "serialization", Name: "closure bytecode " + instruction.Op.String()}
		}
		if start < len(builder.steps) {
			pcStep[instructionPC] = builder.steps[start].object
		}
	}
	graph := &sleepLegacyEncodedBlock{block: builder.object(), function: function, pcStep: pcStep, foreach: foreach, tries: tries}
	state.legacyFunctions[function] = graph
	return graph, nil
}

func (builder *sleepLegacyBlockBuilder) object() *javaser.Object {
	first := javaser.Value(javaser.NullValue)
	last := javaser.Value(javaser.NullValue)
	for index, step := range builder.steps {
		if index == 0 {
			first = step.object
		}
		if index+1 < len(builder.steps) {
			setSleepObjectField(step.object, sleepStepDescriptor.Name, "next", builder.steps[index+1].object)
		} else {
			last = step.object
		}
	}
	return &javaser.Object{
		Descriptor: sleepBlockDescriptor,
		Data: []javaser.ClassData{{
			Descriptor: sleepBlockDescriptor,
			Fields: []javaser.FieldValue{
				{Field: sleepBlockDescriptor.Fields[0], Value: first},
				{Field: sleepBlockDescriptor.Fields[1], Value: last},
				{Field: sleepBlockDescriptor.Fields[2], Value: sleepJavaString(builder.source)},
			},
		}},
	}
}

func setSleepObjectField(object *javaser.Object, className, fieldName string, value javaser.Element) {
	if object == nil {
		return
	}
	for dataIndex := range object.Data {
		data := &object.Data[dataIndex]
		if data.Descriptor == nil || data.Descriptor.Name != className {
			continue
		}
		for fieldIndex := range data.Fields {
			if data.Fields[fieldIndex].Field.Name == fieldName {
				data.Fields[fieldIndex].Value = value
				return
			}
		}
	}
}

func (builder *sleepLegacyBlockBuilder) append(descriptor *javaser.ClassDesc, span Span) *javaser.Object {
	return builder.appendWithFields(descriptor, span)
}

func (builder *sleepLegacyBlockBuilder) appendWithFields(descriptor *javaser.ClassDesc, span Span, values ...javaser.Element) *javaser.Object {
	line := sleepLegacySerializedLine(span)
	stepData := javaser.ClassData{
		Descriptor: sleepStepDescriptor,
		Fields: []javaser.FieldValue{
			{Field: sleepStepDescriptor.Fields[0], Value: javaser.Int(line)},
			{Field: sleepStepDescriptor.Fields[1], Value: javaser.NullValue},
		},
	}
	data := []javaser.ClassData{stepData}
	if descriptor != sleepStepDescriptor {
		fields := make([]javaser.FieldValue, len(descriptor.Fields))
		for index, field := range descriptor.Fields {
			value := javaser.Element(javaser.NullValue)
			if field.TypeCode != javaser.TypeObject && field.TypeCode != javaser.TypeArray {
				value = zeroJavaPrimitive(field.TypeCode)
			}
			if index < len(values) {
				value = values[index]
			}
			fields[index] = javaser.FieldValue{Field: field, Value: value}
		}
		data = append(data, javaser.ClassData{Descriptor: descriptor, Fields: fields})
	}
	object := &javaser.Object{Descriptor: descriptor, Data: data}
	builder.steps = append(builder.steps, &sleepLegacyStep{object: object, line: line})
	return object
}

func sleepLegacySerializedLine(span Span) int {
	line := sleepDisplayLine(span)
	// Sleep compiles eval/expr source under the literal name "eval" and keeps
	// its first Step at line zero. Zero is therefore a real legacy wire value,
	// not a missing public Span, for this one source identity.
	if line == 0 && span.Source == "eval" {
		return 0
	}
	if line <= 0 {
		line = span.Start.Line
	}
	if line <= 0 {
		line = 1
	}
	return line
}

func zeroJavaPrimitive(code byte) javaser.Element {
	switch code {
	case javaser.TypeByte:
		return javaser.Byte(0)
	case javaser.TypeChar:
		return javaser.Char(0)
	case javaser.TypeDouble:
		return javaser.Double(0)
	case javaser.TypeFloat:
		return javaser.Float(0)
	case javaser.TypeInt:
		return javaser.Int(0)
	case javaser.TypeLong:
		return javaser.Long(0)
	case javaser.TypeShort:
		return javaser.Short(0)
	case javaser.TypeBoolean:
		return javaser.Boolean(false)
	default:
		return javaser.Int(0)
	}
}

func (builder *sleepLegacyBlockBuilder) appendScalar(value Value, span Span) error {
	scalar, err := builder.encoder.scalar(value)
	if err != nil {
		return err
	}
	builder.appendWithFields(sleepSValueDescriptor, span, scalar)
	return nil
}

func (builder *sleepLegacyBlockBuilder) expression(expression ast.Expr) error {
	if expression == nil {
		return builder.appendScalar(Null(), Span{})
	}
	span := expression.Span()
	switch node := expression.(type) {
	case *ast.GroupExpr:
		return builder.expression(node.Expr)
	case *ast.VariableExpr:
		if node.Raw == "$null" {
			return builder.appendScalar(Null(), span)
		}
		builder.appendWithFields(sleepGetDescriptor, span, sleepJavaString(node.Raw))
		return nil
	case *ast.FunctionRefExpr:
		name := node.Raw
		if name == "" {
			name = "&" + node.Name
		}
		builder.appendWithFields(sleepGetDescriptor, span, sleepJavaString(name))
		return nil
	case *ast.NullExpr:
		return builder.appendScalar(Null(), span)
	case *ast.BoolExpr:
		return builder.appendScalar(Bool(node.Value), span)
	case *ast.NumberExpr:
		value, err := numberLiteral(node)
		if err != nil {
			return err
		}
		return builder.appendScalar(value, span)
	case *ast.StringExpr:
		if node.Kind == ast.SingleQuotedString {
			return builder.appendScalar(String(decodeSleepSingleQuoted(node.Text)), span)
		}
		if node.Kind == ast.BacktickString {
			return &UnsupportedError{Operation: "serialization", Name: "backtick closure expression"}
		}
		return builder.parsedLiteral(node)
	case *ast.TupleExpr:
		return builder.arrayExpression(node.Elements, span)
	case *ast.ArrayLiteralExpr:
		return builder.arrayExpression(node.Elements, span)
	case *ast.BinaryExpr:
		builder.append(sleepCreateFrameDescriptor, span)
		if err := builder.expression(node.Right); err != nil {
			return err
		}
		if err := builder.expression(node.Left); err != nil {
			return err
		}
		builder.appendWithFields(sleepOperateDescriptor, span, sleepJavaString(node.Op))
		return nil
	case *ast.ParameterTermExpr:
		for _, idea := range node.Ideas {
			if err := builder.expression(idea); err != nil {
				return err
			}
		}
		return nil
	case *ast.ParameterOperatorExpr:
		builder.append(sleepCreateFrameDescriptor, span)
		for _, idea := range node.Right {
			if err := builder.expression(idea); err != nil {
				return err
			}
		}
		if err := builder.expression(node.Left); err != nil {
			return err
		}
		builder.appendWithFields(sleepOperateDescriptor, span, sleepJavaString(node.Op))
		return nil
	case *ast.AdjacentEmptyGroupExpr:
		return builder.expression(node.Value)
	case *ast.AssignExpr:
		builder.append(sleepCreateFrameDescriptor, span)
		if err := builder.expression(node.Value); err != nil {
			return err
		}
		variable, err := builder.assignmentTargetBlock(node.Target)
		if err != nil {
			return err
		}
		var operator javaser.Element = javaser.NullValue
		if node.Op != "=" {
			base := strings.TrimSuffix(node.Op, "=")
			if base == "" || base == node.Op {
				return &UnsupportedError{Operation: "serialization", Name: "assignment operator " + node.Op}
			}
			operatorBuilder := &sleepLegacyBlockBuilder{encoder: builder.encoder, source: builder.source}
			operator = operatorBuilder.appendWithFields(sleepOperateDescriptor, span, sleepJavaString(base))
		}
		builder.appendWithFields(sleepAssignDescriptor, span, operator, variable)
		return nil
	case *ast.CallExpr:
		name, err := legacyCallName(node.Callee)
		if err != nil {
			return err
		}
		if strings.TrimPrefix(name, "&") == "iff" || strings.TrimPrefix(name, "&") == "?" {
			return builder.iffExpression(node)
		}
		builder.append(sleepCreateFrameDescriptor, span)
		for _, index := range callArgumentEvaluationOrder(len(node.Args), node.ArgGroups) {
			if err := builder.expression(node.Args[index]); err != nil {
				return err
			}
		}
		builder.appendWithFields(sleepCallDescriptor, span, sleepJavaString(name))
		return nil
	case *ast.ObjectExpr:
		className := ""
		static := node.Message != nil
		switch target := node.Target.(type) {
		case *ast.IdentifierExpr:
			className = resolvePortableClassName(target.Name)
		case *ast.ClassExpr:
			className = resolvePortableClassName(target.Name)
		default:
			static = false
		}
		builder.append(sleepCreateFrameDescriptor, span)
		for index := len(node.Args) - 1; index >= 0; index-- {
			if err := builder.expression(node.Args[index]); err != nil {
				return err
			}
		}
		if !static {
			if node.Target == nil {
				return errors.New("sleep serialization: instance ObjectAccess has no target")
			}
			if err := builder.expression(node.Target); err != nil {
				return err
			}
			var name javaser.Element = javaser.NullValue
			if node.Message != nil {
				name = sleepJavaString(node.Message.Name)
			}
			builder.appendWithFields(sleepObjectAccessDescriptor, span, javaser.NullValue, name)
			return nil
		}
		class, err := sleepPortableStaticClassValue(className)
		if err != nil {
			return err
		}
		builder.appendWithFields(sleepObjectAccessDescriptor, span, class, sleepJavaString(node.Message.Name))
		return nil
	case *ast.ClosureExpr:
		compiled := compiler.CompileBlock("<closure>", node.Body)
		if len(compiled.Diagnostics) != 0 {
			return &CompileError{Diagnostics: compiled.Diagnostics}
		}
		block, err := builder.encoder.legacyBlock(compiled.Function)
		if err != nil {
			return err
		}
		builder.appendWithFields(sleepCreateClosureDescriptor, span, block)
		return nil
	default:
		return &UnsupportedError{Operation: "serialization", Name: fmt.Sprintf("closure expression %T", expression)}
	}
}

func (builder *sleepLegacyBlockBuilder) arrayExpression(elements []ast.Expr, span Span) error {
	builder.append(sleepCreateFrameDescriptor, span)
	for index := len(elements) - 1; index >= 0; index-- {
		if err := builder.expression(elements[index]); err != nil {
			return err
		}
	}
	builder.appendWithFields(sleepCallDescriptor, span, sleepJavaString("&@"))
	return nil
}

func (builder *sleepLegacyBlockBuilder) iffExpression(call *ast.CallExpr) error {
	if len(call.Args) < 1 || len(call.Args) > 3 {
		return &UnsupportedError{Operation: "serialization", Name: fmt.Sprintf("iff closure expression with %d arguments", len(call.Args))}
	}
	span := call.Span()
	condition, err := builder.checkObject(call.Args[0])
	if err != nil {
		return err
	}
	trueExpression := ast.Expr(&ast.BoolExpr{ExprBase: ast.ExprBase{Base: ast.Base{Range: span}}, Value: true})
	if len(call.Args) >= 2 {
		trueExpression = call.Args[1]
	}
	falseExpression := ast.Expr(&ast.BoolExpr{ExprBase: ast.ExprBase{Base: ast.Base{Range: span}}, Value: false})
	if len(call.Args) == 3 {
		falseExpression = call.Args[2]
	}
	trueBlock, err := builder.resultBlock(trueExpression)
	if err != nil {
		return err
	}
	falseBlock, err := builder.resultBlock(falseExpression)
	if err != nil {
		return err
	}
	builder.appendWithFields(sleepDecideDescriptor, span, falseBlock, trueBlock, condition)
	return nil
}

func (builder *sleepLegacyBlockBuilder) resultBlock(expression ast.Expr) (*javaser.Object, error) {
	nested := &sleepLegacyBlockBuilder{encoder: builder.encoder, source: builder.source}
	if err := nested.expression(expression); err != nil {
		return nil, err
	}
	return nested.object(), nil
}

func (builder *sleepLegacyBlockBuilder) checkObject(expression ast.Expr) (*javaser.Object, error) {
	for {
		group, ok := expression.(*ast.GroupExpr)
		if !ok {
			break
		}
		expression = group.Expr
	}
	span := expression.Span()
	if binary, ok := expression.(*ast.BinaryExpr); ok {
		switch strings.ToLower(binary.Op) {
		case "&&", "||":
			left, err := builder.checkObject(binary.Left)
			if err != nil {
				return nil, err
			}
			right, err := builder.checkObject(binary.Right)
			if err != nil {
				return nil, err
			}
			descriptor := sleepCheckAndDescriptor
			if binary.Op == "||" {
				descriptor = sleepCheckOrDescriptor
			}
			return sleepObjectWithFields(descriptor, left, right), nil
		case "==", "!=", "<", "<=", ">", ">=", "eq", "ne", "lt", "le", "gt", "ge", "is", "isnot", "=~", "!~", "=~~", "!~~", "ismatch", "!ismatch", "hasmatch", "!hasmatch":
			setup := &sleepLegacyBlockBuilder{encoder: builder.encoder, source: builder.source}
			if err := setup.expression(binary.Left); err != nil {
				return nil, err
			}
			if err := setup.expression(binary.Right); err != nil {
				return nil, err
			}
			return builder.checkEval(binary.Op, false, setup.object(), span), nil
		}
	}
	setup := &sleepLegacyBlockBuilder{encoder: builder.encoder, source: builder.source}
	if err := setup.expression(expression); err != nil {
		return nil, err
	}
	return builder.checkEval("-istrue", false, setup.object(), span), nil
}

func (builder *sleepLegacyBlockBuilder) checkEval(name string, negate bool, setup *javaser.Object, span Span) *javaser.Object {
	line := sleepLegacySerializedLine(span)
	if strings.HasPrefix(name, "!") && len(name) > 2 {
		name = strings.TrimPrefix(name, "!")
		negate = !negate
	}
	return sleepObjectWithFields(
		sleepCheckEvalDescriptor,
		javaser.Int(line),
		javaser.Boolean(negate),
		javaser.NullValue,
		javaser.NullValue,
		sleepJavaString(name),
		setup,
	)
}

func sleepObjectWithFields(descriptor *javaser.ClassDesc, values ...javaser.Element) *javaser.Object {
	fields := make([]javaser.FieldValue, len(descriptor.Fields))
	for index, field := range descriptor.Fields {
		value := javaser.Element(javaser.NullValue)
		if field.TypeCode != javaser.TypeObject && field.TypeCode != javaser.TypeArray {
			value = zeroJavaPrimitive(field.TypeCode)
		}
		if index < len(values) {
			value = values[index]
		}
		fields[index] = javaser.FieldValue{Field: field, Value: value}
	}
	return &javaser.Object{
		Descriptor: descriptor,
		Data:       []javaser.ClassData{{Descriptor: descriptor, Fields: fields}},
	}
}

func sleepPortableStaticClassValue(name string) (*javaser.Class, error) {
	switch resolvePortableClassName(name) {
	case sleepJavaSystemClassDescriptor.Name:
		return sleepJavaSystemClass, nil
	case sleepSleepUtilsClassDescriptor.Name:
		return sleepSleepUtilsClass, nil
	default:
		return nil, &UnsupportedError{Operation: "serialization", Name: "static ObjectAccess class " + name}
	}
}

func (builder *sleepLegacyBlockBuilder) assignmentTargetBlock(expression ast.Expr) (*javaser.Object, error) {
	for {
		group, ok := expression.(*ast.GroupExpr)
		if !ok {
			break
		}
		expression = group.Expr
	}
	variable, ok := expression.(*ast.VariableExpr)
	if !ok {
		return nil, &UnsupportedError{Operation: "serialization", Name: fmt.Sprintf("assignment target %T", expression)}
	}
	target := &sleepLegacyBlockBuilder{encoder: builder.encoder, source: builder.source}
	target.appendWithFields(sleepGetDescriptor, variable.Span(), sleepJavaString(variable.Raw))
	return target.object(), nil
}

func legacyCallName(expression ast.Expr) (string, error) {
	switch node := expression.(type) {
	case *ast.IdentifierExpr:
		return "&" + strings.TrimPrefix(node.Name, "&"), nil
	case *ast.FunctionRefExpr:
		name := node.Raw
		if name == "" {
			name = "&" + node.Name
		}
		return name, nil
	default:
		return "", &UnsupportedError{Operation: "serialization", Name: fmt.Sprintf("closure call target %T", expression)}
	}
}

func (builder *sleepLegacyBlockBuilder) parsedLiteral(node *ast.StringExpr) error {
	decoded, err := decodeSleepEscapesAt(node.Text, node.TextRange)
	if err != nil {
		return err
	}
	input := decoded.text
	builder.append(sleepCreateFrameDescriptor, node.Span())
	fragments := make([]javaser.Value, 0, 4)
	var literal strings.Builder
	appendString := func() {
		fragments = append(fragments, sleepPLiteralFragment(1, sleepJavaString(literal.String())))
		literal.Reset()
	}
	for index := 0; index < len(input); {
		if strings.HasPrefix(input[index:], " $+ ") {
			index += 4
			continue
		}
		if strings.HasPrefix(input[index:], escapedDollarSentinel) {
			literal.WriteByte('$')
			index += len(escapedDollarSentinel)
			continue
		}
		if input[index] != '$' || index+1 >= len(input) || sleepInterpolationVariableEnd(input, index+1) {
			literal.WriteByte(input[index])
			index++
			continue
		}
		if input[index+1] == '+' {
			return errors.New("opfor: operator $+ must be surrounded with whitespace")
		}
		appendString()
		index++
		if index < len(input) && input[index] == '[' {
			return &UnsupportedError{Operation: "serialization", Name: "aligned parsed-literal closure expression"}
		}
		start := index
		for index < len(input) && !sleepInterpolationVariableEnd(input, index) {
			index++
		}
		if start == index {
			return errors.New("opfor: can not align an empty variable")
		}
		name := "$" + input[start:index]
		builder.appendWithFields(sleepGetDescriptor, node.Span(), sleepJavaString(name))
		fragments = append(fragments, sleepPLiteralFragment(3, javaser.NullValue))
	}
	if literal.Len() != 0 || len(fragments) == 0 {
		appendString()
	}
	listAnnotation := []javaser.Content{javaIntBlock(len(fragments))}
	for _, fragment := range fragments {
		listAnnotation = append(listAnnotation, fragment)
	}
	list := &javaser.Object{
		Descriptor: javaLinkedListDescriptor,
		Data: []javaser.ClassData{{
			Descriptor: javaLinkedListDescriptor,
			Annotation: listAnnotation,
		}},
	}
	builder.appendWithFields(sleepPLiteralDescriptor, node.Span(), list)
	return nil
}

func sleepPLiteralFragment(fragmentType int32, element javaser.Value) *javaser.Object {
	return &javaser.Object{
		Descriptor: sleepPLiteralFragmentDescriptor,
		Data: []javaser.ClassData{{
			Descriptor: sleepPLiteralFragmentDescriptor,
			Fields: []javaser.FieldValue{
				{Field: sleepPLiteralFragmentDescriptor.Fields[0], Value: javaser.Int(fragmentType)},
				{Field: sleepPLiteralFragmentDescriptor.Fields[1], Value: element},
			},
		}},
	}
}
