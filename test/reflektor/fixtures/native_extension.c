#if defined(_WIN32)
#define SLIVER_EXPORT __declspec(dllexport)
#define SLIVER_CALL __cdecl
#if defined(__i386__)
#define SLIVER_CALLBACK __attribute__((stdcall))
#define SLIVER_ENTRY __attribute__((stdcall))
#else
#define SLIVER_CALLBACK
#define SLIVER_ENTRY
#endif
#else
#define SLIVER_EXPORT __attribute__((visibility("default")))
#define SLIVER_CALL
#define SLIVER_CALLBACK
#endif

typedef __UINT32_TYPE__ sliver_uint32;
typedef __INT32_TYPE__ sliver_int32;
typedef sliver_int32(SLIVER_CALLBACK *sliver_callback)(char *, sliver_int32);

typedef char sliver_uint32_must_be_four_bytes[(sizeof(sliver_uint32) == 4) ? 1 : -1];
typedef char sliver_int32_must_be_four_bytes[(sizeof(sliver_int32) == 4) ? 1 : -1];

static int initialized;
static char output[256];

#if defined(_WIN32)
/* Keep the fixture independent of the CRT so Reflektor only maps this DLL. */
int SLIVER_ENTRY _DllMainCRTStartup(void *module, unsigned long reason, void *reserved) {
    (void)module;
    (void)reason;
    (void)reserved;
    return 1;
}
#endif

/* Registration calls this export before the extension is added to the implant. */
SLIVER_EXPORT sliver_int32 SLIVER_CALL Initialize(void) {
    initialized = 1;
    return 0;
}

/* Sliver's native extension ABI is (buffer, uint32 length, callback(char *, int)). */
SLIVER_EXPORT sliver_int32 SLIVER_CALL Echo(
    char *buffer,
    sliver_uint32 buffer_size,
    sliver_callback callback
) {
    static const char initialized_prefix[] = "initialized:";
    static const char uninitialized_prefix[] = "uninitialized:";
    const char *prefix = initialized ? initialized_prefix : uninitialized_prefix;
    sliver_uint32 prefix_size = initialized
        ? sizeof(initialized_prefix) - 1
        : sizeof(uninitialized_prefix) - 1;
    sliver_uint32 index;

    if (callback == 0 || (buffer == 0 && buffer_size != 0)) {
        return 1;
    }
    if (prefix_size + buffer_size > sizeof(output)) {
        return 2;
    }

    for (index = 0; index < prefix_size; index++) {
        output[index] = prefix[index];
    }
    for (index = 0; index < buffer_size; index++) {
        output[prefix_size + index] = buffer[index];
    }

    return callback(output, (sliver_int32)(prefix_size + buffer_size));
}
