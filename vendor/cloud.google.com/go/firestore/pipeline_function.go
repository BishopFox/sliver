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
	"google.golang.org/genproto/googleapis/type/latlng"
)

// FunctionExpression represents Firestore [Pipeline] functions, which can be evaluated within pipeline
// execution.
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
func Add(left, right any) Expression {
	return leftNumericRightToBaseFunction("add", left, right)
}

// Subtract creates an expression that subtracts the right expression from the left expression, returning it as an Expr.
// - left can be a field path string, [FieldPath] or [Expression].
// - right can be a numeric constant or a numeric [Expression].
func Subtract(left, right any) Expression {
	return leftNumericRightToBaseFunction("subtract", left, right)
}

// Multiply creates an expression that multiplies the left and right expressions, returning it as an Expr.
// - left can be a field path string, [FieldPath] or [Expression].
// - right can be a numeric constant or a numeric [Expression].
func Multiply(left, right any) Expression {
	return leftNumericRightToBaseFunction("multiply", left, right)
}

// Divide creates an expression that divides the left expression by the right expression, returning it as an Expr.
// - left can be a field path string, [FieldPath] or [Expression].
// - right can be a numeric constant or a numeric [Expression].
func Divide(left, right any) Expression {
	return leftNumericRightToBaseFunction("divide", left, right)
}

// Abs creates an expression that is the absolute value of the input field or expression.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
func Abs(numericExprOrFieldPath any) Expression {
	return newBaseFunction("abs", []Expression{asFieldExpr(numericExprOrFieldPath)})
}

// Floor creates an expression that is the largest integer that isn't less than the input field or expression.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
func Floor(numericExprOrFieldPath any) Expression {
	return newBaseFunction("floor", []Expression{asFieldExpr(numericExprOrFieldPath)})
}

// Ceil creates an expression that is the smallest integer that isn't less than the input field or expression.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
func Ceil(numericExprOrFieldPath any) Expression {
	return newBaseFunction("ceil", []Expression{asFieldExpr(numericExprOrFieldPath)})
}

// Exp creates an expression that is the Euler's number e raised to the power of the input field or expression.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
func Exp(numericExprOrFieldPath any) Expression {
	return newBaseFunction("exp", []Expression{asFieldExpr(numericExprOrFieldPath)})
}

// Log creates an expression that is logarithm of the left expression to base as the right expression, returning it as an Expr.
// - left can be a field path string, [FieldPath] or [Expression].
// - right can be a constant or an [Expression].
func Log(left, right any) Expression {
	return leftNumericRightToBaseFunction("log", left, right)
}

// Log10 creates an expression that is the base 10 logarithm of the input field or expression.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
func Log10(numericExprOrFieldPath any) Expression {
	return newBaseFunction("log10", []Expression{asFieldExpr(numericExprOrFieldPath)})
}

// Ln creates an expression that is the natural logarithm (base e) of the input field or expression.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
func Ln(numericExprOrFieldPath any) Expression {
	return newBaseFunction("ln", []Expression{asFieldExpr(numericExprOrFieldPath)})
}

// Mod creates an expression that computes the modulo of the left expression by the right expression, returning it as an Expr.
// - left can be a field path string, [FieldPath] or [Expression].
// - right can be a numeric constant or a numeric [Expression].
func Mod(left, right any) Expression {
	return leftNumericRightToBaseFunction("mod", left, right)
}

// Pow creates an expression that computes the left expression raised to the power of the right expression, returning it as an Expr.
// - left can be a field path string, [FieldPath] or [Expression].
// - right can be a numeric constant or a numeric [Expression].
func Pow(left, right any) Expression {
	return leftNumericRightToBaseFunction("pow", left, right)
}

// Round creates an expression that rounds the input field or expression to nearest integer.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
func Round(numericExprOrFieldPath any) Expression {
	return newBaseFunction("round", []Expression{asFieldExpr(numericExprOrFieldPath)})
}

// RoundToPrecision creates an expression that rounds a number to a specified number of decimal places.
// If places is positive, rounds off digits to the right of the decimal point.
// If places is negative, rounds off digits to the left of the decimal point.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
// - places can be an int, int32, int64 or [Expression].
func RoundToPrecision(numericExprOrFieldPath any, places any) Expression {
	return newBaseFunction("round", []Expression{asFieldExpr(numericExprOrFieldPath), asInt64Expr(places)})
}

// Trunc creates an expression that truncates a number to an integer.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
func Trunc(numericExprOrFieldPath any) Expression {
	return newBaseFunction("trunc", []Expression{asFieldExpr(numericExprOrFieldPath)})
}

// TruncToPrecision creates an expression that truncates a number to a specified number of decimal places.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
// - places can be an int, int32, int64 or [Expression].
func TruncToPrecision(numericExprOrFieldPath any, places any) Expression {
	return newBaseFunction("trunc", []Expression{asFieldExpr(numericExprOrFieldPath), asInt64Expr(places)})
}

// Sqrt creates an expression that is the square root of the input field or expression.
// - numericExprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that returns a number when evaluated.
func Sqrt(numericExprOrFieldPath any) Expression {
	return newBaseFunction("sqrt", []Expression{asFieldExpr(numericExprOrFieldPath)})
}

// Cmp creates an expression that compares the left and right expressions, returning it as an Expr.
// Returns -1 if left < right, 0 if left == right, and 1 if left > right.
// - left can be a field path string, [FieldPath] or [Expression].
// - right can be a constant or an [Expression].
func Cmp(left, right any) Expression {
	return leftRightToBaseFunction("cmp", left, right)
}

// TimestampAdd creates an expression that adds a specified amount of time to a timestamp.
// - timestamp can be a field path string, [FieldPath] or [Expression].
// - unit can be a string or an [Expression]. Valid units include "microsecond", "millisecond", "second", "minute", "hour" and "day".
// - amount can be an int, int32, int64 or [Expression].
func TimestampAdd(timestamp, unit, amount any) Expression {
	return newBaseFunction("timestamp_add", []Expression{asFieldExpr(timestamp), validateTimestampUnit(unit), asInt64Expr(amount)})
}

// TimestampSubtract creates an expression that subtracts a specified amount of time from a timestamp.
// - timestamp can be a field path string, [FieldPath] or [Expression].
// - unit can be a string or an [Expression]. Valid units include "microsecond", "millisecond", "second", "minute", "hour" and "day".
// - amount can be an int, int32, int64 or [Expression].
func TimestampSubtract(timestamp, unit, amount any) Expression {
	return newBaseFunction("timestamp_subtract", []Expression{asFieldExpr(timestamp), validateTimestampUnit(unit), asInt64Expr(amount)})
}

// TimestampExtract creates an expression that extracts a part from a timestamp.
//   - timestamp can be a field path string, [FieldPath] or [Expression].
//   - part can be a string or an [Expression]. Valid parts include "microsecond", "millisecond", "second", "minute", "hour", "day",
//     "dayofweek", "dayofyear", "week", "week(monday)", "week(tuesday)", "week(wednesday)", "week(thursday)",
//     "week(friday)", "week(saturday)", "week(sunday)", "month", "quarter", "year", "isoweek", and "isoyear".
func TimestampExtract(timestamp, part any) Expression {
	return newBaseFunction("timestamp_extract", []Expression{asFieldExpr(timestamp), validateTimestampPart(part)})
}

// TimestampExtractWithTimezone creates an expression that extracts a part from a timestamp in a given timezone.
//   - timestamp can be a field path string, [FieldPath] or [Expression].
//   - part can be a string or an [Expression]. Valid parts include "microsecond", "millisecond", "second", "minute", "hour", "day",
//     "dayofweek", "dayofyear", "week", "week(monday)", "week(tuesday)", "week(wednesday)", "week(thursday)",
//     "week(friday)", "week(saturday)", "week(sunday)", "month", "quarter", "year", "isoweek", and "isoyear".
//   - timezone can be a string or an [Expression].
func TimestampExtractWithTimezone(timestamp, part, timezone any) Expression {
	return newBaseFunction("timestamp_extract", []Expression{asFieldExpr(timestamp), validateTimestampPart(part), asStringExpr(timezone)})
}

// TimestampDiff creates an expression that calculates the difference between two timestamps.
// - end can be a field path string, [FieldPath] or [Expression].
// - start can be a field path string, [FieldPath] or [Expression].
// - unit can be a string or an [Expression]. Valid units include "microsecond", "millisecond", "second", "minute", "hour" and "day".
func TimestampDiff(end, start, unit any) Expression {
	return newBaseFunction("timestamp_diff", []Expression{asFieldExpr(end), asFieldExpr(start), validateTimestampUnit(unit)})
}

// TimestampTruncate creates an expression that truncates a timestamp to a specified granularity.
//   - timestamp can be a field path string, [FieldPath] or [Expression].
//   - granularity can be a string or an [Expression]. Valid values are "microsecond",
//     "millisecond", "second", "minute", "hour", "day", "week", "week(monday)", "week(tuesday)",
//     "week(wednesday)", "week(thursday)", "week(friday)", "week(saturday)", "week(sunday)",
//     "isoweek", "month", "quarter", "year", and "isoyear".
func TimestampTruncate(timestamp, granularity any) Expression {
	return newBaseFunction("timestamp_trunc", []Expression{asFieldExpr(timestamp), validateTimestampGranularity(granularity)})
}

// TimestampTruncateWithTimezone creates an expression that truncates a timestamp to a specified granularity in a given timezone.
//   - timestamp can be a field path string, [FieldPath] or [Expression].
//   - granularity can be a string or an [Expression]. Valid values are "microsecond",
//     "millisecond", "second", "minute", "hour", "day", "week", "week(monday)", "week(tuesday)",
//     "week(wednesday)", "week(thursday)", "week(friday)", "week(saturday)", "week(sunday)",
//     "isoweek", "month", "quarter", "year", and "isoyear".
//   - timezone can be a string or an [Expression]. Valid values are from the TZ database
//     (e.g., "America/Los_Angeles") or in the format "Etc/GMT-1".
func TimestampTruncateWithTimezone(timestamp, granularity any, timezone any) Expression {
	return newBaseFunction("timestamp_trunc", []Expression{asFieldExpr(timestamp), validateTimestampGranularity(granularity), asStringExpr(timezone)})
}

// TimestampToUnixMicros creates an expression that converts a timestamp expression to the number of microseconds since
// the Unix epoch (1970-01-01 00:00:00 UTC).
// - timestamp can be a field path string, [FieldPath] or [Expression].
func TimestampToUnixMicros(timestamp any) Expression {
	return newBaseFunction("timestamp_to_unix_micros", []Expression{asFieldExpr(timestamp)})
}

// TimestampToUnixMillis creates an expression that converts a timestamp expression to the number of milliseconds since
// the Unix epoch (1970-01-01 00:00:00 UTC).
// - timestamp can be a field path string, [FieldPath] or [Expression].
func TimestampToUnixMillis(timestamp any) Expression {
	return newBaseFunction("timestamp_to_unix_millis", []Expression{asFieldExpr(timestamp)})
}

// TimestampToUnixSeconds creates an expression that converts a timestamp expression to the number of seconds since
// the Unix epoch (1970-01-01 00:00:00 UTC).
// - timestamp can be a field path string, [FieldPath] or [Expression].
func TimestampToUnixSeconds(timestamp any) Expression {
	return newBaseFunction("timestamp_to_unix_seconds", []Expression{asFieldExpr(timestamp)})
}

// UnixMicrosToTimestamp creates an expression that converts a Unix timestamp in microseconds to a Firestore timestamp.
// - micros can be a field path string, [FieldPath] or [Expression].
func UnixMicrosToTimestamp(micros any) Expression {
	return newBaseFunction("unix_micros_to_timestamp", []Expression{asFieldExpr(micros)})
}

// UnixMillisToTimestamp creates an expression that converts a Unix timestamp in milliseconds to a Firestore timestamp.
// - millis can be a field path string, [FieldPath] or [Expression].
func UnixMillisToTimestamp(millis any) Expression {
	return newBaseFunction("unix_millis_to_timestamp", []Expression{asFieldExpr(millis)})
}

// UnixSecondsToTimestamp creates an expression that converts a Unix timestamp in seconds to a Firestore timestamp.
// - seconds can be a field path string, [FieldPath] or [Expression].
func UnixSecondsToTimestamp(seconds any) Expression {
	return newBaseFunction("unix_seconds_to_timestamp", []Expression{asFieldExpr(seconds)})
}

// CurrentTimestamp creates an expression that returns the current timestamp.
func CurrentTimestamp() Expression {
	return newBaseFunction("current_timestamp", []Expression{})
}

// CurrentDocument creates an expression that represents the current document being processed.
//
// This expression is useful when you need to access the entire document as a map, or pass the
// document itself to a function or subquery.
//
// Example:
//
//	// Define the current document as a variable "doc"
//	client.Pipeline().Collection("books").
//		Define(AliasedExpressions(CurrentDocument().As("doc"))).
//		// Access a field from the defined document variable
//		Select(Fields(GetField(Variable("doc"), "title")))
func CurrentDocument() Expression {
	return newBaseFunction("current_document", []Expression{})
}

// Variable creates an expression that retrieves the value of a variable bound via Define.
//
// Example:
//
//	// Define a variable "discountedPrice" and use it in a filter
//	client.Pipeline().Collection("products").
//		Define(AliasedExpressions(Multiply("price", 0.9).As("discountedPrice"))).
//		Where(LessThan(Variable("discountedPrice"), 100))
func Variable(name string) Expression {
	pbVal := &pb.Value{ValueType: &pb.Value_VariableReferenceValue{VariableReferenceValue: name}}
	return &baseExpression{pbVal: pbVal}
}

// ArrayLength creates an expression that calculates the length of an array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
func ArrayLength(exprOrFieldPath any) Expression {
	return newBaseFunction("array_length", []Expression{asFieldExpr(exprOrFieldPath)})
}

// Array creates an expression that represents a Firestore array.
// - elements can be any number of values or expressions that will form the elements of the array.
func Array(elements ...any) Expression {
	return newBaseFunction("array", toArrayOfExprOrConstant(elements))
}

// ArrayFromSlice creates a new array expression from a slice of elements.
// This function is necessary for creating an array from an existing typed slice (e.g., []int),
// as the [Array] function (which takes variadic arguments) cannot directly accept a typed slice
// using the spread operator (...). It handles the conversion of each element to `any` internally.
func ArrayFromSlice[T any](elements []T) Expression {
	return newBaseFunction("array", toExprsFromSlice(elements))
}

// ArrayGet creates an expression that retrieves an element from an array at a specified index.
//
// This is a typed function. It expects the first argument to be an array. If the
// target is not an array, the query will fail with a type error.
// If the target is null, the result is null.
//
// For a version that returns an absent value (UNSET) instead of failing on type
// mismatch, use [Offset].
//
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - offset is the 0-based index of the element to retrieve. Supports negative indexing.
func ArrayGet(exprOrFieldPath any, offset any) Expression {
	return newBaseFunction("array_get", []Expression{asFieldExpr(exprOrFieldPath), asInt64Expr(offset)})
}

// Offset creates an expression that accesses an element from an array at a specified index.
//
// This is a field access function. If the input is not an array, or if the index
// is out of bounds, it evaluates to an absent value.
//
//   - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
//   - index is the 0-based index of the element to retrieve. It can be an int or an [Expression].
//     Supports negative indexing (e.g., -1 returns the last element).
func Offset(exprOrFieldPath any, index any) Expression {
	return newBaseFunction("offset", []Expression{asFieldExpr(exprOrFieldPath), asInt64Expr(index)})
}

// ArrayReverse creates an expression that reverses the order of elements in an array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
func ArrayReverse(exprOrFieldPath any) Expression {
	return newBaseFunction("array_reverse", []Expression{asFieldExpr(exprOrFieldPath)})
}

// ArrayConcat creates an expression that concatenates multiple arrays into a single array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - otherArrays are the other arrays to concatenate.
func ArrayConcat(exprOrFieldPath any, otherArrays ...any) Expression {
	return newBaseFunction("array_concat", append([]Expression{asFieldExpr(exprOrFieldPath)}, toArrayOfExprOrConstant(otherArrays)...))
}

// ArraySum creates an expression that calculates the sum of all elements in a numeric array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a numeric array.
func ArraySum(exprOrFieldPath any) Expression {
	return newBaseFunction("sum", []Expression{asFieldExpr(exprOrFieldPath)})
}

// ArrayMaximum creates an expression that finds the maximum element in a numeric array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a numeric array.
func ArrayMaximum(exprOrFieldPath any) Expression {
	return newBaseFunction("maximum", []Expression{asFieldExpr(exprOrFieldPath)})
}

// ArrayMinimum creates an expression that finds the minimum element in a numeric array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a numeric array.
func ArrayMinimum(exprOrFieldPath any) Expression {
	return newBaseFunction("minimum", []Expression{asFieldExpr(exprOrFieldPath)})
}

// ArrayMaximumN creates an expression that finds the N maximum elements in an array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - n can be an int, int32, int64 or [Expression].
func ArrayMaximumN(exprOrFieldPath any, n any) Expression {
	return newBaseFunction("maximum_n", []Expression{asFieldExpr(exprOrFieldPath), asInt64Expr(n)})
}

// ArrayMinimumN creates an expression that finds the N minimum elements in an array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - n can be an int, int32, int64 or [Expression].
func ArrayMinimumN(exprOrFieldPath any, n any) Expression {
	return newBaseFunction("minimum_n", []Expression{asFieldExpr(exprOrFieldPath), asInt64Expr(n)})
}

// ArrayFirst creates an expression that returns the first element of an array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
func ArrayFirst(exprOrFieldPath any) Expression {
	return newBaseFunction("array_first", []Expression{asFieldExpr(exprOrFieldPath)})
}

// ArrayFirstN creates an expression that returns the first N elements of an array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - n can be an int, int32, int64 or [Expression].
func ArrayFirstN(exprOrFieldPath any, n any) Expression {
	return newBaseFunction("array_first_n", []Expression{asFieldExpr(exprOrFieldPath), asInt64Expr(n)})
}

// ArrayLast creates an expression that returns the last element of an array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
func ArrayLast(exprOrFieldPath any) Expression {
	return newBaseFunction("array_last", []Expression{asFieldExpr(exprOrFieldPath)})
}

// ArrayLastN creates an expression that returns the last N elements of an array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - n can be an int, int32, int64 or [Expression].
func ArrayLastN(exprOrFieldPath any, n any) Expression {
	return newBaseFunction("array_last_n", []Expression{asFieldExpr(exprOrFieldPath), asInt64Expr(n)})
}

// ArraySliceToEnd creates an expression that returns a slice of an array starting from the specified offset.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - offset is the 0-based index of the first element to include. It can be an int, int32, int64 or [Expression].
func ArraySliceToEnd(exprOrFieldPath any, offset any) Expression {
	return newBaseFunction("array_slice", []Expression{asFieldExpr(exprOrFieldPath), asInt64Expr(offset)})
}

// ArraySlice creates an expression that returns a slice of an array starting from the specified offset with a given length.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - offset is the 0-based index of the first element to include. It can be an int, int32, int64 or [Expression].
// - length is the number of elements to include. It can be an int, int32, int64 or [Expression].
func ArraySlice(exprOrFieldPath any, offset any, length any) Expression {
	return newBaseFunction("array_slice", []Expression{asFieldExpr(exprOrFieldPath), asInt64Expr(offset), asInt64Expr(length)})
}

// ReferenceSliceToEnd creates an expression that returns a subset of segments from a document reference.
// - exprOrField can be a field path string, [FieldPath] or an [Expression] that evaluates to a document reference.
// - offset is the 0-based index of the first segment to include. It can be an int, int32, int64 or [Expression].
func ReferenceSliceToEnd(exprOrFieldPath any, offset any) Expression {
	return newBaseFunction("reference_slice", []Expression{asFieldExpr(exprOrFieldPath), asInt64Expr(offset)})
}

// ReferenceSlice creates an expression that returns a subset of segments from a document reference with a given length.
// - exprOrField can be a field path string, [FieldPath] or an [Expression] that evaluates to a document reference.
// - offset is the 0-based index of the first segment to include. It can be an int, int32, int64 or [Expression].
// - length is the number of segments to include. It can be an int, int32, int64 or [Expression].
func ReferenceSlice(exprOrFieldPath any, offset any, length any) Expression {
	return newBaseFunction("reference_slice", []Expression{asFieldExpr(exprOrFieldPath), asInt64Expr(offset), asInt64Expr(length)})
}

// ArrayFilter creates an expression for array_filter(array, param, body).
// - array can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - param is the name of the parameter to use in the body expression.
// - body is the expression to evaluate for each element of the array.
func ArrayFilter(array any, param string, body BooleanExpression) Expression {
	return newBaseFunction("array_filter", []Expression{asFieldExpr(array), ConstantOf(param), body})
}

// ArrayTransform applies a transformation to each element of an array.
// - array can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - param is the name of the parameter to use in the transform expression.
// - body is the expression to evaluate for each element of the array.
func ArrayTransform(array any, param string, body Expression) Expression {
	return newBaseFunction("array_transform", []Expression{asFieldExpr(array), ConstantOf(param), body})
}

// ArrayTransformWithIndex applies a transformation to each element of an array, providing the index.
// - array can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - param is the name of the parameter to use in the transform expression for the element.
// - indexParam is the name of the parameter to use in the transform expression for the index.
// - body is the expression to evaluate for each element of the array.
func ArrayTransformWithIndex(array any, param, indexParam string, body Expression) Expression {
	return newBaseFunction("array_transform", []Expression{asFieldExpr(array), ConstantOf(param), ConstantOf(indexParam), body})
}

// ArrayIndexOf creates an expression that returns the first index of a search value in an array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - search is the value to search for. It can be a constant or [Expression].
func ArrayIndexOf(exprOrFieldPath any, search any) Expression {
	return newBaseFunction("array_index_of", []Expression{asFieldExpr(exprOrFieldPath), toExprOrConstant(search), toExprOrConstant("first")})
}

// ArrayLastIndexOf creates an expression that returns the last index of a search value in an array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - search is the value to search for. It can be a constant or [Expression].
func ArrayLastIndexOf(exprOrFieldPath any, search any) Expression {
	return newBaseFunction("array_index_of", []Expression{asFieldExpr(exprOrFieldPath), toExprOrConstant(search), toExprOrConstant("last")})
}

// ArrayIndexOfAll creates an expression that returns the indices of all occurrences of a search value in an array.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to an array.
// - search is the value to search for. It can be a constant or [Expression].
func ArrayIndexOfAll(exprOrFieldPath any, search any) Expression {
	return newBaseFunction("array_index_of_all", []Expression{asFieldExpr(exprOrFieldPath), toExprOrConstant(search)})
}

// StorageSize creates an expression that calculates the storage size of a field or [Expression] in bytes.
//   - exprOrFieldPath can be a field path string, [FieldPath] or [Expression].
func StorageSize(exprOrFieldPath any) Expression {
	return newBaseFunction("storage_size", []Expression{asFieldExpr(exprOrFieldPath)})
}

// ByteLength creates an expression that calculates the length of a string represented by a field or [Expression] in UTF-8
// bytes.
//   - exprOrFieldPath can be a field path string, [FieldPath] or [Expression].
func ByteLength(exprOrFieldPath any) Expression {
	return newBaseFunction("byte_length", []Expression{asFieldExpr(exprOrFieldPath)})
}

// CharLength creates an expression that calculates the character length of a string field or expression in UTF8.
//   - exprOrFieldPath can be a field path string, [FieldPath] or [Expression].
func CharLength(exprOrFieldPath any) Expression {
	return newBaseFunction("char_length", []Expression{asFieldExpr(exprOrFieldPath)})
}

// StringConcat creates an expression that concatenates multiple strings into a single string.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
// - otherStrings are optional additional string expressions or string constants to concatenate.
func StringConcat(exprOrFieldPath any, otherStrings ...any) Expression {
	args := make([]Expression, 1+len(otherStrings))
	args[0] = asFieldExpr(exprOrFieldPath)
	for i, v := range otherStrings {
		args[i+1] = asStringExpr(v)
	}
	return newBaseFunction("string_concat", args)
}

// StringRepeat creates an expression that repeats a string a specified number of times.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
// - repetition is the number of times to repeat the string. It can be an int, int32, int64 or [Expression].
func StringRepeat(exprOrFieldPath any, repetition any) Expression {
	return newBaseFunction("string_repeat", []Expression{asFieldExpr(exprOrFieldPath), asInt64Expr(repetition)})
}

// StringReplaceOne creates an expression that replaces the first occurrence of a search value with a replacement value.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
// - search is the value to search for. It can be a string or [Expression].
// - replacement is the value to replace with. It can be a string or [Expression].
func StringReplaceOne(exprOrFieldPath any, search, replacement any) Expression {
	return newBaseFunction("string_replace_one", []Expression{asFieldExpr(exprOrFieldPath), asStringExpr(search), asStringExpr(replacement)})
}

// StringReplaceAll creates an expression that replaces all occurrences of a search value with a replacement value.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
// - search is the value to search for. It can be a string or [Expression].
// - replacement is the value to replace with. It can be a string or [Expression].
func StringReplaceAll(exprOrFieldPath any, search, replacement any) Expression {
	return newBaseFunction("string_replace_all", []Expression{asFieldExpr(exprOrFieldPath), asStringExpr(search), asStringExpr(replacement)})
}

// StringReverse creates an expression that reverses a string.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
func StringReverse(exprOrFieldPath any) Expression {
	return newBaseFunction("string_reverse", []Expression{asFieldExpr(exprOrFieldPath)})
}

// StringIndexOf creates an expression that returns the index of a search value in a string.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
// - search is the value to search for. It can be a string or [Expression].
func StringIndexOf(exprOrFieldPath any, search any) Expression {
	return newBaseFunction("string_index_of", []Expression{asFieldExpr(exprOrFieldPath), asStringExpr(search)})
}

// Join creates an expression that joins the elements of a string array into a single string.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string array.
// - delimiter is the string to use as a separator between elements.
func Join(exprOrFieldPath any, delimiter any) Expression {
	return newBaseFunction("join", []Expression{asFieldExpr(exprOrFieldPath), asStringExpr(delimiter)})
}

// Substring creates an expression that returns a substring of a string.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
// - index is the starting index of the substring.
// - length is the length of the substring.
func Substring(exprOrFieldPath any, index any, length any) Expression {
	return newBaseFunction("substring", []Expression{asFieldExpr(exprOrFieldPath), asInt64Expr(index), asInt64Expr(length)})
}

// ToLower creates an expression that converts a string to lowercase.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
func ToLower(exprOrFieldPath any) Expression {
	return newBaseFunction("to_lower", []Expression{asFieldExpr(exprOrFieldPath)})
}

// ToUpper creates an expression that converts a string to uppercase.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
func ToUpper(exprOrFieldPath any) Expression {
	return newBaseFunction("to_upper", []Expression{asFieldExpr(exprOrFieldPath)})
}

// Trim creates an expression that removes leading and trailing whitespace from a string.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
func Trim(exprOrFieldPath any) Expression {
	return newBaseFunction("trim", []Expression{asFieldExpr(exprOrFieldPath)})
}

// TrimValue creates an expression that removes specified characters from the beginning and end of a string.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
// - charsOrExprToTrim is a string or [Expression] specifying characters to remove.
func TrimValue(exprOrFieldPath any, charsOrExprToTrim any) Expression {
	return newBaseFunction("trim", []Expression{asFieldExpr(exprOrFieldPath), asStringExpr(charsOrExprToTrim)})
}

// LTrim creates an expression that removes leading whitespace from a string.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
func LTrim(exprOrFieldPath any) Expression {
	return newBaseFunction("ltrim", []Expression{asFieldExpr(exprOrFieldPath)})
}

// LTrimValue creates an expression that removes leading whitespace or specified characters from a string.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
// - charsOrExprToTrim is a string or [Expression] specifying characters to remove.
func LTrimValue(exprOrFieldPath any, charsOrExprToTrim any) Expression {
	return newBaseFunction("ltrim", []Expression{asFieldExpr(exprOrFieldPath), asStringExpr(charsOrExprToTrim)})
}

// RTrim creates an expression that removes trailing whitespace from a string.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
func RTrim(exprOrFieldPath any) Expression {
	return newBaseFunction("rtrim", []Expression{asFieldExpr(exprOrFieldPath)})
}

// RTrimValue creates an expression that removes trailing whitespace or specified characters from a string.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
// - charsOrExprToTrim is a string or [Expression] specifying characters to remove.
func RTrimValue(exprOrFieldPath any, charsOrExprToTrim any) Expression {
	return newBaseFunction("rtrim", []Expression{asFieldExpr(exprOrFieldPath), asStringExpr(charsOrExprToTrim)})
}

// Split creates an expression that splits a string by a delimiter.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression] that evaluates to a string.
// - delimiter is the string to use to split by.
func Split(exprOrFieldPath any, delimiter any) Expression {
	return newBaseFunction("split", []Expression{asFieldExpr(exprOrFieldPath), asStringExpr(delimiter)})
}

// Type creates an expression that returns the type of the expression.
// - exprOrFieldPath can be a field path string, [FieldPath] or an [Expression].
func Type(exprOrFieldPath any) Expression {
	return newBaseFunction("type", []Expression{asFieldExpr(exprOrFieldPath)})
}

// CosineDistance creates an expression that calculates the cosine distance between two vectors.
//   - vector1 can be a field path string, [FieldPath] or [Expression].
//   - vector2 can be [Vector32], [Vector64], []float32, []float64 or [Expression].
func CosineDistance(vector1 any, vector2 any) Expression {
	return newBaseFunction("cosine_distance", []Expression{asFieldExpr(vector1), asVectorExpr(vector2)})
}

// DotProduct creates an expression that calculates the dot product of two vectors.
//   - vector1 can be a field path string, [FieldPath] or [Expression].
//   - vector2 can be [Vector32], [Vector64], []float32, []float64 or [Expression].
func DotProduct(vector1 any, vector2 any) Expression {
	return newBaseFunction("dot_product", []Expression{asFieldExpr(vector1), asVectorExpr(vector2)})
}

// EuclideanDistance creates an expression that calculates the euclidean distance between two vectors.
//   - vector1 can be a field path string, [FieldPath] or [Expression].
//   - vector2 can be [Vector32], [Vector64], []float32, []float64 or [Expression].
func EuclideanDistance(vector1 any, vector2 any) Expression {
	return newBaseFunction("euclidean_distance", []Expression{asFieldExpr(vector1), asVectorExpr(vector2)})
}

// VectorLength creates an expression that calculates the length of a vector.
//   - exprOrFieldPath can be a field path string, [FieldPath] or [Expression].
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
func Concat(exprOrField any, others ...any) Expression {
	return newBaseFunction("concat", append([]Expression{asFieldExpr(exprOrField)}, toArrayOfExprOrConstant(others)...))
}

// GetCollectionID creates an expression that returns the ID of the collection that contains the document.
// - exprOrField can be a field path string, [FieldPath] or an [Expression] that evaluates to a field path.
func GetCollectionID(exprOrField any) Expression {
	return newBaseFunction("collection_id", []Expression{asFieldExpr(exprOrField)})
}

// GetDocumentID creates an expression that returns the ID of the document.
// - exprStringOrDocRef can be a string, a [DocumentRef], or an [Expression] that evaluates to a document reference.
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

// GetParent creates an expression that returns the parent document of a document reference.
// - exprStringOrDocRef can be a string representation of the document path, a [DocumentRef], or an [Expression] that evaluates to a document path.
func GetParent(exprStringOrDocRef any) Expression {
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

	return newBaseFunction("parent", []Expression{expr})
}

// GetField creates an expression that accesses a field/property of a document field using the provided key.
// - exprOrField: The expression representing the document or map.
// - key: The key of the field to access.
func GetField(exprOrField any, key any) Expression {
	return newBaseFunction("get_field", []Expression{asFieldExpr(exprOrField), asStringExpr(key)})
}

// Conditional creates an expression that evaluates a condition and returns one of two expressions.
// - condition is the boolean expression to evaluate.
// - thenValOrExpr is the value or expression to return if the condition is true.
// - elseValOrExpr is the value or expression to return if the condition is false.
func Conditional(condition BooleanExpression, thenValOrExpr, elseValOrExpr any) Expression {
	return newBaseFunction("conditional", []Expression{condition, toExprOrConstant(thenValOrExpr), toExprOrConstant(elseValOrExpr)})
}

// LogicalMaximum creates an expression that evaluates to the maximum value in a list of expressions.
// - exprOrField can be a field path string, [FieldPath] or an [Expression].
// - others can be a list of constants or [Expression].
func LogicalMaximum(exprOrField any, others ...any) Expression {
	return newBaseFunction("maximum", append([]Expression{asFieldExpr(exprOrField)}, toArrayOfExprOrConstant(others)...))
}

// LogicalMinimum creates an expression that evaluates to the minimum value in a list of expressions.
// - exprOrField can be a field path string, [FieldPath] or an [Expression].
// - others can be a list of constants or [Expression].
func LogicalMinimum(exprOrField any, others ...any) Expression {
	return newBaseFunction("minimum", append([]Expression{asFieldExpr(exprOrField)}, toArrayOfExprOrConstant(others)...))
}

// IfError creates an expression that evaluates and returns `tryExpr` if it does not produce an error;
// otherwise, it evaluates and returns `catchExprOrValue`. It returns a new [Expression] representing
// the if_error operation.
// - tryExpr is the expression to try.
// - catchExprOrValue is the expression or value to return if `tryExpr` errors.
func IfError(tryExpr Expression, catchExprOrValue any) Expression {
	return newBaseFunction("if_error", []Expression{tryExpr, toExprOrConstant(catchExprOrValue)})
}

// IfErrorBoolean creates a boolean expression that evaluates and returns `tryExpr` if it does not produce an error;
// otherwise, it evaluates and returns `catchExpr`. It returns a new [BooleanExpression] representing
// the if_error operation.
// - tryExpr is the boolean expression to try.
// - catchExpr is the boolean expression to return if `tryExpr` errors.
func IfErrorBoolean(tryExpr BooleanExpression, catchExpr BooleanExpression) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunction("if_error", []Expression{tryExpr, catchExpr})}
}

// IfAbsent creates an expression that returns a default value if an expression evaluates to an absent value.
// - exprOrField is the field or expression to check. It can be a field path string, [FieldPath] or an [Expression].
// - elseValueOrExpr is the value or expression to return if the expression is absent. It can be a constant or an [Expression].
func IfAbsent(exprOrField any, elseValueOrExpr any) Expression {
	return newBaseFunction("if_absent", []Expression{asFieldExpr(exprOrField), toExprOrConstant(elseValueOrExpr)})
}

// IfNull creates an expression that returns a default value if an expression evaluates to null.
// Note: This function provides a fallback for both absent and explicit null values. In contrast,
// IfAbsent only triggers for missing fields.
// - exprOrField can be a field path string, [FieldPath] or an [Expression].
// - elseValueOrExpr is the default value or expression to return if the first evaluates to null.
func IfNull(exprOrField any, elseValueOrExpr any) Expression {
	return newBaseFunction("if_null", []Expression{asFieldExpr(exprOrField), toExprOrConstant(elseValueOrExpr)})
}

// Coalesce returns the first non-null, non-absent argument, without evaluating the rest of the arguments.
// When all arguments are null or absent, returns the last argument.
// - exprOrField can be a field path string, [FieldPath] or an [Expression].
// - replacement is the fallback expression or value if the first one evaluates to null or is absent.
// - others are optional additional expressions or values to check.
func Coalesce(exprOrField any, replacement any, others ...any) Expression {
	exprs := make([]Expression, 0, len(others)+2)
	exprs = append(exprs, asFieldExpr(exprOrField), toExprOrConstant(replacement))
	for _, v := range others {
		exprs = append(exprs, toExprOrConstant(v))
	}
	return newBaseFunction("coalesce", exprs)
}

// SwitchOn creates an expression that evaluates to the result corresponding to the first true condition.
//
// This function behaves like a `switch` statement. It accepts an alternating sequence of
// conditions and their corresponding results. If an odd number of arguments is provided, the
// final argument serves as a default fallback result. If no default is provided and no condition
// evaluates to true, it throws an error.
//
//   - condition: The first condition to check. Must be a [BooleanExpression].
//   - result: The result to return if the first condition is true. Can be an [Expression] or a literal value.
//   - others: Additional alternating conditions and results, optionally followed by a default fallback value.
//
// Example:
//
//	firestore.SwitchOn(
//	    firestore.FieldOf("score").GreaterThan(90), "A",
//	    firestore.FieldOf("score").GreaterThan(80), "B",
//	    "F", // Default result
//	)
func SwitchOn(condition BooleanExpression, result any, others ...any) Expression {
	exprs := make([]Expression, 0, len(others)+2)
	exprs = append(exprs, condition, toExprOrConstant(result))
	for _, v := range others {
		exprs = append(exprs, toExprOrConstant(v))
	}
	return newBaseFunction("switch_on", exprs)
}

// Map creates an expression that creates a Firestore map value from an input object.
// - elements: The input map to evaluate in the expression.
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
func MapGet(exprOrField any, strOrExprkey any) Expression {
	return newBaseFunction("map_get", []Expression{asFieldExpr(exprOrField), asStringExpr(strOrExprkey)})
}

// MapMerge creates an expression that merges multiple maps into a single map.
// If multiple maps have the same key, the later value is used.
// - exprOrField: First map field path string, [FieldPath] or an [Expression]
// - secondMap: Second map expression that will be merged.
// - otherMaps: Additional maps to merge.
func MapMerge(exprOrField any, secondMap Expression, otherMaps ...Expression) Expression {
	return newBaseFunction("map_merge", append([]Expression{asFieldExpr(exprOrField), secondMap}, otherMaps...))
}

// MapRemove creates an expression that removes a key from a map.
// - exprOrField: The expression representing the map.
// - strOrExprkey: The key to remove from the map.
func MapRemove(exprOrField any, strOrExprkey any) Expression {
	return newBaseFunction("map_remove", []Expression{asFieldExpr(exprOrField), asStringExpr(strOrExprkey)})
}

// MapSet creates an expression that updates a map with key-value pairs.
// - exprOrField: The expression representing the map.
// - key: The first key to set. It can be a string or an [Expression].
// - value: The first value to set. It can be a literal value or an [Expression].
// - moreKeysAndValues: Optional additional alternating key and value arguments.
func MapSet(exprOrField any, key any, value any, moreKeysAndValues ...any) Expression {
	exprs := make([]Expression, 0, len(moreKeysAndValues)+3)
	exprs = append(exprs, asFieldExpr(exprOrField), toExprOrConstant(key), toExprOrConstant(value))
	for _, v := range moreKeysAndValues {
		exprs = append(exprs, toExprOrConstant(v))
	}
	return newBaseFunction("map_set", exprs)
}

// MapKeys creates an expression that returns the keys of a map as an array.
// - exprOrField: The expression representing the map.
func MapKeys(exprOrField any) Expression {
	return newBaseFunction("map_keys", []Expression{asFieldExpr(exprOrField)})
}

// MapValues creates an expression that returns the values of a map as an array.
// - exprOrField: The expression representing the map.
func MapValues(exprOrField any) Expression {
	return newBaseFunction("map_values", []Expression{asFieldExpr(exprOrField)})
}

// MapEntries creates an expression that returns the entries of a map as an array of key-value maps.
// - exprOrField: The expression representing the map.
func MapEntries(exprOrField any) Expression {
	return newBaseFunction("map_entries", []Expression{asFieldExpr(exprOrField)})
}

// RegexFind creates an expression that returns the first substring that matches the specified regex pattern.
// - exprOrField: The expression representing the string to search.
// - pattern is the regular expression to search for. It can be a string or [Expression] that evaluates to a string.
func RegexFind(exprOrField any, pattern any) Expression {
	return newBaseFunction("regex_find", []Expression{asFieldExpr(exprOrField), asStringExpr(pattern)})
}

// RegexFindAll creates an expression that returns all substrings that match the specified regex pattern.
// - exprOrField: The expression representing the string to search.
// - pattern is the regular expression to search for. It can be a string or [Expression] that evaluates to a string.
//
// This expression uses the [RE2](https://github.com/google/re2/wiki/Syntax) regular expression syntax.
func RegexFindAll(exprOrField any, pattern any) Expression {
	return newBaseFunction("regex_find_all", []Expression{asFieldExpr(exprOrField), asStringExpr(pattern)})
}

// Rand returns a pseudo-random floating point number, chosen uniformly between 0.0 (inclusive) and 1.0 (exclusive).
func Rand() Expression {
	return newBaseFunction("rand", nil)
}

// DocumentMatches creates a boolean expression that performs a full-text search on all indexed search fields in the document.
//
// This Expression can only be used within a Search stage.
//
// Example:
//
//	client.Pipeline().Collection("restaurants").
//		Search(WithSearchQuery(DocumentMatches("waffles OR pancakes")))
//
// - query: Define the search query using the search domain-specific language (DSL).
//
// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func DocumentMatches(query string) BooleanExpression {
	return &baseBooleanExpression{baseFunction: newBaseFunction("document_matches", []Expression{ConstantOf(query)})}
}

// GeoDistance creates an expression that evaluates to the distance in meters between the location in the specified field and the query location.
//
// This Expression can only be used within a Search stage.
//
// Example:
//
//	client.Pipeline().Collection("restaurants").
//		Search(
//			WithSearchQuery("waffles"),
//			WithSearchSort(Ascending(GeoDistance("location", &latlng.LatLng{Latitude: 37.0, Longitude: -122.0}))),
//		)
//
// - field: Specifies the field in the document which contains the GeoPoint for distance computation. It can be a field path string, [FieldPath] or [Expression].
// - location: Compute distance to this GeoPoint.
//
// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func GeoDistance(field any, location *latlng.LatLng) Expression {
	return newBaseFunction("geo_distance", []Expression{asFieldExpr(field), ConstantOf(location)})
}

// Score creates an expression that evaluates to the search score that reflects the topicality of the document to all of the text
// predicates (for example: DocumentMatches) in the search query.
//
// This Expression can only be used within a Search stage.
//
// Example:
//
//	client.Pipeline().Collection("restaurants").
//		Search(WithSearchQuery("waffles"), WithSearchSort(Descending(Score())))
//
// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Score() Expression {
	return newBaseFunction("score", nil)
}
