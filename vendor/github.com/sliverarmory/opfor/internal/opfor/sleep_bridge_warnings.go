package opfor

import (
	"errors"
	"fmt"
)

// Sleep's Java bridges mix BridgeUtilities getters (which synthesize
// defaults) with direct Stack.pop calls (which throw). These constructors are
// intentionally limited to stock Sleep bridge implementations; importer and
// Aggressor-provider errors remain authoritative host failures.
func sleepBridgeEmptyStack() error {
	return &uncaughtScriptWarning{err: errors.New(sleepEmptyStackWarning)}
}

func sleepBridgeIllegalArgument(message string) error {
	return &uncaughtScriptWarning{err: errors.New(message)}
}

// BasicNumbers performs integral division and remainder with Java primitive
// operators. A zero divisor raises ArithmeticException, whose message is
// consumed by Sleep's active Block just like the other uncaught bridge
// warnings. Floating-point zero divisors do not use this path.
func sleepBridgeArithmeticException() error {
	return &uncaughtScriptWarning{err: errors.New("/ by zero")}
}

func sleepBridgeInvalidIndex(message string) error {
	if message == "" {
		return &uncaughtScriptWarning{err: errors.New("attempted an invalid index")}
	}
	return &uncaughtScriptWarning{err: fmt.Errorf("attempted an invalid index: %s", message)}
}

func sleepBridgeNullValue() error {
	return &uncaughtScriptWarning{err: errors.New("null value error")}
}

func sleepBridgeArgument(invocation Invocation, index int) (Value, error) {
	if index < 0 || index >= len(invocation.Arguments) {
		return Null(), sleepBridgeEmptyStack()
	}
	return invocation.Arg(index), nil
}
