// Copyright 2025 Google Inc. All Rights Reserved.
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

package remoteconfig

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

type conditionEvaluator struct {
	evaluationContext map[string]any
	conditions        []namedCondition
}

const (
	maxConditionRecursionDepth = 10
	rootNestingLevel           = 0
	doublePrecision            = 64
	whiteSpace                 = " "
	segmentSeparator           = "."
	maxPossibleSegments        = 5
)

var (
	errTooManySegments     = errors.New("number of segments exceeds maximum allowed length")
	errNegativeSegment     = errors.New("segment cannot be negative")
	errInvalidCustomSignal = errors.New("missing operator, key, or target values for custom signal condition")
)

const (
	randomizationID       = "randomizationID"
	totalMicroPercentiles = 100_000_000
	lessThanOrEqual       = "LESS_OR_EQUAL"
	greaterThan           = "GREATER_THAN"
	between               = "BETWEEN"
)

const (
	stringContains       = "STRING_CONTAINS"
	stringDoesNotContain = "STRING_DOES_NOT_CONTAIN"
	stringExactlyMatches = "STRING_EXACTLY_MATCHES"
	stringContainsRegex  = "STRING_CONTAINS_REGEX"

	numericLessThan      = "NUMERIC_LESS_THAN"
	numericLessThanEqual = "NUMERIC_LESS_EQUAL"
	numericEqual         = "NUMERIC_EQUAL"
	numericNotEqual      = "NUMERIC_NOT_EQUAL"
	numericGreaterThan   = "NUMERIC_GREATER_THAN"
	numericGreaterEqual  = "NUMERIC_GREATER_EQUAL"

	semanticVersionLessThan     = "SEMANTIC_VERSION_LESS_THAN"
	semanticVersionLessEqual    = "SEMANTIC_VERSION_LESS_EQUAL"
	semanticVersionEqual        = "SEMANTIC_VERSION_EQUAL"
	semanticVersionNotEqual     = "SEMANTIC_VERSION_NOT_EQUAL"
	semanticVersionGreaterThan  = "SEMANTIC_VERSION_GREATER_THAN"
	semanticVersionGreaterEqual = "SEMANTIC_VERSION_GREATER_EQUAL"
)

func (ce *conditionEvaluator) evaluateConditions() map[string]bool {
	evaluatedConditions := make(map[string]bool)
	for _, condition := range ce.conditions {
		evaluatedConditions[condition.Name] = ce.evaluateCondition(condition.Condition, rootNestingLevel)
	}
	return evaluatedConditions
}

func (ce *conditionEvaluator) evaluateCondition(condition *oneOfCondition, nestingLevel int) bool {
	if nestingLevel >= maxConditionRecursionDepth {
		log.Println("Maximum recursion depth is exceeded.")
		return false
	}

	if condition.Boolean != nil {
		return *condition.Boolean
	} else if condition.OrCondition != nil {
		return ce.evaluateOrCondition(condition.OrCondition, nestingLevel+1)
	} else if condition.AndCondition != nil {
		return ce.evaluateAndCondition(condition.AndCondition, nestingLevel+1)
	} else if condition.Percent != nil {
		return ce.evaluatePercentCondition(condition.Percent)
	} else if condition.CustomSignal != nil {
		return ce.evaluateCustomSignalCondition(condition.CustomSignal)
	}
	log.Println("Unknown condition type encountered.")
	return false
}

func (ce *conditionEvaluator) evaluateOrCondition(orCondition *orCondition, nestingLevel int) bool {
	for _, condition := range orCondition.Conditions {
		result := ce.evaluateCondition(&condition, nestingLevel+1)
		if result {
			return true
		}
	}
	return false
}

func (ce *conditionEvaluator) evaluateAndCondition(andCondition *andCondition, nestingLevel int) bool {
	for _, condition := range andCondition.Conditions {
		result := ce.evaluateCondition(&condition, nestingLevel+1)
		if !result {
			return false
		}
	}
	return true
}

func (ce *conditionEvaluator) evaluatePercentCondition(percentCondition *percentCondition) bool {
	if rid, ok := ce.evaluationContext[randomizationID].(string); ok {
		if percentCondition.PercentOperator == "" {
			log.Println("Missing percent operator for percent condition.")
			return false
		}
		instanceMicroPercentile := computeInstanceMicroPercentile(percentCondition.Seed, rid)
		switch percentCondition.PercentOperator {
		case lessThanOrEqual:
			return instanceMicroPercentile <= percentCondition.MicroPercent
		case greaterThan:
			return instanceMicroPercentile > percentCondition.MicroPercent
		case between:
			return instanceMicroPercentile > percentCondition.MicroPercentRange.MicroPercentLowerBound && instanceMicroPercentile <= percentCondition.MicroPercentRange.MicroPercentUpperBound
		default:
			log.Printf("Unknown percent operator: %s\n", percentCondition.PercentOperator)
			return false
		}
	}
	log.Println("Missing or invalid randomizationID (requires a string value) for percent condition.")
	return false
}

func computeInstanceMicroPercentile(seed string, randomizationID string) uint32 {
	var sb strings.Builder
	if len(seed) > 0 {
		sb.WriteString(seed)
		sb.WriteRune('.')
	}
	sb.WriteString(randomizationID)
	stringToHash := sb.String()

	hash := sha256.New()
	hash.Write([]byte(stringToHash))
	// Calculate the final SHA-256 hash as a byte slice (32 bytes).
	// Convert to a big.Int. The "0x" prefix is implicit in the conversion from hex to big.Int.
	hashBigInt := new(big.Int).SetBytes(hash.Sum(nil))
	instanceMicroPercentileBigInt := new(big.Int).Mod(hashBigInt, big.NewInt(totalMicroPercentiles))
	// Safely convert to uint32 since the range of instanceMicroPercentile is 0 to 100_000_000; range of uint32 is 0 to 4_294_967_295.
	return uint32(instanceMicroPercentileBigInt.Int64())
}

func (ce *conditionEvaluator) evaluateCustomSignalCondition(customSignalCondition *customSignalCondition) bool {
	if err := customSignalCondition.isValid(); err != nil {
		log.Println(err)
		return false
	}
	actualValue, ok := ce.evaluationContext[customSignalCondition.CustomSignalKey]
	if !ok {
		log.Printf("Custom signal key: %s, missing from context\n", customSignalCondition.CustomSignalKey)
		return false
	}
	switch customSignalCondition.CustomSignalOperator {
	case stringContains:
		return compareStrings(customSignalCondition.TargetCustomSignalValues, actualValue, func(actualValue, target string) bool { return strings.Contains(actualValue, target) })
	case stringDoesNotContain:
		return !compareStrings(customSignalCondition.TargetCustomSignalValues, actualValue, func(actualValue, target string) bool { return strings.Contains(actualValue, target) })
	case stringExactlyMatches:
		return compareStrings(customSignalCondition.TargetCustomSignalValues, actualValue, func(actualValue, target string) bool {
			return strings.Trim(actualValue, whiteSpace) == strings.Trim(target, whiteSpace)
		})
	case stringContainsRegex:
		return compareStrings(customSignalCondition.TargetCustomSignalValues, actualValue, func(actualValue, targetPattern string) bool {
			result, err := regexp.MatchString(targetPattern, actualValue)
			if err != nil {
				return false
			}
			return result
		})

	// For numeric operators only one target value is allowed.
	case numericLessThan:
		return compareNumbers(customSignalCondition.TargetCustomSignalValues[0], actualValue, func(result int) bool { return result < 0 })
	case numericLessThanEqual:
		return compareNumbers(customSignalCondition.TargetCustomSignalValues[0], actualValue, func(result int) bool { return result <= 0 })
	case numericEqual:
		return compareNumbers(customSignalCondition.TargetCustomSignalValues[0], actualValue, func(result int) bool { return result == 0 })
	case numericNotEqual:
		return compareNumbers(customSignalCondition.TargetCustomSignalValues[0], actualValue, func(result int) bool { return result != 0 })
	case numericGreaterThan:
		return compareNumbers(customSignalCondition.TargetCustomSignalValues[0], actualValue, func(result int) bool { return result > 0 })
	case numericGreaterEqual:
		return compareNumbers(customSignalCondition.TargetCustomSignalValues[0], actualValue, func(result int) bool { return result >= 0 })

	// For semantic operators only one target value is allowed.
	case semanticVersionLessThan:
		return compareSemanticVersion(customSignalCondition.TargetCustomSignalValues[0], actualValue, func(result int) bool { return result < 0 })
	case semanticVersionLessEqual:
		return compareSemanticVersion(customSignalCondition.TargetCustomSignalValues[0], actualValue, func(result int) bool { return result <= 0 })
	case semanticVersionEqual:
		return compareSemanticVersion(customSignalCondition.TargetCustomSignalValues[0], actualValue, func(result int) bool { return result == 0 })
	case semanticVersionNotEqual:
		return compareSemanticVersion(customSignalCondition.TargetCustomSignalValues[0], actualValue, func(result int) bool { return result != 0 })
	case semanticVersionGreaterThan:
		return compareSemanticVersion(customSignalCondition.TargetCustomSignalValues[0], actualValue, func(result int) bool { return result > 0 })
	case semanticVersionGreaterEqual:
		return compareSemanticVersion(customSignalCondition.TargetCustomSignalValues[0], actualValue, func(result int) bool { return result >= 0 })
	}
	log.Printf("Unknown custom signal operator: %s\n", customSignalCondition.CustomSignalOperator)
	return false
}

func (cs *customSignalCondition) isValid() error {
	if cs.CustomSignalOperator == "" || cs.CustomSignalKey == "" || len(cs.TargetCustomSignalValues) == 0 {
		return errInvalidCustomSignal
	}
	return nil
}

func compareStrings(targetCustomSignalValues []string, actualValue any, predicateFn func(actualValue, target string) bool) bool {
	csValStr, ok := actualValue.(string)
	if !ok {
		if jsonBytes, err := json.Marshal(actualValue); err == nil {
			csValStr = string(jsonBytes)
		} else {
			log.Printf("Failed to parse custom signal value '%v' as a string : %v\n", actualValue, err)
			return false
		}
	}
	for _, target := range targetCustomSignalValues {
		if predicateFn(csValStr, target) {
			return true
		}
	}
	return false
}

func compareNumbers(targetCustomSignalValue string, actualValue any, predicateFn func(result int) bool) bool {
	targetFloat, err := strconv.ParseFloat(strings.Trim(targetCustomSignalValue, whiteSpace), doublePrecision)
	if err != nil {
		log.Printf("Failed to convert target custom signal value '%v' from string to number: %v", targetCustomSignalValue, err)
		return false
	}
	var actualValFloat float64
	switch actualValue := actualValue.(type) {
	case float32:
		actualValFloat = float64(actualValue)
	case float64:
		actualValFloat = actualValue
	case int8:
		actualValFloat = float64(actualValue)
	case int:
		actualValFloat = float64(actualValue)
	case int16:
		actualValFloat = float64(actualValue)
	case int32:
		actualValFloat = float64(actualValue)
	case int64:
		actualValFloat = float64(actualValue)
	case uint8:
		actualValFloat = float64(actualValue)
	case uint:
		actualValFloat = float64(actualValue)
	case uint16:
		actualValFloat = float64(actualValue)
	case uint32:
		actualValFloat = float64(actualValue)
	case uint64:
		actualValFloat = float64(actualValue)
	case bool:
		if actualValue {
			actualValFloat = 1
		} else {
			actualValFloat = 0
		}
	case string:
		actualValFloat, err = strconv.ParseFloat(strings.Trim(actualValue, whiteSpace), doublePrecision)
		if err != nil {
			log.Printf("Failed to convert custom signal value '%v' from string to number: %v", actualValue, err)
			return false
		}
	default:
		log.Printf("Cannot parse custom signal value '%v' of type %T as a number", actualValue, actualValue)
		return false
	}
	result := 0
	if actualValFloat > targetFloat {
		result = 1
	} else if actualValFloat < targetFloat {
		result = -1
	}
	return predicateFn(result)
}

func compareSemanticVersion(targetValue string, actualValue any, predicateFn func(result int) bool) bool {
	targetSemVer, err := transformVersionToSegments(strings.Trim(targetValue, whiteSpace))
	if err != nil {
		log.Printf("Error transforming target semantic version %q: %v\n", targetValue, err)
		return false
	}
	actualValueStr := fmt.Sprintf("%v", actualValue)
	actualSemVer, err := transformVersionToSegments(strings.Trim(actualValueStr, whiteSpace))
	if err != nil {
		log.Printf("Error transforming custom signal value '%v' to semantic version: %v\n", actualValue, err)
		return false
	}
	for idx := 0; idx < maxPossibleSegments; idx++ {
		if actualSemVer[idx] > targetSemVer[idx] {
			return predicateFn(1)
		} else if actualSemVer[idx] < targetSemVer[idx] {
			return predicateFn(-1)
		}
	}
	return predicateFn(0)
}

func transformVersionToSegments(version string) ([]int, error) {
	// Trim any trailing or leading segment separators (.) and split.
	trimmedVersion := strings.Trim(version, segmentSeparator)
	segments := strings.Split(trimmedVersion, segmentSeparator)

	if len(segments) > maxPossibleSegments {
		return nil, errTooManySegments
	}
	// Initialize with the maximum possible segment length for consistent comparison.
	transformedVersion := make([]int, maxPossibleSegments)
	for idx, segmentStr := range segments {
		segmentInt, err := strconv.Atoi(segmentStr)
		if err != nil {
			return nil, err
		}
		if segmentInt < 0 {
			return nil, errNegativeSegment
		}
		transformedVersion[idx] = segmentInt
	}
	return transformedVersion, nil
}
