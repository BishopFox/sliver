package crack

import (
	"strings"
	"testing"

	"github.com/bishopfox/sliver/client/command/settings"
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
		{
			name: "OpenCL version",
			got: formatOpenCLDevice(&clientpb.OpenCLBackendInfo{
				Name: "Portable GPU", OpenCLVersion: "OpenCL 3.0", Version: "legacy",
			}),
			want: "Portable GPU (OpenCL 3.0)",
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

func TestRenderCrackerBackendInfoLevelTwo(t *testing.T) {
	zero := uint32(0)
	falseValue := false
	station := &clientpb.Crackstation{
		Name:           "station",
		OperatorName:   "operator",
		HostUUID:       "host-uuid",
		Version:        "station-v1",
		GOOS:           "linux",
		GOARCH:         "amd64",
		HashcatVersion: "v7.1.2",
		CUDA: []*clientpb.CUDABackendInfo{{
			Name: "CUDA GPU", CUDAVersion: "CUDA 13", Version: "device-v1", Type: "GPU",
			Vendor: "NVIDIA", VendorID: 4318, BackendDeviceID: 1, BackendDeviceAlias: &zero,
			PreferredThreadSize: &zero, MemoryUnified: &falseValue, LocalMemory: "64 KB",
			CacheSize: "8 MB", PCIAddress: "01:00.0", MemoryFree: "8 GB", MemoryTotal: "12 GB",
		}},
		HIP: []*clientpb.HIPBackendInfo{{Name: "HIP GPU", HIPVersion: "HIP 7", BackendDeviceID: 2}},
		Metal: []*clientpb.MetalBackendInfo{{
			Name: "Metal GPU", MetalVersion: "Metal 410", BackendDeviceID: 3,
			RegistryID: &zero, GPUHeadless: &falseValue, GPULowPower: &falseValue, GPURemovable: &falseValue,
			PhysicalLocation: "built-in", MaxTXRate: "100", MemoryAllocPerBlock: "4 GB",
		}},
		OpenCL: []*clientpb.OpenCLBackendInfo{{
			Name: "OpenCL GPU", OpenCLVersion: "OpenCL 3.0", OpenCLDriverVersion: "driver-v1",
			BackendDeviceID: 4, PlatformVendor: "Portable", PlatformName: "Platform",
			PlatformVersion: "platform-v1", MemoryAllocPerBlock: "2 GB",
		}},
	}

	output := renderCracker(station, 0, settings.SliverDefault, 2)
	for _, expected := range []string{
		"station", "operator", "host-uuid", "station-v1", "linux/amd64", "v7.1.2",
		"CUDA GPU", "HIP GPU", "Metal GPU", "OpenCL GPU", "OpenCL 3.0", "driver-v1",
		"Backend Device ID", "Backend Device Alias", "Preferred Thread Size", "Unified Memory",
		"NVIDIA", "4318", "device-v1", "64 KB", "8 MB", "01:00.0",
		"Registry ID", "Headless", "Low Power", "Removable", "built-in", "100",
		"OpenCL Platform ID", "Portable", "Platform", "platform-v1", "2 GB",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("level-2 backend info does not contain %q:\n%s", expected, output)
		}
	}
}

func TestRenderCrackerBackendInfoLevelOneOmitsDetails(t *testing.T) {
	station := &clientpb.Crackstation{
		Name: "station",
		CUDA: []*clientpb.CUDABackendInfo{{
			Name: "GPU", BackendDeviceID: 7, PCIAddress: "01:00.0",
		}},
	}
	output := renderCracker(station, 0, settings.SliverDefault, 1)
	for _, unexpected := range []string{"Host UUID", "Backend Device ID", "PCI Address"} {
		if strings.Contains(output, unexpected) {
			t.Errorf("level-1 backend info unexpectedly contains %q:\n%s", unexpected, output)
		}
	}
}

func TestRenderCrackerPreservesOptionalPresence(t *testing.T) {
	zero := uint32(0)
	falseValue := false
	withPresence := renderCracker(&clientpb.Crackstation{
		CUDA: []*clientpb.CUDABackendInfo{{
			BackendDeviceAlias: &zero,
			MemoryUnified:      &falseValue,
		}},
	}, 0, settings.SliverDefault, 2)
	if !strings.Contains(withPresence, "Backend Device Alias") || !strings.Contains(withPresence, "Unified Memory") {
		t.Fatalf("present zero/false optional fields were omitted:\n%s", withPresence)
	}

	withoutPresence := renderCracker(&clientpb.Crackstation{
		CUDA: []*clientpb.CUDABackendInfo{{}},
	}, 0, settings.SliverDefault, 2)
	if strings.Contains(withoutPresence, "Backend Device Alias") || strings.Contains(withoutPresence, "Unified Memory") {
		t.Fatalf("absent optional fields were rendered:\n%s", withoutPresence)
	}
}

func TestRenderCrackerReportsNoBackendDevices(t *testing.T) {
	output := renderCracker(&clientpb.Crackstation{
		Name: "station",
		CUDA: []*clientpb.CUDABackendInfo{nil},
	}, 0, settings.SliverDefault, 1)
	if !strings.Contains(output, "No backend devices reported") {
		t.Fatalf("missing no-device message:\n%s", output)
	}
}
