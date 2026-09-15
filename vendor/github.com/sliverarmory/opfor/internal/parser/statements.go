package parser

import (
	"strings"

	"github.com/sliverarmory/opfor/internal/ast"
	"github.com/sliverarmory/opfor/internal/envspec"
	"github.com/sliverarmory/opfor/internal/lexer"
)

func (p *parser) parseStatement() ast.Stmt {
	switch {
	case p.checkWord("if"):
		return p.parseIf()
	case p.checkWord("while"):
		return p.parseWhile()
	case p.checkWord("for"):
		return p.parseFor()
	case p.checkWord("foreach"):
		return p.parseForeach()
	case p.checkWord("try"):
		return p.parseTry()
	case p.checkWord("import"):
		return p.parseImport()
	case p.checkWord("return"):
		return p.parseReturn()
	case p.checkWord("break"):
		return p.parseBareFlow("break")
	case p.checkWord("continue"):
		return p.parseBareFlow("continue")
	case p.checkWord("halt"):
		return p.parseBareFlow("halt")
	case p.checkWord("done"):
		return p.parseBareFlow("done")
	case p.checkWord("yield"):
		return p.parseYield()
	case p.checkWord("throw"):
		return p.parseThrow()
	case p.checkWord("callcc"):
		return p.parseCallCC()
	case p.checkWord("assert"):
		return p.parseAssert()
	case p.checkKind(lexer.LeftBrace):
		return p.parseBlock()
	case p.looksLikeEnvironment():
		return p.parseEnvironment()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *parser) parseBlock() *ast.BlockStmt {
	open, ok := p.expectKind(lexer.LeftBrace, "'{'")
	if !ok {
		return &ast.BlockStmt{StmtBase: ast.StmtBase{Base: ast.Base{Range: open.Span}}}
	}

	script := p.parseScript(true)
	close, closed := p.expectKind(lexer.RightBrace, "'}'")
	end := script.Span()
	if closed {
		end = close.Span
	}
	return &ast.BlockStmt{
		StmtBase:   ast.StmtBase{Base: ast.Base{Range: joinSpans(open.Span, end)}},
		Statements: script.Statements,
	}
}

func (p *parser) parseIf() ast.Stmt {
	start := p.advance()
	condition := p.parseParenthesizedExpression("if condition")
	then := p.requireBlock("if body")

	var alternative ast.Stmt
	end := then.Span()
	if p.matchWord("else") {
		if p.checkWord("if") {
			alternative = p.parseIf()
		} else if p.checkKind(lexer.LeftBrace) {
			alternative = p.parseBlock()
		} else {
			p.errorAt(p.current(), diagnosticExpectedToken, "expected 'if' or '{' after else")
		}
		if alternative != nil {
			end = alternative.Span()
		}
	}

	return &ast.IfStmt{
		StmtBase:  ast.StmtBase{Base: ast.Base{Range: joinSpans(start.Span, end)}},
		Condition: condition,
		Then:      then,
		Else:      alternative,
	}
}

func (p *parser) parseWhile() ast.Stmt {
	start := p.advance()
	var binding *ast.VariableExpr
	current := p.current()
	isBinding := current.Kind == lexer.Scalar ||
		(current.Text != "" && (current.Kind == lexer.Array || current.Kind == lexer.Hash))
	if isBinding && p.peek(1).Kind == lexer.LeftParen {
		binding = p.variableFromToken(p.advance())
	}
	condition := p.parseParenthesizedExpression("while condition")
	body := p.requireBlock("while body")
	return &ast.WhileStmt{
		StmtBase:  ast.StmtBase{Base: ast.Base{Range: joinSpans(start.Span, body.Span())}},
		Binding:   binding,
		Condition: condition,
		Body:      body,
	}
}

func (p *parser) parseFor() ast.Stmt {
	start := p.advance()
	p.expectKind(lexer.LeftParen, "'(' after for")

	init := p.parseExpressionListUntil(lexer.Semicolon, true)
	p.expectKind(lexer.Semicolon, "';' after for initializer")

	var condition ast.Expr
	if !p.checkKind(lexer.Semicolon) && !p.atEnd() {
		condition = p.parseExpression(precedenceLowest)
	}
	p.expectKind(lexer.Semicolon, "';' after for condition")

	post := p.parseExpressionListUntil(lexer.RightParen, true)
	p.expectKind(lexer.RightParen, "')' after for clauses")
	body := p.requireBlock("for body")

	return &ast.ForStmt{
		StmtBase:  ast.StmtBase{Base: ast.Base{Range: joinSpans(start.Span, body.Span())}},
		Init:      init,
		Condition: condition,
		Post:      post,
		Body:      body,
	}
}

func (p *parser) parseForeach() ast.Stmt {
	start := p.advance()
	var first ast.Expr
	if p.checkKind(lexer.Scalar) || (p.checkKind(lexer.Array) && p.current().Text != "") || (p.checkKind(lexer.Hash) && p.current().Text != "") {
		first = p.variableFromToken(p.advance())
	} else {
		first = p.parseExpression(precedencePostfix)
	}
	if first == nil {
		p.errorAt(p.current(), diagnosticInvalidControlFlow, "expected foreach variable")
	}

	var key ast.Expr
	value := first
	if p.matchLexeme("=>") {
		key = first
		if p.checkKind(lexer.Scalar) || (p.checkKind(lexer.Array) && p.current().Text != "") || (p.checkKind(lexer.Hash) && p.current().Text != "") {
			value = p.variableFromToken(p.advance())
		} else {
			value = p.parseExpression(precedencePostfix)
		}
		if value == nil {
			p.errorAt(p.current(), diagnosticInvalidControlFlow, "expected value variable after '=>'")
		}
	}

	iterable := p.parseParenthesizedExpression("foreach iterable")
	body := p.requireBlock("foreach body")

	return &ast.ForeachStmt{
		StmtBase: ast.StmtBase{Base: ast.Base{Range: joinSpans(start.Span, body.Span())}},
		Key:      key,
		Value:    value,
		Iterable: iterable,
		Body:     body,
	}
}

func (p *parser) parseTry() ast.Stmt {
	start := p.advance()
	body := p.requireBlock("try body")
	if !p.matchWord("catch") {
		p.errorAt(p.current(), diagnosticExpectedToken, "expected catch after try body")
		return &ast.TryStmt{
			StmtBase: ast.StmtBase{Base: ast.Base{Range: joinSpans(start.Span, body.Span())}},
			Body:     body,
		}
	}

	catchToken, ok := p.expectKind(lexer.Scalar, "scalar catch variable")
	var catchVariable *ast.VariableExpr
	if ok {
		catchVariable = p.variableFromToken(catchToken)
	}
	catchBody := p.requireBlock("catch body")
	return &ast.TryStmt{
		StmtBase: ast.StmtBase{Base: ast.Base{Range: joinSpans(start.Span, catchBody.Span())}},
		Body:     body,
		CatchVar: catchVariable,
		Catch:    catchBody,
	}
}

func (p *parser) parseImport() ast.Stmt {
	start := p.advance()
	var target strings.Builder
	var targetStart lexer.Span
	var targetEnd lexer.Span
	for !p.atEnd() && !p.checkKind(lexer.Semicolon) && !p.checkKind(lexer.RightBrace) && !p.checkWord("from") {
		token := p.current()
		if token.Span.Start.Line > start.Span.End.Line && p.options.AllowOmittedSemicolons && target.Len() != 0 {
			break
		}
		switch token.Kind {
		case lexer.Identifier, lexer.Keyword, lexer.Operator, lexer.Class:
			if target.Len() == 0 {
				targetStart = token.Span
			}
			target.WriteString(token.Lexeme)
			targetEnd = token.Span
			p.advance()
		default:
			p.errorAt(token, diagnosticInvalidImport, "invalid token in import target: %s", tokenDescription(token))
			p.advance()
		}
	}

	if target.Len() == 0 {
		p.errorAt(p.current(), diagnosticInvalidImport, "expected class or package after import")
		targetStart = lexer.Span{Source: start.Span.Source, Start: start.Span.End, End: start.Span.End}
		targetEnd = start.Span
	}
	terminatorSpan := joinSpans(targetStart, targetEnd)

	var from ast.Expr
	if p.matchWord("from") {
		p.expectKind(lexer.Colon, "':' after from")
		if p.checkKind(lexer.SingleString) || p.checkKind(lexer.DoubleString) {
			from = p.parseExpression(precedenceLowest)
		} else {
			pathStart := p.current()
			var path strings.Builder
			var pathEnd lexer.Span
			for !p.atEnd() && !p.checkKind(lexer.Semicolon) && !p.checkKind(lexer.RightBrace) {
				if path.Len() != 0 && p.current().Span.Start.Line > pathEnd.End.Line {
					break
				}
				token := p.advance()
				path.WriteString(token.Lexeme)
				pathEnd = token.Span
			}
			if path.Len() == 0 {
				p.errorAt(p.current(), diagnosticInvalidImport, "expected path after from:")
			} else {
				from = &ast.ImportPathExpr{
					ExprBase: ast.ExprBase{Base: ast.Base{Range: joinSpans(pathStart.Span, pathEnd)}},
					Raw:      path.String(),
				}
			}
		}
		if from != nil {
			targetEnd = from.Span()
			terminatorSpan = from.Span()
		}
	}

	end := p.finishExplicitStatement(start, terminatorSpan, targetEnd)
	return &ast.ImportStmt{
		StmtBase: ast.StmtBase{Base: ast.Base{Range: joinSpans(start.Span, end)}},
		Target:   target.String(),
		From:     from,
	}
}

func (p *parser) parseReturn() ast.Stmt {
	start := p.advance()
	value := p.parseOptionalFlowValue(start)
	end := start.Span
	terminatorSpan := lexer.Span{Source: start.Span.Source, Start: start.Span.End, End: start.Span.End}
	if value != nil {
		end = value.Span()
		terminatorSpan = value.Span()
	}
	end = p.finishExplicitStatement(start, terminatorSpan, end)
	return &ast.ReturnStmt{
		StmtBase: ast.StmtBase{Base: ast.Base{Range: joinSpans(start.Span, end)}},
		Value:    value,
	}
}

func (p *parser) parseYield() ast.Stmt {
	start := p.advance()
	value := p.parseOptionalFlowValue(start)
	end := start.Span
	if value != nil {
		end = value.Span()
	}
	end = p.finishSimpleStatement(end)
	return &ast.YieldStmt{
		StmtBase: ast.StmtBase{Base: ast.Base{Range: joinSpans(start.Span, end)}},
		Value:    value,
	}
}

func (p *parser) parseThrow() ast.Stmt {
	start := p.advance()
	value := p.parseOptionalFlowOperand()
	end := start.Span
	if value != nil {
		end = value.Span()
	}
	end = p.finishSimpleStatement(end)
	return &ast.ThrowStmt{
		StmtBase: ast.StmtBase{Base: ast.Base{Range: joinSpans(start.Span, end)}},
		Value:    value,
	}
}

func (p *parser) parseCallCC() ast.Stmt {
	start := p.advance()
	closure := p.parseOptionalFlowOperand()
	end := start.Span
	if closure != nil {
		end = closure.Span()
	}
	end = p.finishSimpleStatement(end)
	return &ast.CallCCStmt{
		StmtBase: ast.StmtBase{Base: ast.Base{Range: joinSpans(start.Span, end)}},
		Closure:  closure,
	}
}

func (p *parser) parseAssert() ast.Stmt {
	start := p.advance()
	condition := p.parseExpression(precedenceLowest)
	var message ast.Expr
	if p.matchKind(lexer.Colon) {
		message = p.parseExpression(precedenceLowest)
	}
	end := start.Span
	terminatorSpan := lexer.Span{Source: start.Span.Source, Start: start.Span.End, End: start.Span.End}
	if message != nil {
		end = message.Span()
		if condition != nil {
			terminatorSpan = joinSpans(condition.Span(), message.Span())
		} else {
			terminatorSpan = message.Span()
		}
	} else if condition != nil {
		end = condition.Span()
		terminatorSpan = condition.Span()
	}
	end = p.finishExplicitStatement(start, terminatorSpan, end)
	return &ast.AssertStmt{
		StmtBase:  ast.StmtBase{Base: ast.Base{Range: joinSpans(start.Span, end)}},
		Condition: condition,
		Message:   message,
	}
}

func (p *parser) parseBareFlow(kind string) ast.Stmt {
	start := p.advance()
	var value ast.Expr
	end := start.Span
	if kind == "break" || kind == "continue" {
		value = p.parseOptionalFlowValue(start)
		if value != nil {
			end = value.Span()
		}
	}
	end = p.finishSimpleStatement(end)
	base := ast.StmtBase{Base: ast.Base{Range: joinSpans(start.Span, end)}}
	switch kind {
	case "break":
		return &ast.BreakStmt{StmtBase: base, Value: value}
	case "continue":
		return &ast.ContinueStmt{StmtBase: base, Value: value}
	case "halt":
		return &ast.HaltStmt{StmtBase: base}
	default:
		return &ast.DoneStmt{StmtBase: base}
	}
}

func (p *parser) parseExpressionStatement() ast.Stmt {
	start := p.current()
	if !canStartExpression(start) {
		span := p.consumeInvalidStatement()
		p.addDiagnostic(lexer.SeverityError, diagnosticUnexpectedToken, span, "Syntax error")
		return nil
	}
	expression := p.parseExpression(precedenceLowest)
	if expression == nil {
		p.errorAt(start, diagnosticExpectedExpression, "expected expression, found %s", tokenDescription(start))
		return nil
	}
	if p.checkKind(lexer.LeftParen) {
		// Only syntactic function names consume a following parenthesized term.
		// In a value context 1() and $value() are adjacent ideas, but a block
		// statement cannot contain that multi-idea form.
		remainder := p.consumeInvalidStatement()
		p.addDiagnostic(lexer.SeverityError, diagnosticUnexpectedToken, joinSpans(expression.Span(), remainder), "Syntax error")
		return nil
	}
	if _, ok := expression.(*ast.AdjacentEmptyGroupExpr); ok {
		// A multi-idea value may supply an assignment/return/call argument, but
		// is not itself a valid block statement in the reference grammar.
		end := expression.Span()
		if p.matchKind(lexer.Semicolon) {
			end = p.previous().Span
		}
		p.addDiagnostic(lexer.SeverityError, diagnosticUnexpectedToken, joinSpans(expression.Span(), end), "Syntax error")
		return nil
	}
	expression = p.finishClosureAssignmentSequence(expression)
	end := p.finishSimpleStatement(expression.Span())
	return &ast.ExprStmt{
		StmtBase: ast.StmtBase{Base: ast.Base{Range: joinSpans(start.Span, end)}},
		Expr:     expression,
	}
}

// consumeInvalidStatement mirrors TokenParser's final syntax-error fallback:
// the complete token list through the next explicit terminator is blamed as
// one malformed statement. Consuming the list here also prevents the generic
// parser-progress guard from adding a secondary error for its first token.
func (p *parser) consumeInvalidStatement() lexer.Span {
	start := p.advance().Span
	end := start
	for !p.atEnd() && !p.checkKind(lexer.Semicolon) && !p.checkKind(lexer.RightBrace) {
		end = p.advance().Span
	}
	if p.matchKind(lexer.Semicolon) {
		// The reference EOT token delimits the snippet but is not displayed.
	}
	return joinSpans(start, end)
}

func (p *parser) finishClosureAssignmentSequence(expression ast.Expr) ast.Expr {
	if !p.options.AllowOmittedSemicolons {
		return expression
	}
	assignment, ok := expression.(*ast.AssignExpr)
	if !ok {
		return expression
	}
	if _, ok := assignment.Value.(*ast.ClosureExpr); !ok {
		return expression
	}
	if p.checkKind(lexer.Semicolon) || p.checkKind(lexer.RightBrace) || p.atEnd() || !canStartExpression(p.current()) {
		return expression
	}

	elements := []ast.Expr{assignment.Value}
	for !p.atEnd() && !p.checkKind(lexer.Semicolon) && !p.checkKind(lexer.RightBrace) && canStartExpression(p.current()) {
		next := p.parseExpression(precedenceLowest)
		if next == nil {
			break
		}
		elements = append(elements, next)
	}
	if len(elements) == 1 {
		return expression
	}
	sequence := &ast.SequenceExpr{
		ExprBase: ast.ExprBase{Base: ast.Base{Range: joinSpans(elements[0].Span(), elements[len(elements)-1].Span())}},
		Elements: elements,
	}
	assignment.Value = sequence
	assignment.ExprBase.Range = joinSpans(assignment.Target.Span(), sequence.Span())
	return assignment
}

func (p *parser) looksLikeEnvironment() bool {
	token := p.current()
	if token.Kind != lexer.Identifier && token.Kind != lexer.Keyword {
		return false
	}
	keyword := strings.ToLower(token.Text)
	if keyword == "" {
		keyword = strings.ToLower(token.Lexeme)
	}
	form, known := p.environmentForm(keyword)
	if known {
		if form == ast.PredicateEnvironment {
			return p.peek(1).Kind == lexer.LeftParen
		}
		return p.peek(1).Kind != lexer.LeftParen
	}
	if p.peek(1).Kind == lexer.LeftParen {
		return false
	}

	// Sleep environments are host-extensible. Recognize an unknown keyword
	// only when a selector is followed by a block before a terminator.
	for distance := 1; ; distance++ {
		next := p.peek(distance)
		switch next.Kind {
		case lexer.LeftBrace:
			return distance > 1
		case lexer.Semicolon, lexer.RightBrace, lexer.EOF:
			return false
		}
	}
}

func (p *parser) parseEnvironment() ast.Stmt {
	keyword := p.advance()
	form, _ := p.environmentForm(strings.ToLower(keyword.Lexeme))
	if form == ast.PredicateEnvironment {
		return p.parsePredicateEnvironment(keyword)
	}
	if form == ast.FilterEnvironment {
		return p.parseFilterEnvironment(keyword)
	}
	selectors := p.parseEnvironmentSelectors()
	if len(selectors) == 0 {
		p.errorAt(p.current(), diagnosticInvalidEnvironment, "environment %q requires a selector", keyword.Lexeme)
	}
	body := p.requireBlock("environment body")
	return &ast.EnvironmentStmt{
		StmtBase:  ast.StmtBase{Base: ast.Base{Range: joinSpans(keyword.Span, body.Span())}},
		Keyword:   keyword.Lexeme,
		Form:      ast.OrdinaryEnvironment,
		Selectors: selectors,
		Body:      body,
	}
}

func (p *parser) environmentForm(keyword string) (ast.EnvironmentForm, bool) {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if form, ok := p.options.Environments[keyword]; ok {
		return form, true
	}
	spec, ok := envspec.Lookup(keyword)
	if !ok {
		return ast.OrdinaryEnvironment, false
	}
	switch spec.Form {
	case envspec.Filter:
		return ast.FilterEnvironment, true
	case envspec.Predicate:
		return ast.PredicateEnvironment, true
	default:
		return ast.OrdinaryEnvironment, true
	}
}

func (p *parser) parsePredicateEnvironment(keyword lexer.Token) ast.Stmt {
	open := p.current()
	condition := p.parseParenthesizedExpression("predicate environment condition")
	end := open.Span
	if p.position > 0 {
		end = p.tokens[p.position-1].Span
	}
	raw := p.rawSource(joinSpans(open.Span, end))
	if raw == "" {
		raw = p.rawTokensBetween(open.Span, end)
	}
	body := p.requireBlock("predicate environment body")
	return &ast.EnvironmentStmt{
		StmtBase: ast.StmtBase{Base: ast.Base{Range: joinSpans(keyword.Span, body.Span())}},
		Keyword:  keyword.Lexeme,
		Form:     ast.PredicateEnvironment,
		Selectors: []ast.Selector{{
			Range: joinSpans(open.Span, end), Kind: ast.RawSelector, Value: raw, Raw: raw,
		}},
		Predicate: condition,
		Body:      body,
	}
}

func (p *parser) parseFilterEnvironment(keyword lexer.Token) ast.Stmt {
	selectors := make([]ast.Selector, 0, 2)
	if !p.atEnd() && !p.checkKind(lexer.LeftBrace) && !p.checkKind(lexer.Semicolon) {
		identifier := p.advance()
		selectors = append(selectors, ast.Selector{
			Range: identifier.Span, Kind: ast.RawSelector,
			Value: identifier.Lexeme, Raw: identifier.Lexeme,
		})
	}
	if !p.atEnd() && !p.checkKind(lexer.LeftBrace) && !p.checkKind(lexer.Semicolon) {
		start := p.current().Span
		end := start
		parenDepth, bracketDepth := 0, 0
		for !p.atEnd() && !p.checkKind(lexer.Semicolon) {
			if p.checkKind(lexer.LeftBrace) && parenDepth == 0 && bracketDepth == 0 {
				break
			}
			token := p.advance()
			end = token.Span
			switch token.Kind {
			case lexer.LeftParen:
				parenDepth++
			case lexer.RightParen:
				if parenDepth > 0 {
					parenDepth--
				}
			case lexer.LeftBracket:
				bracketDepth++
			case lexer.RightBracket:
				if bracketDepth > 0 {
					bracketDepth--
				}
			}
		}
		raw := p.rawSource(joinSpans(start, end))
		if raw == "" {
			raw = p.rawTokensBetween(start, end)
		}
		selectors = append(selectors, ast.Selector{
			Range: joinSpans(start, end), Kind: ast.RawSelector, Value: raw, Raw: raw,
		})
	}
	if len(selectors) < 2 {
		p.errorAt(p.current(), diagnosticInvalidEnvironment, "filter environment %q requires an identifier and parameter", keyword.Lexeme)
	}
	body := p.requireBlock("filter environment body")
	return &ast.EnvironmentStmt{
		StmtBase:  ast.StmtBase{Base: ast.Base{Range: joinSpans(keyword.Span, body.Span())}},
		Keyword:   keyword.Lexeme,
		Form:      ast.FilterEnvironment,
		Selectors: selectors,
		Body:      body,
	}
}

func (p *parser) rawSource(span lexer.Span) string {
	if len(p.source) == 0 || span.Start.Offset < 0 || span.End.Offset < span.Start.Offset || span.End.Offset > len(p.source) {
		return ""
	}
	return string(p.source[span.Start.Offset:span.End.Offset])
}

func (p *parser) rawTokensBetween(start, end lexer.Span) string {
	var result strings.Builder
	for _, token := range p.tokens {
		if token.Span.Start.Offset < start.Start.Offset || token.Span.End.Offset > end.End.Offset {
			continue
		}
		if result.Len() != 0 && token.LeadingWhitespace {
			result.WriteByte(' ')
		}
		result.WriteString(token.Lexeme)
	}
	return result.String()
}

func (p *parser) parseEnvironmentSelectors() []ast.Selector {
	selectors := make([]ast.Selector, 0, 2)
	for !p.atEnd() && !p.checkKind(lexer.LeftBrace) && !p.checkKind(lexer.Semicolon) {
		first := p.advance()
		selector := ast.Selector{
			Range: first.Span,
			Kind:  ast.IdentifierSelector,
			Value: first.Text,
			Raw:   first.Lexeme,
		}
		if selector.Value == "" {
			selector.Value = first.Lexeme
		}
		switch first.Kind {
		case lexer.SingleString, lexer.DoubleString, lexer.BacktickString:
			selector.Kind = ast.StringSelector
			kind := ast.SingleQuotedString
			if first.Kind == lexer.DoubleString {
				kind = ast.DoubleQuotedString
				p.validateParsedLiteralAlignments(first)
			} else if first.Kind == lexer.BacktickString {
				kind = ast.BacktickString
			}
			selector.Expr = &ast.StringExpr{
				ExprBase:  ast.ExprBase{Base: ast.Base{Range: first.Span}},
				Kind:      kind,
				Raw:       first.Lexeme,
				Text:      first.Text,
				TextRange: first.TextSpan,
			}
		case lexer.Operator:
			if first.Lexeme == "*" {
				selector.Kind = ast.WildcardSelector
			} else {
				selector.Kind = ast.RawSelector
			}
		case lexer.Identifier, lexer.Keyword:
		default:
			selector.Kind = ast.RawSelector
		}

		// Adjacent tokens form one selector, notably bind Ctrl+H and Java
		// qualified names. Whitespace starts another selector.
		for !p.atEnd() && !p.checkKind(lexer.LeftBrace) && !p.checkKind(lexer.Semicolon) && !p.current().LeadingWhitespace {
			next := p.advance()
			selector.Raw += next.Lexeme
			selector.Value += next.Lexeme
			selector.Range = joinSpans(selector.Range, next.Span)
			selector.Kind = ast.RawSelector
			selector.Expr = nil
		}
		selectors = append(selectors, selector)
	}
	return selectors
}

func (p *parser) requireBlock(description string) *ast.BlockStmt {
	if !p.checkKind(lexer.LeftBrace) {
		p.errorAt(p.current(), diagnosticExpectedToken, "expected '{' for %s", description)
		span := p.current().Span
		return &ast.BlockStmt{StmtBase: ast.StmtBase{Base: ast.Base{Range: span}}}
	}
	return p.parseBlock()
}

func (p *parser) parseParenthesizedExpression(description string) ast.Expr {
	p.expectKind(lexer.LeftParen, "'(' before "+description)
	expression := p.parseExpression(precedenceLowest)
	if expression == nil {
		p.errorAt(p.current(), diagnosticExpectedExpression, "expected %s", description)
	}
	p.expectKind(lexer.RightParen, "')' after "+description)
	return expression
}

func (p *parser) parseOptionalFlowValue(keyword lexer.Token) ast.Expr {
	if p.checkKind(lexer.Semicolon) || p.checkKind(lexer.RightBrace) || p.atEnd() {
		return nil
	}
	if p.options.AllowOmittedSemicolons && p.current().Span.Start.Line > keyword.Span.End.Line {
		return nil
	}
	return p.parseExpression(precedenceLowest)
}

func (p *parser) parseOptionalFlowOperand() ast.Expr {
	if p.checkKind(lexer.Semicolon) || p.checkKind(lexer.RightBrace) || p.atEnd() {
		return nil
	}
	return p.parseExpression(precedenceLowest)
}

func (p *parser) finishSimpleStatement(end lexer.Span) lexer.Span {
	if p.matchKind(lexer.Semicolon) {
		return p.previous().Span
	}

	boundary := p.atEnd() || p.checkKind(lexer.RightBrace) || p.current().Span.Start.Line > end.End.Line
	if p.options.AllowOmittedSemicolons && canStartExpression(p.current()) {
		// The reference parser also treats adjacent complete expressions as
		// separate statements, even on one line.
		boundary = true
	}
	if p.options.AllowOmittedSemicolons && boundary {
		if p.options.ReportCompatibilityWarnings {
			p.warnAt(p.current(), diagnosticExpectedTerminator, "accepted omitted semicolon")
		}
		return end
	}

	p.errorAt(p.current(), diagnosticExpectedTerminator, "expected ';' after statement")
	return end
}

// finishExplicitStatement preserves the reference parser's explicit EOT
// requirement for return, assert, and import statements. Unlike ordinary
// expressions, the Sleep token parser reports a missing terminator when one of
// these statements reaches the end of its enclosing block or input. Keep the
// diagnostic on the statement payload (or immediately after a bare keyword)
// so callers can reproduce the reference error line and snippet without
// blaming the closing brace on the following line.
func (p *parser) finishExplicitStatement(keyword lexer.Token, diagnosticSpan, end lexer.Span) lexer.Span {
	if p.matchKind(lexer.Semicolon) {
		return p.previous().Span
	}

	if diagnosticSpan.Source == "" {
		diagnosticSpan = lexer.Span{
			Source: keyword.Span.Source,
			Start:  keyword.Span.End,
			End:    keyword.Span.End,
		}
	}
	p.errorAt(lexer.Token{Span: diagnosticSpan}, diagnosticExpectedTerminator, "Missing terminator")
	return end
}
