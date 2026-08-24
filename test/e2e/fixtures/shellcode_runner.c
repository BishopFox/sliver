#define _GNU_SOURCE

#include <errno.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifdef _WIN32
#include <windows.h>
#else
#include <sys/mman.h>
#include <unistd.h>
#endif

static unsigned char *read_payload(const char *path, size_t *payload_size) {
    FILE *payload = fopen(path, "rb");
    unsigned char *data = NULL;
    long file_size = 0;

    if (payload == NULL) {
        fprintf(stderr, "open %s: %s\n", path, strerror(errno));
        return NULL;
    }
    if (fseek(payload, 0, SEEK_END) != 0) {
        fprintf(stderr, "seek %s: %s\n", path, strerror(errno));
        goto fail;
    }
    file_size = ftell(payload);
    if (file_size <= 0) {
        fprintf(stderr, "payload %s is empty or unreadable\n", path);
        goto fail;
    }
    if (fseek(payload, 0, SEEK_SET) != 0) {
        fprintf(stderr, "rewind %s: %s\n", path, strerror(errno));
        goto fail;
    }

    data = (unsigned char *)malloc((size_t)file_size);
    if (data == NULL) {
        fprintf(stderr, "allocate %ld payload bytes failed\n", file_size);
        goto fail;
    }
    if (fread(data, 1, (size_t)file_size, payload) != (size_t)file_size) {
        fprintf(stderr, "read %s: %s\n", path, ferror(payload) ? strerror(errno) : "short read");
        free(data);
        data = NULL;
        goto fail;
    }

    *payload_size = (size_t)file_size;

fail:
    if (fclose(payload) != 0 && data != NULL) {
        fprintf(stderr, "close %s: %s\n", path, strerror(errno));
        free(data);
        data = NULL;
    }
    return data;
}

int main(int argc, char **argv) {
    unsigned char *payload = NULL;
    size_t payload_size = 0;
    int writable_executable = 0;

    if (argc != 3 || (strcmp(argv[2], "rx") != 0 && strcmp(argv[2], "rwx") != 0)) {
        fprintf(stderr, "usage: %s payload.bin rx|rwx\n", argv[0]);
        return 2;
    }
    writable_executable = strcmp(argv[2], "rwx") == 0;
    payload = read_payload(argv[1], &payload_size);
    if (payload == NULL) {
        return 1;
    }

#ifdef _WIN32
    void *executable = VirtualAlloc(NULL, payload_size, MEM_RESERVE | MEM_COMMIT, PAGE_READWRITE);
    DWORD old_protection = 0;
    DWORD executable_protection = writable_executable ? PAGE_EXECUTE_READWRITE : PAGE_EXECUTE_READ;
    HANDLE thread = NULL;

    if (executable == NULL) {
        fprintf(stderr, "VirtualAlloc failed: %lu\n", (unsigned long)GetLastError());
        free(payload);
        return 1;
    }
    memcpy(executable, payload, payload_size);
    free(payload);
    if (!VirtualProtect(executable, payload_size, executable_protection, &old_protection)) {
        fprintf(stderr, "VirtualProtect failed: %lu\n", (unsigned long)GetLastError());
        VirtualFree(executable, 0, MEM_RELEASE);
        return 1;
    }
    if (!FlushInstructionCache(GetCurrentProcess(), executable, payload_size)) {
        fprintf(stderr, "FlushInstructionCache failed: %lu\n", (unsigned long)GetLastError());
        VirtualFree(executable, 0, MEM_RELEASE);
        return 1;
    }
    thread = CreateThread(NULL, 0, (LPTHREAD_START_ROUTINE)executable, NULL, 0, NULL);
    if (thread == NULL) {
        fprintf(stderr, "CreateThread failed: %lu\n", (unsigned long)GetLastError());
        VirtualFree(executable, 0, MEM_RELEASE);
        return 1;
    }
    if (WaitForSingleObject(thread, INFINITE) != WAIT_OBJECT_0) {
        fprintf(stderr, "WaitForSingleObject failed: %lu\n", (unsigned long)GetLastError());
        CloseHandle(thread);
        VirtualFree(executable, 0, MEM_RELEASE);
        return 1;
    }
    fprintf(stderr, "shellcode returned before the runner was stopped\n");
    CloseHandle(thread);
    VirtualFree(executable, 0, MEM_RELEASE);
    return 1;
#else
    void *executable = mmap(NULL, payload_size, PROT_READ | PROT_WRITE, MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
    int executable_protection = PROT_READ | PROT_EXEC | (writable_executable ? PROT_WRITE : 0);
    void (*entrypoint)(void);

    if (executable == MAP_FAILED) {
        fprintf(stderr, "mmap failed: %s\n", strerror(errno));
        free(payload);
        return 1;
    }
    memcpy(executable, payload, payload_size);
    free(payload);
    if (mprotect(executable, payload_size, executable_protection) != 0) {
        fprintf(stderr, "mprotect failed: %s\n", strerror(errno));
        (void)munmap(executable, payload_size);
        return 1;
    }
    __builtin___clear_cache((char *)executable, (char *)executable + payload_size);
    entrypoint = (void (*)(void))executable;
    entrypoint();
    fprintf(stderr, "shellcode returned before the runner was stopped\n");
    (void)munmap(executable, payload_size);
    return 1;
#endif
}
