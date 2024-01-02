// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package purego

// Constants as defined in https://github.com/freebsd/freebsd-src/blob/main/include/dlfcn.h
const (
	RTLD_DEFAULT = ^uintptr(0) - 2 // Pseudo-handle for dlsym so search for any loaded symbol
	RTLD_LAZY    = 0x00001         // Relocations are performed at an implementation-dependent time.
	RTLD_NOW     = 0x00002         // Relocations are performed when the object is loaded.
	RTLD_LOCAL   = 0x00000         // All symbols are not made available for relocation processing by other modules.
	RTLD_GLOBAL  = 0x00100         // All symbols are available for relocation processing of other modules.
)

//go:cgo_import_dynamic purego_dlopen dlopen "libc.so.7"
//go:cgo_import_dynamic purego_dlsym dlsym "libc.so.7"
//go:cgo_import_dynamic purego_dlerror dlerror "libc.so.7"
//go:cgo_import_dynamic purego_dlclose dlclose "libc.so.7"
