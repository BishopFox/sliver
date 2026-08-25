//go:build cgo && ((linux && !android && (386 || amd64 || (arm && arm.7) || arm64 || ppc64le || riscv64)) || (freebsd && (amd64 || arm64)))

package memmod

/*
#include <pthread.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

typedef uintptr_t (*reflektor_fn0)(void);
typedef uintptr_t (*reflektor_fn1)(uintptr_t);
typedef uintptr_t (*reflektor_fn2)(uintptr_t, uintptr_t);
typedef uintptr_t (*reflektor_fn3)(uintptr_t, uintptr_t, uintptr_t);
typedef void (*reflektor_void_fn0)(void);
typedef void (*reflektor_init_fn)(int, char **, char **);

static uintptr_t reflektor_call0(uintptr_t fn) {
	return ((reflektor_fn0)fn)();
}

static uintptr_t reflektor_call1(uintptr_t fn, uintptr_t a0) {
	return ((reflektor_fn1)fn)(a0);
}

static uintptr_t reflektor_call2(uintptr_t fn, uintptr_t a0, uintptr_t a1) {
	return ((reflektor_fn2)fn)(a0, a1);
}

static uintptr_t reflektor_call3(uintptr_t fn, uintptr_t a0, uintptr_t a1, uintptr_t a2) {
	return ((reflektor_fn3)fn)(a0, a1, a2);
}

static void reflektor_call_void0(uintptr_t fn) {
	((reflektor_void_fn0)fn)();
}

static void reflektor_call_void3(uintptr_t fn, uintptr_t a0, uintptr_t a1, uintptr_t a2) {
	((reflektor_init_fn)fn)((int)a0, (char **)a1, (char **)a2);
}

typedef struct {
	uintptr_t fn;
} reflektor_thread_call;

static void *reflektor_thread_entry(void *opaque) {
	reflektor_thread_call *call = (reflektor_thread_call *)opaque;
	((reflektor_void_fn0)call->fn)();
	return NULL;
}

static int reflektor_call_on_thread(reflektor_thread_call *call) {
	pthread_t thread;
	int err = pthread_create(&thread, NULL, reflektor_thread_entry, call);
	if (err != 0) {
		return err;
	}
	return pthread_join(thread, NULL);
}

static int reflektor_call_void0_thread(uintptr_t fn) {
	reflektor_thread_call call;
	memset(&call, 0, sizeof(call));
	call.fn = fn;
	return reflektor_call_on_thread(&call);
}

// Go's ELF runtime stores thread state in up to two words of static TLS.
// Reserve independent slots so manually mapped Go runtimes do not alias the
// host Go runtime or each other. Static TLS is instantiated, zeroed, and kept
// at the same thread-pointer-relative offset for every pthread.
#define REFLEKTOR_GO_TLS_SLOTS 64
static __thread uintptr_t reflektor_go_tls_slots[REFLEKTOR_GO_TLS_SLOTS][2]
	__attribute__((tls_model("initial-exec")));

static uintptr_t reflektor_thread_pointer(void) {
#if defined(__powerpc64__)
	// The ELFv2 ABI reserves r13 as the thread pointer. GCC does not expose
	// __builtin_thread_pointer on ppc64le, so read the fixed ABI register.
	register uintptr_t thread_pointer __asm__("r13");
	return thread_pointer;
#else
	return (uintptr_t)__builtin_thread_pointer();
#endif
}

static intptr_t reflektor_go_tls_offset(uintptr_t slot) {
	if (slot >= REFLEKTOR_GO_TLS_SLOTS) {
		return INTPTR_MIN;
	}
	return (intptr_t)(uintptr_t)&reflektor_go_tls_slots[slot][0] -
		(intptr_t)reflektor_thread_pointer();
}

extern char **environ;
static uintptr_t reflektor_empty_init_args[4];

// Build a stable argc/argv/envp/auxv vector for runtimes that expect a
// startup-like layout. argc is zero, argv[0] is NULL, the current environment
// follows it, and a terminating AT_NULL pair follows the environment.
static void reflektor_init_call_args(int include_environment, uintptr_t *argc, uintptr_t *argv, uintptr_t *envp) {
	if (!include_environment) {
		*argc = 0;
		*argv = (uintptr_t)reflektor_empty_init_args;
		*envp = (uintptr_t)(reflektor_empty_init_args + 1);
		return;
	}

	size_t envc = 0;
	while (environ != NULL && environ[envc] != NULL) {
		envc++;
	}
	uintptr_t *vec = (uintptr_t *)calloc(envc + 4, sizeof(uintptr_t));
	if (vec != NULL) {
		for (size_t i = 0; i < envc; i++) {
			vec[i + 1] = (uintptr_t)environ[i];
		}
	}
	*argc = 0;
	*argv = (uintptr_t)vec;
	*envp = vec == NULL ? 0 : (uintptr_t)(vec + 1);
}
*/
import "C"

import "fmt"

func cCall0(fn uintptr) uintptr {
	return uintptr(C.reflektor_call0(C.uintptr_t(fn)))
}

func cCall1(fn, a0 uintptr) uintptr {
	return uintptr(C.reflektor_call1(C.uintptr_t(fn), C.uintptr_t(a0)))
}

func cCall2(fn, a0, a1 uintptr) uintptr {
	return uintptr(C.reflektor_call2(C.uintptr_t(fn), C.uintptr_t(a0), C.uintptr_t(a1)))
}

func cCall3(fn, a0, a1, a2 uintptr) uintptr {
	return uintptr(C.reflektor_call3(C.uintptr_t(fn), C.uintptr_t(a0), C.uintptr_t(a1), C.uintptr_t(a2)))
}

//go:uintptrescapes
func callExportFunction(fn uintptr, args ...uintptr) uintptr {
	switch len(args) {
	case 0:
		return cCall0(fn)
	case 1:
		return cCall1(fn, args[0])
	case 2:
		return cCall2(fn, args[0], args[1])
	case 3:
		return cCall3(fn, args[0], args[1], args[2])
	default:
		panic("validated ELF export argument count is out of range")
	}
}

func cCallVoid0(fn uintptr) {
	C.reflektor_call_void0(C.uintptr_t(fn))
}

func cCallVoid3(fn, a0, a1, a2 uintptr) {
	C.reflektor_call_void3(C.uintptr_t(fn), C.uintptr_t(a0), C.uintptr_t(a1), C.uintptr_t(a2))
}

func cCallVoid0OnThread(fn uintptr) error {
	if errno := int(C.reflektor_call_void0_thread(C.uintptr_t(fn))); errno != 0 {
		return fmt.Errorf("pthread call failed with error %d", errno)
	}
	return nil
}

func linuxGoTLSSlotOffset(slot uintptr) (int64, error) {
	if slot >= 64 {
		return 0, fmt.Errorf("Go TLS slot %d is out of range", slot)
	}
	offset := C.reflektor_go_tls_offset(C.uintptr_t(slot))
	return int64(offset), nil
}

func linuxInitCallArgs(includeEnvironment bool) (uintptr, uintptr, uintptr) {
	var argc C.uintptr_t
	var argv C.uintptr_t
	var envp C.uintptr_t
	include := C.int(0)
	if includeEnvironment {
		include = 1
	}
	C.reflektor_init_call_args(include, &argc, &argv, &envp)
	return uintptr(argc), uintptr(argv), uintptr(envp)
}
