package ast

// Inspect traverses node depth-first. If visit returns false, Inspect skips
// that node's children. A nil callback is a no-op.
func Inspect(node Node, visit func(Node) bool) {
	if node == nil || visit == nil || !visit(node) {
		return
	}
	walkChildren(node, visit)
}

func inspectExprs(expressions []Expr, visit func(Node) bool) {
	for _, expression := range expressions {
		Inspect(expression, visit)
	}
}

func inspectStatements(statements []Stmt, visit func(Node) bool) {
	for _, statement := range statements {
		Inspect(statement, visit)
	}
}

func walkChildren(node Node, visit func(Node) bool) {
	switch node := node.(type) {
	case *Script:
		inspectStatements(node.Statements, visit)
	case *BlockStmt:
		inspectStatements(node.Statements, visit)
	case *ExprStmt:
		Inspect(node.Expr, visit)
	case *ReferenceExpr:
		Inspect(node.Target, visit)
	case *ClosureExpr:
		Inspect(node.Body, visit)
	case *GroupExpr:
		Inspect(node.Expr, visit)
	case *TupleExpr:
		inspectExprs(node.Elements, visit)
	case *ArrayLiteralExpr:
		inspectExprs(node.Elements, visit)
	case *HashLiteralExpr:
		inspectExprs(node.Entries, visit)
	case *PairExpr:
		Inspect(node.Key, visit)
		Inspect(node.Value, visit)
	case *UnaryExpr:
		Inspect(node.Operand, visit)
	case *BinaryExpr:
		Inspect(node.Left, visit)
		Inspect(node.Right, visit)
	case *AssignExpr:
		Inspect(node.Target, visit)
		Inspect(node.Value, visit)
	case *SequenceExpr:
		inspectExprs(node.Elements, visit)
	case *ParameterTermExpr:
		inspectExprs(node.Ideas, visit)
	case *ParameterOperatorExpr:
		Inspect(node.Left, visit)
		inspectExprs(node.Right, visit)
	case *AdjacentEmptyGroupExpr:
		Inspect(node.Value, visit)
	case *IndexExpr:
		Inspect(node.Target, visit)
		Inspect(node.Index, visit)
	case *CallExpr:
		Inspect(node.Callee, visit)
		inspectExprs(node.Args, visit)
	case *ObjectExpr:
		Inspect(node.Target, visit)
		inspectExprs(node.Args, visit)
	case *ImportStmt:
		Inspect(node.From, visit)
	case *IfStmt:
		Inspect(node.Condition, visit)
		Inspect(node.Then, visit)
		Inspect(node.Else, visit)
	case *WhileStmt:
		Inspect(node.Binding, visit)
		Inspect(node.Condition, visit)
		Inspect(node.Body, visit)
	case *ForStmt:
		inspectExprs(node.Init, visit)
		Inspect(node.Condition, visit)
		inspectExprs(node.Post, visit)
		Inspect(node.Body, visit)
	case *ForeachStmt:
		Inspect(node.Key, visit)
		Inspect(node.Value, visit)
		Inspect(node.Iterable, visit)
		Inspect(node.Body, visit)
	case *TryStmt:
		Inspect(node.Body, visit)
		Inspect(node.CatchVar, visit)
		Inspect(node.Catch, visit)
	case *ReturnStmt:
		Inspect(node.Value, visit)
	case *BreakStmt:
		Inspect(node.Value, visit)
	case *ContinueStmt:
		Inspect(node.Value, visit)
	case *YieldStmt:
		Inspect(node.Value, visit)
	case *ThrowStmt:
		Inspect(node.Value, visit)
	case *CallCCStmt:
		Inspect(node.Closure, visit)
	case *AssertStmt:
		Inspect(node.Condition, visit)
		Inspect(node.Message, visit)
	case *EnvironmentStmt:
		for _, selector := range node.Selectors {
			Inspect(selector.Expr, visit)
		}
		Inspect(node.Predicate, visit)
		Inspect(node.Body, visit)
	}
}
