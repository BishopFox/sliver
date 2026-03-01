package proxy

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertErrEqualIgnoreQuotes(t *testing.T, expected error, actual error) {
	t.Helper()
	if expected == nil {
		assert.NoError(t, actual)
		return
	}

	if assert.Error(t, actual) {
		expectedMsg := strings.ReplaceAll(expected.Error(), "\"", "")
		actualMsg := strings.ReplaceAll(actual.Error(), "\"", "")
		assert.Equal(t, expectedMsg, actualMsg)
	}
}
