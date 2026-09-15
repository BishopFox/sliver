package parser

import (
	"strings"
	"unicode/utf8"

	"github.com/sliverarmory/opfor/internal/ast"
	"github.com/sliverarmory/opfor/internal/lexer"
)

const (
	precedenceLowest = iota
	precedenceAssignment
	precedencePair
	precedenceOr
	precedenceAnd
	precedencePredicate
	precedenceAdditive
	precedenceProduct
	precedencePrefix
	precedencePostfix
)

var assignmentOperators = map[string]struct{}{
	"=":   {},
	"+=":  {},
	"-=":  {},
	"*=":  {},
	"/=":  {},
	"%=":  {},
	".=":  {},
	"&=":  {},
	"|=":  {},
	"^=":  {},
	"<<=": {},
	">>=": {},
	"**=": {},
}

var knownWordOperators = map[string]struct{}{
	"eq":       {},
	"ge":       {},
	"gt":       {},
	"hasmatch": {},
	"in":       {},
	"isa":      {},
	"isin":     {},
	"ismatch":  {},
	"is":       {},
	"iswm":     {},
	"le":       {},
	"lt":       {},
	"ne":       {},
	"notin":    {},
}

var symbolicPredicates = map[string]struct{}{
	"==":  {},
	"!=":  {},
	"<":   {},
	">":   {},
	"<=":  {},
	">=":  {},
	"=~":  {},
	"!=~": {},
	"===": {},
	"!==": {},
	"!~":  {},
}

type infixOperator struct {
	op               string
	precedence       int
	rightAssociative bool
	tokenCount       int
}

func (p *parser) parseExpression(minimumPrecedence int) ast.Expr {
	left := p.parsePrefix()
	if left == nil {
		return nil
	}

	for !p.atEnd() {
		// Calls, indexes, and postfix mutation bind tighter than all infix
		// operators and may be chained in any order.
		switch {
		case p.checkKind(lexer.LeftParen) && sleepFunctionCallCallee(left):
			left = p.finishCall(left)
			continue
		case p.checkKind(lexer.LeftParen) && p.peek(1).Kind == lexer.RightParen:
			p.advance()
			close := p.advance()
			left = &ast.AdjacentEmptyGroupExpr{
				ExprBase: ast.ExprBase{Base: ast.Base{Range: joinSpans(left.Span(), close.Span)}},
				Value:    left,
			}
			continue
		case p.checkKind(lexer.LeftBracket) && !(p.options.AllowOmittedSemicolons && whitespaceSeparatedClosureObject(left, p.current())):
			left = p.finishIndex(left)
			continue
		case p.checkLexeme("++", "--"):
			operator := p.advance()
			if !sleepPostfixMutationOperand(left, operator) {
				// Sleep implements increment/decrement as a lexical "hack":
				// Checkers accepts only one scalar token whose spelling itself
				// ends in ++ or --. It is not a general postfix operator, so
				// whitespace, grouping, and indexed collection cells are syntax
				// errors even though each would otherwise be assignable.
				p.addDiagnostic(
					lexer.SeverityError,
					diagnosticUnexpectedToken,
					joinSpans(left.Span(), operator.Span),
					"Syntax error",
				)
				return left
			}
			left = &ast.UnaryExpr{
				ExprBase: ast.ExprBase{Base: ast.Base{Range: joinSpans(left.Span(), operator.Span)}},
				Op:       operator.Lexeme,
				Operand:  left,
				Postfix:  true,
			}
			continue
		}

		operator, ok := p.currentInfixOperator()
		if !ok || operator.precedence < minimumPrecedence {
			break
		}

		operatorTokens := make([]lexer.Token, 0, operator.tokenCount)
		for index := 0; index < operator.tokenCount; index++ {
			operatorTokens = append(operatorTokens, p.advance())
		}
		p.checkBinaryWhitespace(operatorTokens)

		rightMinimum := operator.precedence + 1
		if operator.rightAssociative {
			rightMinimum = operator.precedence
		}
		right := p.parseExpression(rightMinimum)
		if right == nil {
			p.errorAt(p.current(), diagnosticExpectedExpression, "expected expression after %q", operator.op)
			break
		}

		range_ := joinSpans(left.Span(), right.Span())
		switch {
		case operator.op == "=>":
			left = &ast.PairExpr{
				ExprBase: ast.ExprBase{Base: ast.Base{Range: range_}},
				Key:      left,
				Value:    right,
				RawKey:   p.sleepRawPairKey(left),
			}
		case isAssignmentOperator(operator.op):
			if !isAssignable(left) {
				p.addDiagnostic(lexer.SeverityError, diagnosticInvalidAssignment, left.Span(), "expression is not assignable")
			}
			left = &ast.AssignExpr{
				ExprBase: ast.ExprBase{Base: ast.Base{Range: range_}},
				Target:   left,
				Op:       operator.op,
				Value:    right,
			}
		default:
			left = &ast.BinaryExpr{
				ExprBase: ast.ExprBase{Base: ast.Base{Range: range_}},
				Left:     left,
				Op:       operator.op,
				Right:    right,
			}
		}
	}

	return left
}

// sleepPostfixMutationOperand mirrors Checkers.isIncrementHack and
// Checkers.isDecrementHack. The reference lexer retains `$name++` as one term,
// so its test is equivalent to an adjacent, non-empty scalar variable here.
// In particular this intentionally excludes grouped scalars and indexed cells.
func sleepPostfixMutationOperand(expression ast.Expr, operator lexer.Token) bool {
	variable, ok := expression.(*ast.VariableExpr)
	return ok && variable.Kind == ast.ScalarVariable && variable.Name != "" &&
		expression.Span().End.Offset == operator.Span.Start.Offset
}

func sleepFunctionCallCallee(expression ast.Expr) bool {
	switch expression.(type) {
	case *ast.IdentifierExpr, *ast.FunctionRefExpr:
		return true
	default:
		return false
	}
}

func (p *parser) sleepRawPairKey(expression ast.Expr) string {
	unary, ok := expression.(*ast.UnaryExpr)
	if !ok || unary.Postfix {
		return ""
	}
	return p.sleepRawExpression(expression)
}

func (p *parser) sleepRawExpression(expression ast.Expr) string {
	if expression == nil {
		return ""
	}
	span := expression.Span()
	var raw strings.Builder
	var previous lexer.Token
	havePrevious := false
	for _, token := range p.tokens {
		if token.Kind == lexer.EOF || token.Span.Start.Offset < span.Start.Offset || token.Span.End.Offset > span.End.Offset {
			continue
		}
		if havePrevious && !previous.Adjacent(token) {
			raw.WriteByte(' ')
		}
		raw.WriteString(token.Lexeme)
		previous = token
		havePrevious = true
	}
	return raw.String()
}

// A bracket following a closure literal after whitespace is a second Sleep
// idea, not an index operation. The distinction matters for the classic
// missing-semicolon closure assignment: `{ ... } [$func]` shares the Assign
// frame and is rejected at runtime after both ideas have executed.
func whitespaceSeparatedClosureObject(left ast.Expr, bracket lexer.Token) bool {
	if !bracket.LeadingWhitespace {
		return false
	}
	switch node := left.(type) {
	case *ast.ClosureExpr:
		return true
	case *ast.AssignExpr:
		_, ok := node.Value.(*ast.ClosureExpr)
		return ok
	default:
		return false
	}
}

func (p *parser) parsePrefix() ast.Expr {
	if callee := p.parseSleepSpecialConstructorCallee(); callee != nil {
		return callee
	}
	if span, ok := p.sleepMalformedNumericIdentifier(); ok {
		p.addDiagnostic(lexer.SeverityError, diagnosticExpectedExpression, span, "Unknown expression")
		return &ast.NullExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
			Raw:      "null",
		}
	}
	if span, ok := p.sleepMalformedDottedNumber(); ok {
		p.addDiagnostic(lexer.SeverityError, diagnosticExpectedExpression, span, "Unknown expression")
		return &ast.NullExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
			Raw:      "null",
		}
	}

	token := p.current()
	switch token.Kind {
	case lexer.Integer, lexer.Long, lexer.Double:
		p.advance()
		return p.numberExpression(token, token.Lexeme, token.Span)
	case lexer.SingleString, lexer.DoubleString, lexer.BacktickString:
		p.advance()
		kind := ast.SingleQuotedString
		if token.Kind == lexer.DoubleString {
			kind = ast.DoubleQuotedString
		} else if token.Kind == lexer.BacktickString {
			kind = ast.BacktickString
		}
		if token.Kind == lexer.DoubleString {
			p.validateParsedLiteralAlignments(token)
		}
		return &ast.StringExpr{
			ExprBase:  ast.ExprBase{Base: ast.Base{Range: token.Span}},
			Kind:      kind,
			Raw:       token.Lexeme,
			Text:      token.Text,
			TextRange: token.TextSpan,
		}
	case lexer.Scalar:
		p.advance()
		return p.variableFromToken(token)
	case lexer.Array:
		p.advance()
		if token.Text == "" && p.checkKind(lexer.LeftParen) {
			return p.parseArrayLiteral(token)
		}
		return &ast.VariableExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: token.Span}},
			Kind:     ast.ArrayVariable,
			Name:     token.Text,
			Raw:      token.Lexeme,
		}
	case lexer.Hash:
		p.advance()
		if token.Text == "" && p.checkKind(lexer.LeftParen) {
			return p.parseHashLiteral(token)
		}
		return &ast.VariableExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: token.Span}},
			Kind:     ast.HashVariable,
			Name:     token.Text,
			Raw:      token.Lexeme,
		}
	case lexer.Function:
		p.advance()
		return &ast.FunctionRefExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: token.Span}},
			Name:     token.Text,
			Raw:      token.Lexeme,
		}
	case lexer.Reference:
		p.advance()
		return p.referenceFromToken(token)
	case lexer.Class:
		p.advance()
		return &ast.ClassExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: token.Span}},
			Name:     token.Text,
			Raw:      token.Lexeme,
		}
	case lexer.Identifier, lexer.Keyword:
		return p.parseIdentifier()
	case lexer.Operator:
		if p.checkLexeme("+", "-", "!", "~") || p.checkWord("not") || strings.HasPrefix(token.Lexeme, "-") || strings.HasPrefix(token.Lexeme, "!-") {
			return p.parseUnary()
		}
		// Some built-ins (notably not(...)) are lexed as operators but are
		// callable by name in Sleep.
		if p.peek(1).Kind == lexer.LeftParen {
			p.advance()
			return &ast.IdentifierExpr{
				ExprBase: ast.ExprBase{Base: ast.Base{Range: token.Span}},
				Name:     token.Text,
			}
		}
	case lexer.LeftParen:
		return p.parseGroupOrTuple()
	case lexer.LeftBrace:
		return p.parseClosure()
	case lexer.LeftBracket:
		return p.parseObject()
	}

	return nil
}

// Sleep keeps an exact unsigned NaN/Infinity term callable when it touches an
// empty collection sigil followed by parentheses. Thus NaN@(2) dispatches
// &NaN@, while 1@(2) and NaN@name remain unknown numeric terms.
func (p *parser) parseSleepSpecialConstructorCallee() ast.Expr {
	first := p.current()
	if first.Kind != lexer.Identifier || first.Lexeme != "NaN" && first.Lexeme != "Infinity" {
		return nil
	}
	sigil := p.peek(1)
	if (sigil.Kind != lexer.Array && sigil.Kind != lexer.Hash) || sigil.Text != "" || !first.Adjacent(sigil) {
		return nil
	}
	open := p.peek(2)
	if open.Kind != lexer.LeftParen || !sigil.Adjacent(open) {
		return nil
	}
	p.advance()
	p.advance()
	return &ast.IdentifierExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: joinSpans(first.Span, sigil.Span)}},
		Name:     first.Lexeme + sigil.Lexeme,
	}
}

func (p *parser) parseIdentifier() ast.Expr {
	token := p.advance()
	if (token.Lexeme == "NaN" || token.Lexeme == "Infinity") && !p.checkKind(lexer.LeftParen) {
		numeric := token
		numeric.Kind = lexer.Double
		return p.numberExpression(numeric, token.Lexeme, token.Span)
	}
	// TokenParser checks function-call syntax before isBoolean. Consequently
	// true()/false() are ordinary calls even though the same exact lowercase
	// spellings are boolean values when bare. Case variants are identifiers,
	// and Sleep has no bare `null` literal (the scalar spelling is $null).
	if p.checkKind(lexer.LeftParen) {
		return &ast.IdentifierExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: token.Span}},
			Name:     token.Text,
		}
	}
	switch token.Lexeme {
	case "true":
		return &ast.BoolExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: token.Span}},
			Value:    true,
			Raw:      token.Lexeme,
		}
	case "false":
		return &ast.BoolExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: token.Span}},
			Value:    false,
			Raw:      token.Lexeme,
		}
	default:
		return &ast.IdentifierExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: token.Span}},
			Name:     token.Text,
		}
	}
}

func (p *parser) parseUnary() ast.Expr {
	first := p.advance()
	op := first.Lexeme
	if (op == "+" || op == "-") && first.Adjacent(p.current()) {
		number := p.current()
		if number.Kind == lexer.Integer || number.Kind == lexer.Long || number.Kind == lexer.Double {
			p.advance()
			return p.numberExpression(number, op+number.Lexeme, joinSpans(first.Span, number.Span))
		}
		if sleepNumericLookingIdentifier(number) {
			p.advance()
			span := joinSpans(first.Span, number.Span)
			p.addDiagnostic(lexer.SeverityError, diagnosticExpectedExpression, span, "Unknown expression")
			return &ast.NullExpr{
				ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
				Raw:      "null",
			}
		}
	}
	// Unary predicates are extensible and spelled -predicate value, with an
	// optional leading !. The lexer deliberately leaves unknown predicate
	// names generic, so combine adjacent pieces here.
	if op == "!" && p.checkLexeme("-") && p.current().Adjacent(p.peek(1)) && p.peek(1).Kind == lexer.Identifier {
		minus := p.advance()
		name := p.advance()
		op += minus.Lexeme + name.Lexeme
	} else if op == "-" && p.current().Kind == lexer.Identifier && first.Adjacent(p.current()) {
		name := p.advance()
		op += name.Lexeme
	}

	operand := p.parseExpression(precedencePrefix)
	if op == "+" || op == "-" {
		span := first.Span
		if operand != nil {
			span = joinSpans(first.Span, operand.Span())
		}
		p.addDiagnostic(lexer.SeverityError, diagnosticExpectedExpression, span, "Unknown expression")
		return &ast.NullExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
			Raw:      "null",
		}
	}
	if operand == nil {
		p.errorAt(p.current(), diagnosticExpectedExpression, "expected operand after %q", op)
		return nil
	}
	return &ast.UnaryExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: joinSpans(first.Span, operand.Span())}},
		Op:       op,
		Operand:  operand,
		Postfix:  false,
	}
}

func sleepNumericLookingIdentifier(token lexer.Token) bool {
	if token.Kind != lexer.Identifier || token.Lexeme == "" {
		return false
	}
	first, _ := utf8.DecodeRuneInString(token.Lexeme)
	return lexer.IsJavaDecimalDigit(first)
}

func (p *parser) numberExpression(token lexer.Token, raw string, span lexer.Span) ast.Expr {
	kind, ok := lexer.ClassifyNumericLiteral(raw, token.Kind)
	if !ok {
		p.addDiagnostic(lexer.SeverityError, diagnosticExpectedExpression, span, "Unknown expression")
		return &ast.NullExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
			Raw:      "null",
		}
	}

	numberKind := ast.IntegerNumber
	if kind == lexer.Long {
		numberKind = ast.LongNumber
	} else if kind == lexer.Double {
		numberKind = ast.DoubleNumber
	}
	return &ast.NumberExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: span}},
		Kind:     numberKind,
		Raw:      raw,
		Text:     raw,
	}
}

func (p *parser) parseGroupOrTuple() ast.Expr {
	open := p.advance()
	if p.matchKind(lexer.RightParen) {
		close := p.previous()
		return &ast.TupleExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: joinSpans(open.Span, close.Span)}},
		}
	}

	elements := p.parseExpressionListUntil(lexer.RightParen, false)
	close, closed := p.expectKind(lexer.RightParen, "')'")
	end := open.Span
	if closed {
		end = close.Span
	} else if len(elements) != 0 {
		end = elements[len(elements)-1].Span()
	}
	range_ := joinSpans(open.Span, end)
	if len(elements) == 1 {
		return &ast.GroupExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: range_}},
			Expr:     elements[0],
		}
	}
	return &ast.TupleExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: range_}},
		Elements: elements,
	}
}

func (p *parser) parseClosure() ast.Expr {
	body := p.parseBlock()
	return &ast.ClosureExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: body.Span()}},
		Body:     body,
	}
}

func (p *parser) parseArrayLiteral(sigil lexer.Token) ast.Expr {
	p.advance() // '('
	elements := p.parseExpressionListUntil(lexer.RightParen, false)
	close, closed := p.expectKind(lexer.RightParen, "')' after array literal")
	end := sigil.Span
	if closed {
		end = close.Span
	} else if len(elements) != 0 {
		end = elements[len(elements)-1].Span()
	}
	return &ast.ArrayLiteralExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: joinSpans(sigil.Span, end)}},
		Elements: elements,
	}
}

func (p *parser) parseHashLiteral(sigil lexer.Token) ast.Expr {
	p.advance() // '('
	entries := p.parseExpressionListUntil(lexer.RightParen, false)
	close, closed := p.expectKind(lexer.RightParen, "')' after hash literal")
	end := sigil.Span
	if closed {
		end = close.Span
	} else if len(entries) != 0 {
		end = entries[len(entries)-1].Span()
	}
	return &ast.HashLiteralExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: joinSpans(sigil.Span, end)}},
		Entries:  entries,
	}
}

func (p *parser) finishCall(callee ast.Expr) ast.Expr {
	p.advance() // '('
	arguments, groups := p.parseCallArgumentList(lexer.RightParen)
	close, closed := p.expectKind(lexer.RightParen, "')' after arguments")
	end := callee.Span()
	if closed {
		end = close.Span
	} else if len(arguments) != 0 {
		end = arguments[len(arguments)-1].Span()
	}
	return &ast.CallExpr{
		ExprBase:  ast.ExprBase{Base: ast.Base{Range: joinSpans(callee.Span(), end)}},
		Callee:    callee,
		Args:      arguments,
		ArgGroups: groups,
	}
}

func (p *parser) finishIndex(target ast.Expr) ast.Expr {
	p.advance() // '['
	index := p.parseExpression(precedenceLowest)
	if index == nil {
		p.errorAt(p.current(), diagnosticExpectedExpression, "expected index expression")
	}
	close, closed := p.expectKind(lexer.RightBracket, "']' after index")
	end := target.Span()
	if closed {
		end = close.Span
	} else if index != nil {
		end = index.Span()
	}
	return &ast.IndexExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: joinSpans(target.Span(), end)}},
		Target:   target,
		Index:    index,
	}
}

func (p *parser) parseObject() ast.Expr {
	open := p.advance()
	if p.checkKind(lexer.RightBracket) {
		p.errorAt(p.current(), diagnosticInvalidObject, "object expression requires a target")
		close := p.advance()
		return &ast.ObjectExpr{ExprBase: ast.ExprBase{Base: ast.Base{Range: joinSpans(open.Span, close.Span)}}}
	}

	var target ast.Expr
	if p.isQualifiedIdentifierStart() {
		target = p.parseQualifiedIdentifier()
	} else {
		target = p.parseExpression(precedenceLowest)
	}
	if target == nil {
		p.errorAt(p.current(), diagnosticInvalidObject, "expected object target")
	}

	var message *ast.ObjectMessage
	if !p.checkKind(lexer.Colon) && !p.checkKind(lexer.RightBracket) && !p.atEnd() {
		commaSeparatedArguments := p.objectHasCommaBeforeSeparator()
		message = p.parseObjectMessage()
		if commaSeparatedArguments && message != nil {
			span := message.Span()
			if target != nil {
				span = joinSpans(target.Span(), span)
			}
			p.addDiagnostic(
				lexer.SeverityError,
				diagnosticInvalidObject,
				span,
				"Object Access: parameter separator is :",
			)
		}
	}

	var arguments []ast.Expr
	if p.matchKind(lexer.Colon) {
		if p.checkKind(lexer.RightBracket) {
			p.addDiagnostic(
				lexer.SeverityError,
				diagnosticInvalidObject,
				joinSpans(open.Span, p.current().Span),
				"Object Access: can not specify empty arg list after :",
			)
		} else {
			arguments = p.parseExpressionListUntil(lexer.RightBracket, false)
		}
	}
	close, closed := p.expectKind(lexer.RightBracket, "']' after object expression")
	end := open.Span
	if closed {
		end = close.Span
	} else if len(arguments) != 0 {
		end = arguments[len(arguments)-1].Span()
	} else if message != nil {
		end = message.Span()
	} else if target != nil {
		end = target.Span()
	}
	return &ast.ObjectExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: joinSpans(open.Span, end)}},
		Target:   target,
		Message:  message,
		Args:     arguments,
	}
}

func (p *parser) objectHasCommaBeforeSeparator() bool {
	for distance := 0; ; distance++ {
		token := p.peek(distance)
		switch token.Kind {
		case lexer.Comma:
			return true
		case lexer.Colon, lexer.RightBracket, lexer.EOF:
			return false
		}
	}
}

// validateParsedLiteralAlignments follows the reference CodeGenerator's
// parsed-literal walk. An alignment expression must be followed by a variable,
// and a present variable must not use an empty alignment expression. Sleep's
// variable boundary here is intentionally wider than source identifiers.
func (p *parser) validateParsedLiteralAlignments(token lexer.Token) {
	text := token.Text
	for index := 0; index < len(text); {
		if text[index] == '\\' {
			index++
			if index < len(text) {
				_, size := utf8.DecodeRuneInString(text[index:])
				index += size
			}
			continue
		}
		if text[index] != '$' || index+1 >= len(text) || parsedLiteralVariableEnd(text[index+1]) {
			_, size := utf8.DecodeRuneInString(text[index:])
			index += size
			continue
		}

		variableStart := index + 1
		if text[variableStart] == '[' {
			close := matchingParsedLiteralBracket(text, variableStart)
			if close < 0 {
				p.addDiagnostic(
					lexer.SeverityError,
					diagnosticInvalidParsedLiteral,
					parsedLiteralSpan(token, text, index, len(text)),
					"missing close brace for variable alignment",
				)
				return
			}

			variableStart = close + 1
			if variableStart >= len(text) || parsedLiteralVariableEnd(text[variableStart]) {
				p.addDiagnostic(
					lexer.SeverityError,
					diagnosticInvalidParsedLiteral,
					parsedLiteralPointSpan(token, text, close),
					"can not align an empty variable",
				)
				index = variableStart
				continue
			}

			variableEnd := variableStart
			for variableEnd < len(text) && !parsedLiteralVariableEnd(text[variableEnd]) {
				_, size := utf8.DecodeRuneInString(text[variableEnd:])
				variableEnd += size
			}
			if close == index+2 {
				_, lastSize := utf8.DecodeLastRuneInString(text[variableStart:variableEnd])
				last := variableEnd - lastSize
				p.addDiagnostic(
					lexer.SeverityError,
					diagnosticInvalidParsedLiteral,
					parsedLiteralPointSpan(token, text, last),
					"Empty alignment specification for $%s",
					text[variableStart:variableEnd],
				)
			}
			index = variableEnd
			continue
		}

		index = variableStart
		for index < len(text) && !parsedLiteralVariableEnd(text[index]) {
			_, size := utf8.DecodeRuneInString(text[index:])
			index += size
		}
	}
}

func parsedLiteralVariableEnd(value byte) bool {
	switch value {
	case ' ', '\t', '\n', '$', '\\':
		return true
	default:
		return false
	}
}

func matchingParsedLiteralBracket(text string, start int) int {
	depth := 0
	for index := start; index < len(text); index++ {
		switch text[index] {
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

func parsedLiteralPointSpan(token lexer.Token, text string, index int) lexer.Span {
	_, size := utf8.DecodeRuneInString(text[index:])
	return parsedLiteralSpan(token, text, index, index+size)
}

func parsedLiteralSpan(token lexer.Token, text string, start, end int) lexer.Span {
	return lexer.Span{
		Source: token.Span.Source,
		Start:  advanceParsedLiteralPosition(token.TextSpan.Start, text[:start]),
		End:    advanceParsedLiteralPosition(token.TextSpan.Start, text[:end]),
	}
}

func advanceParsedLiteralPosition(position lexer.Position, text string) lexer.Position {
	for index := 0; index < len(text); {
		if text[index] == '\r' && index+1 < len(text) && text[index+1] == '\n' {
			position.Offset += 2
			position.Line++
			position.Column = 1
			index += 2
			continue
		}
		r, size := utf8.DecodeRuneInString(text[index:])
		position.Offset += size
		if r == '\n' {
			position.Line++
			position.Column = 1
		} else {
			position.Column++
		}
		index += size
	}
	return position
}

func (p *parser) isQualifiedIdentifierStart() bool {
	if p.current().Kind != lexer.Identifier && p.current().Kind != lexer.Keyword {
		return false
	}
	dot := p.peek(1)
	next := p.peek(2)
	return dot.Lexeme == "." && p.current().Adjacent(dot) && dot.Adjacent(next) && (next.Kind == lexer.Identifier || next.Kind == lexer.Keyword)
}

func (p *parser) parseQualifiedIdentifier() ast.Expr {
	first := p.advance()
	var name strings.Builder
	name.WriteString(first.Lexeme)
	end := first.Span
	for p.checkLexeme(".") && p.previousOr(first).Adjacent(p.current()) {
		dot := p.current()
		next := p.peek(1)
		if !dot.Adjacent(next) || (next.Kind != lexer.Identifier && next.Kind != lexer.Keyword) {
			break
		}
		p.advance()
		part := p.advance()
		name.WriteString(dot.Lexeme)
		name.WriteString(part.Lexeme)
		end = part.Span
	}
	return &ast.IdentifierExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: joinSpans(first.Span, end)}},
		Name:     name.String(),
	}
}

func (p *parser) previousOr(fallback lexer.Token) lexer.Token {
	if p.position <= 0 {
		return fallback
	}
	return p.previous()
}

func (p *parser) parseObjectMessage() *ast.ObjectMessage {
	first := p.current()
	var name strings.Builder
	end := first.Span
	for !p.atEnd() && !p.checkKind(lexer.Colon) && !p.checkKind(lexer.RightBracket) {
		token := p.advance()
		if name.Len() != 0 && token.LeadingWhitespace {
			name.WriteByte(' ')
		}
		name.WriteString(token.Lexeme)
		end = token.Span
	}
	if name.Len() == 0 {
		return nil
	}
	return &ast.ObjectMessage{Range: joinSpans(first.Span, end), Name: name.String()}
}

func (p *parser) parseExpressionListUntil(terminator lexer.Kind, allowEmpty bool) []ast.Expr {
	values, _ := p.parseExpressionListUntilGrouped(terminator, allowEmpty, false)
	return values
}

func (p *parser) parseCallArgumentList(terminator lexer.Kind) ([]ast.Expr, []int) {
	return p.parseExpressionListUntilGrouped(terminator, false, true)
}

func (p *parser) parseExpressionListUntilGrouped(terminator lexer.Kind, allowEmpty, callArguments bool) ([]ast.Expr, []int) {
	values := make([]ast.Expr, 0)
	groups := make([]int, 0)
	groupLength := 0
	finishGroup := func() {
		if groupLength == 0 {
			return
		}
		if callArguments {
			start := len(values) - groupLength
			normalized := p.normalizeSleepParameterTerm(values[start:])
			values = append(values[:start], normalized...)
			groupLength = len(normalized)
		}
		groups = append(groups, groupLength)
		groupLength = 0
	}
	var pairBeforeOmittedComma *ast.PairExpr
	for !p.atEnd() && !p.checkKind(terminator) {
		if callArguments && p.matchKind(lexer.Comma) {
			finishGroup()
			continue
		}
		value := p.parseExpression(precedenceLowest)
		if value == nil {
			p.errorAt(p.current(), diagnosticExpectedExpression, "expected expression in list")
			if p.checkKind(terminator) || p.atEnd() {
				break
			}
			p.advance()
			continue
		}
		if pairBeforeOmittedComma != nil {
			if _, isPair := value.(*ast.PairExpr); isPair {
				span := joinSpans(pairBeforeOmittedComma.Value.Span(), value.Span())
				p.addDiagnostic(
					lexer.SeverityError,
					diagnosticMissingComma,
					span,
					"key/value pair specified for '%s', did you forget a comma?",
					sleepPairKey(pairBeforeOmittedComma.Key),
				)
			}
			pairBeforeOmittedComma = nil
		}
		omitEmptyCallIdea := false
		if callArguments {
			if tuple, ok := value.(*ast.TupleExpr); ok && len(tuple.Elements) == 0 {
				// An empty parenthesized idea pushes no scalar in Sleep. This is
				// observable in 1() and $value(), which are adjacent ideas rather
				// than calls on the preceding value.
				omitEmptyCallIdea = true
			}
		}
		if !omitEmptyCallIdea {
			values = append(values, value)
			groupLength++
		}

		if p.matchKind(lexer.Comma) {
			finishGroup()
			if p.checkKind(terminator) {
				break
			}
			continue
		}
		if p.checkKind(terminator) {
			break
		}

		if p.options.AllowOmittedCommas && canStartExpression(p.current()) {
			if p.options.ReportCompatibilityWarnings {
				p.warnAt(p.current(), diagnosticMissingComma, "accepted omitted comma")
			}
			pairBeforeOmittedComma, _ = value.(*ast.PairExpr)
			continue
		}
		p.errorAt(p.current(), diagnosticMissingComma, "expected ',' between expressions")
		if !canStartExpression(p.current()) {
			p.advance()
		}
	}
	finishGroup()
	if !allowEmpty && len(values) == 0 && !p.checkKind(terminator) {
		p.errorAt(p.current(), diagnosticExpectedExpression, "expected expression")
	}
	return values, groups
}

// normalizeSleepParameterTerm mirrors TokenParser.ParseIdea for the parameter
// text between two commas. Two adjacent ideas remain separate stack values.
// Once a third idea is present, however, Sleep interprets the second idea as
// an arbitrary operator and recursively parses everything after it as the RHS.
func (p *parser) normalizeSleepParameterTerm(ideas []ast.Expr) []ast.Expr {
	if len(ideas) < 2 {
		return ideas
	}

	// Pratt parsing has already folded a recognized source operator into the
	// first idea. If another idea follows, pull the outermost operator back out
	// so its RHS includes all remaining source ideas, as Sleep does.
	switch first := ideas[0].(type) {
	case *ast.BinaryExpr:
		right := make([]ast.Expr, 0, len(ideas))
		right = append(right, first.Right)
		right = append(right, ideas[1:]...)
		right = p.normalizeSleepParameterTerm(right)
		return []ast.Expr{&ast.ParameterOperatorExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: joinSpans(first.Left.Span(), ideas[len(ideas)-1].Span())}},
			Left:     first.Left,
			Op:       first.Op,
			Right:    right,
		}}
	case *ast.PairExpr:
		right := make([]ast.Expr, 0, len(ideas))
		right = append(right, first.Value)
		right = append(right, ideas[1:]...)
		right = p.normalizeSleepParameterTerm(right)
		copy := *first
		copy.ExprBase = ast.ExprBase{Base: ast.Base{Range: joinSpans(first.Key.Span(), ideas[len(ideas)-1].Span())}}
		copy.Value = p.sleepParameterTermValue(right)
		return []ast.Expr{&copy}
	}

	if len(ideas) == 2 {
		return ideas
	}
	right := p.normalizeSleepParameterTerm(ideas[2:])
	return []ast.Expr{&ast.ParameterOperatorExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: joinSpans(ideas[0].Span(), ideas[len(ideas)-1].Span())}},
		Left:     ideas[0],
		Op:       p.sleepRawExpression(ideas[1]),
		Right:    right,
	}}
}

func (p *parser) sleepParameterTermValue(ideas []ast.Expr) ast.Expr {
	if len(ideas) == 1 {
		return ideas[0]
	}
	return &ast.ParameterTermExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: joinSpans(ideas[0].Span(), ideas[len(ideas)-1].Span())}},
		Ideas:    ideas,
	}
}

// sleepMalformedNumericIdentifier recognizes one Sleep term which OPFOR's
// typed lexer deliberately splits: a number immediately followed by a Java
// identifier-like word or sigil term. Whitespace is significant; `1 ticks()`
// remains two accepted argument ideas, while `1ticks()` and `1$foo` are one
// unknown expression.
func (p *parser) sleepMalformedNumericIdentifier() (lexer.Span, bool) {
	first := p.current()
	numberOffset := 0
	number := first
	if (first.Lexeme == "+" || first.Lexeme == "-") && first.Adjacent(p.peek(1)) {
		numberOffset = 1
		number = p.peek(1)
	}
	if !sleepNumericTermToken(number) {
		return lexer.Span{}, false
	}

	tail := p.peek(numberOffset + 1)
	if number.Kind == lexer.Identifier && (tail.Kind == lexer.Array || tail.Kind == lexer.Hash) && tail.Text == "" {
		// A special literal followed by a collection constructor is parsed by
		// Sleep as a callable name ending in @/% rather than one numeric term.
		return lexer.Span{}, false
	}
	if !number.Adjacent(tail) || !sleepIdentifierWordToken(tail) {
		return lexer.Span{}, false
	}
	for index := 0; index <= numberOffset+1; index++ {
		p.advance()
	}
	return joinSpans(first.Span, tail.Span), true
}

func sleepNumericTermToken(token lexer.Token) bool {
	if token.Kind == lexer.Integer || token.Kind == lexer.Long || token.Kind == lexer.Double {
		return true
	}
	return token.Kind == lexer.Identifier && (token.Lexeme == "NaN" || token.Lexeme == "Infinity")
}

func sleepIdentifierWordToken(token lexer.Token) bool {
	if token.Kind == lexer.Identifier || token.Kind == lexer.Keyword {
		return true
	}
	switch token.Kind {
	case lexer.Scalar, lexer.Array, lexer.Hash, lexer.Function, lexer.Class, lexer.Reference:
		return true
	}
	if token.Kind != lexer.Operator || token.Lexeme == "" {
		return false
	}
	first, _ := utf8.DecodeRuneInString(token.Lexeme)
	return first == '_' || first >= 'A' && first <= 'Z' || first >= 'a' && first <= 'z'
}

// sleepMalformedDottedNumber recognizes lexer ambiguities preserved by the
// reference implementation. A leading dot is its own term, so .25 is unknown
// rather than a double. Once a term starts with a digit, a dot after a digit
// remains part of that term. Consequently 1.2 is a valid double, while 1.2.3
// is one unknown expression rather than 1.2 concatenated with 3. OPFOR's lexer
// intentionally emits ordinary number and dot tokens, so fold only these
// adjacent malformed shapes here and leave normal concatenation untouched.
func (p *parser) sleepMalformedDottedNumber() (lexer.Span, bool) {
	first := p.current()
	if first.Lexeme == "." {
		number := p.peek(1)
		if first.Adjacent(number) && (number.Kind == lexer.Integer || number.Kind == lexer.Long || number.Kind == lexer.Double) {
			p.advance()
			p.advance()
			return joinSpans(first.Span, number.Span), true
		}
		return lexer.Span{}, false
	}
	numberOffset := 0
	number := first
	if (first.Lexeme == "+" || first.Lexeme == "-") && first.Adjacent(p.peek(1)) {
		numberOffset = 1
		number = p.peek(1)
	}
	if number.Kind != lexer.Integer && number.Kind != lexer.Long && number.Kind != lexer.Double {
		return lexer.Span{}, false
	}
	firstRune, _ := utf8.DecodeRuneInString(number.Lexeme)
	lastRune, _ := utf8.DecodeLastRuneInString(number.Lexeme)
	if !lexer.IsJavaDecimalDigit(firstRune) || !lexer.IsJavaDecimalDigit(lastRune) {
		return lexer.Span{}, false
	}

	last := number
	consumed := numberOffset
	for {
		lastRune, _ = utf8.DecodeLastRuneInString(last.Lexeme)
		if !lexer.IsJavaDecimalDigit(lastRune) {
			break
		}
		dot := p.peek(consumed + 1)
		if dot.Lexeme != "." || !last.Adjacent(dot) {
			break
		}
		consumed++
		last = dot

		next := p.peek(consumed + 1)
		if !dot.Adjacent(next) || next.Kind != lexer.Integer && next.Kind != lexer.Long && next.Kind != lexer.Double {
			break
		}
		consumed++
		last = next
	}
	if consumed == numberOffset {
		return lexer.Span{}, false
	}

	for index := 0; index <= consumed; index++ {
		p.advance()
	}
	return joinSpans(first.Span, last.Span), true
}

func sleepPairKey(expression ast.Expr) string {
	switch node := expression.(type) {
	case *ast.VariableExpr:
		return node.Raw
	case *ast.IdentifierExpr:
		return node.Name
	case *ast.StringExpr:
		return node.Raw
	case *ast.NumberExpr:
		return node.Raw
	default:
		return ""
	}
}

func (p *parser) currentInfixOperator() (infixOperator, bool) {
	token := p.current()
	op := token.Lexeme
	if op == "!=" && token.Adjacent(p.peek(1)) && p.peek(1).Lexeme == "~" {
		return infixOperator{op: "!=~", precedence: precedencePredicate, tokenCount: 2}, true
	}
	if op == "!" && token.Adjacent(p.peek(1)) && (p.peek(1).Kind == lexer.Identifier || p.peek(1).Kind == lexer.Operator) {
		candidate := "!" + p.peek(1).Lexeme
		if _, known := knownWordOperators[strings.TrimPrefix(candidate, "!")]; known {
			return infixOperator{op: candidate, precedence: precedencePredicate, tokenCount: 2}, true
		}
	}
	if token.Kind == lexer.Operator && token.Adjacent(p.peek(1)) &&
		(p.peek(1).Kind == lexer.Identifier || p.peek(1).Kind == lexer.Integer) && p.peek(1).TrailingWhitespace {
		// Sleep permits arbitrary symbolic operator names. Its lexer treats a
		// spelling such as `*8` as one operator rather than multiplication by 8.
		return infixOperator{op: token.Lexeme + p.peek(1).Lexeme, precedence: precedenceProduct, tokenCount: 2}, true
	}

	if _, assignment := assignmentOperators[op]; assignment {
		return infixOperator{op: op, precedence: precedenceAssignment, rightAssociative: true, tokenCount: 1}, true
	}
	switch op {
	case "=>":
		return infixOperator{op: op, precedence: precedencePair, rightAssociative: true, tokenCount: 1}, true
	case "||":
		return infixOperator{op: op, precedence: precedenceOr, tokenCount: 1}, true
	case "&&":
		return infixOperator{op: op, precedence: precedenceAnd, tokenCount: 1}, true
	case "+", "-", ".":
		return infixOperator{op: op, precedence: precedenceAdditive, tokenCount: 1}, true
	case "++", "--", "!", "~":
		return infixOperator{}, false
	}
	if _, predicate := symbolicPredicates[op]; predicate {
		return infixOperator{op: op, precedence: precedencePredicate, tokenCount: 1}, true
	}
	if token.Kind != lexer.Identifier && token.Kind != lexer.Keyword && token.Kind != lexer.Operator {
		return infixOperator{}, false
	}

	word := strings.ToLower(token.Text)
	if word == "" {
		word = strings.ToLower(op)
	}
	if _, known := knownWordOperators[strings.TrimPrefix(word, "!")]; known {
		return infixOperator{op: op, precedence: precedencePredicate, tokenCount: 1}, true
	}
	if word == "cmp" || word == "x" {
		return infixOperator{op: op, precedence: precedenceProduct, tokenCount: 1}, true
	}
	if token.Kind == lexer.Operator {
		return infixOperator{op: op, precedence: precedenceProduct, tokenCount: 1}, true
	}
	if token.Kind == lexer.Identifier && token.LeadingWhitespace && token.TrailingWhitespace && p.peek(1).Kind != lexer.LeftParen && canStartExpression(p.peek(1)) {
		// Host bridges may install arbitrary word predicates. Requiring visible
		// whitespace avoids confusing a following function call with an infix
		// predicate in compatibility mode.
		return infixOperator{op: op, precedence: precedencePredicate, tokenCount: 1}, true
	}
	return infixOperator{}, false
}

func (p *parser) checkBinaryWhitespace(tokens []lexer.Token) {
	if len(tokens) == 0 {
		return
	}
	// The reference lexer has one deliberate exception to Sleep's operator
	// whitespace rule: concatenation may touch either operand.
	if len(tokens) == 1 && tokens[0].Lexeme == "." {
		return
	}
	first := tokens[0]
	last := tokens[len(tokens)-1]
	// The reference parser also accepts assignment with whitespace on only
	// one side (for example `$value ="text"`). It still rejects completely
	// adjacent forms such as `$value=1`, preserving the ambiguity diagnostic.
	if len(tokens) == 1 && isAssignmentOperator(tokens[0].Lexeme) &&
		(first.LeadingWhitespace || last.TrailingWhitespace) {
		return
	}
	if !first.LeadingWhitespace || !last.TrailingWhitespace {
		p.addDiagnostic(lexer.SeverityError, diagnosticOperatorWhitespace, joinSpans(first.Span, last.Span), "binary operator %q must be separated from both operands by whitespace", concatenateLexemes(tokens))
	}
}

func concatenateLexemes(tokens []lexer.Token) string {
	var result strings.Builder
	for _, token := range tokens {
		result.WriteString(token.Lexeme)
	}
	return result.String()
}

func isAssignmentOperator(operator string) bool {
	_, ok := assignmentOperators[operator]
	return ok
}

func isAssignable(expression ast.Expr) bool {
	switch node := expression.(type) {
	case *ast.VariableExpr, *ast.IndexExpr:
		return true
	case *ast.GroupExpr:
		return isAssignable(node.Expr)
	case *ast.TupleExpr:
		if len(node.Elements) < 2 {
			return false
		}
		for _, element := range node.Elements {
			if !isAssignable(element) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func canStartExpression(token lexer.Token) bool {
	switch token.Kind {
	case lexer.Identifier, lexer.Keyword, lexer.Scalar, lexer.Array, lexer.Hash, lexer.Function,
		lexer.Reference, lexer.Class, lexer.Integer, lexer.Long, lexer.Double,
		lexer.SingleString, lexer.DoubleString, lexer.BacktickString,
		lexer.LeftParen, lexer.LeftBrace, lexer.LeftBracket:
		return true
	case lexer.Operator:
		return token.Lexeme == "+" || token.Lexeme == "-" || token.Lexeme == "!" || token.Lexeme == "~" || strings.HasPrefix(token.Lexeme, "-") || strings.HasPrefix(token.Lexeme, "!-") || strings.EqualFold(token.Text, "not")
	default:
		return false
	}
}

func (p *parser) variableFromToken(token lexer.Token) *ast.VariableExpr {
	kind := ast.ScalarVariable
	if token.Kind == lexer.Array {
		kind = ast.ArrayVariable
	} else if token.Kind == lexer.Hash {
		kind = ast.HashVariable
	}
	return &ast.VariableExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: token.Span}},
		Kind:     kind,
		Name:     token.Text,
		Raw:      token.Lexeme,
	}
}

func (p *parser) referenceFromToken(token lexer.Token) ast.Expr {
	text := token.Text
	if text == "" {
		text = strings.TrimPrefix(token.Lexeme, "\\")
	}
	if text == "$null" {
		p.errorAt(token, diagnosticExpectedExpression, "Unknown expression")
	}
	var target ast.Expr
	if len(text) != 0 {
		switch text[0] {
		case '$', '@', '%':
			kind := ast.ScalarVariable
			if text[0] == '@' {
				kind = ast.ArrayVariable
			} else if text[0] == '%' {
				kind = ast.HashVariable
			}
			target = &ast.VariableExpr{
				ExprBase: ast.ExprBase{Base: ast.Base{Range: token.Span}},
				Kind:     kind,
				Name:     text[1:],
				Raw:      text,
			}
		case '&':
			target = &ast.FunctionRefExpr{
				ExprBase: ast.ExprBase{Base: ast.Base{Range: token.Span}},
				Name:     text[1:],
				Raw:      text,
			}
		}
	}
	if target == nil {
		target = &ast.IdentifierExpr{
			ExprBase: ast.ExprBase{Base: ast.Base{Range: token.Span}},
			Name:     text,
		}
	}
	return &ast.ReferenceExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: token.Span}},
		Target:   target,
	}
}
