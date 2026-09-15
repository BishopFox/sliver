// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package firestore

// BooleanExpression is an interface that represents a boolean expression in a pipeline.
type BooleanExpression interface {
	Expression // Embed Expr interface
	isBooleanExpr()

	// Conditional creates an expression that evaluates a condition and returns one of two expressions.
	//
	// The parameter 'thenVal' is the expression to return if the condition is true.
	// The parameter 'elseVal' is the expression to return if the condition is false.
	Conditional(thenVal, elseVal any) Expression
	// IfErrorBoolean creates a boolean expression that evaluates and returns the receiver expression if it does not produce an error;
	// otherwise, it evaluates and returns `catchExpr`.
	//
	// The parameter 'catchExpr' is the boolean expression to return if the receiver expression errors.
	IfErrorBoolean(catchExpr BooleanExpression) BooleanExpression
	// Not creates an expression that negates a boolean expression.
	Not() BooleanExpression
	// CountIf creates an aggregation that counts the number of stage inputs where the this boolean expression
	// evaluates to true.
	CountIf() AggregateFunction
}

// baseBooleanExpression provides common methods for all BooleanExpr implementations.
type baseBooleanExpression struct {
	*baseFunction // Embed Function to get Expr methods and toProto
}

func (b *baseBooleanExpression) isBooleanExpr() {}
func (b *baseBooleanExpression) Conditional(thenVal, elseVal any) Expression {
	return Conditional(b, thenVal, elseVal)
}
func (b *baseBooleanExpression) IfErrorBoolean(catchExpr BooleanExpression) BooleanExpression {
	return IfErrorBoolean(b, catchExpr)
}
func (b *baseBooleanExpression) Not() BooleanExpression {
	return Not(b)
}
func (b *baseBooleanExpression) CountIf() AggregateFunction {
	return CountIf(b)
}

// Ensure that baseBooleanExpr implements the BooleanExpr interface.
var _ BooleanExpression = (*baseBooleanExpression)(nil)

// ArrayContains creates an expression that checks if an array contains a specified element.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - value is the element to check for.
//
// Example:
//
//	// Check if the 'tags' array contains "Go".
//	ArrayContains("tags", "Go")
func ArrayContains(exprOrFieldPath any, value any) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunction("array_contains", []Expression{asFieldExpr(exprOrFieldPath), toExprOrConstant(value)})}
}

// ArrayContainsAll creates an expression that checks if an array contains all of the provided values.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - values can be an array of values or an expression that evaluates to an array.
//
// Example:
//
//	// Check if the 'tags' array contains both "Go" and "Firestore".
//	ArrayContainsAll("tags", []string{"Go", "Firestore"})
func ArrayContainsAll(exprOrFieldPath any, values any) BooleanExpression {
	return newFieldAndArrayBooleanExpr("array_contains_all", exprOrFieldPath, values)
}

// ArrayContainsAny creates an expression that checks if an array contains any of the provided values.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - values can be an array of values or an expression that evaluates to an array.
//
// Example:
//
//	// Check if the 'tags' array contains either "Go" or "Firestore".
//	ArrayContainsAny("tags", []string{"Go", "Firestore"})
func ArrayContainsAny(exprOrFieldPath any, values any) BooleanExpression {
	return newFieldAndArrayBooleanExpr("array_contains_any", exprOrFieldPath, values)
}

// EqualAny creates an expression that checks if a field or expression is equal to any of the provided values.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression].
// - values can be an array of values or an expression that evaluates to an array.
//
// Example:
//
//	// Check if the 'status' field is either "active" or "pending".
//	EqualAny("status", []string{"active", "pending"})
func EqualAny(exprOrFieldPath any, values any) BooleanExpression {
	return newFieldAndArrayBooleanExpr("equal_any", exprOrFieldPath, values)
}

// NotEqualAny creates an expression that checks if a field or expression is not equal to any of the provided values.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression].
// - values can be an array of values or an expression that evaluates to an array.
//
// Example:
//
//	// Check if the 'status' field is not "archived" or "deleted".
//	NotEqualAny("status", []string{"archived", "deleted"})
func NotEqualAny(exprOrFieldPath any, values any) BooleanExpression {
	return newFieldAndArrayBooleanExpr("not_equal_any", exprOrFieldPath, values)
}

// Equal creates an expression that checks if field's value or an expression is equal to an expression or a constant value,
// returning it as a BooleanExpr.
//   - left: The field path string, [FieldPath] or [Expression] to compare.
//   - right: The constant value or [Expression] to compare to.
//
// Example:
//
//		// Check if the 'age' field is equal to 21
//		Equal(FieldOf("age"), 21)
//
//		// Check if the 'age' field is equal to an expression
//	 	Equal(FieldOf("age"), FieldOf("minAge").Add(10))
//
//		// Check if the 'age' field is equal to the 'limit' field
//		Equal("age", FieldOf("limit"))
//
//		// Check if the 'city' field is equal to string constant "London"
//		Equal("city", "London")
func Equal(left, right any) BooleanExpression {
	return &baseBooleanExpression{baseFunction: leftRightToBaseFunction("equal", left, right)}
}

// NotEqual creates an expression that checks if field's value or an expression is not equal to an expression or a constant value,
// returning it as a BooleanExpr.
//   - left: The field path string, [FieldPath] or [Expression] to compare.
//   - right: The constant value or [Expression] to compare to.
//
// Example:
//
//		// Check if the 'age' field is not equal to 21
//		NotEqual(FieldOf("age"), 21)
//
//		// Check if the 'age' field is not equal to an expression
//	 	NotEqual(FieldOf("age"), FieldOf("minAge").Add(10))
//
//		// Check if the 'age' field is not equal to the 'limit' field
//		NotEqual("age", FieldOf("limit"))
//
//		// Check if the 'city' field is not equal to string constant "London"
//		NotEqual("city", "London")
func NotEqual(left, right any) BooleanExpression {
	return &baseBooleanExpression{baseFunction: leftRightToBaseFunction("not_equal", left, right)}
}

// GreaterThan creates an expression that checks if field's value or an expression is greater than an expression or a constant value,
// returning it as a BooleanExpr.
//   - left: The field path string, [FieldPath] or [Expression] to compare.
//   - right: The constant value or [Expression] to compare to.
//
// Example:
//
//		// Check if the 'age' field is greater than 21
//		GreaterThan(FieldOf("age"), 21)
//
//		// Check if the 'age' field is greater than an expression
//	 	GreaterThan(FieldOf("age"), FieldOf("minAge").Add(10))
//
//		// Check if the 'age' field is greater than the 'limit' field
//		GreaterThan("age", FieldOf("limit"))
func GreaterThan(left, right any) BooleanExpression {
	return &baseBooleanExpression{baseFunction: leftRightToBaseFunction("greater_than", left, right)}
}

// GreaterThanOrEqual creates an expression that checks if field's value or an expression is greater than or equal to an expression or a constant value,
// returning it as a BooleanExpr.
//   - left: The field path string, [FieldPath] or [Expression] to compare.
//   - right: The constant value or [Expression] to compare to.
//
// Example:
//
//		// Check if the 'age' field is greater than or equal to 21
//		GreaterThanOrEqual(FieldOf("age"), 21)
//
//		// Check if the 'age' field is greater than or equal to an expression
//	 	GreaterThanOrEqual(FieldOf("age"), FieldOf("minAge").Add(10))
//
//		// Check if the 'age' field is greater than or equal to the 'limit' field
//		GreaterThanOrEqual("age", FieldOf("limit"))
func GreaterThanOrEqual(left, right any) BooleanExpression {
	return &baseBooleanExpression{baseFunction: leftRightToBaseFunction("greater_than_or_equal", left, right)}
}

// LessThan creates an expression that checks if field's value or an expression is less than an expression or a constant value,
// returning it as a BooleanExpr.
//   - left: The field path string, [FieldPath] or [Expression] to compare.
//   - right: The constant value or [Expression] to compare to.
//
// Example:
//
//		// Check if the 'age' field is less than 21
//		LessThan(FieldOf("age"), 21)
//
//		// Check if the 'age' field is less than an expression
//	 	LessThan(FieldOf("age"), FieldOf("minAge").Add(10))
//
//		// Check if the 'age' field is less than the 'limit' field
//		LessThan("age", FieldOf("limit"))
func LessThan(left, right any) BooleanExpression {
	return &baseBooleanExpression{baseFunction: leftRightToBaseFunction("less_than", left, right)}
}

// LessThanOrEqual creates an expression that checks if field's value or an expression is less than or equal to an expression or a constant value,
// returning it as a BooleanExpr.
//   - left: The field path string, [FieldPath] or [Expression] to compare.
//   - right: The constant value or [Expression] to compare to.
//
// Example:
//
//		// Check if the 'age' field is less than or equal to 21
//		LessThanOrEqual(FieldOf("age"), 21)
//
//		// Check if the 'age' field is less than or equal to an expression
//	 	LessThanOrEqual(FieldOf("age"), FieldOf("minAge").Add(10))
//
//		// Check if the 'age' field is less than or equal to the 'limit' field
//		LessThanOrEqual("age", FieldOf("limit"))
func LessThanOrEqual(left, right any) BooleanExpression {
	return &baseBooleanExpression{baseFunction: leftRightToBaseFunction("less_than_or_equal", left, right)}
}

// EndsWith creates an expression that checks if a string field or expression ends with a given suffix.
// - exprOrFieldPath can be a field path string, [FieldPath] or [Expression].
// - suffix string or [Expression] to check for.
//
// Example:
//
//	// Check if the 'filename' field ends with ".go".
//	EndsWith("filename", ".go")
func EndsWith(exprOrFieldPath any, suffix any) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunction("ends_with", []Expression{asFieldExpr(exprOrFieldPath), asStringExpr(suffix)})}
}

// Like creates an expression that performs a case-sensitive wildcard string comparison.
// - exprOrFieldPath can be a field path string, [FieldPath] or [Expression].
// - pattern string or [Expression] to search for. You can use "%" as a wildcard character.
//
// Example:
//
//	// Check if the 'name' field starts with "G".
//	Like("name", "G%")
func Like(exprOrFieldPath any, pattern any) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunction("like", []Expression{asFieldExpr(exprOrFieldPath), asStringExpr(pattern)})}
}

// RegexContains creates an expression that checks if a string contains a match for a regular expression.
// - exprOrFieldPath can be a field path string, [FieldPath] or [Expression].
// - pattern is the regular expression to search for.
//
// Example:
//
//	// Check if the 'email' field contains a gmail address.
//	RegexContains("email", "@gmail\\.com$")
func RegexContains(exprOrFieldPath any, pattern any) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunction("regex_contains", []Expression{asFieldExpr(exprOrFieldPath), asStringExpr(pattern)})}
}

// RegexMatch creates an expression that checks if a string matches a regular expression.
// - exprOrFieldPath can be a field path string, [FieldPath] or [Expression].
// - pattern is the regular expression to match against.
//
// Example:
//
//	// Check if the 'zip_code' field is a 5-digit number.
//	RegexMatch("zip_code", "^[0-9]{5}$")
func RegexMatch(exprOrFieldPath any, pattern any) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunction("regex_match", []Expression{asFieldExpr(exprOrFieldPath), asStringExpr(pattern)})}
}

// StartsWith creates an expression that checks if a string field or expression starts with a given prefix.
// - exprOrFieldPath can be a field path string, [FieldPath] or [Expression].
// - prefix string or [Expression] to check for.
//
// Example:
//
//	// Check if the 'name' field starts with "Mr.".
//	StartsWith("name", "Mr.")
func StartsWith(exprOrFieldPath any, prefix any) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunction("starts_with", []Expression{asFieldExpr(exprOrFieldPath), asStringExpr(prefix)})}
}

// StringContains creates an expression that checks if a string contains a specified substring.
// - exprOrFieldPath can be a field path string, [FieldPath] or [Expression].
// - substring is the string to search for.
//
// Example:
//
//	// Check if the 'description' field contains the word "Firestore".
//	StringContains("description", "Firestore")
func StringContains(exprOrFieldPath any, substring any) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunction("string_contains", []Expression{asFieldExpr(exprOrFieldPath), asStringExpr(substring)})}
}

// And creates an expression that performs a logical 'AND' operation.
func And(condition BooleanExpression, right ...BooleanExpression) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunctionFromBooleans("and", append([]BooleanExpression{condition}, right...))}
}

// FieldExists creates an expression that checks if a field exists.
func FieldExists(exprOrField any) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunction("exists", []Expression{asFieldExpr(exprOrField)})}
}

// Not creates an expression that negates a boolean expression.
func Not(condition BooleanExpression) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunction("not", []Expression{condition})}
}

// Nor creates an expression that performs a logical 'NOR' operation.
func Nor(condition BooleanExpression, right ...BooleanExpression) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunctionFromBooleans("nor", append([]BooleanExpression{condition}, right...))}
}

// Or creates an expression that performs a logical 'OR' operation.
func Or(condition BooleanExpression, right ...BooleanExpression) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunctionFromBooleans("or", append([]BooleanExpression{condition}, right...))}
}

// Xor creates an expression that performs a logical 'XOR' operation.
func Xor(condition BooleanExpression, right ...BooleanExpression) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunctionFromBooleans("xor", append([]BooleanExpression{condition}, right...))}
}

// IsError creates an expression that checks if an expression evaluates to an error.
func IsError(expr Expression) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunction("is_error", []Expression{expr})}
}

// IsAbsent creates an expression that checks if an expression evaluates to an absent value.
func IsAbsent(exprOrField any) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunction("is_absent", []Expression{asFieldExpr(exprOrField)})}
}

// RawBooleanFunction creates a 'raw' boolean function expression.
// This is useful if the expression is available in the backend, but not yet in the current version of the SDK.
func RawBooleanFunction(name string, exprs ...any) BooleanExpression {
	expressions := make([]Expression, len(exprs))
	for i, expr := range exprs {
		expressions[i] = asFieldExpr(expr)
	}
	return &baseBooleanExpression{baseFunction: newBaseFunction(name, expressions)}
}

// IsType creates an expression that checks if an expression is of a specific type.
//   - exprOrField can be a field path string, [FieldPath] or an [Expression].
//   - dataType can be a valid type name. Valid values are
//     "null", "array", "boolean", "bytes", "timestamp", "geo_point", "number", "int32", "int64",
//     "float64", "decimal128", "map", "reference", "string", "vector", "max_key", "min_key",
//     "min_array", "object_id", "regex", "request_timestamp"
func IsType(exprOrField any, dataType string) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunction("is_type", []Expression{asFieldExpr(exprOrField), ConstantOf(dataType)})}
}
