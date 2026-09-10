package crack

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestFormatBackendDevice(t *testing.T) {
	tests := []struct {
		name     string
		device   string
		versions []string
		want     string
	}{
		{
			name:     "specific runtime version wins",
			device:   "Apple M5",
			versions: []string{"Metal 410.7", "legacy-generic"},
			want:     "Apple M5 (Metal 410.7)",
		},
		{
			name:     "legacy generic fallback",
			device:   "NVIDIA GPU",
			versions: []string{"", "13.0"},
			want:     "NVIDIA GPU (13.0)",
		},
		{
			name:     "no empty version suffix",
			device:   "AMD GPU",
			versions: []string{"", ""},
			want:     "AMD GPU",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := formatBackendDevice(test.device, test.versions...); got != test.want {
				t.Fatalf("formatBackendDevice() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestBackendSpecificDisplayVersions(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "CUDA version",
			got: formatCUDADevice(&clientpb.CUDABackendInfo{
				Name: "NVIDIA GPU", CUDAVersion: "13.0", Version: "legacy",
			}),
			want: "NVIDIA GPU (13.0)",
		},
		{
			name: "HIP version",
			got: formatHIPDevice(&clientpb.HIPBackendInfo{
				Name: "AMD GPU", HIPVersion: "7.0.51831", Version: "legacy",
			}),
			want: "AMD GPU (7.0.51831)",
		},
		{
			name: "Metal version",
			got: formatMetalDevice(&clientpb.MetalBackendInfo{
				Name: "Apple M5", MetalVersion: "Metal 410.7", Version: "legacy",
			}),
			want: "Apple M5 (Metal 410.7)",
		},
		{
			name: "legacy CUDA fallback",
			got: formatCUDADevice(&clientpb.CUDABackendInfo{
				Name: "NVIDIA GPU", Version: "12.9",
			}),
			want: "NVIDIA GPU (12.9)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("backend display = %q, want %q", test.got, test.want)
			}
		})
	}
}
