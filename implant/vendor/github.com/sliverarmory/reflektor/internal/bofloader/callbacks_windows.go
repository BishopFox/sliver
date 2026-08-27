//go:build windows && (386 || amd64 || arm64)

package bofloader

import "github.com/ebitengine/purego"

func platformCallbacks() map[string]uintptr {
	return map[string]uintptr{
		"BeaconDataParse": purego.NewCallback(func(_ purego.CDecl, a0, a1, a2 uintptr) uintptr {
			return beaconDataParse(a0, a1, a2)
		}),
		"BeaconDataInt": purego.NewCallback(func(_ purego.CDecl, a0 uintptr) uintptr {
			return beaconDataInt(a0)
		}),
		"BeaconDataShort": purego.NewCallback(func(_ purego.CDecl, a0 uintptr) uintptr {
			return beaconDataShort(a0)
		}),
		"BeaconDataLength": purego.NewCallback(func(_ purego.CDecl, a0 uintptr) uintptr {
			return beaconDataLength(a0)
		}),
		"BeaconDataExtract": purego.NewCallback(func(_ purego.CDecl, a0, a1 uintptr) uintptr {
			return beaconDataExtract(a0, a1)
		}),
		"BeaconDataExtractOrNull": purego.NewCallback(func(_ purego.CDecl, a0, a1 uintptr) uintptr {
			return beaconDataExtractOrNull(a0, a1)
		}),
		"BeaconFormatAlloc": purego.NewCallback(func(_ purego.CDecl, a0, a1 uintptr) uintptr {
			return beaconFormatAlloc(a0, a1)
		}),
		"BeaconFormatReset": purego.NewCallback(func(_ purego.CDecl, a0 uintptr) uintptr {
			return beaconFormatReset(a0)
		}),
		"BeaconFormatFree": purego.NewCallback(func(_ purego.CDecl, a0 uintptr) uintptr {
			return beaconFormatFree(a0)
		}),
		"BeaconFormatAppend": purego.NewCallback(func(_ purego.CDecl, a0, a1, a2 uintptr) uintptr {
			return beaconFormatAppend(a0, a1, a2)
		}),
		"BeaconFormatPrintf": purego.NewCallback(func(_ purego.CDecl, a0, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11 uintptr) uintptr {
			return beaconFormatPrintf(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11)
		}),
		"BeaconFormatToString": purego.NewCallback(func(_ purego.CDecl, a0, a1 uintptr) uintptr {
			return beaconFormatToString(a0, a1)
		}),
		"BeaconFormatInt": purego.NewCallback(func(_ purego.CDecl, a0, a1 uintptr) uintptr {
			return beaconFormatInt(a0, a1)
		}),
		"BeaconPrintf": purego.NewCallback(func(_ purego.CDecl, a0, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11 uintptr) uintptr {
			return beaconPrintf(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11)
		}),
		"BeaconOutput": purego.NewCallback(func(_ purego.CDecl, a0, a1, a2 uintptr) uintptr {
			return beaconOutput(a0, a1, a2)
		}),
		"toWideChar": purego.NewCallback(func(_ purego.CDecl, a0, a1, a2 uintptr) uintptr {
			return toWideChar(a0, a1, a2)
		}),
	}
}
