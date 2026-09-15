// Package compiler lowers parsed Sleep/Aggressor statements into resumable
// OPFOR bytecode.
package compiler

import (
	"fmt"
	"strings"

	"github.com/sliverarmory/opfor/internal/ast"
	"github.com/sliverarmory/opfor/internal/bytecode"
	"github.com/sliverarmory/opfor/internal/lexer"
)

const (
	diagnosticInvalidFlow       = "CMP001"
	diagnosticUnknownExpression = "CMP002"
)

// Result contains a best-effort function and compiler diagnostics.
type Result struct {
	Function    *bytecode.Function
	Diagnostics []lexer.Diagnostic
}

// Compile lowers a parsed script.
func Compile(script *ast.Script) Result {
	if script == nil {
		return Result{}
	}
	compiler := &compiler{}
	function := compiler.compileFunction("<main>", script.Span(), script.Statements)
	return Result{Function: function, Diagnostics: compiler.diagnostics}
}

// CompileBlock lowers a closure body. It is used when a closure literal is
// instantiated at runtime.
func CompileBlock(name string, block *ast.BlockStmt) Result {
	if block == nil {
		return Result{}
	}
	compiler := &compiler{}
	function := compiler.compileFunction(name, block.Span(), block.Statements)
	return Result{Function: function, Diagnostics: compiler.diagnostics}
}

type compiler struct {
	diagnostics []lexer.Diagnostic
}

type loopContext struct {
	breaks         []int
	continues      []int
	continueTarget int
	tryDepth       int
	parent         *loopContext
}

type functionCompiler struct {
	owner         *compiler
	function      *bytecode.Function
	currentLoop   *loopContext
	functionName  string
	tryDepth      int
	iteratorDepth int
}

func (c *compiler) compileFunction(name string, span lexer.Span, statements []ast.Stmt) *bytecode.Function {
	fc := &functionCompiler{
		owner:        c,
		function:     &bytecode.Function{Name: name, Span: span},
		functionName: name,
	}
	start := fc.pc()
	fc.compileStatements(statements)
	end := fc.pc()
	fc.addBlockRecovery(start, end, end, 0, 0)
	fc.emit(bytecode.Instruction{Op: bytecode.OpEnd, Span: span})
	return fc.function
}

func (c *functionCompiler) compileStatements(statements []ast.Stmt) {
	for _, statement := range statements {
		c.compileStatement(statement)
	}
}

func (c *functionCompiler) compileStatement(statement ast.Stmt) {
	if statement == nil {
		return
	}
	switch node := statement.(type) {
	case *ast.ExprStmt:
		c.emit(bytecode.Instruction{Op: bytecode.OpEval, Span: node.Span(), Expr: node.Expr})
	case *ast.BlockStmt:
		c.compileStatements(node.Statements)
	case *ast.IfStmt:
		c.compileIf(node)
	case *ast.WhileStmt:
		c.compileWhile(node)
	case *ast.ForStmt:
		c.compileFor(node)
	case *ast.ForeachStmt:
		c.compileForeach(node)
	case *ast.TryStmt:
		c.compileTry(node)
	case *ast.ReturnStmt:
		c.emit(bytecode.Instruction{Op: bytecode.OpReturn, Span: node.Span(), Expr: node.Value})
	case *ast.YieldStmt:
		c.emit(bytecode.Instruction{Op: bytecode.OpYield, Span: node.Span(), Expr: node.Value})
	case *ast.ThrowStmt:
		c.emit(bytecode.Instruction{Op: bytecode.OpThrow, Span: node.Span(), Expr: node.Value})
	case *ast.AssertStmt:
		c.emit(bytecode.Instruction{Op: bytecode.OpAssert, Span: node.Span(), Expr: node.Condition, Message: node.Message})
	case *ast.ImportStmt:
		c.emit(bytecode.Instruction{Op: bytecode.OpImport, Span: node.Span(), ImportTarget: node.Target, ImportFrom: node.From})
	case *ast.EnvironmentStmt:
		name := node.Keyword
		if len(node.Selectors) != 0 {
			name += " " + node.Selectors[0].Value
		}
		body := c.owner.compileFunction(name, node.Body.Span(), node.Body.Statements)
		c.emit(bytecode.Instruction{
			Op:          bytecode.OpBind,
			Span:        node.Span(),
			Keyword:     node.Keyword,
			Environment: node.Form,
			Selectors:   append([]ast.Selector(nil), node.Selectors...),
			Predicate:   node.Predicate,
			Body:        body,
		})
	case *ast.BreakStmt:
		c.compileLoopControl(node.Span(), node.Value, false)
	case *ast.ContinueStmt:
		c.compileLoopControl(node.Span(), node.Value, true)
	case *ast.HaltStmt:
		c.emit(bytecode.Instruction{Op: bytecode.OpHalt, Span: node.Span()})
	case *ast.DoneStmt:
		c.emit(bytecode.Instruction{Op: bytecode.OpDone, Span: node.Span()})
	case *ast.CallCCStmt:
		c.emit(bytecode.Instruction{Op: bytecode.OpCallCC, Span: node.Span(), Expr: node.Closure})
	default:
		c.error(statement.Span(), "unsupported statement node %T", statement)
	}
}

func (c *functionCompiler) compileIf(statement *ast.IfStmt) {
	jumpFalse := c.emit(bytecode.Instruction{Op: bytecode.OpJumpFalse, Span: statement.Condition.Span(), Expr: statement.Condition, Target: -1})
	thenStart := c.pc()
	c.compileStatements(statement.Then.Statements)
	thenEnd := c.pc()
	if statement.Else == nil {
		end := c.pc()
		c.patch(jumpFalse, end)
		c.addCurrentBlockRecovery(thenStart, thenEnd, end)
		return
	}
	jumpEnd := c.emit(bytecode.Instruction{Op: bytecode.OpJump, Span: statement.Span(), Target: -1})
	c.patch(jumpFalse, c.pc())
	elseStart := c.pc()
	c.compileStatement(statement.Else)
	elseEnd := c.pc()
	end := c.pc()
	c.patch(jumpEnd, end)
	c.addCurrentBlockRecovery(thenStart, thenEnd, end)
	// Sleep represents both an explicit else body and an else-if chain as the
	// false-choice Block owned by the outer Decide Step.
	c.addCurrentBlockRecovery(elseStart, elseEnd, end)
}

func (c *functionCompiler) compileWhile(statement *ast.WhileStmt) {
	condition := c.pc()
	var exit int
	if statement.Binding != nil {
		exit = c.emit(bytecode.Instruction{
			Op:     bytecode.OpAssignWhile,
			Span:   statement.Condition.Span(),
			Expr:   statement.Condition,
			Name:   statement.Binding.Raw,
			Target: -1,
		})
	} else {
		exit = c.emit(bytecode.Instruction{Op: bytecode.OpJumpFalse, Span: statement.Condition.Span(), Expr: statement.Condition, Target: -1})
	}
	loop := c.pushLoop(condition)
	bodyStart := c.pc()
	c.compileStatements(statement.Body.Statements)
	bodyEnd := c.pc()
	c.emit(bytecode.Instruction{Op: bytecode.OpJump, Span: statement.Span(), Target: condition})
	end := c.pc()
	c.patch(exit, end)
	c.popLoop(loop, end, condition)
	c.addLoopRecovery(condition, end, bodyStart, bodyEnd, end, condition, loop.tryDepth, c.iteratorDepth)
	c.addCurrentBlockRecovery(bodyStart, bodyEnd, condition)
}

func (c *functionCompiler) compileFor(statement *ast.ForStmt) {
	for _, expression := range statement.Init {
		c.emit(bytecode.Instruction{Op: bytecode.OpEval, Span: expression.Span(), Expr: expression})
	}
	condition := c.pc()
	exit := -1
	if statement.Condition != nil {
		exit = c.emit(bytecode.Instruction{Op: bytecode.OpJumpFalse, Span: statement.Condition.Span(), Expr: statement.Condition, Target: -1})
	}
	loop := c.pushLoop(-1)
	bodyStart := c.pc()
	c.compileStatements(statement.Body.Statements)
	post := c.pc()
	loop.continueTarget = post
	for _, expression := range statement.Post {
		c.emit(bytecode.Instruction{Op: bytecode.OpEval, Span: expression.Span(), Expr: expression})
	}
	bodyEnd := c.pc()
	c.emit(bytecode.Instruction{Op: bytecode.OpJump, Span: statement.Span(), Target: condition})
	end := c.pc()
	if exit >= 0 {
		c.patch(exit, end)
	}
	c.popLoop(loop, end, post)
	c.addLoopRecovery(condition, end, bodyStart, bodyEnd, end, post, loop.tryDepth, c.iteratorDepth)
	// CodeGenerator appends the for-post expressions to the same Block as the
	// source body. A warning anywhere in that Block skips the post expressions.
	c.addCurrentBlockRecovery(bodyStart, bodyEnd, condition)
}

func (c *functionCompiler) compileForeach(statement *ast.ForeachStmt) {
	key := ""
	if statement.Key != nil {
		key = variableName(statement.Key)
	}
	value := variableName(statement.Value)
	c.emit(bytecode.Instruction{
		Op:    bytecode.OpIterInit,
		Span:  statement.Iterable.Span(),
		Expr:  statement.Iterable,
		Name:  key,
		Name2: value,
	})
	next := c.pc()
	advance := c.emit(bytecode.Instruction{
		Op:     bytecode.OpIterNext,
		Span:   statement.Span(),
		Name:   key,
		Name2:  value,
		Target: -1,
	})
	loop := c.pushLoop(next)
	bodyStart := c.pc()
	c.iteratorDepth++
	c.compileStatements(statement.Body.Statements)
	c.iteratorDepth--
	bodyEnd := c.pc()
	c.emit(bytecode.Instruction{Op: bytecode.OpJump, Span: statement.Span(), Target: next})
	destroy := c.emit(bytecode.Instruction{Op: bytecode.OpIterDestroy, Span: statement.Span()})
	c.patch(advance, destroy)
	c.popLoop(loop, destroy, next)
	c.addLoopRecovery(next, destroy, bodyStart, bodyEnd, destroy, next, loop.tryDepth, c.iteratorDepth+1)
	c.addBlockRecovery(bodyStart, bodyEnd, next, c.tryDepth, c.iteratorDepth+1)
}

func (c *functionCompiler) compileTry(statement *ast.TryStmt) {
	outerTryDepth := c.tryDepth
	outerIteratorDepth := c.iteratorDepth
	enter := c.emit(bytecode.Instruction{Op: bytecode.OpEnterTry, Span: statement.Span(), Target: -1})
	c.tryDepth++
	bodyStart := c.pc()
	c.compileStatements(statement.Body.Statements)
	bodyEnd := c.pc()
	c.tryDepth--
	c.emit(bytecode.Instruction{Op: bytecode.OpLeaveTry, Span: statement.Span()})
	jumpEnd := c.emit(bytecode.Instruction{Op: bytecode.OpJump, Span: statement.Span(), Target: -1})
	catch := c.pc()
	c.patch(enter, catch)
	catchName := ""
	if statement.CatchVar != nil {
		catchName = statement.CatchVar.Raw
	}
	c.emit(bytecode.Instruction{Op: bytecode.OpCatch, Span: statement.Span(), Name: catchName})
	catchStart := c.pc()
	if statement.Catch != nil {
		c.compileStatements(statement.Catch.Statements)
	}
	catchEnd := c.pc()
	end := c.pc()
	c.patch(jumpEnd, end)
	c.addBlockRecovery(bodyStart, bodyEnd, end, outerTryDepth, outerIteratorDepth)
	c.addBlockRecovery(catchStart, catchEnd, end, outerTryDepth, outerIteratorDepth)
}

func variableName(expression ast.Expr) string {
	variable, ok := expression.(*ast.VariableExpr)
	if !ok {
		return ""
	}
	return variable.Raw
}

func (c *functionCompiler) pushLoop(continueTarget int) *loopContext {
	loop := &loopContext{parent: c.currentLoop, continueTarget: continueTarget, tryDepth: c.tryDepth}
	c.currentLoop = loop
	return loop
}

func (c *functionCompiler) compileLoopControl(span lexer.Span, value ast.Expr, continuing bool) {
	if c.currentLoop == nil {
		op := bytecode.OpBreak
		if continuing {
			op = bytecode.OpContinue
		}
		c.emit(bytecode.Instruction{Op: op, Span: span, Expr: value})
		return
	}

	// Sleep evaluates the optional operand before its Return atom requests
	// control flow. Lexically owned jumps can remain static, but must preserve
	// those side effects and any recoverable warning boundary before unwinding.
	if value != nil {
		c.emit(bytecode.Instruction{Op: bytecode.OpEval, Span: value.Span(), Expr: value})
	}
	c.emitTryUnwind(span, c.currentLoop.tryDepth)
	jump := c.emit(bytecode.Instruction{Op: bytecode.OpJump, Span: span, Target: -1, ClearResult: true})
	if continuing {
		c.currentLoop.continues = append(c.currentLoop.continues, jump)
	} else {
		c.currentLoop.breaks = append(c.currentLoop.breaks, jump)
	}
}

func (c *functionCompiler) emitTryUnwind(span lexer.Span, targetDepth int) {
	for depth := c.tryDepth; depth > targetDepth; depth-- {
		c.emit(bytecode.Instruction{Op: bytecode.OpLeaveTry, Span: span})
	}
}

func (c *functionCompiler) popLoop(loop *loopContext, breakTarget, continueTarget int) {
	if loop.continueTarget < 0 {
		loop.continueTarget = continueTarget
	}
	for _, instruction := range loop.breaks {
		c.patch(instruction, breakTarget)
	}
	for _, instruction := range loop.continues {
		c.patch(instruction, loop.continueTarget)
	}
	c.currentLoop = loop.parent
}

func (c *functionCompiler) emit(instruction bytecode.Instruction) int {
	expressionRole := expressionValueRole
	switch instruction.Op {
	case bytecode.OpJumpFalse, bytecode.OpAssignWhile, bytecode.OpAssert:
		expressionRole = expressionPredicateRole
	}
	c.validateExpression(instruction.Expr, expressionRole)
	c.validateExpression(instruction.Message, expressionValueRole)
	c.validateExpression(instruction.ImportFrom, expressionValueRole)
	c.validateExpression(instruction.Predicate, expressionPredicateRole)
	index := len(c.function.Instructions)
	c.function.Instructions = append(c.function.Instructions, instruction)
	return index
}

func (c *functionCompiler) addCurrentBlockRecovery(start, end, target int) {
	c.addBlockRecovery(start, end, target, c.tryDepth, c.iteratorDepth)
}

func (c *functionCompiler) addBlockRecovery(start, end, target, tryDepth, iteratorDepth int) {
	if c == nil || c.function == nil || start >= end {
		return
	}
	c.function.BlockRecoveries = append(c.function.BlockRecoveries, bytecode.BlockRecovery{
		Start:         start,
		End:           end,
		Target:        target,
		TryDepth:      tryDepth,
		IteratorDepth: iteratorDepth,
	})
}

func (c *functionCompiler) addLoopRecovery(start, end, bodyStart, bodyEnd, breakTarget, continueTarget, tryDepth, iteratorDepth int) {
	if c == nil || c.function == nil || start >= end {
		return
	}
	c.function.LoopRecoveries = append(c.function.LoopRecoveries, bytecode.LoopRecovery{
		Start:          start,
		End:            end,
		BodyStart:      bodyStart,
		BodyEnd:        bodyEnd,
		BreakTarget:    breakTarget,
		ContinueTarget: continueTarget,
		TryDepth:       tryDepth,
		IteratorDepth:  iteratorDepth,
	})
}

type expressionValidationRole uint8

const (
	expressionValueRole expressionValidationRole = iota
	expressionPredicateRole
	expressionIdentifierRole
	expressionPairKeyRole
)

func (c *functionCompiler) validateExpression(expression ast.Expr, role expressionValidationRole) {
	if expression == nil {
		return
	}
	switch node := expression.(type) {
	case *ast.IdentifierExpr:
		if role != expressionIdentifierRole && role != expressionPairKeyRole {
			c.unknownExpression(node.Span())
		}
	case *ast.GroupExpr:
		groupRole := expressionValueRole
		if role == expressionPredicateRole {
			groupRole = expressionPredicateRole
		}
		c.validateExpression(node.Expr, groupRole)
	case *ast.TupleExpr:
		for _, item := range node.Elements {
			c.validateExpression(item, expressionValueRole)
		}
	case *ast.ArrayLiteralExpr:
		for _, item := range node.Elements {
			c.validateExpression(item, expressionValueRole)
		}
	case *ast.HashLiteralExpr:
		for _, item := range node.Entries {
			c.validateExpression(item, expressionValueRole)
		}
	case *ast.PairExpr:
		if node.RawKey == "" {
			c.validateExpression(node.Key, expressionPairKeyRole)
		}
		c.validateExpression(node.Value, expressionValueRole)
	case *ast.UnaryExpr:
		c.validateUnaryExpression(node, role)
	case *ast.BinaryExpr:
		operandRole := expressionValueRole
		if role == expressionPredicateRole && (node.Op == "&&" || node.Op == "||") {
			operandRole = expressionPredicateRole
		}
		c.validateExpression(node.Left, operandRole)
		c.validateExpression(node.Right, operandRole)
	case *ast.AssignExpr:
		c.validateExpression(node.Target, expressionValueRole)
		c.validateExpression(node.Value, expressionValueRole)
	case *ast.SequenceExpr:
		for _, item := range node.Elements {
			c.validateExpression(item, expressionValueRole)
		}
	case *ast.ParameterTermExpr:
		for _, idea := range node.Ideas {
			c.validateExpression(idea, expressionValueRole)
		}
	case *ast.ParameterOperatorExpr:
		c.validateExpression(node.Left, expressionValueRole)
		for _, idea := range node.Right {
			c.validateExpression(idea, expressionValueRole)
		}
	case *ast.AdjacentEmptyGroupExpr:
		c.validateExpression(node.Value, expressionValueRole)
	case *ast.IndexExpr:
		c.validateExpression(node.Target, expressionValueRole)
		c.validateExpression(node.Index, expressionValueRole)
	case *ast.CallExpr:
		c.validateExpression(node.Callee, expressionIdentifierRole)
		for index, argument := range node.Args {
			argumentRole := expressionValueRole
			if index == 0 && predicateCallIdentifier(node.Callee) {
				argumentRole = expressionPredicateRole
			}
			c.validateExpression(argument, argumentRole)
		}
	case *ast.ObjectExpr:
		c.validateExpression(node.Target, expressionIdentifierRole)
		for _, argument := range node.Args {
			c.validateExpression(argument, expressionValueRole)
		}
	case *ast.ReferenceExpr:
		c.validateExpression(node.Target, expressionValueRole)
	case *ast.ClosureExpr:
		// Compile anonymous closures with their owning Program. Runtime
		// evaluation creates fresh closure identity/state from this immutable
		// function template instead of reparsing the body on every iteration.
		if c.function.ClosureTemplates == nil {
			c.function.ClosureTemplates = make(map[*ast.ClosureExpr]*bytecode.Function)
		}
		if _, exists := c.function.ClosureTemplates[node]; !exists {
			c.function.ClosureTemplates[node] = c.owner.compileFunction("<closure>", node.Body.Span(), node.Body.Statements)
		}
	}
}

func callIdentifierName(expression ast.Expr) string {
	identifier, ok := expression.(*ast.IdentifierExpr)
	if !ok {
		return ""
	}
	return strings.ToLower(identifier.Name)
}

func predicateCallIdentifier(expression ast.Expr) bool {
	switch callIdentifierName(expression) {
	case "iff", "?":
		return true
	default:
		return false
	}
}

func (c *functionCompiler) validateUnaryExpression(node *ast.UnaryExpr, role expressionValidationRole) {
	if node == nil {
		return
	}
	if node.Postfix {
		c.validateExpression(node.Operand, expressionValueRole)
		return
	}
	if unaryCallArguments(node) {
		if role == expressionPairKeyRole {
			c.unknownExpression(node.Span())
			return
		}
		switch operand := node.Operand.(type) {
		case *ast.GroupExpr:
			c.validateExpression(operand.Expr, expressionValueRole)
		case *ast.TupleExpr:
			for _, argument := range operand.Elements {
				c.validateExpression(argument, expressionValueRole)
			}
		}
		return
	}
	if node.Op == "~" || strings.EqualFold(node.Op, "not") {
		c.unknownExpression(node.Span())
		return
	}
	if unaryPredicateOperator(node.Op) {
		if role != expressionPredicateRole {
			c.unknownExpression(node.Span())
			return
		}
		c.validateExpression(node.Operand, expressionValueRole)
		return
	}
	c.validateExpression(node.Operand, expressionValueRole)
}

func unaryPredicateOperator(operator string) bool {
	return operator == "!" || (operator != "-" && strings.HasPrefix(operator, "-")) ||
		strings.HasPrefix(operator, "!-")
}

func unaryCallArguments(node *ast.UnaryExpr) bool {
	if node == nil || !strings.EqualFold(node.Op, "not") || node.Operand == nil {
		return false
	}
	if node.Operand.Span().Start.Offset != node.Span().Start.Offset+len(node.Op) {
		return false
	}
	switch node.Operand.(type) {
	case *ast.GroupExpr, *ast.TupleExpr:
		return true
	default:
		return false
	}
}

func (c *functionCompiler) unknownExpression(span lexer.Span) {
	c.owner.diagnostics = append(c.owner.diagnostics, lexer.Diagnostic{
		Severity: lexer.SeverityError,
		Code:     diagnosticUnknownExpression,
		Message:  "Unknown expression",
		Span:     span,
	})
}

func (c *functionCompiler) patch(index, target int) {
	c.function.Instructions[index].Target = target
}

func (c *functionCompiler) pc() int { return len(c.function.Instructions) }

func (c *functionCompiler) error(span lexer.Span, format string, args ...any) {
	c.owner.diagnostics = append(c.owner.diagnostics, lexer.Diagnostic{
		Severity: lexer.SeverityError,
		Code:     diagnosticInvalidFlow,
		Message:  fmt.Sprintf(format, args...),
		Span:     span,
	})
}
