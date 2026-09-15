package opfor

import (
	"errors"
	"strings"

	"github.com/sliverarmory/opfor/internal/ast"
)

const (
	diagnosticImportedClassNotFound = "CMP003"
	diagnosticImportArchiveNotFound = "CMP004"

	messageImportedClassNotFound = "imported class was not found"
	messageImportArchiveNotFound = "jar file to import package from was not found!"
)

// sleep21ClassCatalog is the set of top-level classes present in the pinned
// Sleep 2.1 source tree (Cobalt-Strike/sleep commit 60ac3ff9). OPFOR reserves
// sleep.* for that runtime surface, so an explicit missing class there is
// deterministic. Java and application packages remain importer-delegated.
var sleep21ClassCatalog = func() map[string]struct{} {
	const names = `
sleep.bridges.BasicIO
sleep.bridges.BasicNumbers
sleep.bridges.BasicStrings
sleep.bridges.BasicUtilities
sleep.bridges.BridgeUtilities
sleep.bridges.DefaultEnvironment
sleep.bridges.DefaultVariable
sleep.bridges.FileSystemBridge
sleep.bridges.KeyValuePair
sleep.bridges.RegexBridge
sleep.bridges.Semaphore
sleep.bridges.SleepClosure
sleep.bridges.TimeDateBridge
sleep.bridges.Transliteration
sleep.bridges.io.BufferObject
sleep.bridges.io.DataPattern
sleep.bridges.io.FileObject
sleep.bridges.io.IOObject
sleep.bridges.io.ProcessObject
sleep.bridges.io.SocketObject
sleep.console.ConsoleImplementation
sleep.console.ConsoleProxy
sleep.console.TextConsole
sleep.engine.Block
sleep.engine.CallRequest
sleep.engine.Check
sleep.engine.GeneratedSteps
sleep.engine.ObjectUtilities
sleep.engine.ProxyInterface
sleep.engine.Step
sleep.engine.atoms.Assign
sleep.engine.atoms.AssignT
sleep.engine.atoms.Bind
sleep.engine.atoms.BindFilter
sleep.engine.atoms.BindPredicate
sleep.engine.atoms.Call
sleep.engine.atoms.CheckAnd
sleep.engine.atoms.CheckEval
sleep.engine.atoms.CheckOr
sleep.engine.atoms.CreateClosure
sleep.engine.atoms.CreateFrame
sleep.engine.atoms.Decide
sleep.engine.atoms.Get
sleep.engine.atoms.Goto
sleep.engine.atoms.Index
sleep.engine.atoms.Iterate
sleep.engine.atoms.ObjectAccess
sleep.engine.atoms.ObjectNew
sleep.engine.atoms.Operate
sleep.engine.atoms.PLiteral
sleep.engine.atoms.PopTry
sleep.engine.atoms.Return
sleep.engine.atoms.SValue
sleep.engine.atoms.Try
sleep.engine.types.DoubleValue
sleep.engine.types.HashContainer
sleep.engine.types.IntValue
sleep.engine.types.ListContainer
sleep.engine.types.LongValue
sleep.engine.types.MyLinkedList
sleep.engine.types.NullValue
sleep.engine.types.ObjectValue
sleep.engine.types.OrderedHashContainer
sleep.engine.types.StringValue
sleep.error.RuntimeWarningWatcher
sleep.error.ScriptWarning
sleep.error.SyntaxError
sleep.error.YourCodeSucksException
sleep.interfaces.Environment
sleep.interfaces.FilterEnvironment
sleep.interfaces.Function
sleep.interfaces.Loadable
sleep.interfaces.Operator
sleep.interfaces.Predicate
sleep.interfaces.PredicateEnvironment
sleep.interfaces.Variable
sleep.parser.Checkers
sleep.parser.CodeGenerator
sleep.parser.CommentRule
sleep.parser.ImportManager
sleep.parser.LexicalAnalyzer
sleep.parser.Parser
sleep.parser.ParserConfig
sleep.parser.ParserConstants
sleep.parser.ParserUtilities
sleep.parser.Rule
sleep.parser.Statement
sleep.parser.StringIterator
sleep.parser.Token
sleep.parser.TokenList
sleep.parser.TokenParser
sleep.runtime.CollectionWrapper
sleep.runtime.MapWrapper
sleep.runtime.ProxyIterator
sleep.runtime.Scalar
sleep.runtime.ScalarArray
sleep.runtime.ScalarHash
sleep.runtime.ScalarType
sleep.runtime.ScriptEnvironment
sleep.runtime.ScriptInstance
sleep.runtime.ScriptLoader
sleep.runtime.ScriptVariables
sleep.runtime.SleepUtils
sleep.runtime.WatchScalar
sleep.taint.PermeableStep
sleep.taint.Sanitizer
sleep.taint.Sensitive
sleep.taint.TaintArray
sleep.taint.TaintCall
sleep.taint.TaintHash
sleep.taint.TaintModeGeneratedSteps
sleep.taint.TaintObjectAccess
sleep.taint.TaintOperate
sleep.taint.TaintUtils
sleep.taint.TaintedValue
sleep.taint.Tainter
`
	catalog := make(map[string]struct{}, 116)
	for _, name := range strings.Fields(names) {
		catalog[name] = struct{}{}
	}
	return catalog
}()

func validateStaticImports(source Source, script *ast.Script) []Diagnostic {
	if script == nil {
		return nil
	}
	var diagnostics []Diagnostic
	walkImportStatements(script.Statements, func(statement *ast.ImportStmt) {
		target := strings.TrimSpace(statement.Target)
		if statement.From != nil || strings.HasSuffix(target, ".*") || !strings.HasPrefix(target, "sleep.") {
			return
		}
		if sleep21ClassExists(target) {
			return
		}
		diagnostics = append(diagnostics, importDiagnostic(
			diagnosticImportedClassNotFound,
			messageImportedClassNotFound,
			trimImportTerminator(source.Data, statement.Span()),
		))
	})
	return diagnostics
}

func sleep21ClassExists(name string) bool {
	if _, ok := sleep21ClassCatalog[name]; ok {
		return true
	}
	if nested := strings.IndexByte(name, '$'); nested > 0 {
		_, ok := sleep21ClassCatalog[name[:nested]]
		return ok
	}
	return false
}

func importDiagnostic(code, message string, span Span) Diagnostic {
	return Diagnostic{Severity: SeverityError, Code: code, Message: message, Span: span}
}

func importCompileError(code, message string, span Span) error {
	return &CompileError{Diagnostics: []Diagnostic{importDiagnostic(code, message, span)}}
}

func isImportCompileError(err error) bool {
	var compileError *CompileError
	if !errors.As(err, &compileError) {
		return false
	}
	for _, diagnostic := range compileError.Diagnostics {
		if diagnostic.Code == diagnosticImportedClassNotFound || diagnostic.Code == diagnosticImportArchiveNotFound {
			return true
		}
	}
	return false
}

func importInstructionDiagnosticSpan(script *Script, span Span) Span {
	if script == nil || script.program == nil || script.program.source.Name != span.Source {
		return span
	}
	return trimImportTerminator(script.program.source.Data, span)
}

func (r *Runtime) hasImportObjectDelegate() bool {
	if r == nil || r.objectHost == nil {
		return false
	}
	host, wrapped := r.objectHost.(defaultObjectHost)
	if !wrapped {
		return true
	}
	if host.primary == nil {
		return false
	}
	_, unsupported := host.primary.(unsupportedObjectHost)
	return !unsupported
}

func trimImportTerminator(source []byte, span Span) Span {
	if span.End.Offset <= span.Start.Offset || span.End.Offset > len(source) || source[span.End.Offset-1] != ';' {
		return span
	}
	span.End.Offset--
	if span.End.Line == span.Start.Line && span.End.Column > span.Start.Column {
		span.End.Column--
	} else if span.End.Column > 1 {
		span.End.Column--
	}
	return span
}

func walkImportStatements(statements []ast.Stmt, visit func(*ast.ImportStmt)) {
	for _, statement := range statements {
		walkImportStatement(statement, visit)
	}
}

func walkImportStatement(statement ast.Stmt, visit func(*ast.ImportStmt)) {
	switch node := statement.(type) {
	case nil:
		return
	case *ast.ImportStmt:
		visit(node)
		walkImportExpression(node.From, visit)
	case *ast.BlockStmt:
		walkImportStatements(node.Statements, visit)
	case *ast.ExprStmt:
		walkImportExpression(node.Expr, visit)
	case *ast.IfStmt:
		walkImportExpression(node.Condition, visit)
		walkImportStatement(node.Then, visit)
		walkImportStatement(node.Else, visit)
	case *ast.WhileStmt:
		walkImportExpression(node.Condition, visit)
		walkImportStatement(node.Body, visit)
	case *ast.ForStmt:
		walkImportExpressions(node.Init, visit)
		walkImportExpression(node.Condition, visit)
		walkImportExpressions(node.Post, visit)
		walkImportStatement(node.Body, visit)
	case *ast.ForeachStmt:
		walkImportExpression(node.Key, visit)
		walkImportExpression(node.Value, visit)
		walkImportExpression(node.Iterable, visit)
		walkImportStatement(node.Body, visit)
	case *ast.TryStmt:
		walkImportStatement(node.Body, visit)
		walkImportStatement(node.Catch, visit)
	case *ast.ReturnStmt:
		walkImportExpression(node.Value, visit)
	case *ast.YieldStmt:
		walkImportExpression(node.Value, visit)
	case *ast.ThrowStmt:
		walkImportExpression(node.Value, visit)
	case *ast.CallCCStmt:
		walkImportExpression(node.Closure, visit)
	case *ast.AssertStmt:
		walkImportExpression(node.Condition, visit)
		walkImportExpression(node.Message, visit)
	case *ast.EnvironmentStmt:
		walkImportStatement(node.Body, visit)
	}
}

func walkImportExpressions(expressions []ast.Expr, visit func(*ast.ImportStmt)) {
	for _, expression := range expressions {
		walkImportExpression(expression, visit)
	}
}

func walkImportExpression(expression ast.Expr, visit func(*ast.ImportStmt)) {
	switch node := expression.(type) {
	case nil:
		return
	case *ast.ClosureExpr:
		walkImportStatement(node.Body, visit)
	case *ast.ReferenceExpr:
		walkImportExpression(node.Target, visit)
	case *ast.GroupExpr:
		walkImportExpression(node.Expr, visit)
	case *ast.TupleExpr:
		walkImportExpressions(node.Elements, visit)
	case *ast.ArrayLiteralExpr:
		walkImportExpressions(node.Elements, visit)
	case *ast.HashLiteralExpr:
		walkImportExpressions(node.Entries, visit)
	case *ast.PairExpr:
		walkImportExpression(node.Key, visit)
		walkImportExpression(node.Value, visit)
	case *ast.UnaryExpr:
		walkImportExpression(node.Operand, visit)
	case *ast.BinaryExpr:
		walkImportExpression(node.Left, visit)
		walkImportExpression(node.Right, visit)
	case *ast.AssignExpr:
		walkImportExpression(node.Target, visit)
		walkImportExpression(node.Value, visit)
	case *ast.SequenceExpr:
		walkImportExpressions(node.Elements, visit)
	case *ast.IndexExpr:
		walkImportExpression(node.Target, visit)
		walkImportExpression(node.Index, visit)
	case *ast.CallExpr:
		walkImportExpression(node.Callee, visit)
		walkImportExpressions(node.Args, visit)
	case *ast.ObjectExpr:
		walkImportExpression(node.Target, visit)
		walkImportExpressions(node.Args, visit)
	}
}
