//go:build linux && cgo && (386 || amd64 || arm64)

package memmod

/*
#include <stdint.h>
#include <stdlib.h>

typedef uintptr_t (*reflektor_fn0)(void);
typedef uintptr_t (*reflektor_fn1)(uintptr_t);
typedef uintptr_t (*reflektor_fn2)(uintptr_t, uintptr_t);
typedef uintptr_t (*reflektor_fn3)(uintptr_t, uintptr_t, uintptr_t);

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

static uintptr_t reflektor_init_argc = 0;
static uintptr_t reflektor_init_argv = 0;
static uintptr_t reflektor_init_envp = 0;

// Build a tiny synthetic argv/envp/auxv vector for runtimes that expect a
// startup-like layout (argc/argv/envp/auxv). The layout is:
//   argv[0] = NULL
//   envp[0] = NULL
//   auxv[0] = AT_NULL
//   auxv[1] = 0
static void reflektor_init_call_args(uintptr_t *argc, uintptr_t *argv, uintptr_t *envp) {
	if (reflektor_init_argv == 0) {
		uintptr_t *vec = (uintptr_t *)calloc(4, sizeof(uintptr_t));
		if (vec != NULL) {
			reflektor_init_argv = (uintptr_t)vec;
			reflektor_init_envp = (uintptr_t)(vec + 1);
		}
	}
	*argc = reflektor_init_argc;
	*argv = reflektor_init_argv;
	*envp = reflektor_init_envp;
}
*/
import "C"

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

func linuxInitCallArgs() (uintptr, uintptr, uintptr) {
	var argc C.uintptr_t
	var argv C.uintptr_t
	var envp C.uintptr_t
	C.reflektor_init_call_args(&argc, &argv, &envp)
	return uintptr(argc), uintptr(argv), uintptr(envp)
}
