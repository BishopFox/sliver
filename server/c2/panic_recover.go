package c2

import "runtime/debug"

func recoverAndLogPanic(logf func(string, ...interface{}), scope string) {
	if recovered := recover(); recovered != nil {
		logf("Recovered panic in %s: %v\n%s", scope, recovered, string(debug.Stack()))
	}
}
