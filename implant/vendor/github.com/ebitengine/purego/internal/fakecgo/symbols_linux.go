// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package fakecgo

// pthread_attr_init will get us the wrong version on glibc - but this doesn't matter, since the memory we
// provide is zeroed - which will lead the correct result again

//go:cgo_import_dynamic purego_pthread_attr_init pthread_attr_init "libpthread.so.0"
//go:cgo_import_dynamic purego_pthread_attr_getstacksize pthread_attr_getstacksize "libpthread.so.0"
//go:cgo_import_dynamic purego_pthread_attr_destroy pthread_attr_destroy "libpthread.so.0"
//go:cgo_import_dynamic purego_pthread_sigmask pthread_sigmask "libpthread.so.0"
//go:cgo_import_dynamic purego_pthread_create pthread_create "libpthread.so.0"
//go:cgo_import_dynamic purego_pthread_detach pthread_detach "libpthread.so.0"
//go:cgo_import_dynamic purego_setenv setenv "libc.so.6"
//go:cgo_import_dynamic purego_unsetenv unsetenv "libc.so.6"
//go:cgo_import_dynamic purego_malloc malloc "libc.so.6"
//go:cgo_import_dynamic purego_free free "libc.so.6"
//go:cgo_import_dynamic purego_nanosleep nanosleep "libc.so.6"
//go:cgo_import_dynamic purego_sigfillset sigfillset "libc.so.6"
//go:cgo_import_dynamic purego_abort abort "libc.so.6"

// on amd64 we don't need the following lines - on 386 we do...
// anyway - with those lines the output is better (but doesn't matter) - without it on amd64 we get multiple DT_NEEDED with "libc.so.6" etc

//go:cgo_import_dynamic _ _ "libpthread.so.0"
//go:cgo_import_dynamic _ _ "libc.so.6"
