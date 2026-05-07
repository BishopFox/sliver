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

// AggregateFunction represents an aggregation function in a pipeline.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type AggregateFunction interface {
	toProto() (*pb.Value, error)
	getBaseAggregateFunction() *baseAggregateFunction
	isAggregateFunction()
	As(alias string) *AliasedAggregate
}

// baseAggregateFunction provides common methods for all AggregateFunction implementations.
type baseAggregateFunction struct {
	pbVal *pb.Value
	err   error
}

func newBaseAggregateFunction(name string, fieldOrExpr any) *baseAggregateFunction {
	var argsPbVals []*pb.Value
	var err error

	if fieldOrExpr != nil {
		var valueExpr Expression
		switch value := fieldOrExpr.(type) {
		case string:
			valueExpr = FieldOf(value)
		case FieldPath:
			valueExpr = FieldOf(value)
		case Expression:
			valueExpr = value
		default:
			err = fmt.Errorf("firestore: invalid type for parameter 'value' for %s: expected string, FieldPath, or Expr, but got %T", name, value)
		}

		if err == nil {
			var pbVal *pb.Value
			pbVal, err = valueExpr.toProto()
			if err == nil {
				argsPbVals = append(argsPbVals, pbVal)
			}
		}
	}

	if err != nil {
		return &baseAggregateFunction{err: err}
	}

	pbVal := &pb.Value{ValueType: &pb.Value_FunctionValue{
		FunctionValue: &pb.Function{
			Name: name,
			Args: argsPbVals,
		},
	}}
	return &baseAggregateFunction{pbVal: pbVal}
}

func (b *baseAggregateFunction) toProto() (*pb.Value, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.pbVal, nil
}

func (b *baseAggregateFunction) getBaseAggregateFunction() *baseAggregateFunction { return b }
func (b *baseAggregateFunction) isAggregateFunction()                             {}
func (b *baseAggregateFunction) As(alias string) *AliasedAggregate {
	return &AliasedAggregate{baseAggregateFunction: b, alias: alias}
}

// Ensure that baseAggregateFunction implements the AggregateFunction interface.
var _ AggregateFunction = (*baseAggregateFunction)(nil)

// AliasedAggregate is an aliased [AggregateFunction].
// It's used to give a name to the result of an aggregation.
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
type AliasedAggregate struct {
	*baseAggregateFunction
	alias string
}

// Sum creates an aggregation that calculates the sum of values from an expression or a field's values
// across multiple stage inputs.
//
// Example:
//
//		// Calculate the total revenue from a set of orders
//		Sum(FieldOf("orderAmount")).As("totalRevenue") // FieldOf returns Expr
//	 	Sum("orderAmount").As("totalRevenue")          // String implicitly becomes FieldOf(...).As(...)
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Sum(fieldOrExpr any) AggregateFunction {
	return newBaseAggregateFunction("sum", fieldOrExpr)
}

// Average creates an aggregation that calculates the average (mean) of values from an expression or a field's values
// across multiple stage inputs.
// fieldOrExpr can be a field path string, [FieldPath] or [Expression]
// Example:
//
//		// Calculate the average age of users
//		Average(FieldOf("info.age")).As("averageAge")       // FieldOf returns Expr
//		Average(FieldOfPath("info.age")).As("averageAge") // FieldOfPath returns Expr
//	    Average("info.age").As("averageAge")              // String implicitly becomes FieldOf(...).As(...)
//	    Average(FieldPath([]string{"info", "age"})).As("averageAge")
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Average(fieldOrExpr any) AggregateFunction {
	return newBaseAggregateFunction("average", fieldOrExpr)
}

// Count creates an aggregation that counts the number of stage inputs with valid evaluations of the
// provided field or expression.
// fieldOrExpr can be a field path string, [FieldPath] or [Expression]
// Example:
//
//		// Count the number of items where the price is greater than 10
//		Count(FieldOf("price").Gt(10)).As("expensiveItemCount") // FieldOf("price").Gt(10) is a BooleanExpr
//	    // Count the total number of products
//		Count("productId").As("totalProducts")                  // String implicitly becomes FieldOf(...).As(...)
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Count(fieldOrExpr any) AggregateFunction {
	return newBaseAggregateFunction("count", fieldOrExpr)
}

// CountAll creates an aggregation that counts the total number of stage inputs.
//
// Example:
//
//		// Count the total number of users
//	    CountAll().As("totalUsers")
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func CountAll() AggregateFunction {
	return newBaseAggregateFunction("count", nil)
}

// CountDistinct creates an aggregation that counts the number of distinct values of the
// provided field or expression.
// fieldOrExpr can be a field path string, [FieldPath] or [Expression]
// Example:
//
//		// CountDistinct the number of distinct items where the price is greater than 10
//		CountDistinct(FieldOf("price").Gt(10)).As("expensiveItemCount") // FieldOf("price").Gt(10) is a BooleanExpr
//	    // CountDistinct the total number of distinct products
//		CountDistinct("productId").As("totalProducts")                  // String implicitly becomes FieldOf(...).As(...)
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func CountDistinct(fieldOrExpr any) AggregateFunction {
	return newBaseAggregateFunction("count_distinct", fieldOrExpr)
}

// CountIf creates an aggregation that counts the number of stage inputs where the provided boolean
// expression evaluates to true.
// Example:
//
//	CountIf(FieldOf("published").Equal(true)).As("publishedCount")
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func CountIf(condition BooleanExpression) AggregateFunction {
	return newBaseAggregateFunction("count_if", condition)
}

// Maximum creates an aggregation that calculates the maximum of values from an expression or a field's values
// across multiple stage inputs.
//
// Example:
//
//		// Find the highest order amount
//		Maximum(FieldOf("orderAmount")).As("maxOrderAmount") // FieldOf returns Expr
//	 	Maximum("orderAmount").As("maxOrderAmount")          // String implicitly becomes FieldOf(...).As(...)
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Maximum(fieldOrExpr any) AggregateFunction {
	return newBaseAggregateFunction("maximum", fieldOrExpr)
}

// Minimum creates an aggregation that calculates the minimum of values from an expression or a field's values
// across multiple stage inputs.
//
// Example:
//
//		// Find the lowest order amount
//		Minimum(FieldOf("orderAmount")).As("minOrderAmount") // FieldOf returns Expr
//	 	Minimum("orderAmount").As("minOrderAmount")          // String implicitly becomes FieldOf(...).As(...)
//
// Experimental: Firestore Pipelines is currently in preview and is subject to potential breaking changes in future versions,
// regardless of any other documented package stability guarantees.
func Minimum(fieldOrExpr any) AggregateFunction {
	return newBaseAggregateFunction("minimum", fieldOrExpr)
}
