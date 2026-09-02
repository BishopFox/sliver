package opfor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/sliverarmory/opfor/internal/ast"
)

const debugTraceLogic int32 = 64

const sleepEmptyStackWarning = "internal error - class java.util.EmptyStackException"

// predicateDecisionWarning marks a Java-bridge warning raised after predicate
// operands have already populated CheckEval's setup frame. It still unwraps to
// uncaughtScriptWarning for the VM's ordinary Block recovery, but must not be
// mistaken for an operand-setup failure that corrupts the predicate frame and
// produces Sleep's second EmptyStackException warning.
type predicateDecisionWarning struct {
	err error
}

func (warning *predicateDecisionWarning) Error() string {
	if warning == nil || warning.err == nil {
		return ""
	}
	return warning.err.Error()
}

func (warning *predicateDecisionWarning) Unwrap() error {
	if warning == nil {
		return nil
	}
	return warning.err
}

// evalBlockPredicate models CheckEval's nested setup Block. If evaluating an
// operand raises a Java-bridge warning, that setup Block reports it and
// returns empty; CheckEval then invokes the predicate with its corrupted
// evaluator frame, which raises the EmptyStackException consumed by the
// containing Block.
func (f *fiber) evalBlockPredicate(ctx context.Context, expression ast.Expr) (Value, error) {
	value, err := f.evalPredicate(ctx, expression)
	return value, f.predicateSetupError(err, expression)
}

func (f *fiber) predicateSetupError(err error, expression ast.Expr) error {
	if err == nil {
		return nil
	}
	var decisionWarning *predicateDecisionWarning
	if errors.As(err, &decisionWarning) {
		return err
	}
	var warning *uncaughtScriptWarning
	if !errors.As(err, &warning) {
		return err
	}
	span := Span{}
	if expression != nil {
		span = expression.Span()
	}
	var nested *RuntimeError
	if errors.As(err, &nested) && nested.Span.Source != "" {
		span = nested.Span
	}
	if f != nil && f.closure != nil && f.closure.script != nil && f.closure.script.runtime != nil {
		f.closure.script.runtime.writeWarning(warning.Error(), span)
	}
	return &uncaughtScriptWarning{err: errors.New(sleepEmptyStackWarning)}
}

// evalPredicate evaluates an expression in Sleep's predicate grammar. This is
// deliberately distinct from ordinary expression evaluation: a three-term
// form such as "2 + 2" asks the registered "+" predicate (which returns
// false), while a scalar idea is tested through -istrue.
func (f *fiber) evalPredicate(ctx context.Context, expression ast.Expr) (Value, error) {
	if expression == nil {
		return Bool(false), nil
	}
	switch node := expression.(type) {
	case *ast.GroupExpr:
		return f.evalPredicate(ctx, node.Expr)
	case *ast.BinaryExpr:
		op := strings.ToLower(node.Op)
		if op == "&&" {
			left, err := f.evalPredicate(ctx, node.Left)
			if err != nil || !left.Truth() {
				return Bool(false), err
			}
			return f.evalPredicate(ctx, node.Right)
		}
		if op == "||" {
			left, err := f.evalPredicate(ctx, node.Left)
			if err != nil || left.Truth() {
				return Bool(err == nil && left.Truth()), err
			}
			return f.evalPredicate(ctx, node.Right)
		}
		left, err := f.eval(ctx, node.Left)
		if err != nil {
			return Null(), err
		}
		right, err := f.eval(ctx, node.Right)
		if err != nil {
			return Null(), err
		}
		result, known, err := f.decideBinaryPredicate(ctx, node, left, right)
		if err != nil {
			return Null(), err
		}
		if known {
			f.traceBinaryPredicate(left, node.Op, right, result.Truth(), node.Span())
		}
		return Bool(result.Truth()), nil
	case *ast.UnaryExpr:
		op := strings.ToLower(node.Op)
		if strings.HasPrefix(op, "-") || strings.HasPrefix(op, "!-") {
			operand, err := f.eval(ctx, node.Operand)
			if err != nil {
				return Null(), err
			}
			value, err := f.evalUnaryValue(ctx, node, operand)
			if err != nil {
				return Null(), err
			}
			if knownUnaryPredicate(op) {
				f.traceUnaryPredicate(node.Op, operand, value.Truth(), node.Span())
			}
			return Bool(value.Truth()), nil
		}
		value, err := f.eval(ctx, node)
		if err != nil {
			return Null(), err
		}
		f.traceTruthPredicate(value, node.Span())
		return Bool(value.Truth()), nil
	default:
		value, err := f.eval(ctx, expression)
		if err != nil {
			return Null(), err
		}
		f.traceTruthPredicate(value, expression.Span())
		return Bool(value.Truth()), nil
	}
}

func (f *fiber) decideBinaryPredicate(ctx context.Context, node *ast.BinaryExpr, left, right Value) (Value, bool, error) {
	switch strings.ToLower(node.Op) {
	case "+", "-", "*", "/", "%", "**", "<<", ">>", "&", "|", "^":
		// BasicNumbers implements both Operator and Predicate. Its decide method
		// returns false for arithmetic spellings instead of performing them.
		return Bool(false), true, nil
	case ".", "x", "cmp", "<=>":
		// BasicStrings installs four distinct helper objects which implement
		// Operator only. ScriptEnvironment.getPredicate performs a Java cast, so
		// using any of these keys in predicate position raises the cast warning
		// consumed by the active Sleep Block.
		return Null(), false, sleepBasicStringsPredicateCastWarning(strings.ToLower(node.Op))
	case "==", "!=", "<", "<=", ">", ">=":
		return Bool(numericCompare(left, strings.ToLower(node.Op), right)), true, nil
	case "eq", "ne", "lt", "gt", "!eq", "!ne", "!lt", "!gt":
		op := strings.ToLower(node.Op)
		negated := strings.HasPrefix(op, "!")
		op = strings.TrimPrefix(op, "!")
		comparison := sleepStringCompareValues(left, right)
		matched := comparison == 0
		switch op {
		case "ne":
			matched = comparison != 0
		case "lt":
			matched = comparison < 0
		case "gt":
			matched = comparison > 0
		}
		if negated {
			matched = !matched
		}
		return Bool(matched), true, nil
	case "is", "=~":
		return Bool(left.IdentityEqual(right)), true, nil
	case "!is", "!=~":
		return Bool(!left.IdentityEqual(right)), true, nil
	case "isin", "!isin":
		matched := sleepStringContains(right, left)
		if strings.HasPrefix(node.Op, "!") {
			matched = !matched
		}
		return Bool(matched), true, nil
	case "in", "!in":
		matched, err := f.contains(ctx, left, right)
		if strings.ToLower(node.Op) != "in" {
			matched = !matched
		}
		return Bool(matched), true, err
	case "ismatch", "!ismatch", "hasmatch", "!hasmatch":
		matched, err := f.regexMatch(ctx, strings.ToLower(node.Op), sleepCanonicalString(left), sleepCanonicalString(right))
		f.applyRegexTaint(left, right)
		return Bool(matched), true, err
	case "iswm", "!iswm":
		matched := wildcardMatchValues(left, right)
		if strings.HasPrefix(node.Op, "!") {
			matched = !matched
		}
		return Bool(matched), true, nil
	case "isa", "!isa":
		class, ok := portableClassOperand(right)
		if !ok {
			// The JVM predicate throws before CheckEval emits its logic trace.
			return Null(), false, newPortableInvalidClassCast(right)
		}
		value, err := f.closure.script.runtime.objectHost.Object(ctx, ObjectInvocation{
			Runtime: f.closure.script.runtime, Script: f.closure.script.id,
			Op: ObjectTypeCheck, Class: class, Target: left, Span: node.Span(),
		})
		matched := value.Truth()
		if strings.HasPrefix(node.Op, "!") {
			matched = !matched
		}
		return Bool(matched), true, err
	default:
		value, err := f.closure.script.runtime.invoke(ctx, Invocation{
			Script: f.closure.script.id, Name: node.Op, Span: node.Span(),
			Arguments: []Argument{{Value: left}, {Value: right}},
		})
		if err != nil {
			if f.warnsForMissingSleepBridge(err) {
				f.closure.script.runtime.writeWarning("Attempted to use non-existent predicate: "+node.Op, node.Span())
				return Bool(false), true, nil
			}
			return Null(), false, err
		}
		return Bool(value.Truth()), true, nil
	}
}

func sleepBasicStringsPredicateCastWarning(operator string) error {
	helper := map[string]string{
		".":   "oper_concat",
		"x":   "oper_multiply",
		"cmp": "oper_compare",
		"<=>": "oper_spaceship",
	}[operator]
	actual := "sleep.bridges.BasicStrings$" + helper
	target := "sleep.interfaces.Predicate"
	return &predicateDecisionWarning{err: sleepBridgeIllegalArgument(fmt.Sprintf(
		"attempted an invalid cast: class %s cannot be cast to class %s (%s and %s are in unnamed module of loader 'app')",
		actual, target, actual, target,
	))}
}

func knownUnaryPredicate(operator string) bool {
	operator = strings.TrimPrefix(strings.ToLower(operator), "!")
	switch operator {
	case "-istrue", "-isarray", "-ishash", "-isfunction", "-isnumber", "-isletter",
		"-istainted", "-isupper", "-islower", "-e", "-exists", "-f", "-d",
		"-canread", "-canwrite", "-isdir", "-isfile", "-ishidden", "-eof":
		return true
	default:
		return false
	}
}

func (f *fiber) predicateTraceScript() *Script {
	if f == nil || f.closure == nil || f.closure.script == nil {
		return nil
	}
	script := f.closure.script
	script.mu.RLock()
	enabled := script.debug&debugTraceLogic == debugTraceLogic
	script.mu.RUnlock()
	if !enabled || script.runtime == nil || script.runtime.stderr == nil {
		return nil
	}
	return script
}

func (f *fiber) traceBinaryPredicate(left Value, operator string, right Value, result bool, span Span) {
	script := f.predicateTraceScript()
	if script == nil {
		return
	}
	f.writePredicateTrace(script, fmt.Sprintf("%s %s %s ? %s", left.Describe(), operator, right.Describe(), truthWord(result)), span)
}

func (f *fiber) traceUnaryPredicate(operator string, operand Value, result bool, span Span) {
	script := f.predicateTraceScript()
	if script == nil {
		return
	}
	f.writePredicateTrace(script, fmt.Sprintf("%s %s ? %s", operator, operand.Describe(), truthWord(result)), span)
}

func (f *fiber) traceTruthPredicate(value Value, span Span) {
	script := f.predicateTraceScript()
	if script == nil {
		return
	}
	f.writePredicateTrace(script, fmt.Sprintf("-istrue %s ? %s", value.Describe(), truthWord(value.Truth())), span)
}

func (f *fiber) tracePresencePredicate(value Value, present bool, span Span) {
	script := f.predicateTraceScript()
	if script == nil {
		return
	}
	f.writePredicateTrace(script, fmt.Sprintf("%s !is $null ? %s", value.Describe(), truthWord(present)), span)
}

func (f *fiber) writePredicateTrace(script *Script, message string, span Span) {
	if span.Source != "" {
		_, _ = fmt.Fprintf(script.runtime.stderr, "Trace: %s at %s:%d\n", message, span.Source, sleepDisplayLine(span))
		return
	}
	_, _ = fmt.Fprintf(script.runtime.stderr, "Trace: %s\n", message)
}

func truthWord(value bool) string {
	if value {
		return "TRUE"
	}
	return "FALSE"
}
