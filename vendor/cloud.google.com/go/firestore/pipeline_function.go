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

import (
	"fmt"

	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
)

// FunctionExpression represents Firestore [Pipeline] functions, which can be evaluated within pipeline
// execution.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type FunctionExpression interface {
	Expression
	isFunction()
}

type baseFunction struct {
	*baseExpression
}

func (b *baseFunction) isFunction() {}

// Ensure that *baseFunction implements the FunctionExpression interface.
var _ FunctionExpression = (*baseFunction)(nil)

func newBaseFunction(name string, params []Expression) *baseFunction {
	argsPbVals := make([]*pb.Value, 0, len(params))
	for i, param := range params {
		paramExpr := asFieldExpr(param)
		pbVal, err := paramExpr.toProto()
		if err != nil {
			return &baseFunction{baseExpression: &baseExpression{err: fmt.Errorf("firestore: error converting arg %d for function %q: %w", i, name, err)}}
		}
		argsPbVals = append(argsPbVals, pbVal)
	}
	pbVal := &pb.Value{ValueType: &pb.Value_FunctionValue{
		FunctionValue: &pb.Function{
			Name: name,
			Args: argsPbVals,
		},
	}}

	return &baseFunction{baseExpression: &baseExpression{pbVal: pbVal}}
}

func newBaseFunctionFromBooleans(name string, params []BooleanExpression) *baseFunction {
	exprs := make([]Expression, len(params))
	for i, p := range params {
		exprs[i] = p
	}
	return newBaseFunction(name, exprs)
}

// Add creates an expression that adds two expressions together, returning it as an Expr.
// - left can be a field path string, [FieldPath] or [Expression].
// - right can be a numeric constant or a numeric [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Add(left, right any) Expression {
	return leftRightToBaseFunction("add", left, right)
}

// Subtract creates an expression that subtracts the right expression from the left expression, returning it as an Expr.
// - left can be a field path string, [FieldPath] or [Expression].
// - right can be a constant or an [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Subtract(left, right any) Expression {
	return leftRightToBaseFunction("subtract", left, right)
}

// Multiply creates an expression that multiplies the left and right expressions, returning it as an Expr.
// - left can be a field path string, [FieldPath] or [Expression].
// - right can be a constant or an [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Multiply(left, right any) Expression {
	return leftRightToBaseFunction("multiply", left, right)
}

// Divide creates an expression that divides the left expression by the right expression, returning it as an Expr.
// - left can be a field path string, [FieldPath] or [Expression].
// - right can be a constant or an [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Divide(left, right any) Expression {
	return leftRightToBaseFunction("divide", left, right)
}

// Abs creates an expression that is the absolute value of the input field or expression.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Abs(numericExprOrFieldPath any) Expression {
	return newBaseFunction("abs", []Expression{asFieldExpr(numericExprOrFieldPath)})
}

// Floor creates an expression that is the largest integer that isn't less than the input field or expression.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Floor(numericExprOrFieldPath any) Expression {
	return newBaseFunction("floor", []Expression{asFieldExpr(numericExprOrFieldPath)})
}

// Ceil creates an expression that is the smallest integer that isn't less than the input field or expression.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Ceil(numericExprOrFieldPath any) Expression {
	return newBaseFunction("ceil", []Expression{asFieldExpr(numericExprOrFieldPath)})
}

// Exp creates an expression that is the Euler's number e raised to the power of the input field or expression.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Exp(numericExprOrFieldPath any) Expression {
	return newBaseFunction("exp", []Expression{asFieldExpr(numericExprOrFieldPath)})
}

// Log creates an expression that is logarithm of the left expression to base as the right expression, returning it as an Expr.
// - left can be a field path string, [FieldPath] or [Expression].
// - right can be a constant or an [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Log(left, right any) Expression {
	return leftRightToBaseFunction("log", left, right)
}

// Log10 creates an expression that is the base 10 logarithm of the input field or expression.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Log10(numericExprOrFieldPath any) Expression {
	return newBaseFunction("log10", []Expression{asFieldExpr(numericExprOrFieldPath)})
}

// Ln creates an expression that is the natural logarithm (base e) of the input field or expression.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Ln(numericExprOrFieldPath any) Expression {
	return newBaseFunction("ln", []Expression{asFieldExpr(numericExprOrFieldPath)})
}

// Mod creates an expression that computes the modulo of the left expression by the right expression, returning it as an Expr.
// - left can be a field path string, [FieldPath] or [Expression].
// - right can be a constant or an [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Mod(left, right any) Expression {
	return leftRightToBaseFunction("mod", left, right)
}

// Pow creates an expression that computes the left expression raised to the power of the right expression, returning it as an Expr.
// - left can be a field path string, [FieldPath] or [Expression].
// - right can be a constant or an [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Pow(left, right any) Expression {
	return leftRightToBaseFunction("pow", left, right)
}

// Round creates an expression that rounds the input field or expression to nearest integer.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Round(numericExprOrFieldPath any) Expression {
	return newBaseFunction("round", []Expression{asFieldExpr(numericExprOrFieldPath)})
}

// Sqrt creates an expression that is the square root of the input field or expression.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Sqrt(numericExprOrFieldPath any) Expression {
	return newBaseFunction("sqrt", []Expression{asFieldExpr(numericExprOrFieldPath)})
}

// TimestampAdd creates an expression that adds a specified amount of time to a timestamp.
// - timestamp can be a field path string, [FieldPath] or [Expression].
// - unit can be a string or an [Expression]. Valid units include "microsecond", "millisecond", "second", "minute", "hour" and "day".
// - amount can be an int, int32, int64 or [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func TimestampAdd(timestamp, unit, amount any) Expression {
	return newBaseFunction("timestamp_add", []Expression{asFieldExpr(timestamp), asStringExpr(unit), asInt64Expr(amount)})
}

// TimestampSubtract creates an expression that subtracts a specified amount of time from a timestamp.
// - timestamp can be a field path string, [FieldPath] or [Expression].
// - unit can be a string or an [Expression]. Valid units include "microsecond", "millisecond", "second", "minute", "hour" and "day".
// - amount can be an int, int32, int64 or [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func TimestampSubtract(timestamp, unit, amount any) Expression {
	return newBaseFunction("timestamp_subtract", []Expression{asFieldExpr(timestamp), asStringExpr(unit), asInt64Expr(amount)})
}

// TimestampTruncate creates an expression that truncates a timestamp to a specified granularity.
//   - timestamp can be a field path string, [FieldPath] or [Expression].
//   - granularity can be a string or an [Expression]. Valid values are "microsecond",
//     "millisecond", "second", "minute", "hour", "day", "week", "week(monday)", "week(tuesday)",
//     "week(wednesday)", "week(thursday)", "week(friday)", "week(saturday)", "week(sunday)",
//     "isoweek", "month", "quarter", "year", and "isoyear".
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func TimestampTruncate(timestamp, granularity any) Expression {
	return newBaseFunction("timestamp_trunc", []Expression{asFieldExpr(timestamp), asStringExpr(granularity)})
}

// TimestampTruncateWithTimezone creates an expression that truncates a timestamp to a specified granularity in a given timezone.
//   - timestamp can be a field path string, [FieldPath] or [Expression].
//   - granularity can be a string or an [Expression]. Valid values are "microsecond",
//     "millisecond", "second", "minute", "hour", "day", "week", "week(monday)", "week(tuesday)",
//     "week(wednesday)", "week(thursday)", "week(friday)", "week(saturday)", "week(sunday)",
//     "isoweek", "month", "quarter", "year", and "isoyear".
//   - timezone can be a string or an [Expression]. Valid values are from the TZ database
//     (e.g., "America/Los_Angeles") or in the format "Etc/GMT-1".
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func TimestampTruncateWithTimezone(timestamp, granularity any, timezone string) Expression {
	return newBaseFunction("timestamp_trunc", []Expression{asFieldExpr(timestamp), asStringExpr(granularity), asStringExpr(timezone)})
}

// TimestampToUnixMicros creates an expression that converts a timestamp expression to the number of microseconds since
// the Unix epoch (1970-01-01 00:00:00 UTC).
// - timestamp can be a field path string, [FieldPath] or [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func TimestampToUnixMicros(timestamp any) Expression {
	return newBaseFunction("timestamp_to_unix_micros", []Expression{asFieldExpr(timestamp)})
}

// TimestampToUnixMillis creates an expression that converts a timestamp expression to the number of milliseconds since
// the Unix epoch (1970-01-01 00:00:00 UTC).
// - timestamp can be a field path string, [FieldPath] or [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func TimestampToUnixMillis(timestamp any) Expression {
	return newBaseFunction("timestamp_to_unix_millis", []Expression{asFieldExpr(timestamp)})
}

// TimestampToUnixSeconds creates an expression that converts a timestamp expression to the number of seconds since
// the Unix epoch (1970-01-01 00:00:00 UTC).
// - timestamp can be a field path string, [FieldPath] or [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func TimestampToUnixSeconds(timestamp any) Expression {
	return newBaseFunction("timestamp_to_unix_seconds", []Expression{asFieldExpr(timestamp)})
}

// UnixMicrosToTimestamp creates an expression that converts a Unix timestamp in microseconds to a Firestore timestamp.
// - micros can be a field path string, [FieldPath] or [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func UnixMicrosToTimestamp(micros any) Expression {
	return newBaseFunction("unix_micros_to_timestamp", []Expression{asFieldExpr(micros)})
}

// UnixMillisToTimestamp creates an expression that converts a Unix timestamp in milliseconds to a Firestore timestamp.
// - millis can be a field path string, [FieldPath] or [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func UnixMillisToTimestamp(millis any) Expression {
	return newBaseFunction("unix_millis_to_timestamp", []Expression{asFieldExpr(millis)})
}

// UnixSecondsToTimestamp creates an expression that converts a Unix timestamp in seconds to a Firestore timestamp.
// - seconds can be a field path string, [FieldPath] or [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func UnixSecondsToTimestamp(seconds any) Expression {
	return newBaseFunction("unix_seconds_to_timestamp", []Expression{asFieldExpr(seconds)})
}

// CurrentTimestamp creates an expression that returns the current timestamp.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func CurrentTimestamp() Expression {
	return newBaseFunction("current_timestamp", []Expression{})
}

// ArrayLength creates an expression that calculates the length of an array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func ArrayLength(exprOrFieldPath any) Expression {
	return newBaseFunction("array_length", []Expression{asFieldExpr(exprOrFieldPath)})
}

// Array creates an expression that represents a Firestore array.
// - elements can be any number of values or expressions that will form the elements of the array.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Array(elements ...any) Expression {
	return newBaseFunction("array", toExprs(elements))
}

// ArrayFromSlice creates a new array expression from a slice of elements.
// This function is necessary for creating an array from an existing typed slice (e.g., []int),
// as the [Array] function (which takes variadic arguments) cannot directly accept a typed slice
// using the spread operator (...). It handles the conversion of each element to `any` internally.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func ArrayFromSlice[T any](elements []T) Expression {
	return newBaseFunction("array", toExprsFromSlice(elements))
}

// ArrayGet creates an expression that retrieves an element from an array at a specified index.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - offset is the 0-based index of the element to retrieve.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func ArrayGet(exprOrFieldPath any, offset any) Expression {
	return newBaseFunction("array_get", []Expression{asFieldExpr(exprOrFieldPath), asInt64Expr(offset)})
}

// ArrayReverse creates an expression that reverses the order of elements in an array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func ArrayReverse(exprOrFieldPath any) Expression {
	return newBaseFunction("array_reverse", []Expression{asFieldExpr(exprOrFieldPath)})
}

// ArrayConcat creates an expression that concatenates multiple arrays into a single array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - otherArrays are the other arrays to concatenate.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func ArrayConcat(exprOrFieldPath any, otherArrays ...any) Expression {
	return newBaseFunction("array_concat", append([]Expression{asFieldExpr(exprOrFieldPath)}, toExprs(otherArrays)...))
}

// ArraySum creates an expression that calculates the sum of all elements in a numeric array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a numeric array.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func ArraySum(exprOrFieldPath any) Expression {
	return newBaseFunction("sum", []Expression{asFieldExpr(exprOrFieldPath)})
}

// ArrayMaximum creates an expression that finds the maximum element in a numeric array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a numeric array.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func ArrayMaximum(exprOrFieldPath any) Expression {
	return newBaseFunction("maximum", []Expression{asFieldExpr(exprOrFieldPath)})
}

// ArrayMinimum creates an expression that finds the minimum element in a numeric array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a numeric array.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func ArrayMinimum(exprOrFieldPath any) Expression {
	return newBaseFunction("minimum", []Expression{asFieldExpr(exprOrFieldPath)})
}

// ByteLength creates an expression that calculates the length of a string represented by a field or [Expression] in UTF-8
// bytes.
//   - exprOrFieldPath can be a field path string, [FieldPath] or [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func ByteLength(exprOrFieldPath any) Expression {
	return newBaseFunction("byte_length", []Expression{asFieldExpr(exprOrFieldPath)})
}

// CharLength creates an expression that calculates the character length of a string field or expression in UTF8.
//   - exprOrFieldPath can be a field path string, [FieldPath] or [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func CharLength(exprOrFieldPath any) Expression {
	return newBaseFunction("char_length", []Expression{asFieldExpr(exprOrFieldPath)})
}

// StringConcat creates an expression that concatenates multiple strings into a single string.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
// - otherStrings are the other strings to concatenate.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func StringConcat(exprOrFieldPath any, otherStrings ...any) Expression {
	return newBaseFunction("string_concat", append([]Expression{asFieldExpr(exprOrFieldPath)}, toExprs(otherStrings)...))
}

// StringReverse creates an expression that reverses a string.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func StringReverse(exprOrFieldPath any) Expression {
	return newBaseFunction("string_reverse", []Expression{asFieldExpr(exprOrFieldPath)})
}

// Join creates an expression that joins the elements of a string array into a single string.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string array.
// - delimiter is the string to use as a separator between elements.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Join(exprOrFieldPath any, delimiter any) Expression {
	return newBaseFunction("join", []Expression{asFieldExpr(exprOrFieldPath), asStringExpr(delimiter)})
}

// Substring creates an expression that returns a substring of a string.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
// - index is the starting index of the substring.
// - offset is the length of the substring.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Substring(exprOrFieldPath any, index any, offset any) Expression {
	return newBaseFunction("substring", []Expression{asFieldExpr(exprOrFieldPath), asInt64Expr(index), asInt64Expr(offset)})
}

// ToLower creates an expression that converts a string to lowercase.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func ToLower(exprOrFieldPath any) Expression {
	return newBaseFunction("to_lower", []Expression{asFieldExpr(exprOrFieldPath)})
}

// ToUpper creates an expression that converts a string to uppercase.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func ToUpper(exprOrFieldPath any) Expression {
	return newBaseFunction("to_upper", []Expression{asFieldExpr(exprOrFieldPath)})
}

// Trim creates an expression that removes leading and trailing whitespace from a string.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Trim(exprOrFieldPath any) Expression {
	return newBaseFunction("trim", []Expression{asFieldExpr(exprOrFieldPath)})
}

// Split creates an expression that splits a string by a delimiter.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
// - delimiter is the string to use to split by.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Split(exprOrFieldPath any, delimiter any) Expression {
	return newBaseFunction("split", []Expression{asFieldExpr(exprOrFieldPath), asStringExpr(delimiter)})
}

// Type creates an expression that returns the type of the expression.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Type(exprOrFieldPath any) Expression {
	return newBaseFunction("type", []Expression{asFieldExpr(exprOrFieldPath)})
}

// CosineDistance creates an expression that calculates the cosine distance between two vectors.
//   - vector1 can be a field path string, [FieldPath] or [Expression].
//   - vector2 can be [Vector32], [Vector64], []float32, []float64 or [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func CosineDistance(vector1 any, vector2 any) Expression {
	return newBaseFunction("cosine_distance", []Expression{asFieldExpr(vector1), asVectorExpr(vector2)})
}

// DotProduct creates an expression that calculates the dot product of two vectors.
//   - vector1 can be a field path string, [FieldPath] or [Expression].
//   - vector2 can be [Vector32], [Vector64], []float32, []float64 or [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func DotProduct(vector1 any, vector2 any) Expression {
	return newBaseFunction("dot_product", []Expression{asFieldExpr(vector1), asVectorExpr(vector2)})
}

// EuclideanDistance creates an expression that calculates the euclidean distance between two vectors.
//   - vector1 can be a field path string, [FieldPath] or [Expression].
//   - vector2 can be [Vector32], [Vector64], []float32, []float64 or [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func EuclideanDistance(vector1 any, vector2 any) Expression {
	return newBaseFunction("euclidean_distance", []Expression{asFieldExpr(vector1), asVectorExpr(vector2)})
}

// VectorLength creates an expression that calculates the length of a vector.
//   - exprOrFieldPath can be a field path string, [FieldPath] or [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func VectorLength(exprOrFieldPath any) Expression {
	return newBaseFunction("vector_length", []Expression{asFieldExpr(exprOrFieldPath)})
}

// Length creates an expression that calculates the length of string, array, map or vector.
// - exprOrField can be a field path string, [FieldPath] or an [Expression] that returns a string, array, map or vector when evaluated.
//
// Example:
//
//	// Length of the 'name' field.
//	Length("name")
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Length(exprOrField any) Expression {
	return newBaseFunction("length", []Expression{asFieldExpr(exprOrField)})
}

// Reverse creates an expression that reverses a string, or array.
// - exprOrField can be a field path string, [FieldPath] or an [Expression] that returns a string, or array when evaluated.
//
// Example:
//
//	// Reverse the 'name' field.
//
// Reverse("name")
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Reverse(exprOrField any) Expression {
	return newBaseFunction("reverse", []Expression{asFieldExpr(exprOrField)})
}

// Concat creates an expression that concatenates expressions together.
// - exprOrField can be a field path string, [FieldPath] or an [Expression].
// - others can be a list of constants or [Expression].
//
// Example:
//
//	// Concat the 'name' field with a constant string.
//	Concat("name", "-suffix")
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Concat(exprOrField any, others ...any) Expression {
	return newBaseFunction("concat", append([]Expression{asFieldExpr(exprOrField)}, toArrayOfExprOrConstant(others)...))
}

// GetCollectionID creates an expression that returns the ID of the collection that contains the document.
// - exprOrField can be a field path string, [FieldPath] or an [Expression] that evaluates to a field path.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func GetCollectionID(exprOrField any) Expression {
	return newBaseFunction("collection_id", []Expression{asFieldExpr(exprOrField)})
}

// GetDocumentID creates an expression that returns the ID of the document.
// - exprStringOrDocRef can be a string, a [DocumentRef], or an [Expression] that evaluates to a document reference.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func GetDocumentID(exprStringOrDocRef any) Expression {
	var expr Expression
	switch v := exprStringOrDocRef.(type) {
	case string:
		expr = ConstantOf(v)
	case *DocumentRef:
		expr = ConstantOf(v)
	case Expression:
		expr = v
	default:
		return &baseFunction{baseExpression: &baseExpression{err: fmt.Errorf("firestore: value must be a string, DocumentRef, or Expr, but got %T", exprStringOrDocRef)}}
	}

	return newBaseFunction("document_id", []Expression{expr})
}

// Conditional creates an expression that evaluates a condition and returns one of two expressions.
// - condition is the boolean expression to evaluate.
// - thenVal is the expression to return if the condition is true.
// - elseVal is the expression to return if the condition is false.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Conditional(condition BooleanExpression, thenVal, elseVal any) Expression {
	return newBaseFunction("conditional", []Expression{condition, toExprOrConstant(thenVal), toExprOrConstant(elseVal)})
}

// LogicalMaximum creates an expression that evaluates to the maximum value in a list of expressions.
// - exprOrField can be a field path string, [FieldPath] or an [Expression].
// - others can be a list of constants or [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func LogicalMaximum(exprOrField any, others ...any) Expression {
	return newBaseFunction("maximum", append([]Expression{asFieldExpr(exprOrField)}, toArrayOfExprOrConstant(others)...))
}

// LogicalMinimum creates an expression that evaluates to the minimum value in a list of expressions.
// - exprOrField can be a field path string, [FieldPath] or an [Expression].
// - others can be a list of constants or [Expression].
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func LogicalMinimum(exprOrField any, others ...any) Expression {
	return newBaseFunction("minimum", append([]Expression{asFieldExpr(exprOrField)}, toArrayOfExprOrConstant(others)...))
}

// IfError creates an expression that evaluates and returns `tryExpr` if it does not produce an error;
// otherwise, it evaluates and returns `catchExprOrValue`. It returns a new [Expression] representing
// the if_error operation.
// - tryExpr is the expression to try.
// - catchExprOrValue is the expression or value to return if `tryExpr` errors.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func IfError(tryExpr Expression, catchExprOrValue any) Expression {
	return newBaseFunction("if_error", []Expression{tryExpr, toExprOrConstant(catchExprOrValue)})
}

// IfErrorBoolean creates a boolean expression that evaluates and returns `tryExpr` if it does not produce an error;
// otherwise, it evaluates and returns `catchExpr`. It returns a new [BooleanExpression] representing
// the if_error operation.
// - tryExpr is the boolean expression to try.
// - catchExpr is the boolean expression to return if `tryExpr` errors.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func IfErrorBoolean(tryExpr BooleanExpression, catchExpr BooleanExpression) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunction("if_error", []Expression{tryExpr, catchExpr})}
}

// IfAbsent creates an expression that returns a default value if an expression evaluates to an absent value.
// - exprOrField can be a field path string, [FieldPath] or an [Expression].
// - elseValue is the value to return if the expression is absent.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func IfAbsent(exprOrField any, elseValue any) Expression {
	return newBaseFunction("if_absent", []Expression{asFieldExpr(exprOrField), toExprOrConstant(elseValue)})
}

// Map creates an expression that creates a Firestore map value from an input object.
// - elements: The input map to evaluate in the expression.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Map(elements map[string]any) Expression {
	exprs := make([]Expression, 0, len(elements)*2)
	for k, v := range elements {
		exprs = append(exprs, ConstantOf(k), toExprOrConstant(v))
	}
	return newBaseFunction("map", exprs)
}

// MapGet creates an expression that accesses a value from a map (object) field using the provided key.
// - exprOrField: The expression representing the map.
// - strOrExprkey: The key to access in the map.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func MapGet(exprOrField any, strOrExprkey any) Expression {
	return newBaseFunction("map_get", []Expression{asFieldExpr(exprOrField), asStringExpr(strOrExprkey)})
}

// MapMerge creates an expression that merges multiple maps into a single map.
// If multiple maps have the same key, the later value is used.
// - exprOrField: First map expression that will be merged.
// - secondMap: Second map expression that will be merged.
// - otherMaps: Additional maps to merge.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func MapMerge(exprOrField any, secondMap Expression, otherMaps ...Expression) Expression {
	return newBaseFunction("map_merge", append([]Expression{asFieldExpr(exprOrField), secondMap}, otherMaps...))
}

// MapRemove creates an expression that removes a key from a map.
// - exprOrField: The expression representing the map.
// - strOrExprkey: The key to remove from the map.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func MapRemove(exprOrField any, strOrExprkey any) Expression {
	return newBaseFunction("map_remove", []Expression{asFieldExpr(exprOrField), asStringExpr(strOrExprkey)})
}
