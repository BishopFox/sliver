//go:build (darwin && !ios && (amd64 || arm64)) || (freebsd && (amd64 || arm64)) || (linux && !android && (386 || amd64 || (arm && arm.7) || arm64 || ppc64le || riscv64))

package bofloader

import "github.com/ebitengine/purego"

func platformCallbacks() map[string]uintptr {
	return map[string]uintptr{
		"BeaconDataParse":         purego.NewCallback(beaconDataParse),
		"BeaconDataInt":           purego.NewCallback(beaconDataInt),
		"BeaconDataShort":         purego.NewCallback(beaconDataShort),
		"BeaconDataLength":        purego.NewCallback(beaconDataLength),
		"BeaconDataExtract":       purego.NewCallback(beaconDataExtract),
		"BeaconDataExtractOrNull": purego.NewCallback(beaconDataExtractOrNull),
		"BeaconFormatAlloc":       purego.NewCallback(beaconFormatAlloc),
		"BeaconFormatReset":       purego.NewCallback(beaconFormatReset),
		"BeaconFormatFree":        purego.NewCallback(beaconFormatFree),
		"BeaconFormatAppend":      purego.NewCallback(beaconFormatAppend),
		"BeaconFormatPrintf":      purego.NewCallback(beaconFormatPrintf),
		"BeaconFormatToString":    purego.NewCallback(beaconFormatToString),
		"BeaconFormatInt":         purego.NewCallback(beaconFormatInt),
		"BeaconPrintf":            purego.NewCallback(beaconPrintf),
		"BeaconOutput":            purego.NewCallback(beaconOutput),
		"toWideChar":              purego.NewCallback(toWideChar),
	}
}
