package opfor

import (
	"context"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sliverarmory/opfor/internal/ast"
	"github.com/sliverarmory/opfor/internal/compiler"
	"github.com/sliverarmory/opfor/internal/lexer"
)

func (f *fiber) eval(ctx context.Context, expression ast.Expr) (Value, error) {
	if expression == nil {
		return Null(), nil
	}
	if simpleEvaluatorExpression(expression) {
		return f.evalValue(ctx, expression)
	}
	if err := scriptExecutionError(ctx); err != nil {
		return Null(), err
	}
	result, err := f.evalValue(ctx, expression)
	if err != nil {
		return result, err
	}
	if err := scriptExecutionError(ctx); err != nil {
		return Null(), err
	}
	return result, nil
}

// simpleEvaluatorExpression has no host-side effect of its own. Variable
// provider reads retain their own before/after execution checks in scope;
// built-in scope reads and immutable literals rely on the enclosing VM or
// compound expression safe point. Avoiding two general context walks for each
// arithmetic operand materially reduces interpreter-loop overhead without
// weakening callback, mutation, or object-call cancellation boundaries.
func simpleEvaluatorExpression(expression ast.Expr) bool {
	switch expression.(type) {
	case *ast.IdentifierExpr, *ast.VariableExpr, *ast.NumberExpr, *ast.BoolExpr, *ast.NullExpr, *ast.ImportPathExpr:
		return true
	default:
		return false
	}
}

func (f *fiber) evalValue(ctx context.Context, expression ast.Expr) (Value, error) {
	switch node := expression.(type) {
	case *ast.IdentifierExpr:
		return String(node.Name), nil
	case *ast.VariableExpr:
		if node.Raw == "$null" {
			return Null(), nil
		}
		return f.readVariable(ctx, node.Raw, node.Span())
	case *ast.NumberExpr:
		if f != nil && f.closure != nil && f.closure.script != nil && f.closure.script.program != nil {
			if literal, ok := f.closure.script.program.numberLiterals[node]; ok {
				return literal.value, literal.err
			}
		}
		return numberLiteral(node)
	case *ast.BoolExpr:
		return Bool(node.Value), nil
	case *ast.NullExpr:
		return Null(), nil
	case *ast.StringExpr:
		return f.stringLiteral(ctx, node)
	case *ast.ImportPathExpr:
		// Sleep's unquoted import-from form is a literal path token. It is kept
		// as an expression in the AST so quoted and unquoted paths share the
		// same import instruction without granting the path interpolation or
		// object semantics of an ordinary identifier.
		return String(node.Raw), nil
	case *ast.FunctionRefExpr:
		result := FunctionValue(f.callable(node.Name, node.Span()))
		if f.callTraceEnabled() {
			f.writeCallTrace("function("+String("&"+strings.TrimPrefix(node.Name, "&")).Describe()+")", result, nil, node.Span())
		}
		return result, nil
	case *ast.ReferenceExpr:
		cell, err := f.referenceCell(ctx, node.Target, true)
		if err != nil {
			return Null(), err
		}
		name, ok := referenceArgumentName(node.Target)
		if !ok {
			return Null(), fmt.Errorf("opfor: referenced expression has no variable name")
		}
		return ObjectValue(sleepKeyValue{key: String(name), value: cell.Get()}), nil
	case *ast.ClassExpr:
		return ObjectValue(classReference(resolvePortableClassName(f.closure.script.resolveClass(node.Name)))), nil
	case *ast.ClosureExpr:
		function := f.function.ClosureTemplates[node]
		if function == nil {
			// Serialized or importer-synthesized functions may not belong to a
			// Program compiled by the current compiler. Preserve that compatibility
			// boundary while ordinary source uses the immutable template fast path.
			compiled := compiler.CompileBlock("<closure>", node.Body)
			if len(compiled.Diagnostics) != 0 {
				return Null(), &CompileError{Diagnostics: compiled.Diagnostics}
			}
			function = compiled.Function
		}
		// SleepClosure constructs a fresh internal variable container for every
		// closure. It does not retain the caller's active local or closure level;
		// unresolved names fall through from the new internal level directly to
		// the script globals. Explicit lambda()/let() bindings and this() remain
		// the language mechanisms for installing closure-owned cells.
		closure := f.closure.script.newClosure(function, f.scope.root)
		return FunctionValue(closure), nil
	case *ast.GroupExpr:
		return f.eval(ctx, node.Expr)
	case *ast.TupleExpr:
		values := make([]Value, len(node.Elements))
		for index := len(node.Elements) - 1; index >= 0; index-- {
			element := node.Elements[index]
			value, err := f.eval(ctx, element)
			if err != nil {
				return Null(), err
			}
			values[index] = value
		}
		array, err := newRuntimeArray(f.closure.script.runtime, values...)
		if err != nil {
			return Null(), err
		}
		return ArrayValue(array), nil
	case *ast.ArrayLiteralExpr:
		values := make([]Value, len(node.Elements))
		for index := len(node.Elements) - 1; index >= 0; index-- {
			element := node.Elements[index]
			value, err := f.eval(ctx, element)
			if err != nil {
				return Null(), err
			}
			values[index] = value
		}
		array, err := newRuntimeArray(f.closure.script.runtime, values...)
		if err != nil {
			return Null(), err
		}
		return ArrayValue(array), nil
	case *ast.HashLiteralExpr:
		type hashEntry struct {
			key   string
			value Value
		}
		entries := make([]hashEntry, len(node.Entries))
		for index := len(node.Entries) - 1; index >= 0; index-- {
			entry := node.Entries[index]
			var key string
			var value Value
			var err error
			if pair, ok := entry.(*ast.PairExpr); ok {
				value, err = f.eval(ctx, pair.Value)
				if err == nil {
					key, err = f.pairExpressionKey(ctx, pair)
				}
			} else {
				var evaluated Value
				evaluated, err = f.eval(ctx, entry)
				if err == nil {
					key, value, err = hashArgument(Argument{Value: evaluated})
				}
			}
			if err != nil {
				return Null(), err
			}
			entries[index] = hashEntry{key: key, value: value}
		}
		unique := make(map[string]struct{}, len(entries))
		for _, entry := range entries {
			unique[sleepCanonicalString(String(entry.key))] = struct{}{}
		}
		if err := reserveCollectionEntries(f.closure.script.runtime, len(unique)); err != nil {
			return Null(), err
		}
		hash := NewHash()
		for _, entry := range entries {
			hash.Set(entry.key, entry.value)
		}
		return HashValue(hash), nil
	case *ast.PairExpr:
		value, err := f.eval(ctx, node.Value)
		if err != nil {
			return Null(), err
		}
		key, err := f.pairExpressionKey(ctx, node)
		if err != nil {
			return Null(), err
		}
		return ObjectValue(namedValue{name: key, value: value}), nil
	case *ast.UnaryExpr:
		return f.evalUnary(ctx, node)
	case *ast.BinaryExpr:
		return f.evalBinary(ctx, node)
	case *ast.AssignExpr:
		return f.evalAssignment(ctx, node)
	case *ast.SequenceExpr:
		return f.evalSequence(ctx, node)
	case *ast.ParameterTermExpr:
		return f.evalParameterTerm(ctx, node.Ideas)
	case *ast.ParameterOperatorExpr:
		return f.evalParameterOperator(ctx, node)
	case *ast.AdjacentEmptyGroupExpr:
		return f.eval(ctx, node.Value)
	case *ast.IndexExpr:
		return f.evalIndex(ctx, node)
	case *ast.CallExpr:
		return f.evalCall(ctx, node)
	case *ast.ObjectExpr:
		return f.evalObject(ctx, node)
	default:
		return Null(), fmt.Errorf("opfor: unsupported expression %T", expression)
	}
}

func (f *fiber) evalSequence(ctx context.Context, node *ast.SequenceExpr) (Value, error) {
	if node == nil || len(node.Elements) == 0 {
		return Null(), nil
	}
	for _, expression := range node.Elements {
		var err error
		if object, ok := expression.(*ast.ObjectExpr); ok {
			// The reference code generator attributes every Step inside this
			// malformed assignment frame to the beginning of the RHS token.
			_, err = f.evalObjectAt(ctx, object, node.Span())
		} else {
			_, err = f.eval(ctx, expression)
		}
		if err != nil {
			return Null(), err
		}
	}
	return Null(), &uncaughtScriptWarning{err: errors.New("assignment is corrupted, did you forget a semicolon?")}
}

func (f *fiber) evalParameterTerm(ctx context.Context, ideas []ast.Expr) (Value, error) {
	result := Null()
	for _, idea := range ideas {
		value, err := f.eval(ctx, idea)
		if err != nil {
			return Null(), err
		}
		result = value
	}
	return result, nil
}

func (f *fiber) evalParameterOperator(ctx context.Context, node *ast.ParameterOperatorExpr) (Value, error) {
	if node == nil || len(node.Right) == 0 {
		return Null(), nil
	}
	f.binaryDepth++
	defer func() { f.binaryDepth-- }()
	right, err := f.evalParameterTerm(ctx, node.Right)
	if err != nil {
		return Null(), err
	}
	left, err := f.eval(ctx, node.Left)
	if err != nil {
		return Null(), err
	}
	return f.evalBinaryValues(ctx, node.Op, node.Span(), left, right)
}

func numberLiteral(node *ast.NumberExpr) (Value, error) {
	raw := strings.TrimSpace(node.Text)
	if raw == "" {
		raw = strings.TrimSpace(node.Raw)
	}
	switch node.Kind {
	case ast.DoubleNumber:
		value, err := lexer.ParseJavaDoubleLiteral(raw)
		if err != nil {
			return Null(), err
		}
		return Double(value), nil
	case ast.LongNumber:
		raw = strings.TrimSuffix(raw, "L")
		value, err := lexer.ParseJavaIntegerLiteral(raw, 64)
		if err != nil {
			return Null(), err
		}
		return Long(value), nil
	default:
		value, err := lexer.ParseJavaIntegerLiteral(raw, 32)
		if err != nil {
			return Null(), err
		}
		return Int(int32(value)), nil
	}
}

func (f *fiber) stringLiteral(ctx context.Context, node *ast.StringExpr) (Value, error) {
	var template compiledStringLiteral
	var cached bool
	if f != nil && f.closure != nil && f.closure.script != nil && f.closure.script.program != nil {
		template, cached = f.closure.script.program.stringLiterals[node]
	}
	if cached {
		if template.err != nil {
			return Null(), template.err
		}
		if template.static {
			if node.Kind == ast.DoubleQuotedString {
				return f.closure.script.runtime.permeateResultFrom(ctx, template.value, nil, node.Span()), nil
			}
			return template.value, nil
		}
	} else if node.Kind == ast.SingleQuotedString {
		return String(decodeSleepSingleQuoted(node.Text)), nil
	}
	decoded := template.decoded
	if !cached {
		var err error
		decoded, err = decodeSleepEscapesAt(node.Text, node.TextRange)
		if err != nil {
			return Null(), err
		}
	}
	switch node.Kind {
	case ast.DoubleQuotedString:
		interpolated, tainted, err := f.interpolate(ctx, decoded, node.Span())
		if err != nil {
			return Null(), err
		}
		value := interpolated
		if strings.Contains(decoded.text, escapedDollarSentinel) {
			value = sleepStringReplaceAll(value, String(escapedDollarSentinel), String("$"))
		}
		return f.closure.script.runtime.permeateResultFrom(ctx, value, tainted, node.Span()), nil
	case ast.BacktickString:
		command, tainted, err := f.interpolate(ctx, decoded, node.Span())
		if err != nil {
			return Null(), err
		}
		command = sleepStringReplaceAll(command, String(escapedDollarSentinel), String("$"))
		commandValue := f.closure.script.runtime.permeateResultFrom(ctx, command, tainted, node.Span())
		return f.closure.script.runtime.invoke(ctx, Invocation{
			Script:    f.closure.script.id,
			Name:      "__EXEC__",
			Span:      node.Span(),
			Arguments: []Argument{{Value: commandValue}},
		})
	default:
		return String(strings.ReplaceAll(decoded.text, escapedDollarSentinel, "$")), nil
	}
}

func decodeSleepSingleQuoted(value string) string {
	var builder strings.Builder
	for index := 0; index < len(value); index++ {
		if value[index] == '\\' && index+1 < len(value) &&
			(value[index+1] == '\\' || value[index+1] == '\'') {
			index++
			builder.WriteByte(value[index])
			continue
		}
		builder.WriteByte(value[index])
	}
	return builder.String()
}

const escapedDollarSentinel = "\x00opfor:escaped-dollar\x00"

func decodeSleepEscapes(value string) (string, error) {
	decoded, err := decodeSleepEscapesAt(value, Span{})
	return decoded.text, err
}

// decodedSleepLiteral retains the source origin of every decoded byte. Sleep
// parses interpolated variables while walking the original quoted token with a
// StringIterator, so an escaped newline must not move a later variable to a new
// source line while an actual newline must. Keeping this map also lets public
// spans remain one-based and source-oriented after escape decoding.
type decodedSleepLiteral struct {
	text             string
	origins          []Position
	escapeBoundaries map[int]struct{}
	segments         []decodedSleepSegment
	end              Position
}

type decodedSleepSegment struct {
	start int
	end   int
	value Value
}

func (d decodedSleepLiteral) valueRange(start, end int) Value {
	result := String("")
	for _, segment := range d.segments {
		if segment.end <= start || segment.start >= end {
			continue
		}
		if segment.start >= start && segment.end <= end {
			result = sleepStringConcat(result, segment.value)
			continue
		}
		left := start
		if left < segment.start {
			left = segment.start
		}
		right := end
		if right > segment.end {
			right = segment.end
		}
		result = sleepStringConcat(result, String(segment.value.String()[left-segment.start:right-segment.start]))
	}
	return result
}

func (d decodedSleepLiteral) sourceSpan(start, end int, fallback Span) Span {
	span := fallback
	if start >= 0 && start < len(d.origins) {
		span.Start = d.origins[start]
	}
	if end >= 0 && end < len(d.origins) {
		span.End = d.origins[end]
	} else {
		span.End = d.end
	}
	return span
}

// interpolationVariableEnd preserves the source-level backslash boundary of
// an escape after decoding. Sleep finds the end of an interpolated variable
// while it is still walking the original parsed literal, where any backslash
// ends the variable name. Without this metadata, for example, "$value\o"
// would be decoded to "$value<reset>" before interpolation and the reset byte
// would incorrectly become part of the variable name.
func (d decodedSleepLiteral) interpolationVariableEnd(index int) bool {
	if _, ok := d.escapeBoundaries[index]; ok {
		return true
	}
	return sleepInterpolationVariableEnd(d.text, index)
}

func decodeSleepEscapesAt(value string, textRange Span) (decodedSleepLiteral, error) {
	var builder strings.Builder
	origins := make([]Position, 0, len(value))
	escapeBoundaries := make(map[int]struct{})
	segments := make([]decodedSleepSegment, 0, len(value))
	position := textRange.Start
	appendDecodedValue := func(origin Position, decoded Value) {
		start := builder.Len()
		text := decoded.String()
		builder.WriteString(text)
		segments = append(segments, decodedSleepSegment{start: start, end: builder.Len(), value: decoded})
		for range len(text) {
			origins = append(origins, origin)
		}
	}
	appendDecoded := func(origin Position, decoded string) { appendDecodedValue(origin, String(decoded)) }
	appendEscaped := func(origin Position, decoded string) {
		escapeBoundaries[builder.Len()] = struct{}{}
		appendDecoded(origin, decoded)
	}
	appendEscapedValue := func(origin Position, decoded Value) {
		escapeBoundaries[builder.Len()] = struct{}{}
		appendDecodedValue(origin, decoded)
	}

	for index := 0; index < len(value); {
		origin := position
		if value[index] != '\\' || index+1 >= len(value) {
			next, size := advanceSleepLiteralPosition(value, index, position)
			appendDecoded(origin, value[index:index+size])
			position = next
			index += size
			continue
		}

		position, _ = advanceSleepLiteralPosition(value, index, position)
		index++
		escapedOrigin := position
		next, size := advanceSleepLiteralPosition(value, index, position)
		escaped := value[index]
		position = next
		index += size
		switch escaped {
		case 'n':
			appendEscaped(origin, "\n")
		case 'r':
			appendEscaped(origin, "\r")
		case 't':
			appendEscaped(origin, "\t")
		case 'b':
			appendEscaped(origin, "\b")
		case 'f':
			appendEscaped(origin, "\f")
		case '$':
			appendEscaped(origin, escapedDollarSentinel)
		case 'c':
			// Cobalt Strike installs these three parsed-literal constants in
			// Sleep's ParserConfig. They are the IRC-style color, underline,
			// and reset control bytes consumed by Cobalt console renderers.
			appendEscaped(origin, "\x03")
		case 'U':
			appendEscaped(origin, "\x1f")
		case 'o':
			appendEscaped(origin, "\x0f")
		case 'u':
			if index+4 > len(value) {
				return decodedSleepLiteral{}, errors.New("opfor: incomplete \\u escape")
			}
			digits := value[index : index+4]
			code, err := strconv.ParseUint(digits, 16, 16)
			if err != nil {
				return decodedSleepLiteral{}, fmt.Errorf("opfor: invalid \\u escape: %w", err)
			}
			for range 4 {
				position, _ = advanceSleepLiteralPosition(value, index, position)
				index++
			}
			if code == '$' {
				appendEscaped(origin, escapedDollarSentinel)
			} else {
				appendEscapedValue(origin, sleepUTF16CharacterValue(uint16(code)))
			}
		case 'x':
			if index+2 > len(value) {
				return decodedSleepLiteral{}, errors.New("opfor: incomplete \\x escape")
			}
			digits := value[index : index+2]
			code, err := strconv.ParseUint(digits, 16, 8)
			if err != nil {
				return decodedSleepLiteral{}, fmt.Errorf("opfor: invalid \\x escape: %w", err)
			}
			for range 2 {
				position, _ = advanceSleepLiteralPosition(value, index, position)
				index++
			}
			if code == '$' {
				appendEscaped(origin, escapedDollarSentinel)
			} else {
				appendEscapedValue(origin, sleepUTF16CharacterValue(uint16(code)))
			}
		default:
			appendEscaped(escapedOrigin, value[index-size:index])
		}
	}
	return decodedSleepLiteral{
		text:             builder.String(),
		origins:          origins,
		escapeBoundaries: escapeBoundaries,
		segments:         segments,
		end:              position,
	}, nil
}

func advanceSleepLiteralPosition(value string, index int, position Position) (Position, int) {
	if value[index] == '\r' && index+1 < len(value) && value[index+1] == '\n' {
		position.Offset += 2
		position.Line++
		position.Column = 1
		return position, 2
	}
	r, size := utf8.DecodeRuneInString(value[index:])
	position.Offset += size
	if r == '\n' {
		position.Line++
		position.Column = 1
	} else {
		position.Column++
	}
	return position, size
}

func (f *fiber) interpolate(ctx context.Context, literal decodedSleepLiteral, span Span) (Value, []Value, error) {
	input := literal.text
	result := String("")
	var tainted []Value
	taintMode := f != nil && f.closure != nil && f.closure.script != nil &&
		f.closure.script.runtime != nil && f.closure.script.runtime.taintMode
	literalStart := 0
	appendLiteral := func(end int) {
		if end > literalStart {
			result = sleepStringConcat(result, literal.valueRange(literalStart, end))
		}
		literalStart = end
	}
	for index := 0; index < len(input); {
		if strings.HasPrefix(input[index:], " $+ ") {
			appendLiteral(index)
			index += 4
			literalStart = index
			continue
		}
		if input[index] != '$' {
			index++
			continue
		}
		appendLiteral(index)
		if index+1 < len(input) && input[index+1] == '+' {
			return Null(), nil, errors.New("opfor: operator $+ must be surrounded with whitespace")
		}
		if index+1 >= len(input) || literal.interpolationVariableEnd(index+1) {
			result = sleepStringConcat(result, String("$"))
			index++
			literalStart = index
			continue
		}

		index++
		width := 0
		aligned := false
		if index < len(input) && input[index] == '[' {
			end := matchingBracket(input, index)
			if end < 0 {
				return Null(), nil, errors.New("opfor: missing close bracket in string alignment")
			}
			expression := input[index+1 : end]
			value, err := f.evalInlineExpression(ctx, expression)
			if err != nil {
				return Null(), nil, err
			}
			if taintMode && value.IsTainted() {
				tainted = append(tainted, value)
			}
			width = int(value.Int32())
			aligned = true
			index = end + 1
		}
		if index >= len(input) || literal.interpolationVariableEnd(index) {
			return Null(), nil, errors.New("opfor: can not align an empty variable")
		}
		nameStart := index
		for index < len(input) && !literal.interpolationVariableEnd(index) {
			index++
		}
		variableSpan := literal.sourceSpan(nameStart, index, span)
		rawVariable := "$" + input[nameStart:index]
		variable := Null()
		if strings.Contains(rawVariable, "[") {
			// CodeGenerator feeds the complete fragment back through parseIdea,
			// so parsed literals support indexed references such as $ds[0].
			var err error
			variable, err = f.evalInlineExpression(ctx, rawVariable)
			if err != nil {
				return Null(), nil, err
			}
		} else {
			var err error
			variable, err = f.readVariable(ctx, rawVariable, variableSpan)
			if err != nil {
				return Null(), nil, err
			}
		}
		if taintMode && variable.IsTainted() {
			tainted = append(tainted, variable)
		}
		value := sleepStringCoercion(variable)
		if aligned {
			value = sleepStringAlign(value, width)
		}
		result = sleepStringConcat(result, value)
		literalStart = index
	}
	appendLiteral(len(input))
	return result, tainted, nil
}

// readVariable mirrors Get's strict-variable diagnostic and its otherwise
// easy-to-miss side effect: an unresolved access installs the default value in
// global scope. Sleep also evaluates ordinary assignment/reference targets
// through Get, so reads and writes share variableCell and only the first access
// of that variable warns.
func (f *fiber) readVariable(ctx context.Context, name string, span Span) (Value, error) {
	cell, err := f.variableCell(ctx, name, span)
	if err != nil {
		return Null(), err
	}
	if cell == nil {
		return Null(), nil
	}
	return cell.Get(), nil
}

func (f *fiber) variableCell(ctx context.Context, name string, span Span) (*Cell, error) {
	if f == nil || f.scope == nil {
		return nil, nil
	}
	if cell, ok, err := f.scope.lookupAt(ctx, name, span); err != nil || ok {
		return cell, err
	}
	name = normalizeVariableName(name)
	cell := NewCell(defaultVariableValue(name))
	if err := f.scope.root.putCellAt(ctx, name, cell, span); err != nil {
		return nil, err
	}
	if f.strictVariablesEnabled() && f.closure.script.runtime != nil {
		f.closure.script.runtime.writeWarning("variable '"+name+"' not declared", span)
	}
	return cell, nil
}

func (f *fiber) strictVariablesEnabled() bool {
	if f == nil || f.closure == nil || f.closure.script == nil {
		return false
	}
	script := f.closure.script
	script.mu.RLock()
	enabled := script.debug&debugRequireStrict == debugRequireStrict
	script.mu.RUnlock()
	return enabled
}

// Sleep parsed-literal variables deliberately have a much wider spelling than
// source-level identifiers. Checkers.isEndOfVar only stops at ASCII whitespace,
// another dollar, or a backslash, so punctuation immediately following a name
// is part of that name. The sentinel represents an escaped dollar whose source
// backslash likewise terminated the preceding variable.
func sleepInterpolationVariableEnd(input string, index int) bool {
	if index >= len(input) || strings.HasPrefix(input[index:], escapedDollarSentinel) {
		return true
	}
	switch input[index] {
	case ' ', '\t', '\n', '$', '\\', '\x03', '\x0f', '\x1f':
		return true
	default:
		return false
	}
}

func matchingBracket(value string, start int) int {
	depth := 0
	for index := start; index < len(value); index++ {
		switch value[index] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func (f *fiber) evalInlineExpression(ctx context.Context, code string) (Value, error) {
	runtime := f.closure.script.runtime
	const prefix, suffix = "return ", ";"
	if _, err := runtime.reserveSourceLength(len(prefix)+len(code)+len(suffix), 0); err != nil {
		return Null(), err
	}
	program, err := runtime.compileReservedSource(NewSource("<interpolation>", []byte(prefix+code+suffix)))
	if err != nil {
		return Null(), err
	}
	closure := &scriptClosure{script: f.closure.script, function: program.function, captured: f.scope}
	return closure.invokeFresh(ctx, nil)
}

func (f *fiber) evalUnary(ctx context.Context, node *ast.UnaryExpr) (Value, error) {
	if node.Postfix && (node.Op == "++" || node.Op == "--") {
		cell, err := f.referenceCell(ctx, node.Operand, true)
		if err != nil {
			return Null(), err
		}
		current := cell.Get()
		delta := Int(1)
		if node.Op == "--" {
			delta = Int(-1)
		}
		updated, err := numericBinary(current, "+", delta)
		if err != nil {
			return Null(), err
		}
		if err := f.setCellAtExecution(ctx, cell, updated, node.Span()); err != nil {
			return Null(), err
		}
		// Sleep calls this syntax the increment/decrement "hack": the parser
		// rewrites `$x++` into `$x = $x + 1`, so its expression result is the
		// newly assigned value rather than C-style postfix's previous value.
		return updated, nil
	}
	if arguments, ok := unaryCallArguments(node); ok {
		evaluated, err := f.callArguments(ctx, arguments)
		if err != nil {
			return Null(), err
		}
		// Sleep's lexer also classifies "not" as an operator, but adjacent
		// function spelling (not(...)) is dispatched through FunctionCallRequest.
		// Preserve that observable distinction here without weakening ordinary
		// unary `not value` parsing.
		return f.invokeNamed(ctx, nil, node.Op, node.Span(), evaluated)
	}
	value, err := f.eval(ctx, node.Operand)
	if err != nil {
		return Null(), err
	}
	return f.evalUnaryValue(ctx, node, value)
}

// evalUnaryValue applies a unary operator to an operand which has already been
// evaluated. Predicate evaluation uses this split so logic tracing can describe
// the operand without executing a side-effecting expression a second time.
func (f *fiber) evalUnaryValue(ctx context.Context, node *ast.UnaryExpr, value Value) (Value, error) {
	switch strings.ToLower(node.Op) {
	case "+":
		return value, nil
	case "-":
		switch value.Kind() {
		case KindDouble:
			return Double(-value.Float64()), nil
		case KindLong:
			return Long(-value.Int64()), nil
		default:
			return Int(-value.Int32()), nil
		}
	case "!", "not":
		return Bool(!value.Truth()), nil
	case "~":
		if value.Kind() == KindLong {
			return Long(^value.Int64()), nil
		}
		return Int(^value.Int32()), nil
	case "-istrue":
		return Bool(value.Truth()), nil
	case "!-istrue":
		return Bool(!value.Truth()), nil
	case "-isarray":
		return Bool(value.Kind() == KindArray), nil
	case "!-isarray":
		return Bool(value.Kind() != KindArray), nil
	case "-ishash":
		return Bool(value.Kind() == KindHash), nil
	case "!-ishash":
		return Bool(value.Kind() != KindHash), nil
	case "-isfunction":
		return Bool(value.Kind() == KindFunction), nil
	case "!-isfunction":
		return Bool(value.Kind() != KindFunction), nil
	case "-istainted":
		return Bool(f.closure.script.runtime.taintMode && value.IsTainted()), nil
	case "!-istainted":
		return Bool(!f.closure.script.runtime.taintMode || !value.IsTainted()), nil
	case "-isnumber":
		return Bool(isSleepNumber(value)), nil
	case "!-isnumber":
		return Bool(!isSleepNumber(value)), nil
	case "-isletter":
		return Bool(sleepStringLength(value) != 0 && allUTF16Units(value, unicode.IsLetter)), nil
	case "!-isletter":
		return Bool(sleepStringLength(value) == 0 || !allUTF16Units(value, unicode.IsLetter)), nil
	case "-isupper":
		return Bool(sleepStringValuesEqual(sleepStringMapCase(value, true), sleepStringCoercion(value))), nil
	case "!-isupper":
		return Bool(!sleepStringValuesEqual(sleepStringMapCase(value, true), sleepStringCoercion(value))), nil
	case "-islower":
		return Bool(sleepStringValuesEqual(sleepStringMapCase(value, false), sleepStringCoercion(value))), nil
	case "!-islower":
		return Bool(!sleepStringValuesEqual(sleepStringMapCase(value, false), sleepStringCoercion(value))), nil
	}
	negated := strings.HasPrefix(node.Op, "!-")
	name := node.Op
	if negated {
		name = strings.TrimPrefix(name, "!")
	}
	result, err := f.closure.script.runtime.invoke(ctx, Invocation{
		Script:    f.closure.script.id,
		Name:      name,
		Span:      node.Span(),
		Arguments: []Argument{{Value: value}},
	})
	if err != nil {
		var unsupported *UnsupportedError
		if errors.As(err, &unsupported) {
			f.closure.script.runtime.writeWarning("Attempted to use non-existent predicate: "+name, node.Span())
			return Bool(negated), nil
		}
		return Null(), err
	}
	if negated {
		return Bool(!result.Truth()), nil
	}
	return Bool(result.Truth()), nil
}

func unaryCallArguments(node *ast.UnaryExpr) ([]ast.Expr, bool) {
	if node == nil || !strings.EqualFold(node.Op, "not") || node.Operand == nil {
		return nil, false
	}
	operandSpan := node.Operand.Span()
	if operandSpan.Start.Offset != node.Span().Start.Offset+len(node.Op) {
		return nil, false
	}
	switch operand := node.Operand.(type) {
	case *ast.GroupExpr:
		return []ast.Expr{operand.Expr}, true
	case *ast.TupleExpr:
		return operand.Elements, true
	default:
		return nil, false
	}
}

func isSleepNumber(value Value) bool {
	// BasicStrings walks a Java String one char at a time and calls
	// Character.isDigit(char). Preserve that UTF-16 boundary: a supplementary
	// decimal digit consists of two surrogate chars, neither of which is a
	// digit. Decoding to Go runes here would incorrectly accept it.
	units := sleepStringUnits(value)
	if len(units) == 0 {
		return false
	}
	dots := 0
	for index, unit := range units {
		if sleepJavaDigit(unit, 10) >= 0 {
			continue
		}
		if unit == '.' && dots == 0 && index+1 < len(units) {
			dots++
			continue
		}
		return false
	}
	return true
}

func allRunes(value string, predicate func(rune) bool) bool {
	for _, r := range value {
		if !predicate(r) {
			return false
		}
	}
	return true
}

func allUTF16Units(value Value, predicate func(rune) bool) bool {
	for _, unit := range sleepStringUnits(value) {
		if !predicate(rune(unit)) {
			return false
		}
	}
	return true
}

func (f *fiber) evalBinary(ctx context.Context, node *ast.BinaryExpr) (Value, error) {
	op := strings.ToLower(node.Op)
	if op == "&&" {
		left, err := f.evalPredicate(ctx, node.Left)
		if err != nil || !left.Truth() {
			return Bool(false), err
		}
		right, err := f.evalPredicate(ctx, node.Right)
		return Bool(err == nil && right.Truth()), err
	}
	if op == "||" {
		left, err := f.evalPredicate(ctx, node.Left)
		if err != nil || left.Truth() {
			return Bool(err == nil && left.Truth()), err
		}
		right, err := f.evalPredicate(ctx, node.Right)
		return Bool(err == nil && right.Truth()), err
	}
	// Inline yield needs to know whether Sleep's non-resumable binary operand
	// frame surrounds the call. See inlineAbort and the canonical inlineb.sl.
	f.binaryDepth++
	defer func() { f.binaryDepth-- }()
	right, err := f.eval(ctx, node.Right)
	if err != nil {
		return Null(), err
	}
	left, err := f.eval(ctx, node.Left)
	if err != nil {
		return Null(), err
	}
	return f.evalBinaryValues(ctx, node.Op, node.Span(), left, right)
}

func (f *fiber) evalBinaryValues(ctx context.Context, operator string, span Span, left, right Value) (Value, error) {
	op := strings.ToLower(operator)
	switch op {
	case "+", "-", "*", "/", "%", "**", "<<", ">>", "&", "|", "^":
		return numericBinary(left, op, right)
	case ".":
		result := sleepStringConcat(left, right)
		runtime := f.closure.script.runtime
		return runtime.permeateResultFrom(ctx, result, runtime.taintedValues(left, right), span), nil
	case "x":
		count := int(right.Int32())
		if count < 0 {
			count = 0
		}
		result := sleepStringRepeat(left, count)
		runtime := f.closure.script.runtime
		return runtime.permeateResultFrom(ctx, result, runtime.taintedValues(left, right), span), nil
	case "==", "!=", "<", "<=", ">", ">=":
		result := numericCompare(left, op, right)
		return Bool(result), nil
	case "<=>":
		return Int(numericSpaceship(left, right)), nil
	case "eq", "ne", "lt", "le", "gt", "ge", "cmp":
		comparison := sleepStringCompareValues(left, right)
		switch op {
		case "eq":
			return Bool(comparison == 0), nil
		case "ne":
			return Bool(comparison != 0), nil
		case "lt":
			return Bool(comparison < 0), nil
		case "le":
			return Bool(comparison <= 0), nil
		case "gt":
			return Bool(comparison > 0), nil
		case "ge":
			return Bool(comparison >= 0), nil
		default:
			return Int(int32(comparison)), nil
		}
	case "!eq", "!ne", "!lt", "!le", "!gt", "!ge":
		comparison := sleepStringCompareValues(left, right)
		var matched bool
		switch strings.TrimPrefix(op, "!") {
		case "eq":
			matched = comparison == 0
		case "ne":
			matched = comparison != 0
		case "lt":
			matched = comparison < 0
		case "le":
			matched = comparison <= 0
		case "gt":
			matched = comparison > 0
		case "ge":
			matched = comparison >= 0
		}
		return Bool(!matched), nil
	case "is", "=~":
		return Bool(left.IdentityEqual(right)), nil
	case "!is", "!=~":
		return Bool(!left.IdentityEqual(right)), nil
	case "isin":
		return Bool(sleepStringContains(right, left)), nil
	case "!isin":
		return Bool(!sleepStringContains(right, left)), nil
	case "in", "notin", "!in":
		contained, err := f.contains(ctx, left, right)
		if op != "in" {
			contained = !contained
		}
		return Bool(contained), err
	case "ismatch", "!ismatch", "hasmatch", "!hasmatch":
		matched, err := f.regexMatch(ctx, op, sleepCanonicalString(left), sleepCanonicalString(right))
		f.applyRegexTaint(left, right)
		return Bool(matched), err
	case "iswm", "!iswm":
		matched := wildcardMatchValues(left, right)
		if op == "!iswm" {
			matched = !matched
		}
		return Bool(matched), nil
	case "isa", "!isa":
		class, ok := portableClassOperand(right)
		if !ok {
			return Null(), newPortableInvalidClassCast(right)
		}
		result, err := f.closure.script.runtime.objectHost.Object(ctx, ObjectInvocation{
			Runtime: f.closure.script.runtime, Script: f.closure.script.id,
			Op: ObjectTypeCheck, Class: class, Target: left, Span: span,
		})
		if err != nil {
			return Null(), err
		}
		truth := result.Truth()
		if op == "!isa" {
			truth = !truth
		}
		return Bool(truth), nil
	}
	if _, err := normalizeFunctionName(operator); err != nil {
		f.closure.script.runtime.writeWarning(fmt.Sprintf("Attempting to use non-existent operator: '%s'", operator), span)
		return Null(), nil
	}
	result, err := f.closure.script.runtime.invoke(ctx, Invocation{
		Script: f.closure.script.id, Name: operator, Span: span,
		Arguments: []Argument{{Value: left}, {Value: right}},
	})
	if err != nil {
		if f.warnsForMissingSleepBridge(err) {
			f.closure.script.runtime.writeWarning(fmt.Sprintf("Attempting to use non-existent operator: '%s'", operator), span)
			return Null(), nil
		}
		return Null(), err
	}
	return result, nil
}

func numericSpaceship(left, right Value) int32 {
	if left.Kind() == KindDouble || right.Kind() == KindDouble {
		a, b := left.Float64(), right.Float64()
		switch {
		case a > b:
			return 1
		case a < b:
			return -1
		default:
			return 0
		}
	}
	a, b := left.Int64(), right.Int64()
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

func numericBinary(left Value, operator string, right Value) (Value, error) {
	if left.Kind() == KindDouble || right.Kind() == KindDouble {
		a, b := left.Float64(), right.Float64()
		switch operator {
		case "+":
			return Double(a + b), nil
		case "-":
			return Double(a - b), nil
		case "*":
			return Double(a * b), nil
		case "/":
			return Double(a / b), nil
		case "%":
			return Double(math.Mod(a, b)), nil
		case "**":
			return Double(math.Pow(a, b)), nil
		}
	}
	if left.Kind() == KindLong || right.Kind() == KindLong {
		a, b := left.Int64(), right.Int64()
		switch operator {
		case "+":
			return Long(a + b), nil
		case "-":
			return Long(a - b), nil
		case "*":
			return Long(a * b), nil
		case "/":
			if b == 0 {
				return Null(), sleepBridgeArithmeticException()
			}
			return Long(a / b), nil
		case "%":
			if b == 0 {
				return Null(), sleepBridgeArithmeticException()
			}
			return Long(a % b), nil
		case "**":
			return Double(math.Pow(float64(a), float64(b))), nil
		case "<<":
			// BasicNumbers delegates to Java's long shift operators, which
			// retain only the low six bits of the shift distance.
			return Long(a << (uint64(b) & 63)), nil
		case ">>":
			return Long(a >> (uint64(b) & 63)), nil
		case "&":
			return Long(a & b), nil
		case "|":
			return Long(a | b), nil
		case "^":
			return Long(a ^ b), nil
		}
	}
	a, b := left.Int32(), right.Int32()
	switch operator {
	case "+":
		return Int(a + b), nil
	case "-":
		return Int(a - b), nil
	case "*":
		return Int(a * b), nil
	case "/":
		if b == 0 {
			return Null(), sleepBridgeArithmeticException()
		}
		return Int(a / b), nil
	case "%":
		if b == 0 {
			return Null(), sleepBridgeArithmeticException()
		}
		return Int(a % b), nil
	case "**":
		return Double(math.Pow(float64(a), float64(b))), nil
	case "<<":
		// The corresponding Java int operators retain the low five bits.
		return Int(a << (uint32(b) & 31)), nil
	case ">>":
		return Int(a >> (uint32(b) & 31)), nil
	case "&":
		return Int(a & b), nil
	case "|":
		return Int(a | b), nil
	case "^":
		return Int(a ^ b), nil
	default:
		return Null(), fmt.Errorf("opfor: unsupported numeric operator %q", operator)
	}
}

func numericCompare(left Value, operator string, right Value) bool {
	if left.Kind() == KindDouble || right.Kind() == KindDouble {
		a, b := left.Float64(), right.Float64()
		switch operator {
		case "==":
			return a == b
		case "!=":
			return a != b
		case "<":
			return a < b
		case "<=":
			return a <= b
		case ">":
			return a > b
		case ">=":
			return a >= b
		}
	}
	a, b := left.Int64(), right.Int64()
	switch operator {
	case "==":
		return a == b
	case "!=":
		return a != b
	case "<":
		return a < b
	case "<=":
		return a <= b
	case ">":
		return a > b
	case ">=":
		return a >= b
	}
	return false
}

func (f *fiber) contains(ctx context.Context, needle, collection Value) (bool, error) {
	switch collection.Kind() {
	case KindHash:
		hash, _ := collection.Hash()
		cell, ok := hash.Cell(needle.String())
		return ok && !cell.Get().IsNull(), nil
	case KindArray:
		array, _ := collection.Array()
		for _, value := range array.Values() {
			if needle.IdentityEqual(value) {
				return true, nil
			}
		}
		return false, nil
	case KindString:
		return sleepStringContains(collection, needle), nil
	case KindFunction:
		iterator, err := f.iterator(ctx, collection, Span{})
		if err != nil {
			return false, err
		}
		for {
			item, ok, nextErr := iterator.next(ctx)
			if nextErr != nil || !ok {
				return false, nextErr
			}
			if needle.IdentityEqual(item.value) {
				return true, nil
			}
		}
	default:
		return needle.IdentityEqual(collection), nil
	}
}

type regexCursor struct {
	matches [][]Value
	index   int
}

func (f *fiber) regexMatch(ctx context.Context, operator, text, pattern string) (bool, error) {
	negated := strings.HasPrefix(operator, "!")
	operator = strings.TrimPrefix(operator, "!")
	expression, err := f.closure.script.runtime.compileSleepRegexBridge(pattern, operator == "ismatch")
	if err != nil {
		var warning *uncaughtScriptWarning
		if errors.As(err, &warning) {
			return false, &predicateDecisionWarning{err: err}
		}
		return false, err
	}
	if operator == "ismatch" {
		match, matchErr := expression.FindStringSubmatchIndexContext(ctx, text)
		if matchErr != nil {
			return false, fmt.Errorf("opfor: regular expression match: %w", matchErr)
		}
		f.lastMatch = sleepRegexCaptures(text, match)
		result := match != nil
		if negated {
			result = !result
		}
		return result, nil
	}
	if f.regexCursors == nil {
		f.regexCursors = make(map[string]*regexCursor)
	}
	// RegexBridge concatenates these values without a separator. Preserve the
	// collision behavior because hasmatch cursors are observable script state.
	key := text + pattern
	cursor := f.regexCursors[key]
	if cursor == nil {
		indices, matchErr := expression.FindAllStringSubmatchIndexContext(ctx, text, -1)
		if matchErr != nil {
			return false, fmt.Errorf("opfor: regular expression match: %w", matchErr)
		}
		cursor = &regexCursor{}
		for _, match := range indices {
			cursor.matches = append(cursor.matches, sleepRegexCaptures(text, match))
		}
		f.regexCursors[key] = cursor
	}
	result := cursor.index < len(cursor.matches)
	if result {
		f.lastMatch = cursor.matches[cursor.index]
		cursor.index++
	} else {
		delete(f.regexCursors, key)
		f.lastMatch = nil
	}
	if negated {
		result = !result
	}
	return result, nil
}

func (f *fiber) applyRegexTaint(values ...Value) {
	if f == nil || f.closure == nil || f.closure.script == nil || !f.closure.script.runtime.taintMode || len(f.lastMatch) == 0 {
		return
	}
	if len(taintedValues(values...)) == 0 {
		return
	}
	tainted := make([]Value, len(f.lastMatch))
	for index, value := range f.lastMatch {
		tainted[index] = f.closure.script.runtime.TaintAll(value)
	}
	f.lastMatch = tainted
}

func sleepCountedQuantifier(pattern string, start int) (int, bool) {
	end := strings.IndexByte(pattern[start+1:], '}')
	if end < 0 {
		return 0, false
	}
	end += start + 1
	body := pattern[start+1 : end]
	if body == "" {
		return 0, false
	}
	comma := false
	digits := false
	for _, current := range body {
		switch {
		case current >= '0' && current <= '9':
			digits = true
		case current == ',' && !comma:
			comma = true
		default:
			return 0, false
		}
	}
	return end, digits
}

func sleepRegexCaptures(text string, match []int) []Value {
	if match == nil || len(match) < 2 {
		return nil
	}
	captures := make([]Value, 0, len(match)/2-1)
	for part := 2; part+1 < len(match); part += 2 {
		if match[part] < 0 {
			captures = append(captures, Null())
			continue
		}
		captures = append(captures, sleepStringValueFromCanonical(text[match[part]:match[part+1]]))
	}
	return captures
}

func wildcardMatch(pattern, value string) bool {
	return wildcardMatchValues(String(pattern), String(value))
}

func wildcardMatchValues(pattern, value Value) bool {
	// BasicStrings.pred_iswm operates on Java UTF-16 chars. Port its pointer
	// algorithm directly, including the unusual rule that * takes the first
	// following literal occurrence while ** takes the last.
	a := sleepStringUnits(pattern)
	b := sleepStringUnits(value)
	if (len(a) == 0 || len(b) == 0) && len(a) != len(b) {
		return false
	}
	aptr, bptr := 0, 0
	for aptr < len(a) {
		if a[aptr] == '*' {
			greedy := aptr+1 < len(a) && a[aptr+1] == '*'
			for aptr < len(a) && a[aptr] == '*' {
				aptr++
				if aptr == len(a) {
					return true
				}
			}
			cptr := aptr
			for cptr < len(a) && a[cptr] != '?' && a[cptr] != '\\' && a[cptr] != '*' {
				cptr++
			}
			if cptr != aptr {
				position := sleepUTF16Index(b, a[aptr:cptr], bptr, greedy)
				if position < bptr {
					return false
				}
				bptr = position
			}
			if a[aptr] == '?' {
				aptr--
			}
		} else if bptr >= len(b) {
			return false
		} else if a[aptr] == '\\' {
			aptr++
			if aptr < len(a) && a[aptr] != b[bptr] {
				return false
			}
		} else if a[aptr] != '?' && a[aptr] != b[bptr] {
			return false
		}
		aptr++
		bptr++
	}
	return bptr == len(b)
}

func sleepUTF16Index(value, literal []uint16, start int, last bool) int {
	if len(literal) == 0 {
		return start
	}
	position := -1
	for index := start; index+len(literal) <= len(value); index++ {
		matched := true
		for offset := range literal {
			if value[index+offset] != literal[offset] {
				matched = false
				break
			}
		}
		if matched {
			position = index
			if !last {
				return position
			}
		}
	}
	return position
}

func sleepUTF16Length(value string) int {
	return sleepRegexUTF16Length(value)
}

func sleepUTF16ToByteIndex(value string, target int) (int, bool) {
	return sleepRegexUTF16ToByteIndex(value, target)
}

func (f *fiber) evalAssignment(ctx context.Context, node *ast.AssignExpr) (Value, error) {
	value, err := f.eval(ctx, node.Value)
	if err != nil {
		return Null(), err
	}
	if tuple, ok := node.Target.(*ast.TupleExpr); ok {
		return f.assignTuple(ctx, tuple.Elements, node.Op, value)
	}
	if group, ok := node.Target.(*ast.GroupExpr); ok && node.Op == "=" {
		if array, isArray := value.Array(); isArray {
			first, exists, accessErr := array.getAtExecution(ctx, f.closure.script, 0)
			if accessErr != nil {
				return Null(), accessErr
			}
			if !exists {
				first = Null()
			}
			return f.assignOne(ctx, group.Expr, node.Op, first)
		}
	}
	return f.assignOne(ctx, node.Target, node.Op, value)
}

func (f *fiber) assignTuple(ctx context.Context, targets []ast.Expr, operator string, value Value) (Value, error) {
	var values []Value
	if array, ok := value.Array(); ok {
		values = array.Values()
	}
	for index, target := range targets {
		candidate := value
		if values != nil {
			candidate = Null()
			if index < len(values) {
				candidate = values[index]
			}
		}
		if _, err := f.assignOne(ctx, target, operator, candidate); err != nil {
			return Null(), err
		}
	}
	return value, nil
}

func (f *fiber) assignOne(ctx context.Context, target ast.Expr, operator string, value Value) (Value, error) {
	if group, ok := target.(*ast.GroupExpr); ok {
		target = group.Expr
	}
	var cell *Cell
	var err error
	if ordinaryIndexReferenceArgument(target) {
		cell, err = f.indexCell(ctx, target)
		var warning *uncaughtScriptWarning
		if errors.As(err, &warning) {
			// Index.evaluate reports its Java exception and leaves Assign's
			// operand frame empty. The following Assign step consequently emits
			// EmptyStackException before the active block is abandoned.
			f.closure.script.runtime.writeWarning(warning.Error(), target.Span())
			err = &uncaughtScriptWarning{err: errors.New(sleepEmptyStackWarning)}
		}
	} else {
		cell, err = f.referenceCell(ctx, target, true)
	}
	if err != nil {
		return Null(), err
	}
	if operator == "=" {
		if err := f.setCellAtExecution(ctx, cell, value, target.Span()); err != nil {
			return Null(), err
		}
		return value, nil
	}
	baseOperator := strings.TrimSuffix(operator, "=")
	current := cell.Get()
	if left, leftOK := current.Array(); leftOK {
		if right, rightOK := value.Array(); rightOK {
			limit := left.Len()
			if right.Len() < limit {
				limit = right.Len()
			}
			for index := 0; index < limit; index++ {
				a, _, accessErr := left.getAtExecution(ctx, f.closure.script, index)
				if accessErr != nil {
					return Null(), accessErr
				}
				b, _, accessErr := right.getAtExecution(ctx, f.closure.script, index)
				if accessErr != nil {
					return Null(), accessErr
				}
				updated, applyErr := applyCompound(a, baseOperator, b)
				if applyErr != nil {
					return Null(), applyErr
				}
				if baseOperator == "." || baseOperator == "x" {
					runtime := f.closure.script.runtime
					updated = runtime.permeateResultFrom(ctx, updated, runtime.taintedValues(a, b), Span{Source: target.Span().Source})
				}
				item, ok, accessErr := left.cellAtExecution(ctx, f.closure.script, index)
				if accessErr != nil {
					return Null(), accessErr
				}
				if !ok {
					return Null(), ErrIndexOutOfRange
				}
				if err := f.setCellAtExecution(ctx, item, updated, Span{}); err != nil {
					return Null(), err
				}
			}
			return current, nil
		}
	}
	updated, err := applyCompound(current, baseOperator, value)
	if err != nil {
		return Null(), err
	}
	if baseOperator == "." || baseOperator == "x" {
		// Sleep's compound-assignment helper does not copy the source line onto
		// its synthetic operation step; DEBUG_TRACE_TAINT therefore reports 0.
		runtime := f.closure.script.runtime
		updated = runtime.permeateResultFrom(ctx, updated, runtime.taintedValues(current, value), Span{Source: target.Span().Source})
	}
	if err := f.setCellAtExecution(ctx, cell, updated, target.Span()); err != nil {
		return Null(), err
	}
	return updated, nil
}

func taintedValues(values ...Value) []Value {
	tainted := make([]Value, 0, len(values))
	for _, value := range values {
		if value.IsTainted() {
			tainted = append(tainted, value)
		}
	}
	return tainted
}

func applyCompound(left Value, operator string, right Value) (Value, error) {
	if operator == "." {
		return sleepStringConcat(left, right), nil
	}
	return numericBinary(left, operator, right)
}

func (f *fiber) evalIndex(ctx context.Context, node *ast.IndexExpr) (Value, error) {
	cell, err := f.indexExprCell(ctx, node)
	if err != nil {
		return Null(), err
	}
	return cell.Get(), nil
}

func (f *fiber) referenceCell(ctx context.Context, expression ast.Expr, create bool) (*Cell, error) {
	switch node := expression.(type) {
	case *ast.VariableExpr:
		return f.variableCell(ctx, node.Raw, node.Span())
	case *ast.GroupExpr:
		return f.referenceCell(ctx, node.Expr, create)
	case *ast.AdjacentEmptyGroupExpr:
		return f.referenceCell(ctx, node.Value, create)
	case *ast.IndexExpr:
		var target Value
		var parent *Cell
		var err error
		if create {
			// Resolving an assignable parent as a reference first lets nested
			// chains such as %hash["x"]["y"] autovivify from the inside out.
			// Non-lvalue targets still fall back to ordinary expression
			// evaluation below.
			parent, err = f.referenceCell(ctx, node.Target, true)
			if err == nil {
				target = parent.Get()
			}
		}
		if parent == nil {
			target, err = f.eval(ctx, node.Target)
			if err != nil {
				return nil, err
			}
		}
		index, err := f.eval(ctx, node.Index)
		if err != nil {
			return nil, err
		}
		if target.IsNull() && create {
			if parent == nil {
				parent, err = f.referenceCell(ctx, node.Target, true)
				if err != nil {
					return nil, err
				}
			}
			if autovivifiesArray(node.Target, index) {
				target = ArrayValue(NewArray())
			} else {
				target = HashValue(NewHash())
			}
			if err := f.setCellAtExecution(ctx, parent, target, node.Span()); err != nil {
				return nil, err
			}
		}
		switch target.Kind() {
		case KindArray:
			array, _ := target.Array()
			position := int(index.Int32())
			cell, ok, accessErr := array.cellAtExecution(ctx, f.closure.script, position)
			if accessErr != nil {
				return nil, accessErr
			}
			if ok {
				return cell, nil
			}
			if create && position >= 0 {
				return array.ensureAtExecution(ctx, f.closure.script, position)
			}
			return nil, ErrIndexOutOfRange
		case KindHash:
			hash, _ := target.Hash()
			if create {
				return hash.EnsureValueContext(ctx, index)
			}
			if cell, ok := hash.Cell(index.String()); ok {
				return cell, nil
			}
			return nil, ErrIndexOutOfRange
		case KindFunction:
			callable, _ := target.Function()
			closure, ok := callable.(*scriptClosure)
			if !ok || closure == nil {
				return nil, fmt.Errorf("opfor: cannot index non-script function")
			}
			return closure.variableCellAt(ctx, index.String(), node.Span())
		default:
			return nil, fmt.Errorf("opfor: cannot assign through %s index", target.Kind())
		}
	default:
		return nil, fmt.Errorf("opfor: %T is not assignable", expression)
	}
}

func autovivifiesArray(target ast.Expr, index Value) bool {
	for {
		switch node := target.(type) {
		case *ast.VariableExpr:
			if node.Kind == ast.ArrayVariable {
				return true
			}
			if node.Kind == ast.HashVariable {
				return false
			}
			return index.Kind() == KindInt || index.Kind() == KindLong || index.Kind() == KindDouble
		case *ast.IndexExpr:
			target = node.Target
		case *ast.GroupExpr:
			target = node.Expr
		default:
			return index.Kind() == KindInt || index.Kind() == KindLong || index.Kind() == KindDouble
		}
	}
}

func (f *fiber) evalCall(ctx context.Context, node *ast.CallExpr) (Value, error) {
	if suspended := f.inlineFiber(node); suspended != nil {
		return f.resumeInlineAt(ctx, node, suspended)
	}
	if identifier, ok := node.Callee.(*ast.IdentifierExpr); ok {
		name := strings.TrimPrefix(identifier.Name, "&")
		if f.useExplicitSpecialCallOverride(name) {
			arguments, err := f.callExpressionArguments(ctx, node)
			if err != nil {
				return Null(), err
			}
			arguments = sleepLexicalCallArguments(name, node.Span(), arguments)
			return f.invokeNamed(ctx, node, identifier.Name, node.Span(), arguments)
		}
		if value, handled, err := f.specialCall(ctx, node, name, node.Args); handled {
			return value, err
		}
		arguments, err := f.callExpressionArguments(ctx, node)
		if err != nil {
			return Null(), err
		}
		arguments = sleepLexicalCallArguments(name, node.Span(), arguments)
		return f.invokeNamed(ctx, node, identifier.Name, node.Span(), arguments)
	}
	if function, ok := node.Callee.(*ast.FunctionRefExpr); ok {
		arguments, err := f.callExpressionArguments(ctx, node)
		if err != nil {
			return Null(), err
		}
		arguments = sleepLexicalCallArguments(strings.TrimPrefix(function.Name, "&"), node.Span(), arguments)
		return f.invokeNamed(ctx, node, function.Name, node.Span(), arguments)
	}
	callee, err := f.eval(ctx, node.Callee)
	if err != nil {
		return Null(), err
	}
	arguments, err := f.callExpressionArguments(ctx, node)
	if err != nil {
		return Null(), err
	}
	return f.invokeCallableAt(ctx, node, callee, arguments)
}

// CodeGenerator pushes a warn call's lexical source line into its argument
// frame before resolving the environment function. Keep the hidden value in
// the actual call arguments so setf, script definitions, importer overrides,
// and the stock BasicUtilities object all observe the same frame.
func sleepLexicalCallArguments(name string, span Span, arguments []Argument) []Argument {
	if strings.TrimPrefix(name, "&") != "warn" {
		return arguments
	}
	return append(arguments, Argument{Value: Int(int32(sleepDisplayLine(span)))})
}

func (f *fiber) useExplicitSpecialCallOverride(name string) bool {
	if !overridableSpecialCall(name) || f == nil || f.closure == nil || f.closure.script == nil {
		return false
	}
	script := f.closure.script
	resolved := script.resolveFunction(name)
	if intrinsic, ok := resolved.(*intrinsicFunctionCallable); ok && intrinsic != nil && intrinsic.name == name {
		return false
	}
	explicitRuntime := script.runtime != nil && script.runtime.hasExplicitFunction(name)
	// ScriptLoader materializes Sleep's stock global bridge table for Java-side
	// introspection. Its default entries resolve through this runtime-callable
	// marker, but they are not importer or script overrides: evaluator intrinsics
	// such as lambda still need their AST-sensitive implementation. An explicit
	// WithFunction/RegisterFunction registration remains authoritative even
	// though the shared table uses the same marker to route it.
	if shared, ok := resolved.(*portableSharedRuntimeCallable); ok && shared != nil &&
		strings.TrimPrefix(shared.name, "&") == name && !explicitRuntime {
		resolved = nil
	}
	return resolved != nil || script.functionWasRemoved(name) || explicitRuntime
}

func overridableSpecialCall(name string) bool {
	switch name {
	case "local", "global", "this", "checkError", "getStackTrace",
		"lambda", "let", "compile_closure", "function", "setf", "invoke",
		"inline", "matched", "matches", "find":
		return true
	default:
		return false
	}
}

func (f *fiber) specialCall(ctx context.Context, call *ast.CallExpr, name string, arguments []ast.Expr) (Value, bool, error) {
	// These names are ordinary Sleep Function bridges even though OPFOR needs
	// evaluator-aware implementations. Evaluate their complete argument list
	// once, in Sleep's right-to-left call order, then dispatch over preserved
	// Arguments. This also keeps function()/setf() bridge handles and direct
	// syntax on one value/reference/named-pair contract.
	switch name {
	case "local", "global", "this", "checkError", "getStackTrace",
		"lambda", "let", "compile_closure", "function", "setf",
		"matched", "matches", "find":
		evaluated, err := f.callExpressionArguments(ctx, call)
		if err != nil {
			return Null(), true, err
		}
		bridge := newIntrinsicFunctionCallable(name)
		value, err := bridge.invokeNamedArgumentsAt(ctx, name, call.Span(), evaluated)
		if f.callTraceEnabled() {
			f.writeCallTrace(formatCall(name, evaluated), value, err, call.Span())
		}
		return value, true, err
	}

	switch name {
	case "local", "global", "this":
		if len(arguments) == 0 {
			return Null(), true, sleepBridgeEmptyStack()
		}
		evaluated, err := f.callExpressionArguments(ctx, call)
		if err != nil {
			if f.callTraceEnabled() {
				f.writeCallTrace(formatCall(name, evaluated), Null(), err, call.Span())
			}
			return Null(), true, err
		}
		for _, argument := range evaluated {
			declaration := argument.Resolve().String()
			for _, variable := range strings.Fields(declaration) {
				if variable == "" || (variable[0] != '$' && variable[0] != '@' && variable[0] != '%') {
					return Null(), true, &uncaughtScriptWarning{err: fmt.Errorf("&%s: malformed variable name '%s' from '%s'", name, variable, declaration)}
				}
				switch name {
				case "local":
					if _, err := f.scope.localAt(ctx, variable, call.Span()); err != nil {
						return Null(), true, err
					}
				case "global":
					if _, err := f.scope.globalAt(ctx, variable, call.Span()); err != nil {
						return Null(), true, err
					}
				case "this":
					if err := f.closure.declareThis(ctx, variable); err != nil {
						return Null(), true, err
					}
				}
			}
		}
		if f.callTraceEnabled() {
			f.writeCallTrace(formatCall(name, evaluated), Null(), nil, call.Span())
		}
		return Null(), true, nil
	case "checkError":
		invocation := Invocation{Script: f.closure.script.id, Name: "checkError", Span: call.Span()}
		if len(arguments) != 0 {
			expression := arguments[0]
			if reference, ok := expression.(*ast.ReferenceExpr); ok {
				expression = reference.Target
			}
			cell, err := f.referenceCell(ctx, expression, true)
			if err != nil {
				return Null(), true, err
			}
			invocation.Arguments = []Argument{{Reference: cell}}
		}
		value, err := f.closure.script.runtime.checkError(ctx, invocation)
		return value, true, err
	case "getStackTrace":
		frames := f.closure.script.getStackTrace()
		values := make([]Value, len(frames))
		for index, frame := range frames {
			values[index] = ObjectValue(portableSleepStackElement{text: frame})
		}
		array, err := newRuntimeArray(f.closure.script.runtime, values...)
		if err != nil {
			return Null(), true, err
		}
		value := ArrayValue(array)
		if f.callTraceEnabled() {
			f.writeCallTrace(formatCall(name, nil), value, nil, call.Span())
		}
		return value, true, nil
	case "iff":
		if len(arguments) < 1 || len(arguments) > 3 {
			return Null(), true, fmt.Errorf("opfor: iff expects 1 to 3 arguments")
		}
		condition, err := f.evalBlockPredicate(ctx, arguments[0])
		if err != nil {
			return Null(), true, err
		}
		if condition.Truth() {
			if len(arguments) >= 2 {
				value, err := f.eval(ctx, arguments[1])
				return value, true, err
			}
			return Bool(true), true, nil
		}
		if len(arguments) == 3 {
			value, err := f.eval(ctx, arguments[2])
			return value, true, err
		}
		return Bool(false), true, nil
	case "lambda", "let", "compile_closure":
		if len(arguments) == 0 {
			if name == "compile_closure" {
				return Null(), true, sleepBridgeEmptyStack()
			}
			return Null(), true, sleepBridgeIllegalArgument("expected &closure--received: $null")
		}
		var closure *scriptClosure
		traceArguments := make([]Argument, 0, len(arguments))
		if name == "compile_closure" {
			code, err := f.eval(ctx, arguments[0])
			if err != nil {
				return Null(), true, err
			}
			if err := f.closure.script.runtime.rejectTaintedCall(ctx, name, []Argument{{Value: code}}); err != nil {
				return Null(), true, err
			}
			traceArguments = append(traceArguments, Argument{Value: code})
			program, err := f.closure.script.runtime.CompileString("eval", code.String())
			if err != nil {
				invocation := Invocation{Script: f.closure.script.id, Name: name, Span: arguments[0].Span()}
				value, flaggedErr := f.closure.script.runtime.flagSourceError(invocation, err)
				return value, true, flaggedErr
			}
			function := *program.function
			function.Name = "<closure>"
			closure = f.closure.script.newClosure(&function, f.scope.root)
		} else {
			value, err := f.eval(ctx, arguments[0])
			if err != nil {
				return Null(), true, err
			}
			template, ok := value.Function()
			if !ok {
				return Null(), true, fmt.Errorf("opfor: %s expected a closure, received %s", name, value.Describe())
			}
			var scriptTemplate *scriptClosure
			scriptTemplate, ok = template.(*scriptClosure)
			if !ok || scriptTemplate == nil {
				return Null(), true, fmt.Errorf("opfor: %s expected a script closure", name)
			}
			if name == "lambda" {
				closure = f.closure.script.newClosure(scriptTemplate.function, f.scope.root)
			} else {
				closure = scriptTemplate
			}
			traceArguments = append(traceArguments, Argument{Value: value})
		}
		bindings, err := f.bindClosureVariables(ctx, closure, arguments[1:])
		if err != nil {
			return Null(), true, err
		}
		traceArguments = append(traceArguments, bindings...)
		result := FunctionValue(closure)
		if f.callTraceEnabled() {
			f.writeCallTrace(formatCall(name, traceArguments), result, nil, call.Span())
		}
		return result, true, nil
	case "function":
		if len(arguments) == 0 {
			return Null(), true, sleepBridgeIllegalArgument("&function: requested function name must begin with '&'")
		}
		value, err := f.eval(ctx, arguments[0])
		if err != nil {
			return Null(), true, err
		}
		if err := f.closure.script.runtime.rejectTaintedCall(ctx, name, []Argument{{Value: value}}); err != nil {
			return Null(), true, err
		}
		functionName := value.String()
		if !strings.HasPrefix(functionName, "&") || len(functionName) == 1 {
			return Null(), true, &uncaughtScriptWarning{err: errors.New("&function: requested function name must begin with '&'")}
		}
		result := FunctionValue(f.callable(functionName, arguments[0].Span()))
		if f.callTraceEnabled() {
			f.writeCallTrace(formatCall(name, []Argument{{Value: value}}), result, nil, call.Span())
		}
		return result, true, nil
	case "setf":
		evaluated, err := f.callExpressionArguments(ctx, call)
		if err != nil {
			return Null(), true, err
		}
		value, err := f.setIntrinsicFunction(evaluated)
		return value, true, err
	case "invoke":
		if len(arguments) == 0 {
			return Null(), true, sleepBridgeIllegalArgument("expected &closure--received: $null")
		}
		evaluated, err := f.callExpressionArguments(ctx, call)
		if err != nil {
			return Null(), true, err
		}
		trace := f.callTraceEnabled()
		traceCall := ""
		if trace {
			traceCall = formatCall(name, evaluated)
		}
		callee := evaluated[0].Resolve()
		positional := make([]Argument, 0, len(evaluated)-1)
		named := make(map[string]Argument)
		for _, argument := range evaluated[1:] {
			if argument.Name != "" {
				named[argument.Name] = argument
			} else {
				positional = append(positional, argument)
			}
		}

		invokeArguments := make([]Argument, 0)
		if len(positional) != 0 {
			values, iteratorErr := iteratorValues(ctx, positional[0].Resolve(), "invoke")
			if iteratorErr != nil {
				return Null(), true, iteratorErr
			}
			for _, value := range values {
				invokeArguments = append(invokeArguments, Argument{Value: value})
			}
		}

		message := Null()
		if len(positional) > 1 {
			message = String(positional[1].Resolve().String())
		}
		if argument, ok := named["message"]; ok {
			message = String(argument.Resolve().String())
		}
		if argument, ok := named["parameters"]; ok {
			hash, hashOK := argument.Resolve().Hash()
			if !hashOK || hash == nil {
				return Null(), true, fmt.Errorf("opfor: invoke parameters expected a hash")
			}
			keys, keysErr := activeHashKeysAtExecution(ctx, f.closure.script, hash, true)
			if keysErr != nil {
				return Null(), true, keysErr
			}
			for _, key := range keys {
				value, accessErr := hash.HashAtValue(ctx, key)
				if accessErr != nil {
					return Null(), true, accessErr
				}
				invokeArguments = append(invokeArguments, Argument{Name: key.String(), Value: value})
			}
		}
		invokeArguments = append(invokeArguments, Argument{Name: "$0", Value: message})

		environment := Null()
		if argument, ok := named["$this"]; ok {
			environment = argument.Resolve()
		}
		var traceFrame *callTraceFrame
		if trace {
			traceFrame = f.beginCallTrace(traceCall, call.Span())
		}
		var internalTrace *callTraceFrame
		if trace {
			if callable, ok := callee.Function(); ok {
				if closure, ok := callable.(*scriptClosure); ok && closure != nil {
					internalTrace = f.beginCallTrace(
						formatClosureCall(callee, "", invokeArguments[:len(invokeArguments)-1]),
						Span{Source: "<internal>", Start: Position{Line: -1}},
					)
				}
			}
		}
		value, err := f.invokeCallableWithEnvironment(ctx, callee, invokeArguments, environment)
		if internalTrace != nil {
			f.finishCallTrace(internalTrace, value, err)
		}
		if traceFrame != nil {
			f.finishCallTrace(traceFrame, value, err)
		}
		var thrown *scriptThrow
		if errors.As(err, &thrown) {
			thrown.addFrame(fmt.Sprintf("   <internal>:-1 %s", describeTraceValue(callee)))
			frame := "&invoke()"
			if span := call.Span(); span.Source != "" {
				frame = fmt.Sprintf("   %s:%d %s", span.Source, sleepDisplayLine(span), frame)
			}
			thrown.addFrame(frame)
		}
		return value, true, err
	case "inline":
		if len(arguments) == 0 {
			return Null(), true, sleepBridgeIllegalArgument("expected &closure--received: $null")
		}
		evaluated, err := f.callExpressionArguments(ctx, call)
		if err != nil {
			return Null(), true, err
		}
		callable, ok := evaluated[0].Resolve().Function()
		if !ok {
			return Null(), true, ErrInvalidCallable
		}
		closure, ok := callable.(*scriptClosure)
		if !ok || closure == nil {
			return Null(), true, errors.New("opfor: inline expected a script closure")
		}
		value, err := f.invokeInlineAt(ctx, call, closure, nil, false)
		return value, true, err
	case "matched":
		// RegexBridge ignores arguments, but Sleep still evaluates them before
		// invoking the bridge. matched() always returns capture groups 1..N;
		// the full group 0 match is intentionally omitted.
		for _, expression := range arguments {
			if _, err := f.eval(ctx, expression); err != nil {
				return Null(), true, err
			}
		}
		array, err := newRuntimeArray(f.closure.script.runtime, f.lastMatch...)
		if err != nil {
			return Null(), true, err
		}
		return ArrayValue(array), true, nil
	case "matches":
		if len(arguments) < 2 {
			return Null(), true, sleepBridgeEmptyStack()
		}
		values := make([]Value, len(arguments))
		for index, expression := range arguments {
			value, err := f.eval(ctx, expression)
			if err != nil {
				return Null(), true, err
			}
			values[index] = value
		}
		expression, err := f.closure.script.runtime.compileSleepRegexBridge(sleepCanonicalString(values[1]), false)
		if err != nil {
			return Null(), true, err
		}
		first, last := int32(-1), int32(-1)
		if len(values) > 2 {
			first = values[2].Int32()
			last = first
		}
		if len(values) > 3 {
			last = values[3].Int32()
		}
		text := sleepCanonicalString(values[0])
		allMatches, matchErr := expression.FindAllStringSubmatchIndexContext(ctx, text, -1)
		if matchErr != nil {
			return Null(), true, fmt.Errorf("opfor: regular expression match: %w", matchErr)
		}
		selectedCount := 0
		for index, match := range allMatches {
			if int32(index) == first {
				selectedCount = 0
			}
			if len(match) >= 2 {
				selectedCount += len(match)/2 - 1
			}
			if int32(index) == last {
				break
			}
		}
		if err := reserveCollectionEntries(f.closure.script.runtime, selectedCount); err != nil {
			return Null(), true, err
		}
		matches := make([]Value, 0, selectedCount)
		for index, match := range allMatches {
			if int32(index) == first {
				matches = nil
			}
			matches = append(matches, sleepRegexCaptures(text, match)...)
			if int32(index) == last {
				break
			}
		}
		result := ArrayValue(NewArray(matches...))
		if len(f.closure.script.runtime.taintedValues(values[0], values[1])) != 0 {
			result = f.closure.script.runtime.TaintAll(result)
		}
		return result, true, nil
	case "find":
		valueCount := len(arguments)
		if valueCount < 2 {
			valueCount = 2
		}
		values := make([]Value, valueCount)
		for index := range values {
			values[index] = Null()
		}
		for index, argument := range arguments {
			value, err := f.eval(ctx, argument)
			if err != nil {
				return Null(), true, err
			}
			values[index] = value
		}
		text := sleepCanonicalString(values[0])
		expression, err := f.closure.script.runtime.compileSleepRegexBridge(sleepCanonicalString(values[1]), false)
		if err != nil {
			return Null(), true, err
		}
		start := 0
		if len(values) > 2 {
			start = int(values[2].Int32())
			if start < 0 {
				start += sleepUTF16Length(text)
			}
		}
		byteStart, ok := sleepUTF16ToByteIndex(text, start)
		if !ok {
			return Null(), true, fmt.Errorf("opfor: find start %d is outside the UTF-16 string boundary", start)
		}
		match, matchErr := expression.FindStringSubmatchIndexAtContext(ctx, text, byteStart)
		if matchErr != nil {
			return Null(), true, fmt.Errorf("opfor: regular expression match: %w", matchErr)
		}
		if match == nil {
			f.lastMatch = nil
			return Null(), true, nil
		}
		f.lastMatch = sleepRegexCaptures(text, match)
		f.applyRegexTaint(values[0], values[1])
		return Int(int32(sleepUTF16Length(text[:match[0]]))), true, nil
	}
	return Null(), false, nil
}

func (f *fiber) callArguments(ctx context.Context, expressions []ast.Expr) ([]Argument, error) {
	return f.callArgumentsGrouped(ctx, expressions, nil)
}

func (f *fiber) callExpressionArguments(ctx context.Context, call *ast.CallExpr) ([]Argument, error) {
	if call == nil {
		return nil, nil
	}
	return f.callArgumentsGrouped(ctx, call.Args, call.ArgGroups)
}

func (f *fiber) callArgumentsGrouped(ctx context.Context, expressions []ast.Expr, groupLengths []int) ([]Argument, error) {
	// A single argument has no cross-group ordering to reconstruct. This is the
	// dominant Sleep-to-Sleep call shape and avoids allocating the temporary
	// group-offset and evaluation slices used by the general parameter-term
	// algorithm below.
	if len(expressions) == 1 {
		argument, err := f.callArgument(ctx, expressions[0])
		if err != nil {
			return nil, err
		}
		return []Argument{argument}, nil
	}
	groups := normalizedCallArgumentGroups(len(expressions), groupLengths)
	starts := make([]int, len(groups))
	offset := 0
	for index, length := range groups {
		starts[index] = offset
		offset += length
	}

	evaluated := make([]Argument, len(expressions))
	// CodeGenerator evaluates comma-delimited parameter terms right-to-left,
	// but evaluates the omitted-comma ideas within each term left-to-right.
	for group := len(groups) - 1; group >= 0; group-- {
		start := starts[group]
		end := start + groups[group]
		for index := start; index < end; index++ {
			argument, err := f.callArgument(ctx, expressions[index])
			if err != nil {
				return nil, err
			}
			evaluated[index] = argument
		}
	}

	// Function bridges pop the shared frame. Consequently ideas from one
	// omitted-comma term are observed in reverse order, while comma-delimited
	// terms retain source order.
	arguments := make([]Argument, 0, len(expressions))
	for group, length := range groups {
		start := starts[group]
		for index := start + length - 1; index >= start; index-- {
			arguments = append(arguments, evaluated[index])
		}
	}
	return arguments, nil
}

func callArgumentEvaluationOrder(argumentCount int, groupLengths []int) []int {
	groups := normalizedCallArgumentGroups(argumentCount, groupLengths)
	starts := make([]int, len(groups))
	offset := 0
	for index, length := range groups {
		starts[index] = offset
		offset += length
	}
	order := make([]int, 0, argumentCount)
	for group := len(groups) - 1; group >= 0; group-- {
		start := starts[group]
		for index := start; index < start+groups[group]; index++ {
			order = append(order, index)
		}
	}
	return order
}

func normalizedCallArgumentGroups(argumentCount int, groupLengths []int) []int {
	if argumentCount == 0 {
		return nil
	}
	total := 0
	valid := len(groupLengths) != 0
	for _, length := range groupLengths {
		if length <= 0 {
			valid = false
			break
		}
		total += length
	}
	if valid && total == argumentCount {
		return groupLengths
	}
	groups := make([]int, argumentCount)
	for index := range groups {
		groups[index] = 1
	}
	return groups
}

func (f *fiber) callArgument(ctx context.Context, expression ast.Expr) (Argument, error) {
	if pair, ok := expression.(*ast.PairExpr); ok {
		argument, err := f.argument(ctx, pair.Value)
		if err != nil {
			return Argument{}, err
		}
		key, err := f.pairExpressionKey(ctx, pair)
		if err != nil {
			return Argument{}, err
		}
		argument.Name = key
		argument.syntax = argumentSyntaxPair
		argument.syntaxName = key
		return argument, nil
	}
	return f.argument(ctx, expression)
}

func (f *fiber) argument(ctx context.Context, expression ast.Expr) (Argument, error) {
	if reference, ok := expression.(*ast.ReferenceExpr); ok {
		cell, err := f.referenceCell(ctx, reference.Target, true)
		if err != nil {
			return Argument{}, err
		}
		name, ok := referenceArgumentName(reference.Target)
		if !ok {
			return Argument{}, errors.New("opfor: referenced argument has no variable name")
		}
		return Argument{
			Name: name, Reference: cell,
			syntax: argumentSyntaxReference, syntaxName: name,
		}, nil
	}
	if variable, ok := ordinaryReferenceArgument(expression); ok && variable.Raw != "$null" {
		cell, err := f.referenceCell(ctx, expression, true)
		if err != nil {
			return Argument{}, err
		}
		return Argument{Reference: cell}, nil
	}
	if ordinaryIndexReferenceArgument(expression) {
		// Sleep's Index atom puts the actual array/hash/closure Scalar on the
		// argument frame. Preserve that cell identity so stock bridges such as
		// clear can replace an indexed scalar without mutating its old value.
		// The same atom also serves ordinary reads and assignments.
		cell, err := f.indexCell(ctx, expression)
		if err != nil {
			return Argument{}, err
		}
		return Argument{Reference: cell}, nil
	}
	value, err := f.eval(ctx, expression)
	return Argument{Value: value}, err
}

func ordinaryReferenceArgument(expression ast.Expr) (*ast.VariableExpr, bool) {
	for {
		switch node := expression.(type) {
		case *ast.VariableExpr:
			return node, true
		case *ast.GroupExpr:
			expression = node.Expr
		case *ast.AdjacentEmptyGroupExpr:
			expression = node.Value
		default:
			return nil, false
		}
	}
}

func ordinaryIndexReferenceArgument(expression ast.Expr) bool {
	for {
		switch node := expression.(type) {
		case *ast.IndexExpr:
			return true
		case *ast.GroupExpr:
			expression = node.Expr
		case *ast.AdjacentEmptyGroupExpr:
			expression = node.Value
		default:
			return false
		}
	}
}

// indexCell mirrors the stock Index atom. Sleep propagates the exact Scalar
// returned by ScalarArray/ScalarHash/closure lookup through both ordinary
// expressions and bridge-call frames. Index only autovivifies an empty
// structure when the original indexed spelling begins with @ or %, and
// ListContainer.getAt grows an out-of-range array by exactly one element rather
// than through the queried numeric position.
func (f *fiber) indexCell(ctx context.Context, expression ast.Expr) (*Cell, error) {
	for {
		switch node := expression.(type) {
		case *ast.GroupExpr:
			expression = node.Expr
		case *ast.AdjacentEmptyGroupExpr:
			expression = node.Value
		case *ast.IndexExpr:
			return f.indexExprCell(ctx, node)
		default:
			return nil, fmt.Errorf("opfor: %T is not indexed", expression)
		}
	}
}

func (f *fiber) indexExprCell(ctx context.Context, node *ast.IndexExpr) (*Cell, error) {
	structure, err := f.indexStructureCell(ctx, node.Target)
	if err != nil {
		return nil, err
	}
	target := structure.Get()
	if target.IsNull() {
		switch sleepIndexRootSigil(node) {
		case '@':
			target = ArrayValue(NewArray())
		case '%':
			target = HashValue(NewHash())
		}
		if !target.IsNull() {
			if err := f.setCellAtExecution(ctx, structure, target, node.Span()); err != nil {
				return nil, err
			}
		}
	}

	index, err := f.eval(ctx, node.Index)
	if err != nil {
		return nil, err
	}
	switch target.Kind() {
	case KindArray:
		array, _ := target.Array()
		position := int(index.Int32())
		size := array.Len()
		if position < 0 && size > 0 {
			// Index.java repeatedly adds the array size. Modulo preserves the
			// exact wrapped position without making a very negative script value
			// consume billions of interpreter iterations.
			position %= size
			if position < 0 {
				position += size
			}
		}
		cell, exists, accessErr := array.cellAtExecution(ctx, f.closure.script, position)
		if accessErr != nil {
			return nil, accessErr
		}
		if exists {
			return cell, nil
		}
		if position < 0 {
			return nil, ErrIndexOutOfRange
		}
		if array.backend != nil {
			return nil, sleepBridgeInvalidIndex(fmt.Sprintf(
				"Index %d out of bounds for length %d", position, array.Len(),
			))
		}
		var appended *Cell
		err = array.mutateCellsAtExecution(ctx, f.closure.script, true, func(cells []*Cell) ([]*Cell, error) {
			if err := reserveCollectionEntriesAtExecution(ctx, f.closure.script, 1); err != nil {
				return nil, err
			}
			appended = NewCell(Null())
			return append(cells, appended), nil
		})
		return appended, err
	case KindHash:
		hash, _ := target.Hash()
		return hash.ensureValueAtExecution(ctx, f.closure.script, index)
	case KindFunction:
		callable, _ := target.Function()
		closure, ok := callable.(*scriptClosure)
		if ok && closure != nil {
			return closure.variableCellAt(ctx, index.String(), node.Span())
		}
	}
	return nil, &uncaughtScriptWarning{err: fmt.Errorf(
		"invalid use of index operator: %s[%s]", target.Describe(), index.Describe(),
	)}
}

func (f *fiber) indexStructureCell(ctx context.Context, expression ast.Expr) (*Cell, error) {
	for {
		switch node := expression.(type) {
		case *ast.VariableExpr:
			if node.Raw == "$null" {
				return NewCell(Null()), nil
			}
			return f.variableCell(ctx, node.Raw, node.Span())
		case *ast.GroupExpr:
			expression = node.Expr
		case *ast.AdjacentEmptyGroupExpr:
			expression = node.Value
		case *ast.IndexExpr:
			return f.indexExprCell(ctx, node)
		default:
			value, err := f.eval(ctx, expression)
			if err != nil {
				return nil, err
			}
			return NewCell(value), nil
		}
	}
}

func sleepIndexRootSigil(node *ast.IndexExpr) byte {
	var expression ast.Expr = node
	for {
		switch current := expression.(type) {
		case *ast.IndexExpr:
			expression = current.Target
		case *ast.GroupExpr:
			expression = current.Expr
		case *ast.AdjacentEmptyGroupExpr:
			expression = current.Value
		case *ast.VariableExpr:
			if current.Raw != "" {
				return current.Raw[0]
			}
			return 0
		default:
			return 0
		}
	}
}

func referenceArgumentName(expression ast.Expr) (string, bool) {
	for {
		switch node := expression.(type) {
		case *ast.VariableExpr:
			return node.Raw, true
		case *ast.GroupExpr:
			expression = node.Expr
		default:
			return "", false
		}
	}
}

func (f *fiber) bindClosureVariables(ctx context.Context, closure *scriptClosure, expressions []ast.Expr) ([]Argument, error) {
	if closure == nil {
		return nil, errors.New("opfor: closure is nil")
	}
	bound := make([]Argument, 0, len(expressions))
	for _, expression := range expressions {
		name, value, err := f.closureBinding(ctx, expression)
		if err != nil {
			return nil, err
		}
		bound = append(bound, Argument{Name: name, Value: value})
		if name == "$this" {
			callable, ok := value.Function()
			if !ok {
				return nil, fmt.Errorf("opfor: $this closure binding expected a closure, received %s", value.Describe())
			}
			source, ok := callable.(*scriptClosure)
			if !ok || source == nil {
				return nil, errors.New("opfor: $this closure binding expected a script closure")
			}
			if source == closure {
				continue
			}
			if err := source.ensureStateAt(ctx, expression.Span()); err != nil {
				return nil, err
			}
			source.mu.Lock()
			sharedState, sharedThis := source.state, source.thisHash
			source.mu.Unlock()
			closure.mu.Lock()
			closure.state, closure.thisHash = sharedState, sharedThis
			closure.mu.Unlock()
			continue
		}
		if err := closure.ensureStateAt(ctx, expression.Span()); err != nil {
			return nil, err
		}
		closure.mu.Lock()
		state := closure.state
		closure.mu.Unlock()
		cell, err := state.localAt(ctx, name, expression.Span())
		if err != nil {
			return nil, err
		}
		if err := f.setCellAtExecution(ctx, cell, value, Span{}); err != nil {
			return nil, err
		}
	}
	return bound, nil
}

func (f *fiber) closureBinding(ctx context.Context, expression ast.Expr) (string, Value, error) {
	switch node := expression.(type) {
	case *ast.PairExpr:
		name, err := f.pairExpressionKey(ctx, node)
		if err != nil {
			return "", Null(), err
		}
		value, err := f.eval(ctx, node.Value)
		return name, value, err
	case *ast.ReferenceExpr:
		name, ok := referenceArgumentName(node.Target)
		if !ok {
			return "", Null(), errors.New("opfor: referenced closure binding has no variable name")
		}
		cell, err := f.referenceCell(ctx, node.Target, true)
		if err != nil {
			return "", Null(), err
		}
		return name, cell.Get(), nil
	default:
		value, err := f.eval(ctx, expression)
		if err != nil {
			return "", Null(), err
		}
		object, ok := value.Object()
		if !ok {
			return "", Null(), fmt.Errorf("opfor: malformed closure binding %s", value.Describe())
		}
		switch pair := object.(type) {
		case sleepKeyValue:
			return pair.key.String(), pair.value, nil
		case *sleepKeyValue:
			if pair != nil {
				return pair.key.String(), pair.value, nil
			}
		}
		return "", Null(), fmt.Errorf("opfor: malformed closure binding %s", value.Describe())
	}
}

func (f *fiber) pairKey(ctx context.Context, expression ast.Expr) (string, error) {
	switch node := expression.(type) {
	case *ast.IdentifierExpr:
		return node.Name, nil
	case *ast.VariableExpr:
		return node.Raw, nil
	case *ast.StringExpr:
		value, err := f.stringLiteral(ctx, node)
		return value.String(), err
	default:
		value, err := f.eval(ctx, expression)
		return value.String(), err
	}
}

func (f *fiber) pairExpressionKey(ctx context.Context, pair *ast.PairExpr) (string, error) {
	if pair == nil {
		return "", nil
	}
	if pair.RawKey != "" {
		return pair.RawKey, nil
	}
	return f.pairKey(ctx, pair.Key)
}

func (f *fiber) callable(name string, span Span) Callable {
	if closure := f.closure.script.resolveFunction(name); closure != nil {
		return closure
	}
	normalized := strings.TrimPrefix(name, "&")
	if f.closure.script.functionWasRemoved(normalized) {
		return nil
	}
	if overridableSpecialCall(normalized) {
		// Keep a stable marker for function()/setf() round trips. When this
		// marker is installed in the mutable script function table, direct call
		// syntax still selects the AST-sensitive intrinsic fallback.
		return newIntrinsicFunctionCallable(normalized)
	}
	runtime := f.closure.script.runtime
	if family, ok := sleepBuiltinFamilyForFunction(normalized); ok && runtime != nil &&
		runtime.hasStockFunction(normalized) && !runtime.hasExplicitFunction(normalized) {
		return newSleepBuiltinFunctionCallable(normalized, family, runtime, f.closure.script.id, span)
	}
	if runtime != nil && runtime.hasRegisteredFunction(normalized) {
		return &runtimeCallable{runtime: runtime, script: f.closure.script.id, name: name, span: span}
	}
	return nil
}

type intrinsicFunctionCallable struct {
	name   string
	family intrinsicFunctionFamily
}

func (c *intrinsicFunctionCallable) Invoke(context.Context, ...Value) (Value, error) {
	if c == nil {
		return Null(), ErrInvalidCallable
	}
	return Null(), fmt.Errorf("opfor: intrinsic function &%s requires named script call syntax", c.name)
}

func (c *intrinsicFunctionCallable) String() string {
	if c == nil {
		return "&"
	}
	return "&" + c.name
}

func (f *fiber) invokeNamed(ctx context.Context, callExpr *ast.CallExpr, name string, span Span, arguments []Argument) (Value, error) {
	// Sleep's CallRequest deliberately excludes the aggregate constructors and
	// &warn from call tracing. In particular, tracing &warn would duplicate the
	// warning that the function itself emits.
	trace := f.callTraceEnabled() && name != "@" && name != "%" && name != "warn"
	call := ""
	var traceFrame *callTraceFrame
	if trace {
		call = formatCall(name, arguments)
		traceFrame = f.beginCallTrace(call, span)
	}
	f.flushForkLaunchTraceBeforeCall(name)
	var value Value
	var err error
	closure := f.closure.script.resolveFunction(name)
	runtime := f.closure.script.runtime
	taintedDescription := ""
	if runtime.taintMode {
		taintedDescription = describeTaintedValues(taintedArgumentValues(arguments))
	}
	if closure != nil {
		if scriptClosure, ok := closure.(*scriptClosure); ok {
			if invalid := validateNamedParameters(arguments); invalid != nil {
				err = invalid
			} else if scriptClosure.inline {
				if trace {
					call = "<inline> " + call
					if traceFrame != nil {
						traceFrame.call = call
					}
				}
				value, err = f.invokeInlineAt(ctx, callExpr, scriptClosure, arguments, true)
			} else {
				scriptArguments := make([]Argument, 0, len(arguments)+1)
				scriptArguments = append(scriptArguments, Argument{Name: "$0", Value: String("&" + strings.TrimPrefix(name, "&"))})
				scriptArguments = append(scriptArguments, arguments...)
				value, err = scriptClosure.invokeArguments(ctx, scriptArguments)
			}
		} else if native, ok := closure.(intrinsicNamedArgumentCallable); ok {
			value, err = native.invokeNamedArgumentsAt(ctx, name, span, arguments)
		} else if native, ok := closure.(portableArgumentCallable); ok {
			value, err = native.invokeArgumentsAt(ctx, span, arguments)
		} else if native, ok := closure.(interface {
			invokeArguments(context.Context, []Argument) (Value, error)
		}); ok {
			value, err = native.invokeArguments(ctx, arguments)
		} else {
			value, err = closure.Invoke(ctx, resolvedArguments(arguments)...)
		}
	} else if f.closure.script.functionWasRemoved(name) {
		f.closure.script.runtime.writeWarning("Attempted to call non-existent function &"+strings.TrimPrefix(name, "&"), span)
		value = Null()
	} else {
		value, err = runtime.invoke(ctx, Invocation{Script: f.closure.script.id, Name: name, Span: span, Arguments: arguments})
		if err != nil && f.warnsForMissingSleepBridge(err) {
			runtime.writeWarning("Attempted to call non-existent function &"+strings.TrimPrefix(name, "&"), span)
			value, err = Null(), nil
		}
	}
	if err == nil && closure != nil {
		if policyCallable, ok := closure.(interface{ appliesTaintPolicy() bool }); !ok || !policyCallable.appliesTaintPolicy() {
			value = runtime.permeateResultWithDescription(ctx, value, taintedDescription, span)
		}
	}
	var leak *localScopeLeakError
	if errors.As(err, &leak) {
		f.closure.script.runtime.writeWarning(leak.Error(), span)
		value, err = Null(), nil
	}
	var thrown *scriptThrow
	if errors.As(err, &thrown) {
		frame := formatCall(name, nil)
		if scriptClosure, ok := f.closure.script.resolveFunction(name).(*scriptClosure); ok && scriptClosure.inline {
			frame = "<inline> " + frame
		}
		if span.Source != "" {
			frame = fmt.Sprintf("   %s:%d %s", span.Source, sleepDisplayLine(span), frame)
		}
		thrown.addFrame(frame)
	}
	traceErr := err
	var returned *inlineReturn
	var yielded *inlineYield
	var controlled *loopControl
	if errors.As(err, &returned) {
		value, traceErr = returned.value, nil
	} else if errors.As(err, &yielded) {
		value, traceErr = yielded.value, nil
	} else if errors.As(err, &controlled) {
		value, traceErr = Null(), nil
	}
	f.flushForkLaunchTraceAfterCall()
	if traceFrame != nil {
		f.finishCallTrace(traceFrame, value, traceErr)
	}
	return value, err
}

func (f *fiber) warnsForMissingSleepBridge(err error) bool {
	if f == nil || f.closure == nil || f.closure.script == nil || f.closure.script.runtime == nil {
		return false
	}
	if _, defaultHost := f.closure.script.runtime.host.(unsupportedHost); !defaultHost {
		return false
	}
	name := ""
	if f.closure.script.program != nil {
		name = strings.ToLower(strings.TrimSpace(f.closure.script.program.source.Name))
	}
	if strings.HasSuffix(name, ".cna") {
		return false
	}
	var unsupported *UnsupportedError
	return errors.As(err, &unsupported)
}

func (f *fiber) invokeCallableAt(ctx context.Context, call *ast.CallExpr, value Value, arguments []Argument) (Value, error) {
	callable, ok := value.Function()
	if !ok {
		return Null(), ErrInvalidCallable
	}
	traceFrame := f.beginCallTrace(formatClosureCall(value, "", arguments), call.Span())
	runtime := f.closure.script.runtime
	taintedDescription := ""
	if runtime.taintMode {
		taintedInputs := taintedArgumentValues(arguments)
		if value.IsTainted() {
			taintedInputs = append([]Value{value}, taintedInputs...)
		}
		taintedDescription = describeTaintedValues(taintedInputs)
	}
	var result Value
	var err error
	if closure, ok := callable.(*scriptClosure); ok && closure.inline {
		if invalid := validateNamedParameters(arguments); invalid != nil {
			err = invalid
		} else {
			result, err = f.invokeInlineAt(ctx, call, closure, arguments, true)
		}
	} else {
		result, err = f.invokeCallable(ctx, value, arguments)
	}
	if err == nil {
		result = runtime.permeateResultWithDescription(ctx, result, taintedDescription, call.Span())
	}
	if traceFrame != nil {
		f.finishCallTrace(traceFrame, result, err)
	}
	return result, err
}

func (f *fiber) invokeCallable(ctx context.Context, value Value, arguments []Argument) (Value, error) {
	callable, ok := value.Function()
	if !ok {
		return Null(), ErrInvalidCallable
	}
	if closure, ok := callable.(*scriptClosure); ok {
		if invalid := validateNamedParameters(arguments); invalid != nil {
			return Null(), invalid
		}
		if closure.inline {
			return f.invokeInline(ctx, closure, arguments)
		}
		return closure.invokeArguments(ctx, arguments)
	}
	if callable, ok := callable.(interface {
		invokeArguments(context.Context, []Argument) (Value, error)
	}); ok {
		return callable.invokeArguments(ctx, arguments)
	}
	return callable.Invoke(ctx, resolvedArguments(arguments)...)
}

func (f *fiber) invokeCallableWithEnvironment(ctx context.Context, value Value, arguments []Argument, environment Value) (Value, error) {
	if environment.IsNull() {
		return f.invokeCallable(ctx, value, arguments)
	}
	callable, ok := value.Function()
	if !ok {
		return Null(), ErrInvalidCallable
	}
	target, ok := callable.(*scriptClosure)
	if !ok || target == nil {
		return Null(), errors.New("opfor: invoke $this requires a script closure target")
	}
	environmentCallable, ok := environment.Function()
	if !ok {
		return Null(), fmt.Errorf("opfor: invoke $this expected a closure, received %s", environment.Describe())
	}
	source, ok := environmentCallable.(*scriptClosure)
	if !ok || source == nil {
		return Null(), errors.New("opfor: invoke $this expected a script closure")
	}
	if err := source.ensureStateAt(ctx, Span{}); err != nil {
		return Null(), err
	}
	source.mu.Lock()
	sharedState, sharedThis := source.state, source.thisHash
	source.mu.Unlock()
	temporary := &scriptClosure{
		script: target.script, function: target.function, captured: target.captured,
		state: sharedState, thisHash: sharedThis, id: target.id, inline: target.inline,
	}
	arguments = append(arguments, Argument{Name: "$this", Value: FunctionValue(source)})
	return f.invokeCallable(ctx, FunctionValue(temporary), arguments)
}

func partitionArguments(arguments []Argument) ([]Value, []Argument) {
	values := make([]Value, 0, len(arguments))
	named := make([]Argument, 0)
	for _, argument := range arguments {
		if argument.Name != "" {
			named = append(named, argument)
		} else {
			values = append(values, argument.Resolve())
		}
	}
	return values, named
}

func validateNamedParameters(arguments []Argument) error {
	for _, argument := range arguments {
		if argument.Name == "" {
			continue
		}
		switch argument.Name[0] {
		case '$', '@', '%':
			continue
		default:
			return &uncaughtScriptWarning{err: fmt.Errorf("unreachable named parameter: %s", argument.Name)}
		}
	}
	return nil
}

func resolvedArguments(arguments []Argument) []Value {
	values := make([]Value, len(arguments))
	for index, argument := range arguments {
		values[index] = argument.Resolve()
	}
	return values
}

type runtimeCallable struct {
	runtime *Runtime
	script  ScriptID
	name    string
	span    Span
}

func (c *runtimeCallable) Invoke(ctx context.Context, values ...Value) (Value, error) {
	arguments := make([]Argument, len(values))
	for i, value := range values {
		arguments[i] = Argument{Value: value}
	}
	return c.runtime.invoke(ctx, Invocation{Script: c.script, Name: c.name, Span: c.span, Arguments: arguments})
}

func (c *runtimeCallable) invokeArguments(ctx context.Context, arguments []Argument) (Value, error) {
	return c.runtime.invoke(ctx, Invocation{Script: c.script, Name: c.name, Span: c.span, Arguments: arguments})
}

func (c *runtimeCallable) String() string { return "&" + strings.TrimPrefix(c.name, "&") }

type namedValue struct {
	name  string
	value Value
}
type classReference string

func (f *fiber) evalObject(ctx context.Context, node *ast.ObjectExpr) (Value, error) {
	return f.evalObjectAt(ctx, node, node.Span())
}

func (f *fiber) evalObjectAt(ctx context.Context, node *ast.ObjectExpr, warningSpan Span) (Value, error) {
	arguments, err := f.callArguments(ctx, node.Args)
	if err != nil {
		return Null(), err
	}
	message := ""
	if node.Message != nil {
		message = strings.TrimSpace(node.Message.Name)
	}

	if identifier, ok := node.Target.(*ast.IdentifierExpr); ok && strings.EqualFold(identifier.Name, "new") {
		invocation := ObjectInvocation{Runtime: f.closure.script.runtime, Script: f.closure.script.id, Op: ObjectConstruct, Class: f.closure.script.resolveClass(message), Span: node.Span()}
		invocation.Arguments = arguments
		traceCall, trace := portableObjectCallTrace(invocation)
		var traceFrame *callTraceFrame
		if trace {
			traceFrame = f.beginCallTrace(traceCall, node.Span())
		}
		taintedInputs := f.closure.script.runtime.taintedArguments(arguments)
		value, callErr := f.closure.script.runtime.objectHost.Object(ctx, invocation)
		if traceFrame != nil {
			f.finishCallTrace(traceFrame, value, callErr)
		}
		if callErr != nil {
			return f.handlePortableObjectError(callErr, node.Span())
		}
		value = f.closure.script.runtime.permeateResultFrom(ctx, value, taintedInputs, node.Span())
		return value, nil
	}
	target, err := f.eval(ctx, node.Target)
	if err != nil {
		return Null(), err
	}
	if callable, ok := target.Function(); ok {
		runtime := f.closure.script.runtime
		taintedInputs := runtime.taintedArguments(arguments)
		if runtime.taintMode && target.IsTainted() {
			taintedInputs = append([]Value{target}, taintedInputs...)
		}
		traceFrame := f.beginCallTrace(formatClosureCall(target, message, arguments), node.Span())
		callArguments := arguments
		if message != "" {
			callArguments = append(append([]Argument(nil), arguments...), Argument{Name: "$0", Value: String(message)})
		}
		var value Value
		var callErr error
		if closure, ok := callable.(*scriptClosure); ok {
			if invalid := validateNamedParameters(arguments); invalid != nil {
				callErr = invalid
			} else {
				value, callErr = closure.invokeArguments(ctx, callArguments)
			}
		} else {
			value, callErr = callable.Invoke(ctx, resolvedArguments(callArguments)...)
		}
		if traceFrame != nil {
			f.finishCallTrace(traceFrame, value, callErr)
		}
		if callErr != nil {
			// Sleep routes direct bracket invocation through
			// CallRequest.ClosureCallRequest. When the invoked closure leaves a
			// thrown value in the script environment, CallRequest records the
			// closure itself at the bracket call site before the exception
			// continues to the enclosing try frame.
			var thrown *scriptThrow
			if errors.As(callErr, &thrown) {
				frame := target.String()
				span := node.Span()
				if span.Source != "" {
					frame = fmt.Sprintf("   %s:%d %s", span.Source, sleepDisplayLine(span), frame)
				}
				thrown.addFrame(frame)
			}
			return f.handleBracketCallError(callErr, node.Span())
		}
		value = runtime.permeateResultFrom(ctx, value, taintedInputs, node.Span())
		return value, nil
	}
	if target.IsNull() {
		f.closure.script.runtime.writeWarning("Attempted to call a non-static method on a null reference", warningSpan)
		return Null(), nil
	}

	invocation := ObjectInvocation{Runtime: f.closure.script.runtime, Script: f.closure.script.id, Op: ObjectInvoke, Target: target, Message: message, Span: node.Span(), Arguments: arguments}
	if identifier, ok := node.Target.(*ast.IdentifierExpr); ok {
		invocation.Class = f.closure.script.resolveClass(identifier.Name)
		invocation.Target = Null()
	}
	if message == "" {
		invocation.Op = ObjectGet
	}
	runtime := f.closure.script.runtime
	taintedInputs := runtime.taintedArguments(arguments)
	targetTainted := runtime.taintMode && target.IsTainted()
	if targetTainted {
		taintedInputs = append([]Value{target}, taintedInputs...)
	}
	if len(taintedInputs) != 0 && !targetTainted && !invocation.Target.IsNull() {
		target = runtime.Taint(target)
		invocation.Target = target
		if cell, referenceErr := f.referenceCell(ctx, node.Target, false); referenceErr == nil && cell != nil {
			cell.setTaintValue(target)
		}
		runtime.traceTaint(ctx, "tainted object: "+target.Describe()+" from: "+describeTaintedValues(taintedInputs), node.Span())
	}
	traceCall, trace := portableObjectCallTrace(invocation)
	var traceFrame *callTraceFrame
	if trace {
		traceFrame = f.beginCallTrace(traceCall, node.Span())
	}
	value, callErr := f.closure.script.runtime.objectHost.Object(ctx, invocation)
	if traceFrame != nil {
		f.finishCallTrace(traceFrame, value, callErr)
	}
	if callErr != nil {
		return f.handlePortableObjectError(callErr, node.Span())
	}
	value = runtime.permeateResultFrom(ctx, value, taintedInputs, node.Span())
	return value, nil
}

func (f *fiber) handlePortableObjectError(err error, span Span) (Value, error) {
	var exception *portableJavaException
	if !errors.As(err, &exception) {
		// Importer ObjectHost errors and unsupported operations retain the
		// public host boundary's ordinary fatal-error behavior.
		return Null(), err
	}
	if f == nil || f.closure == nil || f.closure.script == nil {
		return Null(), err
	}

	script := f.closure.script
	value := ObjectValue(exception)
	script.mu.Lock()
	script.lastError = value
	debug := script.debug
	if debug&2 == 2 && debug&32 == 32 {
		// ScriptEnvironment.flagError calls checkError before requesting a
		// throw, so the pending soft-error slot is empty inside the catch body.
		script.lastError = Null()
	}
	script.mu.Unlock()

	if debug&2 != 2 {
		return Null(), nil
	}
	if debug&32 == 32 {
		thrown := &scriptThrow{value: value}
		frames := append([]string(nil), exception.frames...)
		if exception.frame != "" {
			frame := exception.frame
			if span.Source != "" {
				frame = fmt.Sprintf("   %s:%d %s", filepath.Base(span.Source), sleepDisplayLine(span), frame)
			}
			frames = append([]string{frame}, frames...)
		}
		thrown.frames = frames
		return Null(), thrown
	}
	if script.runtime != nil {
		script.runtime.writeWarning("checkError(): "+exception.Error(), span)
	}
	return Null(), nil
}

func (f *fiber) handleBracketCallError(err error, fallback Span) (Value, error) {
	// These are VM control transfers, not bridge failures. In particular a
	// thrown Sleep value must reach the nearest saved try frame, and a callcc
	// handoff must reach the source scriptClosure so it can park its fiber.
	// Converting either into bracket-call $error state changes control flow.
	var transfer *callCCTransfer
	var thrown *scriptThrow
	var returned *inlineReturn
	var yielded *inlineYield
	var controlled *loopControl
	var exited *scriptExit
	if errors.As(err, &transfer) || errors.As(err, &thrown) ||
		errors.As(err, &returned) || errors.As(err, &yielded) || errors.As(err, &exited) {
		return Null(), err
	}
	if errors.As(err, &controlled) {
		return Null(), err
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, ErrResourceLimit) || errors.Is(err, ErrScriptUnloaded) {
		return Null(), err
	}

	span := fallback
	message := err.Error()
	var runtimeError *RuntimeError
	if errors.As(err, &runtimeError) {
		if runtimeError.Span.Source != "" {
			span = runtimeError.Span
		}
		if runtimeError.Cause != nil {
			message = runtimeError.Cause.Error()
		}
	}
	if f != nil && f.closure != nil && f.closure.script != nil {
		script := f.closure.script
		script.mu.Lock()
		script.lastError = String(message)
		showErrors := script.debug&1 == 1
		script.mu.Unlock()
		if showErrors && script.runtime != nil {
			script.runtime.writeWarning(message, span)
		}
	}
	return Null(), nil
}

func (c *scriptClosure) declareThis(ctx context.Context, name string) error {
	if c == nil {
		return nil
	}
	if err := c.ensureStateAt(ctx, Span{}); err != nil {
		return err
	}
	c.mu.Lock()
	state, thisHash := c.state, c.thisHash
	c.mu.Unlock()
	name = normalizeVariableName(name)
	_, exists, err := state.ownCellAt(ctx, name, Span{})
	if err != nil {
		return err
	}
	if exists {
		return scriptExecutionError(ctx)
	}
	key := strings.TrimLeft(name, "$@%")
	cell, err := thisHash.ensureDirectAtExecution(ctx, c.script, String(key))
	if err != nil {
		return err
	}
	if cell.Get().IsNull() {
		if err := cell.setAtExecution(ctx, c.script, defaultVariableValue(name), Span{}); err != nil {
			return err
		}
	}
	if err := scriptExecutionError(ctx); err != nil {
		return err
	}
	// A lambda binding may have installed the closure variable between the
	// read above and this write. Sleep's this() leaves any existing closure
	// variable untouched, including a null-valued one.
	if _, exists, err := state.ownCellAt(ctx, name, Span{}); err != nil {
		return err
	} else if !exists {
		if err := state.putCellAt(ctx, name, cell, Span{}); err != nil {
			return err
		}
	}
	return nil
}

func (c *scriptClosure) ensureStateAt(ctx context.Context, span Span) error {
	if c == nil {
		return errors.New("opfor: closure is nil")
	}
	c.mu.Lock()
	if c.state != nil {
		c.mu.Unlock()
		return nil
	}
	if c.stateErr != nil {
		err := c.stateErr
		c.mu.Unlock()
		return err
	}
	captured := c.captured
	c.mu.Unlock()
	if captured == nil {
		return errors.New("opfor: closure has no captured variable scope")
	}

	// Do not hold the coroutine mutex while calling importer code. stateInit
	// serializes the one factory call, while the second state check permits a
	// provider to execute unrelated callbacks without blocking this closure's
	// suspension machinery.
	c.stateInit.Lock()
	defer c.stateInit.Unlock()
	c.mu.Lock()
	if c.state != nil {
		c.mu.Unlock()
		return nil
	}
	if c.stateErr != nil {
		err := c.stateErr
		c.mu.Unlock()
		return err
	}
	c.mu.Unlock()
	state, err := captured.internalChildAt(ctx, span)
	if err != nil {
		c.mu.Lock()
		c.stateErr = err
		c.mu.Unlock()
		return err
	}
	c.mu.Lock()
	c.state = state
	c.thisHash = NewHash()
	c.mu.Unlock()
	return nil
}

func (c *scriptClosure) ensureStateLocked() {
	if c.state != nil {
		return
	}
	c.state = c.captured.child()
	c.thisHash = NewHash()
}
