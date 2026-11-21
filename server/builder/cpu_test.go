package builder

import "testing"

func TestCombineCPULimits(t *testing.T) {
	tests := []struct {
		name                               string
		host, concurrency, affinity, quota int
		quotaKnown                         bool
		want                               int
	}{
		{"all limits", 32, 16, 8, 3, true, 3},
		{"affinity wins", 32, 16, 4, 8, true, 4},
		{"concurrency wins", 32, 2, 4, 8, true, 2},
		{"missing limits ignored", 16, 8, 0, 0, false, 8},
		{"zero usable quota", 16, 8, 4, 0, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := combineCPULimits(tt.host, tt.concurrency, tt.affinity, tt.quota, tt.quotaKnown); got != tt.want {
				t.Fatalf("effective CPUs: got %d, want %d", got, tt.want)
			}
		})
	}
}
