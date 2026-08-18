//go:build darwin && (amd64 || arm64) && cgo

package memmod

/*
#include <pthread.h>
#include <stdint.h>
#include <string.h>

typedef uintptr_t (*reflektor_fn10_t)(
	uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t,
	uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t
);

typedef void (*reflektor_void_fn0_t)(void);
typedef void *(*reflektor_dlopen_fn_t)(const char *, int);
typedef const char *(*reflektor_dlerror_fn_t)(void);

static void reflektor_call_void0(uintptr_t fn) {
	((reflektor_void_fn0_t)fn)();
}

static uintptr_t reflektor_call_dlopen(uintptr_t fn, uintptr_t name, int flags) {
	return (uintptr_t)((reflektor_dlopen_fn_t)fn)((const char *)name, flags);
}

static uintptr_t reflektor_call_dlerror(uintptr_t fn) {
	return (uintptr_t)((reflektor_dlerror_fn_t)fn)();
}

static uintptr_t reflektor_call10(
	uintptr_t fn,
	uintptr_t a0, uintptr_t a1, uintptr_t a2, uintptr_t a3, uintptr_t a4,
	uintptr_t a5, uintptr_t a6, uintptr_t a7, uintptr_t a8, uintptr_t a9
) {
	return ((reflektor_fn10_t)fn)(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9);
}

typedef struct {
	uintptr_t fn;
} reflektor_void_thread_call;

static void *reflektor_void_thread_entry(void *opaque) {
	reflektor_void_thread_call *call = (reflektor_void_thread_call *)opaque;
	((reflektor_void_fn0_t)call->fn)();
	return NULL;
}

static int reflektor_call_void0_thread(uintptr_t fn) {
	pthread_t thread;
	reflektor_void_thread_call call;
	memset(&call, 0, sizeof(call));
	call.fn = fn;
	int err = pthread_create(&thread, NULL, reflektor_void_thread_entry, &call);
	if (err != 0) {
		return err;
	}
	return pthread_join(thread, NULL);
}
*/
import "C"

import "fmt"

func callVoid0(fn uintptr) {
	C.reflektor_call_void0(C.uintptr_t(fn))
}

func callVoid0OnThread(fn uintptr) error {
	if errno := int(C.reflektor_call_void0_thread(C.uintptr_t(fn))); errno != 0 {
		return fmt.Errorf("pthread call failed with error %d", errno)
	}
	return nil
}

func callDlopen(fn, name uintptr, flags int) uintptr {
	return uintptr(C.reflektor_call_dlopen(C.uintptr_t(fn), C.uintptr_t(name), C.int(flags)))
}

func callDlerror(fn uintptr) uintptr {
	return uintptr(C.reflektor_call_dlerror(C.uintptr_t(fn)))
}

func call0(fn uintptr) uintptr {
	return call10(fn, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
}

func call1(fn, a0 uintptr) uintptr {
	return call10(fn, a0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
}

func call2(fn, a0, a1 uintptr) uintptr {
	return call10(fn, a0, a1, 0, 0, 0, 0, 0, 0, 0, 0)
}

func call4(fn, a0, a1, a2, a3 uintptr) uintptr {
	return call10(fn, a0, a1, a2, a3, 0, 0, 0, 0, 0, 0)
}

func call6(fn, a0, a1, a2, a3, a4, a5 uintptr) uintptr {
	return call10(fn, a0, a1, a2, a3, a4, a5, 0, 0, 0, 0)
}

func call10(fn, a0, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr) uintptr {
	return uintptr(C.reflektor_call10(
		C.uintptr_t(fn),
		C.uintptr_t(a0),
		C.uintptr_t(a1),
		C.uintptr_t(a2),
		C.uintptr_t(a3),
		C.uintptr_t(a4),
		C.uintptr_t(a5),
		C.uintptr_t(a6),
		C.uintptr_t(a7),
		C.uintptr_t(a8),
		C.uintptr_t(a9),
	))
}
