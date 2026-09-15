package clientpb

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestBackendInfoFieldNumbers(t *testing.T) {
	tests := []struct {
		message protoreflect.Message
		fields  map[protoreflect.Name]protoreflect.FieldNumber
	}{
		{
			message: (&Crackstation{}).ProtoReflect(),
			fields:  map[protoreflect.Name]protoreflect.FieldNumber{"HIP": 103},
		},
		{
			message: (&CUDABackendInfo{}).ProtoReflect(),
			fields: map[protoreflect.Name]protoreflect.FieldNumber{
				"BackendDeviceID": 11, "BackendDeviceAlias": 12, "PreferredThreadSize": 13,
				"MemoryUnified": 14, "LocalMemory": 15, "CacheSize": 16, "PCIAddress": 17,
			},
		},
		{
			message: (&MetalBackendInfo{}).ProtoReflect(),
			fields: map[protoreflect.Name]protoreflect.FieldNumber{
				"BackendDeviceID": 11, "BackendDeviceAlias": 12, "PreferredThreadSize": 13,
				"MemoryUnified": 14, "LocalMemory": 15, "CacheSize": 16,
				"MemoryAllocPerBlock": 17, "PhysicalLocation": 18, "RegistryID": 19,
				"MaxTXRate": 20, "GPUHeadless": 21, "GPULowPower": 22, "GPURemovable": 23,
			},
		},
		{
			message: (&OpenCLBackendInfo{}).ProtoReflect(),
			fields: map[protoreflect.Name]protoreflect.FieldNumber{
				"BackendDeviceID": 12, "BackendDeviceAlias": 13, "PreferredThreadSize": 14,
				"MemoryUnified": 15, "LocalMemory": 16, "MemoryAllocPerBlock": 17,
				"PCIAddress": 18, "PlatformID": 19, "PlatformVendor": 20,
				"PlatformName": 21, "PlatformVersion": 22,
			},
		},
		{
			message: (&HIPBackendInfo{}).ProtoReflect(),
			fields: map[protoreflect.Name]protoreflect.FieldNumber{
				"HIPVersion": 10, "BackendDeviceID": 11, "BackendDeviceAlias": 12,
				"PreferredThreadSize": 13, "MemoryUnified": 14, "LocalMemory": 15,
				"CacheSize": 16, "PCIAddress": 17,
			},
		},
	}

	for _, test := range tests {
		for name, want := range test.fields {
			field := test.message.Descriptor().Fields().ByName(name)
			if field == nil {
				t.Fatalf("%s is missing field %s", test.message.Descriptor().FullName(), name)
			}
			if got := field.Number(); got != want {
				t.Fatalf("%s.%s tag = %d, want %d", test.message.Descriptor().FullName(), name, got, want)
			}
		}
	}
}

func TestBackendInfoOptionalZeroPresenceRoundTrip(t *testing.T) {
	zero := uint32(0)
	falseValue := false
	want := &Crackstation{
		HIP: []*HIPBackendInfo{{
			Name:                "AMD GPU",
			BackendDeviceID:     1,
			BackendDeviceAlias:  &zero,
			PreferredThreadSize: &zero,
			MemoryUnified:       &falseValue,
		}},
	}

	data, err := proto.Marshal(want)
	if err != nil {
		t.Fatalf("marshal backend info: %v", err)
	}
	got := &Crackstation{}
	if err := proto.Unmarshal(data, got); err != nil {
		t.Fatalf("unmarshal backend info: %v", err)
	}
	if len(got.HIP) != 1 {
		t.Fatalf("HIP devices = %d, want 1", len(got.HIP))
	}
	hip := got.HIP[0]
	if hip.BackendDeviceAlias == nil || *hip.BackendDeviceAlias != 0 {
		t.Fatalf("BackendDeviceAlias = %v, want present zero", hip.BackendDeviceAlias)
	}
	if hip.PreferredThreadSize == nil || *hip.PreferredThreadSize != 0 {
		t.Fatalf("PreferredThreadSize = %v, want present zero", hip.PreferredThreadSize)
	}
	if hip.MemoryUnified == nil || *hip.MemoryUnified {
		t.Fatalf("MemoryUnified = %v, want present false", hip.MemoryUnified)
	}
}
