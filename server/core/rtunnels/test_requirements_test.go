package rtunnels

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The vendored testify subset includes assert but not require. Keep fatal test
// assertions local so repository tests remain fully vendored.
var must testRequirements

type testRequirements struct{}

func (testRequirements) NoError(t testing.TB, err error, message ...any) {
	t.Helper()
	if !assert.NoError(t, err, message...) {
		t.FailNow()
	}
}

func (testRequirements) ErrorIs(t testing.TB, err error, target error, message ...any) {
	t.Helper()
	if !assert.ErrorIs(t, err, target, message...) {
		t.FailNow()
	}
}

func (testRequirements) True(t testing.TB, value bool, message ...any) {
	t.Helper()
	if !assert.True(t, value, message...) {
		t.FailNow()
	}
}

func (testRequirements) NotEmpty(t testing.TB, value any, message ...any) {
	t.Helper()
	if !assert.NotEmpty(t, value, message...) {
		t.FailNow()
	}
}

func (testRequirements) Len(t testing.TB, value any, length int, message ...any) {
	t.Helper()
	if !assert.Len(t, value, length, message...) {
		t.FailNow()
	}
}
