package matcher

import (
	"runtime"
	"testing"
)

func TestMatchUploadPatterns(t *testing.T) {
	for _, test := range []struct {
		pattern string
		path    string
		want    bool
	}{
		{"report-*.txt", "report-1.txt", true},
		{"report-*.txt", "other.txt", false},
		{"REPORT.TXT", "report.txt", runtime.GOOS == "windows"},
	} {
		got, err := Match(test.pattern, test.path)
		if err != nil || got != test.want {
			t.Fatalf("Match(%q, %q) = %t, %v; want %t", test.pattern, test.path, got, err, test.want)
		}
	}
}
