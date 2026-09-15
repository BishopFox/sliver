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
	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
	"google.golang.org/genproto/googleapis/type/latlng"
)

// Selectable is an interface for expressions that can be selected in a pipeline.
type Selectable interface {
	// getSelectionDetails returns the output alias and the underlying expression.
	getSelectionDetails() (alias string, expr Expression)

	isSelectable()
}

// Expression represents an expression that can be evaluated to a value within the execution of a
// [Pipeline].
//
// Expressions are the building blocks for creating complex queries and transformations in
// Firestore pipelines. They can represent:
//
// - Field references: Access values from document fields.
// - Literals: Represent constant values (strings, numbers, booleans).
// - Function calls: Apply functions to one or more expressions.
// - Aggregations: Calculate aggregate values (e.g., sum, average) using [AggregateFunction] instances.
//
// The [Expression] interface provides a fluent API for building expressions. You can chain together
// method calls to create complex expressions.
type Expression interface {
	isExpr()
	toProto() (*pb.Value, error)
	getBaseExpr() *baseExpression

	// Aritmetic operations

	// Add creates an expression that adds two expressions together, returning it as an Expr.
	//
	// The parameter 'other' can be a numeric constant or a numeric [Expression].
	Add(other any) Expression
	// Subtract creates an expression that subtracts the right expression from the left expression, returning it as an Expr.
	//
	// The parameter 'other' can be a numeric constant or a numeric [Expression].
	Subtract(other any) Expression
	// Multiply creates an expression that multiplies the left and right expressions, returning it as an Expr.
	//
	// The parameter 'other' can be a numeric constant or a numeric [Expression].
	Multiply(other any) Expression
	// Divide creates an expression that divides the left expression by the right expression, returning it as an Expr.
	//
	// The parameter 'other' can be a numeric constant or a numeric [Expression].
	Divide(other any) Expression
	// Abs creates an expression that is the absolute value of the input field or expression.
	Abs() Expression
	// Floor creates an expression that is the largest integer that isn't less than the input field or expression.
	Floor() Expression
	// Ceil creates an expression that is the smallest integer that isn't less than the input field or expression.
	Ceil() Expression
	// Exp creates an expression that is the Euler's number e raised to the power of the input field or expression.
	Exp() Expression
	// Log creates an expression that is logarithm of the left expression to base as the right expression, returning it as an Expr.
	//
	// The parameter 'other' can be a numeric constant or a numeric [Expression].
	Log(other any) Expression
	// Log10 creates an expression that is the base 10 logarithm of the input field or expression.
	Log10() Expression
	// Ln creates an expression that is the natural logarithm (base e) of the input field or expression.
	Ln() Expression
	// Mod creates an expression that computes the modulo of the left expression by the right expression, returning it as an Expr.
	//
	// The parameter 'other' can be a numeric constant or a numeric [Expression].
	Mod(other any) Expression
	// Pow creates an expression that computes the left expression raised to the power of the right expression, returning it as an Expr.
	//
	// The parameter 'other' can be a numeric constant or a numeric [Expression].
	Pow(other any) Expression
	// Round creates an expression that rounds the input field or expression to nearest integer.
	Round() Expression
	// RoundToPrecision creates an expression that rounds the input field or expression to a specified number of decimal places.
	RoundToPrecision(places any) Expression
	// Trunc creates an expression that truncates a number to an integer.
	Trunc() Expression
	// TruncToPrecision creates an expression that truncates a number to a specified number of decimal places.
	//
	// The parameter 'places' is the number of decimal places to truncate to. It can be an int, int32, int64 or [Expression].
	TruncToPrecision(places any) Expression

	// Sqrt creates an expression that is the square root of the input field or expression.
	Sqrt() Expression
	// Cmp creates an expression that compares two expressions.
	// Returns -1 if left < right, 0 if left == right, and 1 if left > right.
	//
	// The parameter 'other' can be a constant or an [Expression].
	Cmp(other any) Expression

	// Array operations
	// ArrayContains creates a boolean expression that checks if an array contains a specific value.
	//
	// The parameter 'value' can be a constant (e.g., string, int, bool) or an [Expression].
	ArrayContains(value any) BooleanExpression
	// ArrayContainsAll creates a boolean expression that checks if an array contains all the specified values.
	//
	// The parameter 'values' can be a slice of constants (e.g., []string, []int) or an [Expression] that evaluates to an array.
	ArrayContainsAll(values any) BooleanExpression
	// ArrayContainsAny creates a boolean expression that checks if an array contains any of the specified values.
	//
	// The parameter 'values' can be a slice of constants (e.g., []string, []int) or an [Expression] that evaluates to an array.
	ArrayContainsAny(values any) BooleanExpression
	// ArrayLength creates an expression that calculates the length of an array.
	ArrayLength() Expression
	// EqualAny creates a boolean expression that checks if the expression is equal to any of the specified values.
	//
	// The parameter 'values' can be a slice of constants (e.g., []string, []int) or an [Expression] that evaluates to an array.
	EqualAny(values any) BooleanExpression
	// NotEqualAny creates a boolean expression that checks if the expression is not equal to any of the specified values.
	//
	// The parameter 'values' can be a slice of constants (e.g., []string, []int) or an [Expression] that evaluates to an array.
	NotEqualAny(values any) BooleanExpression
	// ArrayGet creates an expression that retrieves an element from an array at a specified index.
	//
	// The parameter 'offset' is the 0-based index of the element to retrieve.
	// It can be an integer constant or an [Expression] that evaluates to an integer.
	ArrayGet(offset any) Expression
	// Offset creates an expression that accesses an element from an array at a specified index.
	//
	// This is a field access function. If the input is not an array, or if the index
	// is out of bounds, it evaluates to an absent value.
	//
	// The parameter 'index' is the 0-based index of the element to retrieve. It can be an int or an [Expression].
	// Supports negative indexing (e.g., -1 returns the last element).
	Offset(index any) Expression
	// ArrayReverse creates an expression that reverses the order of elements in an array.
	ArrayReverse() Expression
	// ArrayConcat creates an expression that concatenates multiple arrays into a single array.
	//
	// The parameter 'otherArrays' can be a mix of array constants (e.g., []string, []int) or [Expression]s that evaluate to arrays.
	ArrayConcat(otherArrays ...any) Expression
	// ArraySum creates an expression that calculates the sum of all elements in a numeric array.
	ArraySum() Expression
	// ArrayMaximum creates an expression that finds the maximum element in a numeric array.
	ArrayMaximum() Expression
	// ArrayMaximumN creates an expression that finds the N maximum elements in an array.
	//
	// The parameter 'n' can be an integer constant or an [Expression] that evaluates to an integer.
	ArrayMaximumN(n any) Expression
	// ArrayMinimum creates an expression that finds the minimum element in a numeric array.
	ArrayMinimum() Expression
	// ArrayMinimumN creates an expression that finds the N minimum elements in an array.
	//
	// The parameter 'n' can be an integer constant or an [Expression] that evaluates to an integer.
	ArrayMinimumN(n any) Expression
	// ArrayFirst creates an expression that returns the first element of an array.
	ArrayFirst() Expression
	// ArrayFirstN creates an expression that returns the first N elements of an array.
	//
	// The parameter 'n' can be an integer constant or an [Expression] that evaluates to an integer.
	ArrayFirstN(n any) Expression
	// ArrayLast creates an expression that returns the last element of an array.
	ArrayLast() Expression
	// ArrayLastN creates an expression that returns the last N elements of an array.
	//
	// The parameter 'n' can be an integer constant or an [Expression] that evaluates to an integer.
	ArrayLastN(n any) Expression
	// ArraySliceToEnd creates an expression that returns a slice of an array starting from the specified offset.
	//
	// The parameter 'offset' is the 0-based index of the first element to include. It can be an int, int32, int64 or [Expression].
	ArraySliceToEnd(offset any) Expression
	// ArraySlice creates an expression that returns a slice of an array starting from the specified offset with a given length.
	//
	// The parameter 'offset' is the 0-based index of the first element to include. It can be an int, int32, int64 or [Expression].
	// The parameter 'length' is the number of elements to include. It can be an int, int32, int64 or [Expression].
	ArraySlice(offset, length any) Expression
	// ReferenceSliceToEnd creates an expression that returns a subset of segments from a document reference.
	//
	// The parameter 'offset' is the 0-based index of the first element to include. It can be an int, int32, int64 or [Expression].
	ReferenceSliceToEnd(offset any) Expression
	// ReferenceSlice creates an expression that returns a subset of segments from a document reference.
	//
	// The parameter 'offset' is the 0-based index of the first element to include. It can be an int, int32, int64 or [Expression].
	// The parameter 'length' is the number of elements to include. It can be an int, int32, int64 or [Expression].
	ReferenceSlice(offset, length any) Expression

	// ArrayIndexOf creates an expression that returns the first index of a search value in an array.
	//
	// The parameter 'search' is the value to search for. It can be a constant or [Expression].
	ArrayIndexOf(search any) Expression
	// ArrayIndexOfAll creates an expression that returns the indices of all occurrences of a search value in an array.
	//
	// The parameter 'search' is the value to search for. It can be a constant or [Expression].
	ArrayIndexOfAll(search any) Expression
	// ArrayLastIndexOf creates an expression that returns the last index of a search value in an array.
	//
	// The parameter 'search' is the value to search for. It can be a constant or [Expression].
	ArrayLastIndexOf(search any) Expression
	// First returns the value of the expression for the first document in the group.
	First() AggregateFunction
	// Last returns the value of the expression for the last document in the group.
	Last() AggregateFunction
	// ArrayAgg returns an array containing all values of the expression when evaluated on each document in the group.
	//
	// If the expression resolves to an absent value, it is converted to NULL.
	// The order of elements in the output array is not stable and shouldn't be relied upon.
	ArrayAgg() AggregateFunction
	// ArrayAggDistinct returns an array containing all distinct values of the expression when evaluated on each document in the group.
	//
	// If the expression resolves to an absent value, it is converted to NULL.
	// The order of elements in the output array is not stable and shouldn't be relied upon.
	ArrayAggDistinct() AggregateFunction
	// ArrayFilter creates an expression for array_filter(array, param, body).
	//
	// The parameter 'param' is the name of the parameter to use in the body expression.
	// The parameter 'body' is the expression to evaluate for each element of the array.
	ArrayFilter(param string, body BooleanExpression) Expression
	// ArrayTransform applies a transformation to each element of an array.
	//
	// The parameter 'param' is the name of the parameter to use in the transform expression.
	// The parameter 'body' is the expression to evaluate for each element of the array.
	ArrayTransform(param string, body Expression) Expression
	// ArrayTransformWithIndex applies a transformation to each element of an array, providing the index.
	//
	// The parameter 'param' is the name of the parameter to use in the transform expression for the element.
	// The parameter 'indexParam' is the name of the parameter to use in the transform expression for the index.
	// The parameter 'body' is the expression to evaluate for each element of the array.
	ArrayTransformWithIndex(param, indexParam string, body Expression) Expression
	// LogicalMaximum returns the maximum value of the expression and the specified values.
	LogicalMaximum(others ...any) Expression
	// LogicalMinimum returns the minimum value of the expression and the specified values.
	LogicalMinimum(others ...any) Expression

	// Timestamp operations
	// TimestampAdd creates an expression that adds a specified amount of time to a timestamp.
	//
	// The parameter 'unit' can be a string constant (e.g.,  "day") or an [Expression] that evaluates to a valid unit string.
	// Valid units include "microsecond", "millisecond", "second", "minute", "hour" and "day".
	// The parameter 'amount' can be an integer constant or an [Expression] that evaluates to an integer.
	TimestampAdd(unit, amount any) Expression
	// TimestampSubtract creates an expression that subtracts a specified amount of time from a timestamp.
	//
	// The parameter 'unit' can be a string constant (e.g.,  "hour") or an [Expression] that evaluates to a valid unit string.
	// Valid units include "microsecond", "millisecond", "second", "minute", "hour" and "day".
	// The parameter 'amount' can be an integer constant or an [Expression] that evaluates to an integer.
	TimestampSubtract(unit, amount any) Expression
	// TimestampTruncate creates an expression that truncates a timestamp to a specified granularity.
	//
	// The parameter 'granularity' can be a string constant (e.g.,  "month") or an [Expression] that evaluates to a valid granularity string.
	// Valid values are "microsecond", "millisecond", "second", "minute", "hour", "day", "week", "week(monday)", "week(tuesday)",
	// "week(wednesday)", "week(thursday)", "week(friday)", "week(saturday)", "week(sunday)", "isoweek", "month", "quarter", "year", and "isoyear".
	TimestampTruncate(granularity any) Expression
	// TimestampTruncateWithTimezone creates an expression that truncates a timestamp to a specified granularity in a given timezone.
	//
	// The parameter 'granularity' can be a string constant (e.g.,  "week") or an [Expression] that evaluates to a valid granularity string.
	// Valid values are "microsecond", "millisecond", "second", "minute", "hour", "day", "week", "week(monday)", "week(tuesday)",
	// "week(wednesday)", "week(thursday)", "week(friday)", "week(saturday)", "week(sunday)", "isoweek", "month", "quarter", "year", and "isoyear".
	// The parameter 'timezone' can be a string constant (e.g., "America/Los_Angeles") or an [Expression] that evaluates to a valid timezone string.
	// Valid values are from the TZ database or in the format "Etc/GMT-1".
	TimestampTruncateWithTimezone(granularity any, timezone any) Expression
	// TimestampToUnixMicros creates an expression that converts a timestamp expression to the number of microseconds since
	// the Unix epoch (1970-01-01 00:00:00 UTC).
	TimestampToUnixMicros() Expression
	// TimestampToUnixMillis creates an expression that converts a timestamp expression to the number of milliseconds since
	// the Unix epoch (1970-01-01 00:00:00 UTC).
	TimestampToUnixMillis() Expression
	// TimestampToUnixSeconds creates an expression that converts a timestamp expression to the number of seconds since
	// the Unix epoch (1970-01-01 00:00:00 UTC).
	TimestampToUnixSeconds() Expression
	// TimestampExtract creates an expression that extracts a part from a timestamp.
	// - part can be a string or an [Expression]. Valid parts include "microsecond", "millisecond", "second", "minute", "hour", "day",
	//   "dayofweek", "dayofyear", "week", "week(monday)", "week(tuesday)", "week(wednesday)", "week(thursday)",
	//   "week(friday)", "week(saturday)", "week(sunday)", "month", "quarter", "year", "isoweek", and "isoyear".
	TimestampExtract(part any) Expression
	// TimestampExtractWithTimezone creates an expression that extracts a part from a timestamp in a given timezone.
	// - timestamp can be a field path string, [FieldPath] or [Expression].
	// - part can be a string or an [Expression]. Valid parts include "microsecond", "millisecond", "second", "minute", "hour", "day",
	//   "dayofweek", "dayofyear", "week", "week(monday)", "week(tuesday)", "week(wednesday)", "week(thursday)",
	//   "week(friday)", "week(saturday)", "week(sunday)", "month", "quarter", "year", "isoweek", and "isoyear".
	// - timezone can be a string or an [Expression].
	TimestampExtractWithTimezone(part, timezone any) Expression
	// TimestampDiff creates an expression that calculates the difference between two timestamps.
	//
	// The parameter 'start' can be a field path string, [FieldPath] or [Expression].
	// The parameter 'unit' can be a string constant (e.g., "day") or an [Expression] that evaluates to a valid unit string.
	TimestampDiff(start, unit any) Expression
	// UnixMicrosToTimestamp creates an expression that converts a Unix timestamp in microseconds to a Firestore timestamp.
	UnixMicrosToTimestamp() Expression
	// UnixMillisToTimestamp creates an expression that converts a Unix timestamp in milliseconds to a Firestore timestamp.
	UnixMillisToTimestamp() Expression
	// UnixSecondsToTimestamp creates an expression that converts a Unix timestamp in seconds to a Firestore timestamp.
	UnixSecondsToTimestamp() Expression

	// Comparison operations
	// Equal creates a boolean expression that checks if the expression is equal to the other value.
	//
	// The parameter 'other' can be a constant (e.g., string, int, bool) or an [Expression].
	Equal(other any) BooleanExpression
	// NotEqual creates a boolean expression that checks if the expression is not equal to the other value.
	//
	// The parameter 'other' can be a constant (e.g., string, int, bool) or an [Expression].
	NotEqual(other any) BooleanExpression
	// GreaterThan creates a boolean expression that checks if the expression is greater than the other value.
	//
	// The parameter 'other' can be a constant (e.g., string, int, bool) or an [Expression].
	GreaterThan(other any) BooleanExpression
	// GreaterThanOrEqual creates a boolean expression that checks if the expression is greater than or equal to the other value.
	//
	// The parameter 'other' can be a constant (e.g., string, int, bool) or an [Expression].
	GreaterThanOrEqual(other any) BooleanExpression
	// LessThan creates a boolean expression that checks if the expression is less than the other value.
	//
	// The parameter 'other' can be a constant (e.g., string, int, bool) or an [Expression].
	LessThan(other any) BooleanExpression
	// LessThanOrEqual creates a boolean expression that checks if the expression is less than or equal to the other value.
	//
	// The parameter 'other' can be a constant (e.g., string, int, bool) or an [Expression].
	LessThanOrEqual(other any) BooleanExpression

	// General functions
	// Length creates an expression that calculates the length of string, array, map or vector.
	Length() Expression
	// Reverse creates an expression that reverses a string, or array.
	Reverse() Expression
	// Concat creates an expression that concatenates expressions together.
	//
	// The parameter 'others' can be a list of constants (e.g., string, int) or [Expression].
	Concat(others ...any) Expression

	// Key functions
	// GetCollectionID creates an expression that returns the ID of the collection that contains the document.
	GetCollectionID() Expression
	// GetDocumentID creates an expression that returns the ID of the document.
	GetDocumentID() Expression
	// GetParent creates an expression that returns the parent document of a document reference.
	GetParent() Expression
	// GetField creates an expression that accesses a field/property of a document field using the provided key.
	//
	// The parameter 'key' can be a string constant or an [Expression] that evaluates to a string.
	GetField(key any) Expression

	// Logical functions
	// IfError creates an expression that evaluates and returns the receiver expression if it does not produce an error;
	// otherwise, it evaluates and returns `catchExprOrValue`.
	//
	// The parameter 'catchExprOrValue' is the expression or value to return if the receiver expression errors.
	IfError(catchExprOrValue any) Expression
	// IsError returns a boolean expression that checks if the expression evaluates to an error.
	IsError() BooleanExpression
	// FieldExists returns a boolean expression that checks if the field exists.
	FieldExists() BooleanExpression
	// IsAbsent returns a boolean expression that checks if the field is absent.
	IsAbsent() BooleanExpression
	// IfAbsent creates an expression that returns a default value if an expression evaluates to an absent value.
	//
	// The parameter 'catchExprOrValue' is the value to return if the expression is absent.
	// It can be a constant or an [Expression].
	IfAbsent(catchExprOrValue any) Expression
	// IfNull creates an expression that returns a default value if an expression evaluates to null.
	//
	// The parameter 'elseValueOrExpr' can be a constant or [Expression].
	IfNull(elseValueOrExpr any) Expression
	// Coalesce returns the first non-null, non-absent argument, without evaluating the rest of the arguments.
	// When all arguments are null or absent, returns the last argument.
	//
	// The parameter 'others' can be a list of constants or [Expression].
	Coalesce(replacement any, others ...any) Expression

	// Object functions
	// MapGet creates an expression that accesses a value from a map (object) field using the provided key.
	//
	// The parameter 'strOrExprkey' is the key to access in the map.
	// It can be a string constant or an [Expression] that evaluates to a string.
	MapGet(strOrExprkey any) Expression
	// MapMerge creates an expression that merges multiple maps into a single map.
	// If multiple maps have the same key, the later value is used.
	//
	// The parameter 'secondMap' is an [Expression] representing the second map.
	// The parameter 'otherMaps' is a list of additional [Expression]s representing maps to merge.
	MapMerge(secondMap Expression, otherMaps ...Expression) Expression
	// MapRemove creates an expression that removes a key from a map.
	//
	// The parameter 'strOrExprkey' is the key to remove from the map.
	// It can be a string constant or an [Expression] that evaluates to a string.
	MapRemove(strOrExprkey any) Expression
	// MapSet creates an expression that updates a map with key-value pairs.
	//
	// The parameter 'keysAndValues' is a list of alternating key and value arguments.
	MapSet(key any, value any, moreKeysAndValues ...any) Expression
	// MapKeys creates an expression that returns the keys of a map as an array.
	MapKeys() Expression
	// MapValues creates an expression that returns the values of a map as an array.
	MapValues() Expression
	// MapEntries creates an expression that returns the entries of a map as an array of key-value maps.
	MapEntries() Expression

	// Aggregators
	// Sum creates an aggregate function that calculates the sum of the expression.
	Sum() AggregateFunction
	// Average creates an aggregate function that calculates the average of the expression.
	Average() AggregateFunction
	// Count creates an aggregate function that counts the number of documents.
	Count() AggregateFunction
	// CountDistinct creates an aggregate function that counts the distinct values of the expression.
	CountDistinct() AggregateFunction
	// Maximum creates an aggregate function that finds the maximum value of the expression.
	Maximum() AggregateFunction
	// Minimum creates an aggregate function that finds the minimum value of the expression.
	Minimum() AggregateFunction

	// Data size functions
	// StorageSize creates an expression that calculates the storage size of a field or [Expression] in bytes.
	StorageSize() Expression

	// String functions
	// ByteLength creates an expression that calculates the length of a string represented by a field or [Expression] in UTF-8
	// bytes.
	ByteLength() Expression
	// CharLength creates an expression that calculates the character length of a string field or expression in UTF8.
	CharLength() Expression
	// EndsWith creates a boolean expression that checks if the string expression ends with the specified suffix.
	//
	// The parameter 'suffix' can be a string constant or an [Expression] that evaluates to a string.
	EndsWith(suffix any) BooleanExpression
	// Like creates a boolean expression that checks if the string expression matches the specified pattern.
	//
	// The parameter 'suffix' can be a string constant or an [Expression] that evaluates to a string.
	Like(suffix any) BooleanExpression
	// RegexContains creates a boolean expression that checks if the string expression contains a match for the specified regex pattern.
	//
	// The parameter 'pattern' can be a string constant or an [Expression] that evaluates to a string.
	RegexContains(pattern any) BooleanExpression
	// RegexFind creates an expression that returns the first substring that matches the specified regex pattern.
	//
	// The parameter 'pattern' can be a string constant or an [Expression] that evaluates to a string.
	RegexFind(pattern any) Expression
	// RegexFindAll creates an expression that returns all substrings that match the specified regex pattern.
	//
	// The parameter 'pattern' can be a string constant or an [Expression] that evaluates to a string.
	RegexFindAll(pattern any) Expression
	// RegexMatch creates a boolean expression that checks if the string expression matches the specified regex pattern.
	//
	// The parameter 'pattern' can be a string constant or an [Expression] that evaluates to a string.
	RegexMatch(pattern any) BooleanExpression
	// StartsWith creates a boolean expression that checks if the string expression starts with the specified prefix.
	//
	// The parameter 'prefix' can be a string constant or an [Expression] that evaluates to a string.
	StartsWith(prefix any) BooleanExpression
	// StringConcat creates an expression that concatenates multiple strings into a single string.
	//
	// The parameter 'otherStrings' can be a mix of string constants or [Expression]s that evaluate to strings.
	StringConcat(otherStrings ...any) Expression
	// StringContains creates a boolean expression that checks if the string expression contains the specified substring.
	//
	// The parameter 'substring' can be a string constant or an [Expression] that evaluates to a string.
	StringContains(substring any) BooleanExpression
	// StringIndexOf creates an expression that returns the index of a search value in a string.
	//
	// The parameter 'search' can be a string constant or an [Expression] that evaluates to a string.
	StringIndexOf(search any) Expression
	// StringRepeat creates an expression that repeats a string a specified number of times.
	//
	// The parameter 'repetition' can be an integer constant or an [Expression] that evaluates to an integer.
	StringRepeat(repetition any) Expression
	// StringReplaceOne creates an expression that replaces the first occurrence of a search value with a replacement value.
	//
	// The parameter 'search' can be a string constant or an [Expression] that evaluates to a string.
	// The parameter 'replacement' can be a string constant or an [Expression] that evaluates to a string.
	StringReplaceOne(search, replacement any) Expression
	// StringReplaceAll creates an expression that replaces all occurrences of a search value with a replacement value.
	//
	// The parameter 'search' can be a string constant or an [Expression] that evaluates to a string.
	// The parameter 'replacement' can be a string constant or an [Expression] that evaluates to a string.
	StringReplaceAll(search, replacement any) Expression
	// StringReverse creates an expression that reverses a string.
	StringReverse() Expression
	// Join creates an expression that joins the elements of a string array into a single string.
	//
	// The parameter 'delimiter' can be a string constant or an [Expression] that evaluates to a string.
	Join(delimiter any) Expression
	// Substring creates an expression that returns a substring of a string.
	//
	// The parameter 'index' is the starting index of the substring.
	// It can be an integer constant or an [Expression] that evaluates to an integer.
	// The parameter 'offset' is the length of the substring.
	// It can be an integer constant or an [Expression] that evaluates to an integer.
	Substring(index, offset any) Expression
	// ToLower creates an expression that converts a string to lowercase.
	ToLower() Expression
	// ToUpper creates an expression that converts a string to uppercase.
	ToUpper() Expression
	// Trim creates an expression that removes leading and trailing whitespace from a string.
	Trim() Expression
	// TrimValue creates an expression that removes leading and trailing whitespace or specified characters from a string.
	TrimValue(valuesToTrim any) Expression
	// LTrim creates an expression that removes leading whitespace from a string.
	LTrim() Expression
	// LTrimValue creates an expression that removes leading whitespace or specified characters from a string.
	LTrimValue(valuesToTrim any) Expression
	// RTrim creates an expression that removes trailing whitespace from a string.
	RTrim() Expression
	// RTrimValue creates an expression that removes trailing whitespace or specified characters from a string.
	RTrimValue(valuesToTrim any) Expression
	// Split creates an expression that splits a string by a delimiter.
	//
	// The parameter 'delimiter' can be a string constant or an [Expression] that evaluates to a string.
	Split(delimiter any) Expression

	// Type creates an expression that returns the type of the expression.
	Type() Expression
	// IsType creates a boolean expression that checks if the expression is of a specific type.
	//
	// The parameter 'dataType' can be one of the following string constants:
	//   "null", "array", "boolean", "bytes", "timestamp", "geo_point", "number",
	//   "int32", "int64", "float64", "decimal128", "map", "reference", "string",
	//   "vector", "max_key", "min_key", "object_id", "regex", "request_timestamp".
	IsType(dataType string) BooleanExpression

	// Vector functions
	// CosineDistance creates an expression that calculates the cosine distance between two vectors.
	//
	// The parameter 'other' can be [Vector32], [Vector64], []float32, []float64 or an [Expression] that evaluates to a vector.
	CosineDistance(other any) Expression
	// DotProduct creates an expression that calculates the dot product of two vectors.
	//
	// The parameter 'other' can be [Vector32], [Vector64], []float32, []float64 or an [Expression] that evaluates to a vector.
	DotProduct(other any) Expression
	// EuclideanDistance creates an expression that calculates the euclidean distance between two vectors.
	//
	// The parameter 'other' can be [Vector32], [Vector64], []float32, []float64 or an [Expression] that evaluates to a vector.
	EuclideanDistance(other any) Expression
	// VectorLength creates an expression that calculates the length of a vector.
	VectorLength() Expression

	// Ordering
	// Ascending creates an ordering expression for ascending order.
	Ascending() Ordering
	// Descending creates an ordering expression for descending order.
	Descending() Ordering

	// GeoDistance creates an expression that evaluates to the distance in meters between the location in the expression and the query location.
	//
	// The parameter 'location' is the query location.
	//
	// Example:
	//
	//	client.Pipeline().Collection("restaurants").
	//		Search(
	//			WithSearchQuery("waffles"),
	//			WithSearchSort(Ascending(FieldOf("location").GeoDistance(&latlng.LatLng{Latitude: 37.0, Longitude: -122.0}))),
	//		)
	//
	// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
	// and are subject to potential breaking changes in future versions,
	// regardless of any other documented package stability guarantees.
	GeoDistance(location *latlng.LatLng) Expression

	// As assigns an alias to an expression.
	// Aliases are useful for renaming fields in the output of a stage.
	As(alias string) *AliasedExpression
}

// baseExpression provides common methods for all Expr implementations, allowing for method chaining.
type baseExpression struct {
	pbVal *pb.Value
	err   error
}

func (b *baseExpression) isExpr()                      {}
func (b *baseExpression) toProto() (*pb.Value, error)  { return b.pbVal, b.err }
func (b *baseExpression) getBaseExpr() *baseExpression { return b }

// Aritmetic functions
func (b *baseExpression) Add(other any) Expression      { return Add(b, other) }
func (b *baseExpression) Subtract(other any) Expression { return Subtract(b, other) }
func (b *baseExpression) Multiply(other any) Expression { return Multiply(b, other) }
func (b *baseExpression) Divide(other any) Expression   { return Divide(b, other) }
func (b *baseExpression) Abs() Expression               { return Abs(b) }
func (b *baseExpression) Floor() Expression             { return Floor(b) }
func (b *baseExpression) Ceil() Expression              { return Ceil(b) }
func (b *baseExpression) Exp() Expression               { return Exp(b) }
func (b *baseExpression) Log(other any) Expression      { return Log(b, other) }
func (b *baseExpression) Log10() Expression             { return Log10(b) }
func (b *baseExpression) Ln() Expression                { return Ln(b) }
func (b *baseExpression) Mod(other any) Expression      { return Mod(b, other) }
func (b *baseExpression) Pow(other any) Expression      { return Pow(b, other) }
func (b *baseExpression) Round() Expression             { return Round(b) }
func (b *baseExpression) RoundToPrecision(places any) Expression {
	return RoundToPrecision(b, places)
}
func (b *baseExpression) Trunc() Expression { return Trunc(b) }
func (b *baseExpression) TruncToPrecision(places any) Expression {
	return TruncToPrecision(b, places)
}
func (b *baseExpression) Sqrt() Expression         { return Sqrt(b) }
func (b *baseExpression) Cmp(other any) Expression { return Cmp(b, other) }

// Array functions
func (b *baseExpression) ArrayContains(value any) BooleanExpression { return ArrayContains(b, value) }
func (b *baseExpression) ArrayContainsAll(values any) BooleanExpression {
	return ArrayContainsAll(b, values)
}
func (b *baseExpression) ArrayContainsAny(values any) BooleanExpression {
	return ArrayContainsAny(b, values)
}
func (b *baseExpression) ArrayLength() Expression                  { return ArrayLength(b) }
func (b *baseExpression) EqualAny(values any) BooleanExpression    { return EqualAny(b, values) }
func (b *baseExpression) NotEqualAny(values any) BooleanExpression { return NotEqualAny(b, values) }
func (b *baseExpression) ArrayGet(offset any) Expression           { return ArrayGet(b, offset) }
func (b *baseExpression) Offset(index any) Expression              { return Offset(b, index) }
func (b *baseExpression) ArrayReverse() Expression                 { return ArrayReverse(b) }
func (b *baseExpression) ArrayConcat(otherArrays ...any) Expression {
	return ArrayConcat(b, otherArrays...)
}
func (b *baseExpression) ArraySum() Expression                  { return ArraySum(b) }
func (b *baseExpression) ArrayMaximum() Expression              { return ArrayMaximum(b) }
func (b *baseExpression) ArrayMaximumN(n any) Expression        { return ArrayMaximumN(b, n) }
func (b *baseExpression) ArrayMinimum() Expression              { return ArrayMinimum(b) }
func (b *baseExpression) ArrayMinimumN(n any) Expression        { return ArrayMinimumN(b, n) }
func (b *baseExpression) ArrayFirst() Expression                { return ArrayFirst(b) }
func (b *baseExpression) ArrayFirstN(n any) Expression          { return ArrayFirstN(b, n) }
func (b *baseExpression) ArrayLast() Expression                 { return ArrayLast(b) }
func (b *baseExpression) ArrayLastN(n any) Expression           { return ArrayLastN(b, n) }
func (b *baseExpression) ArraySliceToEnd(offset any) Expression { return ArraySliceToEnd(b, offset) }
func (b *baseExpression) ArraySlice(offset, length any) Expression {
	return ArraySlice(b, offset, length)
}

func (b *baseExpression) ArrayIndexOf(search any) Expression {
	return ArrayIndexOf(b, search)
}
func (b *baseExpression) ArrayIndexOfAll(search any) Expression {
	return ArrayIndexOfAll(b, search)
}
func (b *baseExpression) ArrayLastIndexOf(search any) Expression {
	return ArrayLastIndexOf(b, search)
}
func (b *baseExpression) First() AggregateFunction            { return First(b) }
func (b *baseExpression) Last() AggregateFunction             { return Last(b) }
func (b *baseExpression) ArrayAgg() AggregateFunction         { return ArrayAgg(b) }
func (b *baseExpression) ArrayAggDistinct() AggregateFunction { return ArrayAggDistinct(b) }

func (b *baseExpression) ArrayFilter(param string, body BooleanExpression) Expression {
	return ArrayFilter(b, param, body)
}
func (b *baseExpression) ArrayTransform(param string, body Expression) Expression {
	return ArrayTransform(b, param, body)
}
func (b *baseExpression) ArrayTransformWithIndex(param, indexParam string, body Expression) Expression {
	return ArrayTransformWithIndex(b, param, indexParam, body)
}
func (b *baseExpression) LogicalMaximum(others ...any) Expression {
	return LogicalMaximum(b, others...)
}
func (b *baseExpression) LogicalMinimum(others ...any) Expression {
	return LogicalMinimum(b, others...)
}

// Timestamp functions
func (b *baseExpression) TimestampAdd(unit, amount any) Expression {
	return TimestampAdd(b, unit, amount)
}
func (b *baseExpression) TimestampSubtract(unit, amount any) Expression {
	return TimestampSubtract(b, unit, amount)
}
func (b *baseExpression) TimestampTruncate(granularity any) Expression {
	return TimestampTruncate(b, granularity)
}
func (b *baseExpression) TimestampTruncateWithTimezone(granularity any, timezone any) Expression {
	return TimestampTruncateWithTimezone(b, granularity, timezone)
}
func (b *baseExpression) TimestampToUnixMicros() Expression  { return TimestampToUnixMicros(b) }
func (b *baseExpression) TimestampToUnixMillis() Expression  { return TimestampToUnixMillis(b) }
func (b *baseExpression) TimestampToUnixSeconds() Expression { return TimestampToUnixSeconds(b) }
func (b *baseExpression) TimestampExtract(part any) Expression {
	return TimestampExtract(b, part)
}
func (b *baseExpression) TimestampExtractWithTimezone(part, timezone any) Expression {
	return TimestampExtractWithTimezone(b, part, timezone)
}
func (b *baseExpression) TimestampDiff(start, unit any) Expression {
	return TimestampDiff(b, start, unit)
}
func (b *baseExpression) UnixMicrosToTimestamp() Expression  { return UnixMicrosToTimestamp(b) }
func (b *baseExpression) UnixMillisToTimestamp() Expression  { return UnixMillisToTimestamp(b) }
func (b *baseExpression) UnixSecondsToTimestamp() Expression { return UnixSecondsToTimestamp(b) }

// Comparison functions
func (b *baseExpression) Equal(other any) BooleanExpression       { return Equal(b, other) }
func (b *baseExpression) NotEqual(other any) BooleanExpression    { return NotEqual(b, other) }
func (b *baseExpression) GreaterThan(other any) BooleanExpression { return GreaterThan(b, other) }
func (b *baseExpression) GreaterThanOrEqual(other any) BooleanExpression {
	return GreaterThanOrEqual(b, other)
}
func (b *baseExpression) LessThan(other any) BooleanExpression { return LessThan(b, other) }
func (b *baseExpression) LessThanOrEqual(other any) BooleanExpression {
	return LessThanOrEqual(b, other)
}

// General functions
func (b *baseExpression) Length() Expression              { return Length(b) }
func (b *baseExpression) Reverse() Expression             { return Reverse(b) }
func (b *baseExpression) Concat(others ...any) Expression { return Concat(b, others...) }

// Key functions
func (b *baseExpression) GetCollectionID() Expression { return GetCollectionID(b) }
func (b *baseExpression) GetDocumentID() Expression   { return GetDocumentID(b) }
func (b *baseExpression) GetField(key any) Expression { return GetField(b, key) }

// Reference functions
func (b *baseExpression) GetParent() Expression { return GetParent(b) }
func (b *baseExpression) ReferenceSliceToEnd(offset any) Expression {
	return ReferenceSliceToEnd(b, offset)
}
func (b *baseExpression) ReferenceSlice(offset, length any) Expression {
	return ReferenceSlice(b, offset, length)
}

// Logical functions
func (b *baseExpression) IfError(catchExprOrValue any) Expression {
	return IfError(b, catchExprOrValue)
}
func (b *baseExpression) IsError() BooleanExpression {
	return IsError(b)
}
func (b *baseExpression) FieldExists() BooleanExpression {
	return FieldExists(b)
}

func (b *baseExpression) IsAbsent() BooleanExpression {
	return IsAbsent(b)
}
func (b *baseExpression) IfAbsent(catchExprOrValue any) Expression {
	return IfAbsent(b, catchExprOrValue)
}
func (b *baseExpression) IfNull(elseValueOrExpr any) Expression {
	return IfNull(b, elseValueOrExpr)
}
func (b *baseExpression) Coalesce(replacement any, others ...any) Expression {
	return Coalesce(b, replacement, others...)
}

// Object functions
func (b *baseExpression) MapGet(strOrExprkey any) Expression { return MapGet(b, strOrExprkey) }
func (b *baseExpression) MapMerge(secondMap Expression, otherMaps ...Expression) Expression {
	return MapMerge(b, secondMap, otherMaps...)
}
func (b *baseExpression) MapRemove(strOrExprkey any) Expression { return MapRemove(b, strOrExprkey) }
func (b *baseExpression) MapSet(key any, value any, moreKeysAndValues ...any) Expression {
	return MapSet(b, key, value, moreKeysAndValues...)
}
func (b *baseExpression) MapKeys() Expression    { return MapKeys(b) }
func (b *baseExpression) MapValues() Expression  { return MapValues(b) }
func (b *baseExpression) MapEntries() Expression { return MapEntries(b) }

// Aggregation operations
func (b *baseExpression) Sum() AggregateFunction           { return Sum(b) }
func (b *baseExpression) Average() AggregateFunction       { return Average(b) }
func (b *baseExpression) Count() AggregateFunction         { return Count(b) }
func (b *baseExpression) CountDistinct() AggregateFunction { return CountDistinct(b) }
func (b *baseExpression) Maximum() AggregateFunction       { return Maximum(b) }
func (b *baseExpression) Minimum() AggregateFunction       { return Minimum(b) }

// Data size functions
func (b *baseExpression) StorageSize() Expression { return StorageSize(b) }

// String functions
func (b *baseExpression) ByteLength() Expression                { return ByteLength(b) }
func (b *baseExpression) CharLength() Expression                { return CharLength(b) }
func (b *baseExpression) EndsWith(suffix any) BooleanExpression { return EndsWith(b, suffix) }
func (b *baseExpression) Like(suffix any) BooleanExpression     { return Like(b, suffix) }
func (b *baseExpression) RegexContains(pattern any) BooleanExpression {
	return RegexContains(b, pattern)
}
func (b *baseExpression) RegexFind(pattern any) Expression {
	return RegexFind(b, pattern)
}
func (b *baseExpression) RegexFindAll(pattern any) Expression {
	return RegexFindAll(b, pattern)
}
func (b *baseExpression) RegexMatch(pattern any) BooleanExpression { return RegexMatch(b, pattern) }
func (b *baseExpression) StartsWith(prefix any) BooleanExpression  { return StartsWith(b, prefix) }
func (b *baseExpression) StringConcat(otherStrings ...any) Expression {
	return StringConcat(b, otherStrings...)
}
func (b *baseExpression) StringContains(substring any) BooleanExpression {
	return StringContains(b, substring)
}
func (b *baseExpression) StringIndexOf(search any) Expression {
	return StringIndexOf(b, search)
}
func (b *baseExpression) StringRepeat(repetition any) Expression {
	return StringRepeat(b, repetition)
}
func (b *baseExpression) StringReplaceOne(search, replacement any) Expression {
	return StringReplaceOne(b, search, replacement)
}
func (b *baseExpression) StringReplaceAll(search, replacement any) Expression {
	return StringReplaceAll(b, search, replacement)
}
func (b *baseExpression) StringReverse() Expression              { return StringReverse(b) }
func (b *baseExpression) Join(delimiter any) Expression          { return Join(b, delimiter) }
func (b *baseExpression) Substring(index, offset any) Expression { return Substring(b, index, offset) }
func (b *baseExpression) ToLower() Expression                    { return ToLower(b) }
func (b *baseExpression) ToUpper() Expression                    { return ToUpper(b) }
func (b *baseExpression) Trim() Expression                       { return Trim(b) }
func (b *baseExpression) TrimValue(valuesToTrim any) Expression {
	return TrimValue(b, valuesToTrim)
}
func (b *baseExpression) LTrim() Expression { return LTrim(b) }
func (b *baseExpression) LTrimValue(valuesToTrim any) Expression {
	return LTrimValue(b, valuesToTrim)
}
func (b *baseExpression) RTrim() Expression { return RTrim(b) }
func (b *baseExpression) RTrimValue(valuesToTrim any) Expression {
	return RTrimValue(b, valuesToTrim)
}
func (b *baseExpression) Split(delimiter any) Expression { return Split(b, delimiter) }

// Type functions
func (b *baseExpression) Type() Expression                         { return Type(b) }
func (b *baseExpression) IsType(dataType string) BooleanExpression { return IsType(b, dataType) }

// Vector functions
func (b *baseExpression) CosineDistance(other any) Expression    { return CosineDistance(b, other) }
func (b *baseExpression) DotProduct(other any) Expression        { return DotProduct(b, other) }
func (b *baseExpression) EuclideanDistance(other any) Expression { return EuclideanDistance(b, other) }
func (b *baseExpression) VectorLength() Expression               { return VectorLength(b) }

// Ordering
func (b *baseExpression) Ascending() Ordering  { return Ascending(b) }
func (b *baseExpression) Descending() Ordering { return Descending(b) }

// Experimental: Update, Delete and Search stages in pipeline queries are in public preview
// and are subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func (b *baseExpression) GeoDistance(location *latlng.LatLng) Expression {
	return GeoDistance(b, location)
}

func (b *baseExpression) As(alias string) *AliasedExpression {
	return newAliasedExpr(b, alias)
}

// Ensure that baseExpr implements the Expr interface.
var _ Expression = (*baseExpression)(nil)

// AliasedExpression represents an expression with an alias.
// It implements the [Selectable] interface, allowing it to be used in projection stages like `Select` and `AddFields`.
type AliasedExpression struct {
	expr  Expression
	alias string
}

func newAliasedExpr(expr Expression, alias string) *AliasedExpression {
	return &AliasedExpression{expr: expr, alias: alias}
}

// getSelectionDetails returns the alias and the underlying expression for this AliasedExpr.
// This method allows AliasedExpr to satisfy the Selectable interface.
func (e *AliasedExpression) getSelectionDetails() (string, Expression) {
	return e.alias, e.expr
}

func (e *AliasedExpression) isSelectable() {}

// Ensure that AliasedExpr implements the Selectable interface.
var _ Selectable = (*AliasedExpression)(nil)
