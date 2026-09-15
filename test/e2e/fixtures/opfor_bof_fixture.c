#include <stdint.h>

#define BEACON_IMPORT __declspec(dllimport)

typedef struct {
    char *original;
    char *buffer;
    int32_t length;
    int32_t size;
} datap;

BEACON_IMPORT void BeaconDataParse(datap *parser, char *buffer, int32_t size);
BEACON_IMPORT int32_t BeaconDataInt(datap *parser);
BEACON_IMPORT void BeaconOutput(int32_t type, char *data, int32_t length);
BEACON_IMPORT void __stdcall KERNEL32$Sleep(uint32_t milliseconds);

void go(char *buffer, int32_t length) {
    static char alpha[] = "alpha";
    static char beta[] = "beta";
    static char before_error[] = "before-error";
    static char before_timeout[] = "before-timeout";
    static char after_timeout[] = "after-timeout";
    static unsigned char binary[] = {0x41, 0x00, 0xff, 0x42};
    static unsigned char utf8[] = {'s', 'n', 'o', 'w', ':', 0xe2, 0x98, 0x83};
    datap parser = {0};
    int32_t mode;
    int32_t sleep_milliseconds;

    BeaconDataParse(&parser, buffer, length);
    mode = BeaconDataInt(&parser);
    sleep_milliseconds = BeaconDataInt(&parser);

    switch (mode) {
    case 0:
        BeaconOutput(0x00, alpha, (int32_t)(sizeof(alpha) - 1));
        BeaconOutput(0x0d, beta, (int32_t)(sizeof(beta) - 1));
        BeaconOutput(0x1e, (char *)binary, (int32_t)sizeof(binary));
        BeaconOutput(0x20, (char *)utf8, (int32_t)sizeof(utf8));
        BeaconOutput(0x7f, (char *)0, 0);
        return;
    case 1:
        BeaconOutput(0x00, before_error, (int32_t)(sizeof(before_error) - 1));
        BeaconOutput(0x00, (char *)0, 1);
        return;
    case 2:
        BeaconOutput(0x00, before_timeout, (int32_t)(sizeof(before_timeout) - 1));
        KERNEL32$Sleep((uint32_t)sleep_milliseconds);
        BeaconOutput(0x00, after_timeout, (int32_t)(sizeof(after_timeout) - 1));
        return;
    default:
        BeaconOutput(0x0d, (char *)"invalid-mode", 12);
        return;
    }
}
