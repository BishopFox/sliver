#pragma once

#include <windows.h>
#include <ntsecapi.h>

/* ============================================================
 * MSV1_0 Structures - Credential Key & NTLM operations
 * ============================================================ */

#ifndef MSV1_0_PACKAGE_NAME
#define MSV1_0_PACKAGE_NAME "MICROSOFT_AUTHENTICATION_PACKAGE_V1_0"
#endif

/* MSV1_0 protocol message types */
typedef enum _MSV1_0_PROTOCOL_MESSAGE_TYPE_CUSTOM {
    MsV1_0Lm20ChallengeRequest = 0,
    MsV1_0Lm20GetChallengeResponse,
    MsV1_0EnumerateUsers,
    MsV1_0GetUserInfo,
    MsV1_0ReLogonUsers,
    MsV1_0ChangePassword,
    MsV1_0ChangeCachedPassword,
    MsV1_0GenericPassthrough,
    MsV1_0CacheLogon,
    MsV1_0SubAuth,
    MsV1_0DeriveCredential,
    MsV1_0CacheLookup,
    MsV1_0SetProcessOption,
    MsV1_0ConfigLocalAliases,
    MsV1_0ClearCachedCredentials,
    MsV1_0LookupToken,
    MsV1_0ValidateAuth,
    MsV1_0CacheLookupEx,
    MsV1_0GetCredentialKey,
    MsV1_0SetThreadOption,
    MsV1_0DecryptDpapiMasterKey,
    MsV1_0GetStrongCredentialKey,
    MsV1_0TransferCred,
    MsV1_0ProvisionTbal,
    MsV1_0DeleteTbalSecrets
} MSV1_0_PROTOCOL_MESSAGE_TYPE_CUSTOM;

/* Request for GetCredentialKey / GetStrongCredentialKey */
typedef struct _MSV1_0_CREDENTIAL_KEY_REQUEST {
    MSV1_0_PROTOCOL_MESSAGE_TYPE_CUSTOM MessageType;
    LUID LogonId;
} MSV1_0_CREDENTIAL_KEY_REQUEST, *PMSV1_0_CREDENTIAL_KEY_REQUEST;

/* Response for credential key operations */
typedef struct _MSV1_0_CREDENTIAL_KEY_RESPONSE {
    MSV1_0_PROTOCOL_MESSAGE_TYPE_CUSTOM MessageType;
    ULONG KeyLength;
    UCHAR Key[1]; /* Variable length */
} MSV1_0_CREDENTIAL_KEY_RESPONSE, *PMSV1_0_CREDENTIAL_KEY_RESPONSE;

/* Challenge request for NTLMv1 */
typedef struct _MSV1_0_LM20_CHALLENGE_REQUEST {
    MSV1_0_PROTOCOL_MESSAGE_TYPE_CUSTOM MessageType;
} MSV1_0_LM20_CHALLENGE_REQUEST, *PMSV1_0_LM20_CHALLENGE_REQUEST;

/* Challenge response from LSA */
typedef struct _MSV1_0_LM20_CHALLENGE_RESPONSE_OUT {
    MSV1_0_PROTOCOL_MESSAGE_TYPE_CUSTOM MessageType;
    UCHAR ChallengeToClient[8];
} MSV1_0_LM20_CHALLENGE_RESPONSE_OUT, *PMSV1_0_LM20_CHALLENGE_RESPONSE_OUT;

/* NTLMv1 GetChallengeResponse request */
#define RTL_ENCRYPT_MEMORY_SIZE     8
#define RETURN_NT_RESPONSE_ONLY     0x00000002
#define USE_SUPPLIED_CHALLENGE      0x00000008
#define RETURN_NON_NT_RESPONSE      0x00000010
#define GCR_NTLM3_PARMS            0x00000020
#define GCR_ALLOW_NTLM             0x00000100
#define RETURN_PRIMARY_USERNAME     0x00000400

typedef struct _MSV1_0_GETCHALLENRESP_REQUEST {
    MSV1_0_PROTOCOL_MESSAGE_TYPE_CUSTOM MessageType;
    ULONG ParameterControl;
    LUID LogonId;
    UCHAR ChallengeToClient[8];
    UNICODE_STRING UserName;
    UNICODE_STRING LogonDomainName;
    UNICODE_STRING ServerName;
} MSV1_0_GETCHALLENRESP_REQUEST, *PMSV1_0_GETCHALLENRESP_REQUEST;

typedef struct _MSV1_0_GETCHALLENRESP_RESPONSE {
    MSV1_0_PROTOCOL_MESSAGE_TYPE_CUSTOM MessageType;
    STRING CaseSensitiveChallengeResponse;
    STRING CaseInsensitiveChallengeResponse;
    UNICODE_STRING UserName;
    UNICODE_STRING LogonDomainName;
    UCHAR UserSessionKey[16];
    UCHAR LanmanSessionKey[8];
} MSV1_0_GETCHALLENRESP_RESPONSE, *PMSV1_0_GETCHALLENRESP_RESPONSE;


/* ============================================================
 * Kerberos Structures - Ticket operations
 * ============================================================ */

#ifndef MICROSOFT_KERBEROS_NAME_A
#define MICROSOFT_KERBEROS_NAME_A "Kerberos"
#endif

/* Kerberos protocol message types */
typedef enum _KERB_PROTOCOL_MESSAGE_TYPE_CUSTOM {
    KerbDebugRequestMessage = 0,
    KerbQueryTicketCacheMessage,
    KerbChangeMachinePasswordMessage,
    KerbVerifyPacMessage,
    KerbRetrieveTicketMessage,
    KerbUpdateAddressesMessage,
    KerbPurgeTicketCacheMessage,
    KerbChangePasswordMessage,
    KerbRetrieveEncodedTicketMessage,
    KerbDecryptDataMessage,
    KerbAddBindingCacheEntryMessage,
    KerbSetPasswordMessage,
    KerbSetPasswordExMessage,
    KerbVerifyCredentialsMessage,
    KerbQueryTicketCacheExMessage,
    KerbPurgeTicketCacheExMessage,
    KerbRefreshSmartcardCredentialsMessage,
    KerbAddExtraCredentialsMessage,
    KerbQuerySupplementalCredentialsMessage,
    KerbTransferCredentialsMessage,
    KerbQueryTicketCacheEx2Message,
    KerbSubmitTicketMessage,
    KerbAddExtraCredentialsExMessage,
    KerbQueryKdcProxyCacheMessage,
    KerbPurgeKdcProxyCacheMessage,
    KerbQueryTicketCacheEx3Message,
    KerbCleanupMachinePkinitCredsMessage,
    KerbAddBindingCacheEntryExMessage,
    KerbQueryBindingCacheMessage,
    KerbPurgeBindingCacheMessage,
    KerbPinKdcMessage,
    KerbUnpinAllKdcsMessage,
    KerbQueryDomainExtendedPoliciesMessage,
    KerbQueryS4U2ProxyCacheMessage,
    KerbRetrieveKeyTabMessage
} KERB_PROTOCOL_MESSAGE_TYPE_CUSTOM;

/* Kerberos encryption types */
#define KERB_ETYPE_DES_CBC_CRC          1
#define KERB_ETYPE_DES_CBC_MD4          2
#define KERB_ETYPE_DES_CBC_MD5          3
#define KERB_ETYPE_AES128_CTS_HMAC_SHA1 17
#define KERB_ETYPE_AES256_CTS_HMAC_SHA1 18
#define KERB_ETYPE_RC4_HMAC_NT         23
#define KERB_ETYPE_RC4_HMAC_NT_EXP     24

/* Ticket flags */
#define KERB_TICKET_FLAGS_reserved          0x80000000
#define KERB_TICKET_FLAGS_forwardable       0x40000000
#define KERB_TICKET_FLAGS_forwarded         0x20000000
#define KERB_TICKET_FLAGS_proxiable         0x10000000
#define KERB_TICKET_FLAGS_proxy             0x08000000
#define KERB_TICKET_FLAGS_may_postdate      0x04000000
#define KERB_TICKET_FLAGS_postdated         0x02000000
#define KERB_TICKET_FLAGS_invalid           0x01000000
#define KERB_TICKET_FLAGS_renewable         0x00800000
#define KERB_TICKET_FLAGS_initial           0x00400000
#define KERB_TICKET_FLAGS_pre_authent       0x00200000
#define KERB_TICKET_FLAGS_hw_authent        0x00100000
#define KERB_TICKET_FLAGS_ok_as_delegate    0x00040000
#define KERB_TICKET_FLAGS_name_canonicalize 0x00010000
#define KERB_TICKET_FLAGS_enc_pa_rep        0x00010000
#define KERB_TICKET_FLAGS_reserved1         0x00000001

/* Query ticket cache request */
typedef struct _KERB_QUERY_TKT_CACHE_REQUEST {
    KERB_PROTOCOL_MESSAGE_TYPE_CUSTOM MessageType;
    LUID LogonId;
} KERB_QUERY_TKT_CACHE_REQUEST, *PKERB_QUERY_TKT_CACHE_REQUEST;

/* Extended ticket cache entry */
typedef struct _KERB_TICKET_CACHE_INFO_EX {
    UNICODE_STRING ClientName;
    UNICODE_STRING ClientRealm;
    UNICODE_STRING ServerName;
    UNICODE_STRING ServerRealm;
    LARGE_INTEGER  StartTime;
    LARGE_INTEGER  EndTime;
    LARGE_INTEGER  RenewTime;
    LONG           EncryptionType;
    ULONG          TicketFlags;
} KERB_TICKET_CACHE_INFO_EX, *PKERB_TICKET_CACHE_INFO_EX;

/* Query ticket cache response */
typedef struct _KERB_QUERY_TKT_CACHE_EX_RESPONSE {
    KERB_PROTOCOL_MESSAGE_TYPE_CUSTOM MessageType;
    ULONG CountOfTickets;
    KERB_TICKET_CACHE_INFO_EX Tickets[1]; /* Variable length */
} KERB_QUERY_TKT_CACHE_EX_RESPONSE, *PKERB_QUERY_TKT_CACHE_EX_RESPONSE;

/* Retrieve ticket request */
typedef struct _KERB_RETRIEVE_TKT_REQUEST {
    KERB_PROTOCOL_MESSAGE_TYPE_CUSTOM MessageType;
    LUID LogonId;
    UNICODE_STRING TargetName;
    ULONG TicketFlags;
    ULONG CacheOptions;
    LONG  EncryptionType;
    SecHandle CredentialsHandle;
} KERB_RETRIEVE_TKT_REQUEST, *PKERB_RETRIEVE_TKT_CACHE_REQUEST;

/* External ticket structure */
typedef struct _KERB_EXTERNAL_NAME {
    SHORT NameType;
    USHORT NameCount;
    UNICODE_STRING Names[1]; /* Variable */
} KERB_EXTERNAL_NAME, *PKERB_EXTERNAL_NAME;

typedef struct _KERB_EXTERNAL_TICKET {
    PKERB_EXTERNAL_NAME ServiceName;
    PKERB_EXTERNAL_NAME TargetName;
    PKERB_EXTERNAL_NAME ClientName;
    UNICODE_STRING      DomainName;
    UNICODE_STRING      TargetDomainName;
    UNICODE_STRING      AltTargetDomainName;
    KERB_CRYPTO_KEY     SessionKey;
    ULONG               TicketFlags;
    ULONG               Flags;
    LARGE_INTEGER       KeyExpirationTime;
    LARGE_INTEGER       StartTime;
    LARGE_INTEGER       EndTime;
    LARGE_INTEGER       RenewUntil;
    LARGE_INTEGER       TimeSkew;
    ULONG               EncodedTicketSize;
    PUCHAR              EncodedTicket;
} KERB_EXTERNAL_TICKET, *PKERB_EXTERNAL_TICKET;

/* Retrieve ticket response */
typedef struct _KERB_RETRIEVE_TKT_RESPONSE {
    KERB_PROTOCOL_MESSAGE_TYPE_CUSTOM MessageType;
    KERB_EXTERNAL_TICKET Ticket;
} KERB_RETRIEVE_TKT_RESPONSE, *PKERB_RETRIEVE_TKT_RESPONSE;

/* Purge ticket cache request */
typedef struct _KERB_PURGE_TKT_CACHE_REQUEST {
    KERB_PROTOCOL_MESSAGE_TYPE_CUSTOM MessageType;
    LUID LogonId;
    UNICODE_STRING ServerName;
    UNICODE_STRING RealmName;
} KERB_PURGE_TKT_CACHE_REQUEST, *PKERB_PURGE_TKT_CACHE_REQUEST;

/* Cache options */
#define KERB_RETRIEVE_TICKET_AS_KERB_CRED  0x8
#define KERB_RETRIEVE_TICKET_CACHE_TICKET  0x1


/* ============================================================
 * CloudAP Structures - Azure/Entra ID operations
 * ============================================================ */

#define CLOUDAP_NAME_A "CloudAP"

/* CloudAP protocol message types */
typedef enum _CLOUDAP_PROTOCOL_MESSAGE_TYPE {
    CloudApGetUnlockKeyType = 0,
    CloudApPluginUnlockKeyType,
    CloudApGetPrtTokenType,
    CloudApGetSsoCookieType,
    CloudApRenewPrtType,
    CloudApGetDeviceSsoCookieType,
    CloudApGetEnterpriseSsoCookieType,
    CloudApGetProviderInfoType,
    CloudApDecryptDpapiMasterKeyType,
    CloudApDisableOptimizedLogonType
} CLOUDAP_PROTOCOL_MESSAGE_TYPE;

/* Generic CloudAP call request */
typedef struct _CLOUDAP_CALL_PACKAGE_GENERIC_REQUEST {
    ULONG MessageType;
    ULONG CallType;
    LUID  LogonId;
} CLOUDAP_CALL_PACKAGE_GENERIC_REQUEST, *PCLOUDAP_CALL_PACKAGE_GENERIC_REQUEST;

/* SSO cookie request */
typedef struct _CLOUDAP_GET_SSO_COOKIE_REQUEST {
    CLOUDAP_PROTOCOL_MESSAGE_TYPE MessageType;
    LUID LogonId;
} CLOUDAP_GET_SSO_COOKIE_REQUEST, *PCLOUDAP_GET_SSO_COOKIE_REQUEST;

/* SSO cookie response */
typedef struct _CLOUDAP_GET_SSO_COOKIE_RESPONSE {
    CLOUDAP_PROTOCOL_MESSAGE_TYPE MessageType;
    ULONG CookieLength;
    WCHAR Cookie[1]; /* Variable length */
} CLOUDAP_GET_SSO_COOKIE_RESPONSE, *PCLOUDAP_GET_SSO_COOKIE_RESPONSE;

/* Provider info request */
typedef struct _CLOUDAP_GET_PROVIDER_INFO_REQUEST {
    CLOUDAP_PROTOCOL_MESSAGE_TYPE MessageType;
    LUID LogonId;
} CLOUDAP_GET_PROVIDER_INFO_REQUEST, *PCLOUDAP_GET_PROVIDER_INFO_REQUEST;

/* Provider info response */
typedef struct _CLOUDAP_GET_PROVIDER_INFO_RESPONSE {
    CLOUDAP_PROTOCOL_MESSAGE_TYPE MessageType;
    ULONG InfoLength;
    WCHAR Info[1]; /* Variable length */
} CLOUDAP_GET_PROVIDER_INFO_RESPONSE, *PCLOUDAP_GET_PROVIDER_INFO_RESPONSE;


/* ============================================================
 * Base64 encoding table (shared)
 * ============================================================ */
static const char b64_table[] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
